package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	kafkaproducer "github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/producer"
	"github.com/sirupsen/logrus"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka/helpers"

)

func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxRetries:    3,
        RetryInterval: time.Second * 2,
        BackoffFactor: 2.0,
    }
}

func (k *KafkaConfig) PublishEventWithRetry(ctx context.Context, topic string, value kafka.Event, retryConfig RetryConfig) (*PublishResult, error) {
    var lastErr error

    for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
        if attempt > 0 {
            delay := time.Duration(float64(retryConfig.RetryInterval) *
                math.Pow(retryConfig.BackoffFactor, float64(attempt-1)))

            logrus.Warnf("Retrying publish to topic %s, attempt %d/%d after %v",
                topic, attempt, retryConfig.MaxRetries, delay)

            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(delay):
            }
        }

        result, err := k.publishEventOnce(topic, value)
        if err == nil {
            if attempt > 0 {
                logrus.Infof("Successfully published to topic %s after %d retries", topic, attempt)
            }
            return result, nil
        }

        lastErr = err
        logrus.Errorf("Attempt %d failed to publish to topic %s: %v", attempt+1, topic, err)

        if helpers.IsConnectionError(err) {
            k.resetProducer()
        }
    }

    return nil, fmt.Errorf("failed to publish after %d attempts, last error: %w", retryConfig.MaxRetries+1, lastErr)
}

func (k *KafkaConfig) publishEventOnce(topic string, value kafka.Event) (*PublishResult, error) {
	syncProducer, err := k.getOrCreateProducer()
	if err != nil {
		return nil, fmt.Errorf("unable to get kafka producer: %w", err)
	}

	kafkaProducer := &kafkaproducer.KafkaProducer{
		Producer: syncProducer,
	}

	msg, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	partition, offset, err := kafkaProducer.SendMessageSync(topic, string(msg))
	if err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, fmt.Errorf("failed to send message: %w", err)
	}

	return &PublishResult{
		Success:   true,
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now(),
	}, nil
}

func (k *KafkaConfig) PublishEventWithCallback(
	ctx context.Context,
	topic string,
	value kafka.Event,
	onSuccess func(*PublishResult),
	onFailure func(error),
) {
	go func() {
		result, err := k.PublishEventWithRetry(ctx, topic, value, DefaultRetryConfig())

		if err != nil {

			logrus.Errorf("Final failure to publish event to topic %s: %v", topic, err)
			k.handlePublishFailure(topic, value, err)

			if onFailure != nil {
				onFailure(err)
			}
		} else {
			if onSuccess != nil {
				onSuccess(result)
			}
		}
	}()
}

func (k *KafkaConfig) PublishEvent(topic string, value kafka.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := k.PublishEventWithRetry(ctx, topic, value, DefaultRetryConfig())
	if err != nil {
		logrus.Errorf("Final failure to publish event to topic %s: %v", topic, err)
		k.handlePublishFailure(topic, value, err)
		return
	}

	logrus.Infof("Successfully published event to topic %s, partition %d, offset %d",
		topic, result.Partition, result.Offset)
}

func (k *KafkaConfig) handlePublishFailure(topic string, value kafka.Event, err error) {
	logrus.WithFields(logrus.Fields{
		"topic":      topic,
		"event_name": value.EventName,
		"source":     value.Source,
		"error":      err.Error(),
	}).Error("Handling Kafka publish failure")

	// ✅ 1. Store to dead letter queue (dengan retry) - ENABLEDx
	k.storeToDeadLetterQueue(topic, value, err)

	// ✅ 3. Send to monitoring/alerting system - ENABLED
	k.sendAlert("kafka_publish_failed", topic, err)

	 // Write to file for later processing
    helpers.WriteToFailureLog(topic, value, err, k.Address, k.Username)

}

func (k *KafkaConfig) InitializeWithDLQ(dlqConfig DLQConfig) {
	k.DLQConfig = dlqConfig
	k.failureCache = sync.Map{}

	logrus.WithFields(logrus.Fields{
		"dlq_enabled":        dlqConfig.Enabled,
		"max_retries":        dlqConfig.MaxRetries,
		"retry_delay":        dlqConfig.RetryDelay,
		"dead_letter_suffix": dlqConfig.DeadLetterSuffix,
	}).Info("Kafka DLQ initialized")
}

// ✅ Quick setup with defaults
func (k *KafkaConfig) EnableDLQ() {
	k.InitializeWithDLQ(DefaultDLQConfig())
}

func (k *KafkaConfig) storeToDeadLetterQueue(topic string, value kafka.Event, originalErr error) {
    if !k.DLQConfig.Enabled {
        logrus.Debug("DLQ is disabled, skipping dead letter storage")
        return
    }

    deadLetterTopic := fmt.Sprintf("%s%s", topic, k.DLQConfig.DeadLetterSuffix)

    // Check for duplicate failures
    messageKey := helpers.GenerateMessageKey(topic, value)
    if k.DLQConfig.EnableDeduplication && k.dlqHelper.IsDuplicateFailure(messageKey) {
        logrus.WithField("message_key", messageKey).Warn("Duplicate failure detected, skipping DLQ")
        return
    }

    // Mark as processing
    if k.DLQConfig.EnableDeduplication {
        k.dlqHelper.MarkFailureProcessing(messageKey)
        defer k.dlqHelper.ClearFailureProcessing(messageKey)
    }

    // Create enhanced dead letter event
    deadLetterEvent := k.dlqHelper.CreateDeadLetterEvent(topic, value, originalErr, k.Address, k.Username)

    // Retry logic with exponential backoff
    k.retryPublishToDeadLetter(deadLetterTopic, deadLetterEvent, topic, value, originalErr)
}

func (k *KafkaConfig) sendAlert(alertType, topic string, err error) {
    alert := k.dlqHelper.CreateFailureAlert(alertType, topic, err, k.Address)

    // Send to multiple monitoring systems (async)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                logrus.WithField("panic", r).Error("Panic while sending alert")
            }
        }()

        // Send to Slack
        k.slackHelper.SendSlackAlert(alert)
    }()
}


func (k *KafkaConfig) retryPublishToDeadLetter(deadLetterTopic string, deadLetterEvent kafka.Event, originalTopic string, originalValue kafka.Event, originalErr error) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                logrus.WithFields(logrus.Fields{
                    "panic": r,
                    "topic": deadLetterTopic,
                }).Error("Panic in DLQ retry goroutine")
                helpers.WriteToFailureLog(fmt.Sprintf("%s-dlq-panic", originalTopic), originalValue, fmt.Errorf("panic in DLQ: %v", r), k.Address, k.Username)
            }
        }()

        var lastErr error
        delay := k.DLQConfig.RetryDelay

        for attempt := 0; attempt < k.DLQConfig.MaxRetries; attempt++ {
            if eventData, ok := deadLetterEvent.Data.(map[string]interface{}); ok {
                eventData["retry_count"] = attempt
                eventData["retry_timestamp"] = time.Now().Unix()
            }

            _, err := k.publishEventOnce(deadLetterTopic, deadLetterEvent)

            if err == nil {
                logrus.WithFields(logrus.Fields{
                    "dead_letter_topic": deadLetterTopic,
                    "attempt":           attempt + 1,
                    "original_topic":    originalTopic,
                }).Info("Successfully stored to dead letter queue")

                if attempt > 0 {
                    k.sendDLQRecoveryAlert(deadLetterTopic, attempt)
                }
                return
            }

            lastErr = err
            logrus.WithFields(logrus.Fields{
                "attempt":           attempt + 1,
                "max_retries":       k.DLQConfig.MaxRetries,
                "delay":             delay,
                "error":             err.Error(),
                "dead_letter_topic": deadLetterTopic,
            }).Warn("Failed to publish to dead letter queue, retrying...")
        }

        logrus.WithFields(logrus.Fields{
            "dead_letter_topic": deadLetterTopic,
            "original_topic":    originalTopic,
            "final_error":       lastErr.Error(),
            "retries":           k.DLQConfig.MaxRetries,
        }).Error("Failed to store to dead letter queue after all retries")

        // Use helper for failure fallback
        helpers.LogFailureFallback(originalTopic, originalValue, originalErr, lastErr, k.Address, k.Username)
    }()
}


func (k *KafkaConfig) TestDLQ() error {
	if !k.DLQConfig.Enabled {
		return fmt.Errorf("DLQ is not enabled")
	}

	testEvent := kafka.Event{
		EventName: "DLQ_TEST_EVENT",
		Source:    "dlq-test",
		Data: map[string]interface{}{
			"test":      true,
			"timestamp": time.Now().Unix(),
			"message":   "This is a test event for DLQ functionality",
		},
	}

	// ✅ Simulate failure to trigger DLQ
	testError := fmt.Errorf("simulated error for DLQ testing")

	logrus.Info("Testing DLQ functionality...")
	k.handlePublishFailure("test-topic", testEvent, testError)

	logrus.Info("DLQ test completed - check logs and monitoring for results")
	return nil
}


func (k *KafkaConfig) sendDLQRecoveryAlert(deadLetterTopic string, attempts int) {
    recoveryAlert := k.dlqHelper.CreateDLQRecoveryAlert(deadLetterTopic, attempts, k.Address)
    
    k.slackHelper.SendSlackAlert(recoveryAlert)
}

func (k *KafkaConfig) sendSlackAlert(alert map[string]interface{}) {

	
	slackWebhook := k.SlackWebhookURL

	if slackWebhook == "" {
		logrus.Warn("SLACK_WEBHOOK_URL not configured, skipping Slack alert")
		return
	}

	// ✅ Check if alerts enabled
	if !k.AlertEnabled {
		logrus.Debug("Slack alerts disabled, skipping")
		return
	}

	logrus.WithFields(logrus.Fields{
		"webhook_configured": slackWebhook != "",
		"alert_enabled":      k.AlertEnabled,
		"topic":              alert["topic"],
	}).Info("Preparing to send Slack alert")

	// ✅ Build Slack message
	message := k.buildSlackMessage(alert)

	// ✅ Send to Slack
	if err := k.sendToSlack(slackWebhook, message); err != nil {
		logrus.WithError(err).Error("Failed to send Slack alert")
	} else {
		logrus.WithFields(logrus.Fields{
			"topic":    alert["topic"],
			"severity": alert["severity"],
		}).Info("✅ Successfully sent Slack alert")
	}
}

func (k *KafkaConfig) buildSlackMessage(alert map[string]interface{}) SlackMessage {

	severity := alert["severity"].(string)
	topic := alert["topic"].(string)
	errorMsg := alert["error"].(string)
	timestamp := alert["timestamp"].(time.Time)

	// ✅ Determine color based on severity
	color := k.slackHelper.GetSlackColor(severity)

	// ✅ Build main message
	mainText := fmt.Sprintf("🚨 *Kafka Failure Alert* - %s", strings.ToUpper(severity))

	// ✅ Build attachment with detailed info
	attachment := SlackAttachment{
		Color:     color,
		Title:     fmt.Sprintf("Failed to publish to topic: %s", topic),
		Text:      fmt.Sprintf("```%s```", errorMsg),
		Timestamp: timestamp.Unix(),
		Footer:    "Kafka Alert System",
		Fields: []SlackField{
			{
				Title: "Service",
				Value: getEnvOrDefault("APP_NAME", "unknown-service"),
				Short: true,
			},
			{
				Title: "Environment",
				Value: getEnvOrDefault("ENVIRONMENT", "unknown"),
				Short: true,
			},
			{
				Title: "Topic",
				Value: topic,
				Short: true,
			},
			{
				Title: "Severity",
				Value: strings.ToUpper(severity),
				Short: true,
			},
			{
				Title: "Kafka Hosts",
				Value: fmt.Sprintf("%v", alert["kafka_hosts"]),
				Short: false,
			},
			{
				Title: "Timestamp",
				Value: timestamp.Format("2006-01-02 15:04:05 MST"),
				Short: true,
			},
			{
				Title: "Alert Type",
				Value: fmt.Sprintf("%s", alert["alert_type"]),
				Short: true,
			},
		},
	}

	// ✅ Add action buttons for critical alerts
	if severity == "critical" {
		attachment.Text += "\n\n⚠️ *This is a critical alert requiring immediate attention!*"
	}

	return SlackMessage{
		Text:        mainText,
		Username:    getEnvOrDefault("SLACK_USERNAME", "Kafka Alert Bot"),
		IconEmoji:   getEnvOrDefault("SLACK_ICON_EMOJI", ":warning:"),
		Channel:     getEnvOrDefault("SLACK_CHANNEL", "#kafka-alerts"),
		Attachments: []SlackAttachment{attachment},
	}
}

// ✅ Helper function
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (k *KafkaConfig) sendToSlack(webhookURL string, message SlackMessage) error {
	// ✅ Marshal message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"webhook_url":  maskWebhookURL(webhookURL),
		"message_size": len(jsonData),
	}).Debug("Sending Slack message")

	// ✅ Create HTTP request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kafka-Alert-Bot/1.0")

	// ✅ Send request with timeout
	client := &http.Client{
		Timeout: 15 * time.Second, // Increase timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// ✅ Read response body for debugging
	body, _ := io.ReadAll(resp.Body)

	// ✅ Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	logrus.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"response":    string(body),
	}).Debug("Slack message sent successfully")

	return nil
}

// ✅ Helper untuk mask webhook URL
func maskWebhookURL(url string) string {
	if len(url) > 30 {
		return url[:30] + "***MASKED***"
	}
	return "***MASKED***"
}


func (k *KafkaConfig) SendRecoveryAlert(topic string) {
    // Create recovery message using helper
    message := helpers.CreateRecoveryMessage(topic, k.Address)
    
    // Send using Slack helper
    webhookURL := k.SlackWebhookURL
    if webhookURL == "" {
        webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
    }
    
    if webhookURL != "" && k.slackHelper.IsSlackEnabled() {
        k.slackHelper.SendToSlack(webhookURL, message)
    }
}

func (k *KafkaConfig) IsDeadLetterTopicAvailable(topic string) bool {
	_, err := k.getOrCreateProducer()
	logrus.WithField("topic", topic).Debug("Checking DLQ topic availability")
	if err != nil {
		logrus.WithError(err).Error("Failed to get Kafka producer for DLQ check")
		return false
	}

	testMsg := kafka.Event{
		EventName: "DLQ_HEALTH_CHECK",
		Source:    "system",
		Data:      map[string]interface{}{"ping": "pong"},
	}

	_, err = k.publishEventOnce(topic+"-dead-letter", testMsg)
	if err != nil {
		logrus.WithError(err).Error("DLQ health check failed")
		return false
	}

	logrus.Info("DLQ topic is available and accepting messages")
	return true
}


func (k *KafkaConfig) getOrCreateProducer() (sarama.SyncProducer, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.producer == nil {
		brokers := k.Address
		if len(brokers) == 0 {
			return nil, fmt.Errorf("kafka address is not configured")
		}

		producer, err := sarama.NewSyncProducer(brokers, createConfig(k))
		if err != nil {
			return nil, fmt.Errorf("unable to create kafka producer: %w", err)
		}
		k.producer = producer
		logrus.WithField("brokers", brokers).Info("Created new Kafka producer")
	}
	return k.producer, nil
}


func (k *KafkaConfig) resetProducer() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.producer != nil {
		k.producer.Close()
		k.producer = nil
		logrus.Warn("Reset Kafka producer due to error")
	}
}

func (k *KafkaConfig) Validate() error {
	if len(k.Address) == 0 {
		return fmt.Errorf("kafka address cannot be empty")
	}

	for i, addr := range k.Address {
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("kafka address at index %d cannot be empty", i)
		}
	}

	if k.Config.Retry.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if k.Config.Retry.RetryInterval < 0 {
		return fmt.Errorf("retry interval cannot be negative")
	}

	if k.Config.FlushFrequency < 0 {
		return fmt.Errorf("flush frequency cannot be negative")
	}

	return nil
}

func (k *KafkaConfig) GetInfo() map[string]interface{} {
	k.mu.Lock()
	defer k.mu.Unlock()

	return map[string]interface{}{
		"addresses":          k.Address,
		"username":           k.Username,
		"has_password":       k.Password != "",
		"producer_connected": k.producer != nil,
		"config": map[string]interface{}{
			"required_acks":   k.Config.RequiredAcks,
			"compression":     k.Config.Compression.String(),
			"flush_frequency": k.Config.FlushFrequency.String(),
			"flush_messages":  k.Config.FlushMessages,
			"flush_bytes":     k.Config.FlushBytes,
			"enable_tls":      k.Config.EnableTLS,
			"enable_sasl":     k.Config.EnableSASL,
			"sasl_mechanism":  k.Config.SASLMechanism,
		},
	}
}

func (k *KafkaConfig) UpdateConfig(options KafkaConfigOptions) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Close existing producer if config changes
	if k.producer != nil {
		if err := k.producer.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close existing producer during config update")
		}
		k.producer = nil
	}

	k.Config = options

	// Validate new configuration
	if err := k.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	logrus.Info("Kafka configuration updated successfully")
	return nil
}

func (k *KafkaConfig) ForceReconnect() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.producer != nil {
		if err := k.producer.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close producer during force reconnect")
		}
		k.producer = nil
	}

	logrus.Info("Forced Kafka producer reconnection")
	return nil
}
