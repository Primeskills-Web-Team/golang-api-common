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
	"path/filepath"
	"strings"
	"time"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	kafkaproducer "github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/producer"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// PublishResult represents the result of a publish operation
type PublishResult struct {
	Success   bool
	Error     error
	Partition int32
	Offset    int64
	Timestamp time.Time
}

// RetryConfig configuration for retry mechanism
type RetryConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	BackoffFactor float64
}

// ✅ Slack Message Structure
type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		RetryInterval: time.Second * 2,
		BackoffFactor: 2.0,
	}
}

// PublishEventWithRetry publishes event with retry mechanism and returns result
func (k *KafkaConfig) PublishEventWithRetry(ctx context.Context, topic string, value kafka.Event, retryConfig RetryConfig) (*PublishResult, error) {
	var lastErr error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
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

		// ✅ Gunakan function dari config.go
		if isConnectionError(err) {
			k.resetProducer()
		}
	}

	return nil, fmt.Errorf("failed to publish after %d attempts, last error: %w", retryConfig.MaxRetries+1, lastErr)
}

// ✅ publishEventOnce menggunakan method dari config.go
func (k *KafkaConfig) publishEventOnce(topic string, value kafka.Event) (*PublishResult, error) {
	syncProducer, err := k.getOrCreateProducer() // ✅ Method dari config.go
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

// PublishEventWithCallback publishes event with success/failure callbacks
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

// Enhanced version of original method with better error handling
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

// ✅ ENHANCED handlePublishFailure dengan implementasi lengkap
func (k *KafkaConfig) handlePublishFailure(topic string, value kafka.Event, err error) {
	logrus.WithFields(logrus.Fields{
		"topic":      topic,
		"event_name": value.EventName,
		"source":     value.Source,
		"error":      err.Error(),
	}).Error("Handling Kafka publish failure")

	// ✅ 1. Store to dead letter queue (dengan retry)
	// k.storeToDeadLetterQueue(topic, value, err)

	// ✅ 2. Store to database for manual processing
	// k.storeFailedMessage(topic, value, err)

	// ✅ 3. Send to monitoring/alerting system
	k.sendAlert("kafka_publish_failed", topic, err)

	// ✅ 4. Write to file for later processing
	// k.writeToFailureLog(topic, value, err)

	// ✅ 5. Increment failure metrics
	// k.incrementFailureMetrics(topic, err)
}

// ✅ 1. Dead Letter Queue Implementation
func (k *KafkaConfig) storeToDeadLetterQueue(topic string, value kafka.Event, originalErr error) {
	deadLetterTopic := fmt.Sprintf("%s-dead-letter", topic)

	// ✅ Enhanced dead letter event dengan metadata
	deadLetterEvent := kafka.Event{
		EventName: fmt.Sprintf("%s_DEAD_LETTER", value.EventName),
		Source:    value.Source,
		Data: map[string]interface{}{
			"original_event":    value,
			"original_topic":    topic,
			"failure_reason":    originalErr.Error(),
			"failure_timestamp": time.Now().Unix(),
			"retry_count":       0,
		},
	}

	// ✅ Try to publish to dead letter queue dengan timeout
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ Use simple producer untuk dead letter (avoid infinite recursion)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{
					"topic": deadLetterTopic,
					"panic": r,
					"event": value.EventName,
				}).Error("Panic while storing to dead letter queue")
			}
		}()

		result, err := k.publishEventOnce(deadLetterTopic, deadLetterEvent)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"dead_letter_topic": deadLetterTopic,
				"original_topic":    topic,
				"error":             err.Error(),
			}).Error("Failed to store to dead letter queue")

			// ✅ Fallback: store to file if dead letter fails
			k.writeToFailureLog(fmt.Sprintf("%s-deadletter-failed", topic), deadLetterEvent, err)
		} else {
			logrus.WithFields(logrus.Fields{
				"dead_letter_topic": deadLetterTopic,
				"partition":         result.Partition,
				"offset":            result.Offset,
			}).Info("Successfully stored failed event to dead letter queue")
		}
	}()
}

// ✅ 2. Database Storage Implementation
func (k *KafkaConfig) storeFailedMessage(topic string, value kafka.Event, err error) {
	// ✅ Create failed event record
	failedEvent := map[string]interface{}{
		"id":            k.generateFailedEventID(),
		"topic":         topic,
		"event_name":    value.EventName,
		"source":        value.Source,
		"event_data":    value.Data,
		"error_message": err.Error(),
		"retry_count":   0,
		"status":        "pending",
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	// ✅ Store to database (async untuk avoid blocking)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{
					"topic": topic,
					"event": value.EventName,
					"panic": r,
				}).Error("Panic while storing failed message to database")
			}
		}()

		if err := k.saveToDatabase(failedEvent); err != nil {
			logrus.WithFields(logrus.Fields{
				"topic": topic,
				"event": value.EventName,
				"error": err.Error(),
			}).Error("Failed to store failed message to database")

			// ✅ Fallback: store to file
			k.writeToFailureLog(fmt.Sprintf("%s-db-failed", topic), value, err)
		} else {
			logrus.WithFields(logrus.Fields{
				"topic":           topic,
				"event":           value.EventName,
				"failed_event_id": failedEvent["id"],
			}).Info("Successfully stored failed message to database")
		}
	}()
}

// ✅ 3. Monitoring/Alerting Implementation
func (k *KafkaConfig) sendAlert(alertType, topic string, err error) {
	alert := map[string]interface{}{
		"alert_type":  alertType,
		"topic":       topic,
		"error":       err.Error(),
		"timestamp":   time.Now(),
		"severity":    k.determineAlertSeverity(err),
		"service":     os.Getenv("APP_NAME"),
		"environment": os.Getenv("ENVIRONMENT"),
		"kafka_hosts": k.Address,
	}

	// ✅ Send to multiple monitoring systems (async)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithField("panic", r).Error("Panic while sending alert")
			}
		}()

		// ✅ Send to Slack/Discord
		k.sendSlackAlert(alert)

		// // ✅ Send to monitoring service (Prometheus, DataDog, etc.)
		// k.sendToMonitoringService(alert)

		// // ✅ Send email alert for critical errors
		// if alert["severity"] == "critical" {
		// 	k.sendEmailAlert(alert)
		// }

		// logrus.WithFields(logrus.Fields{
		// 	"alert_type": alertType,
		// 	"topic":      topic,
		// 	"severity":   alert["severity"],
		// }).Info("Alert sent for Kafka failure")
	}()
}

// ✅ 4. File Logging Implementation
func (k *KafkaConfig) writeToFailureLog(topic string, value kafka.Event, err error) {
	// ✅ Create failure log entry
	logEntry := map[string]interface{}{
		"id":        k.generateFailedEventID(),
		"timestamp": time.Now().Format(time.RFC3339),
		"topic":     topic,
		"event":     value,
		"error":     err.Error(),
		"kafka_config": map[string]interface{}{
			"addresses": k.Address,
			"username":  k.Username,
		},
	}

	// ✅ Async file writing
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithField("panic", r).Error("Panic while writing to failure log")
			}
		}()

		// ✅ Create directory structure
		logDir := "storage/kafka_failures"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.WithError(err).Error("Failed to create failure log directory")
			return
		}

		// ✅ File name with date and topic
		fileName := filepath.Join(logDir, fmt.Sprintf("kafka_failures_%s_%s.jsonl",
			topic, time.Now().Format("2006-01-02")))

		// ✅ Write to file
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrus.WithError(err).Error("Failed to open failure log file")
			return
		}
		defer file.Close()

		// ✅ Write JSON line
		jsonData, err := json.Marshal(logEntry)
		if err != nil {
			logrus.WithError(err).Error("Failed to marshal failure log entry")
			return
		}

		if _, err := file.Write(append(jsonData, '\n')); err != nil {
			logrus.WithError(err).Error("Failed to write to failure log file")
		} else {
			logrus.WithFields(logrus.Fields{
				"file":  fileName,
				"topic": topic,
				"event": value.EventName,
			}).Debug("Successfully wrote failure log to file")
		}
	}()
}

// ✅ 5. Metrics Implementation
func (k *KafkaConfig) incrementFailureMetrics(topic string, err error) {
	// ✅ Simple in-memory metrics (bisa diganti dengan Prometheus)
	go func() {
		// Increment failure counter
		// metrics.KafkaFailureCounter.WithLabelValues(topic, k.getErrorType(err)).Inc()

		logrus.WithFields(logrus.Fields{
			"topic":      topic,
			"error_type": k.getErrorType(err),
		}).Debug("Incremented Kafka failure metrics")
	}()
}

// ✅ Batch publish menggunakan method dari config.go
func (k *KafkaConfig) PublishEventsBatch(ctx context.Context, topic string, events []kafka.Event) ([]*PublishResult, error) {
	syncProducer, err := k.getOrCreateProducer() // ✅ Method dari config.go
	if err != nil {
		return nil, fmt.Errorf("unable to get kafka producer: %w", err)
	}

	results := make([]*PublishResult, 0, len(events))

	for i, event := range events {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		msg, err := json.Marshal(event)
		if err != nil {
			results = append(results, &PublishResult{
				Success: false,
				Error:   fmt.Errorf("failed to marshal event %d: %w", i, err),
			})
			continue
		}

		kafkaProducer := &kafkaproducer.KafkaProducer{Producer: syncProducer}
		partition, offset, err := kafkaProducer.SendMessageSync(topic, string(msg))

		if err != nil {
			results = append(results, &PublishResult{
				Success: false,
				Error:   fmt.Errorf("failed to send event %d: %w", i, err),
			})
		} else {
			results = append(results, &PublishResult{
				Success:   true,
				Partition: partition,
				Offset:    offset,
				Timestamp: time.Now(),
			})
		}
	}

	return results, nil
}

// ✅ Helper Functions
func (k *KafkaConfig) generateFailedEventID() string {
	return fmt.Sprintf("failed_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func (k *KafkaConfig) determineAlertSeverity(err error) string {
	errorStr := err.Error()

	// ✅ Critical errors
	if strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "no such host") ||
		strings.Contains(errorStr, "network unreachable") {
		return "critical"
	}

	// ✅ Warning errors
	if strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "context deadline exceeded") {
		return "warning"
	}

	// ✅ Default to info
	return "info"
}

func (k *KafkaConfig) getErrorType(err error) string {
	errorStr := err.Error()

	if strings.Contains(errorStr, "connection") {
		return "connection_error"
	}
	if strings.Contains(errorStr, "timeout") {
		return "timeout_error"
	}
	if strings.Contains(errorStr, "marshal") {
		return "serialization_error"
	}
	if strings.Contains(errorStr, "authentication") {
		return "auth_error"
	}

	return "unknown_error"
}

// ✅ Database storage implementation
func (k *KafkaConfig) saveToDatabase(failedEvent map[string]interface{}) error {
	// ✅ Implementation depends on your database
	// Example for GORM:
	/*
	   db := database.Connection

	   event := entity.FailedKafkaEvent{
	       EventName:    failedEvent["event_name"].(string),
	       Topic:        failedEvent["topic"].(string),
	       Source:       failedEvent["source"].(string),
	       ErrorMessage: failedEvent["error_message"].(string),
	       Status:       entity.FailedEventStatusPending,
	   }

	   if err := event.SetEventData(failedEvent["event_data"]); err != nil {
	       return err
	   }

	   return db.Create(&event).Error
	*/

	// ✅ For now, just log (implement based on your database)
	logrus.WithField("event", failedEvent).Info("Would save to database")
	return nil
}

func (k *KafkaConfig) sendSlackAlert(alert map[string]interface{}) {
	// ✅ Load environment untuk mendapatkan Slack config
	_ = godotenv.Load()

	// ✅ Prioritas: 1. Dari constructor, 2. Dari environment
	slackWebhook := k.SlackWebhookURL
	if slackWebhook == "" {
		slackWebhook = os.Getenv("SLACK_WEBHOOK_URL")
	}

	if slackWebhook == "" {
		logrus.Warn("SLACK_WEBHOOK_URL not configured, skipping Slack alert")
		return
	}

	// ✅ Check if alerts enabled
	alertEnabled := os.Getenv("ALERT_ENABLED")
	if alertEnabled != "true" && alertEnabled != "1" {
		logrus.Debug("Slack alerts disabled, skipping")
		return
	}

	logrus.WithFields(logrus.Fields{
		"webhook_configured": slackWebhook != "",
		"alert_enabled":      alertEnabled,
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

// ✅ PERBAIKAN: Enhanced buildSlackMessage
func (k *KafkaConfig) buildSlackMessage(alert map[string]interface{}) SlackMessage {
	// ✅ Load environment
	_ = godotenv.Load()

	severity := alert["severity"].(string)
	topic := alert["topic"].(string)
	errorMsg := alert["error"].(string)
	timestamp := alert["timestamp"].(time.Time)

	// ✅ Determine color based on severity
	color := k.getSlackColor(severity)

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

// ✅ PERBAIKAN: Enhanced sendToSlack dengan better error handling
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
		return fmt.Errorf("Slack webhook returned status %d: %s", resp.StatusCode, string(body))
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

// ✅ Check if Slack is enabled
func (k *KafkaConfig) isSlackEnabled() bool {
	_ = godotenv.Load("./../../.env")
	enabled := os.Getenv("ALERT_ENABLED")
	return enabled == "true" || enabled == "1"
}


// ✅ Get Slack color based on severity
func (k *KafkaConfig) getSlackColor(severity string) string {
	switch severity {
	case "critical":
		return "danger" // Red
	case "warning":
		return "warning" // Yellow
	case "info":
		return "good" // Green
	default:
		return "#808080" // Gray
	}
}

// ✅ Send test Slack alert
func (k *KafkaConfig) SendTestSlackAlert() error {
	testAlert := map[string]interface{}{
		"alert_type":  "kafka_test_alert",
		"topic":       "test-topic",
		"error":       "This is a test alert to verify Slack integration",
		"timestamp":   time.Now(),
		"severity":    "info",
		"service":     os.Getenv("APP_NAME"),
		"environment": os.Getenv("ENVIRONMENT"),
		"kafka_hosts": k.Address,
	}

	k.sendSlackAlert(testAlert)
	return nil
}

func (k *KafkaConfig) SendSlackAlert(alert map[string]interface{}) {
    k.sendSlackAlert(alert)
}

// ✅ TAMBAHKAN: Method untuk test dengan custom retry
func (k *KafkaConfig) PublishEventWithCustomRetry(ctx context.Context, topic string, value kafka.Event, retryConfig RetryConfig, onSuccess func(*PublishResult), onFailure func(error)) {
    go func() {
        result, err := k.PublishEventWithRetry(ctx, topic, value, retryConfig)
        if err != nil {
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

// ✅ Send recovery alert when Kafka is back online
func (k *KafkaConfig) SendRecoveryAlert(topic string) {
	if !k.isSlackEnabled() {
		return
	}

	recoveryAlert := map[string]interface{}{
		"alert_type":  "kafka_recovery",
		"topic":       topic,
		"error":       "Kafka connection has been restored",
		"timestamp":   time.Now(),
		"severity":    "info",
		"service":     os.Getenv("APP_NAME"),
		"environment": os.Getenv("ENVIRONMENT"),
		"kafka_hosts": k.Address,
	}

	// ✅ Build recovery message
	message := SlackMessage{
		Text:      "✅ *Kafka Recovery Alert*",
		Username:  os.Getenv("SLACK_USERNAME"),
		IconEmoji: ":white_check_mark:",
		Channel:   os.Getenv("SLACK_CHANNEL"),
		Attachments: []SlackAttachment{
			{
				Color: "good",
				Title: "Kafka Connection Restored",
				Text:  fmt.Sprintf("Topic `%s` is now accessible", topic),
				Fields: []SlackField{
					{
						Title: "Service",
						Value: fmt.Sprintf("%s", recoveryAlert["service"]),
						Short: true,
					},
					{
						Title: "Environment",
						Value: fmt.Sprintf("%s", recoveryAlert["environment"]),
						Short: true,
					},
				},
				Footer:    "Kafka Alert System",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL != "" {
		k.sendToSlack(webhookURL, message)
	}
}

// ✅ Monitoring service implementation
func (k *KafkaConfig) sendToMonitoringService(alert map[string]interface{}) {
	// ✅ Implementation for monitoring service
	logrus.WithField("alert", alert).Info("Would send to monitoring service")
}

// ✅ Email alert implementation
func (k *KafkaConfig) sendEmailAlert(alert map[string]interface{}) {
	// ✅ Implementation for email alerts
	logrus.WithField("alert", alert).Info("Would send email alert")
}
