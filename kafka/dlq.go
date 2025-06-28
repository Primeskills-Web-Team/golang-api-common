package kafka

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// DLQMessage represents a message stored in the dead letter queue
type DLQMessage struct {
	ConsumedBy    string            `json:"consumed_by"`
	OriginalTopic string            `json:"original_topic"`
	Partition     int32             `json:"partition"`
	Offset        int64             `json:"offset"`
	Key           []byte            `json:"key"`
	Value         []byte            `json:"value"`
	Headers       map[string]string `json:"headers"`
	Error         string            `json:"error"`
	FailedAt      time.Time         `json:"failed_at"`
	Timestamp     time.Time         `json:"timestamp"`
}

// SendToDLQ sends a message to the dead letter queue
func (c *Kafka) SendToDLQ(originalMsg *sarama.ConsumerMessage, processingError error) error {
	if !c.config.WithDlq || c.config.Storage == nil {
		return nil
	}

	// Convert headers to map for JSON serialization
	headers := make(map[string]string)
	for _, header := range originalMsg.Headers {
		if header != nil {
			headers[string(header.Key)] = string(header.Value)
		}
	}

	// Create DLQ message
	dlqMsg := DLQMessage{
		ConsumedBy:    c.config.AppName,
		OriginalTopic: originalMsg.Topic,
		Partition:     originalMsg.Partition,
		Offset:        originalMsg.Offset,
		Key:           originalMsg.Key,
		Value:         originalMsg.Value,
		Headers:       headers,
		Error:         processingError.Error(),
		FailedAt:      time.Now(),
		Timestamp:     originalMsg.Timestamp,
	}

	// Serialize to JSON
	msgBytes, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	// Generate unique key for storage
	dlqKey := fmt.Sprintf("dlq:%s:%s", c.config.AppName, uuid.New().String())

	// Store in configured storage (no expiration)
	if err := (*c.config.Storage).Set(dlqKey, msgBytes, 0); err != nil {
		return fmt.Errorf("failed to store message in DLQ: %w", err)
	}

	// Send alert to telegram
	if c.config.WithAlert && c.config.TelegramNotifier != nil {
		message := formatDLQMessage(&dlqMsg)
		if err := c.config.TelegramNotifier.SendMessage(message); err != nil {
			log.Error().Err(err).Msg("Failed to send alert to telegram")
		}
	}

	log.Info().
		Str("dlq_key", dlqKey).
		Str("original_topic", originalMsg.Topic).
		Int32("partition", originalMsg.Partition).
		Int64("offset", originalMsg.Offset).
		Str("error", processingError.Error()).
		Msg("Message sent to DLQ")

	return nil
}

// GetDLQMessage retrieves a DLQ message by key from storage
func (c *Kafka) GetDLQMessage(dlqKey string) (*DLQMessage, error) {
	if c.config.Storage == nil {
		return nil, fmt.Errorf("no storage configured")
	}

	msgBytes, err := (*c.config.Storage).Get(dlqKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ message: %w", err)
	}
	if msgBytes == nil {
		return nil, nil // Message not found
	}

	var dlqMsg DLQMessage
	if err := json.Unmarshal(msgBytes, &dlqMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ message: %w", err)
	}

	return &dlqMsg, nil
}

// DeleteDLQMessage removes a DLQ message from storage
func (c *Kafka) DeleteDLQMessage(dlqKey string) error {
	if c.config.Storage == nil {
		return fmt.Errorf("no storage configured")
	}

	if err := (*c.config.Storage).Delete(dlqKey); err != nil {
		return fmt.Errorf("failed to delete DLQ message: %w", err)
	}

	log.Info().
		Str("dlq_key", dlqKey).
		Msg("DLQ message deleted")

	return nil
}

// prettifyJSON formats JSON bytes into a pretty-printed string
func prettifyJSON(data []byte) string {
	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		// If it's not valid JSON, return as is
		return string(data)
	}

	prettyBytes, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		// If prettifying fails, return original
		return string(data)
	}

	return string(prettyBytes)
}

// escapeMarkdownV2 escapes special characters for Telegram's MarkdownV2 format
func escapeMarkdownV2(text string) string {
	// Characters that need to be escaped in MarkdownV2
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

func formatDLQMessage(dlqMsg *DLQMessage) string {
	// Truncate error message if too long
	errorMsg := dlqMsg.Error
	if len(errorMsg) > 200 {
		errorMsg = errorMsg[:200] + "..."
	}

	// Escape all text content for Telegram MarkdownV2
	escapedTopic := escapeMarkdownV2(dlqMsg.OriginalTopic)
	escapedError := escapeMarkdownV2(errorMsg)
	escapedFailedAt := escapeMarkdownV2(dlqMsg.FailedAt.Format(time.RFC3339))
	escapedTimestamp := escapeMarkdownV2(dlqMsg.Timestamp.Format(time.RFC3339))
	escapedConsumedBy := escapeMarkdownV2(dlqMsg.ConsumedBy)

	// Prettify JSON value
	prettyValue := prettifyJSON(dlqMsg.Value)
	escapedValue := escapeMarkdownV2(prettyValue)

	// Format headers as escaped text
	headersStr := ""
	for k, v := range dlqMsg.Headers {
		headersStr += fmt.Sprintf("%s: %s\n", escapeMarkdownV2(k), escapeMarkdownV2(v))
	}
	if headersStr == "" {
		headersStr = "None"
	}

	return fmt.Sprintf("*Topic:* %s\n*Partition:* %d\n*Offset:* %d\n*Error:* %s\n*Failed At:* %s\n*Timestamp:* %s\n*Consumed By:* %s\n\n*Headers:*\n%s\n*Value:*\n```json\n%s\n```",
		escapedTopic,
		dlqMsg.Partition,
		dlqMsg.Offset,
		escapedError,
		escapedFailedAt,
		escapedTimestamp,
		escapedConsumedBy,
		headersStr,
		escapedValue,
	)
}
