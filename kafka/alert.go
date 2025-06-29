package kafka

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

func (k *Kafka) SendAlert(message *sarama.ConsumerMessage, err error) error {
	if !k.config.WithAlert || k.config.TelegramNotifier == nil {
		return nil
	}

	dlqMsg := k.constructDLQMessage(message, err)

	// Send alert to telegram
	msg := formatDLQMessage(dlqMsg)
	if err := k.config.TelegramNotifier.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send alert to telegram: %w", err)
	}

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
