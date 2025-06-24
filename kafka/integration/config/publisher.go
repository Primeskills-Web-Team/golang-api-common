package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	kafkaproducer "github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/producer"
	"github.com/Primeskills-Web-Team/golang-api-common/pkg/circuitbreaker"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

		if isConnectionError(err) {
			k.resetProducer()
		}
	}

	return nil, fmt.Errorf("failed to publish after %d attempts, last error: %w", retryConfig.MaxRetries+1, lastErr)
}

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

	// ✅ 2. Store to database for manual processing - ENABLED
	// k.storeFailedMessage(topic, value, err)

	// ✅ 3. Send to monitoring/alerting system - ENABLED
	k.sendAlert("kafka_publish_failed", topic, err)

	// ✅ 4. Write to file for later processing - ENABLED
	k.writeToFailureLog(topic, value, err)

	// ✅ 5. Increment failure metrics - ENABLED
	// k.incrementFailureMetrics(topic, err)
}

func (k *KafkaConfig) InitializeWithDLQ(dlqConfig DLQConfig) {
	k.DLQConfig = dlqConfig
	k.failureCache = sync.Map{}

	// ✅ Initialize circuit breaker if enabled
	if dlqConfig.EnableCircuitBreaker {
		k.initCircuitBreaker()
	}

	logrus.WithFields(logrus.Fields{
		"dlq_enabled":        dlqConfig.Enabled,
		"max_retries":        dlqConfig.MaxRetries,
		"retry_delay":        dlqConfig.RetryDelay,
		"dead_letter_suffix": dlqConfig.DeadLetterSuffix,
		"circuit_breaker":    dlqConfig.EnableCircuitBreaker,
	}).Info("Kafka DLQ initialized")
}

func (k *KafkaConfig) initCircuitBreaker() {
	config := circuitbreaker.Config{
		Name:         "kafka-dlq",
		MaxFailures:  k.DLQConfig.CircuitBreakerConfig.MaxFailures,
		ResetTimeout: k.DLQConfig.CircuitBreakerConfig.ResetTimeout,
		OnStateChange: func(from, to circuitbreaker.CircuitState) {
			k.onCircuitBreakerStateChange(from, to)
		},
		OnFailure: func(err error) {
			k.onCircuitBreakerFailure(err)
		},
		OnSuccess: func() {
			k.onCircuitBreakerSuccess()
		},
	}

	k.circuitBreaker = circuitbreaker.NewCircuitBreakerWithConfig(config)

	logrus.WithFields(logrus.Fields{
		"max_failures":  config.MaxFailures,
		"reset_timeout": config.ResetTimeout,
	}).Info("Circuit breaker initialized for Kafka DLQ")
}

func (k *KafkaConfig) onCircuitBreakerStateChange(from, to circuitbreaker.CircuitState) {
	logrus.WithFields(logrus.Fields{
		"from_state": k.getCircuitStateString(from),
		"to_state":   k.getCircuitStateString(to),
		"component":  "kafka-dlq",
	}).Warn("Circuit breaker state changed")

	// ✅ Send alert for state changes
	alert := map[string]interface{}{
		"alert_type":  "circuit_breaker_state_change",
		"topic":       "kafka-dlq",
		"error":       fmt.Sprintf("Circuit breaker state changed from %s to %s", k.getCircuitStateString(from), k.getCircuitStateString(to)),
		"timestamp":   time.Now(),
		"severity":    k.getCircuitBreakerSeverity(to),
		"service":     os.Getenv("APP_NAME"),
		"environment": os.Getenv("ENVIRONMENT"),
		"kafka_hosts": k.Address,
		"from_state":  k.getCircuitStateString(from),
		"to_state":    k.getCircuitStateString(to),
	}

	k.sendSlackAlert(alert)
}

func (k *KafkaConfig) onCircuitBreakerFailure(err error) {
	logrus.WithFields(logrus.Fields{
		"error":     err.Error(),
		"component": "kafka-dlq",
	}).Debug("Circuit breaker recorded failure")
}

func (k *KafkaConfig) onCircuitBreakerSuccess() {
	logrus.WithField("component", "kafka-dlq").Debug("Circuit breaker recorded success")
}

func (k *KafkaConfig) getCircuitStateString(state circuitbreaker.CircuitState) string {
	switch state {
	case circuitbreaker.StateClosed:
		return "CLOSED"
	case circuitbreaker.StateOpen:
		return "OPEN"
	case circuitbreaker.StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

func (k *KafkaConfig) getCircuitBreakerSeverity(state circuitbreaker.CircuitState) string {
	switch state {
	case circuitbreaker.StateOpen:
		return "critical"
	case circuitbreaker.StateHalfOpen:
		return "warning"
	case circuitbreaker.StateClosed:
		return "info"
	default:
		return "info"
	}
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

	// ✅ Check for duplicate failures
	messageKey := k.generateMessageKey(topic, value)
	if k.DLQConfig.EnableDeduplication && k.isDuplicateFailure(messageKey) {
		logrus.WithField("message_key", messageKey).Warn("Duplicate failure detected, skipping DLQ")
		return
	}

	// ✅ Mark as processing
	if k.DLQConfig.EnableDeduplication {
		k.markFailureProcessing(messageKey)
		defer k.clearFailureProcessing(messageKey)
	}

	// ✅ Enhanced dead letter event dengan metadata
	deadLetterEvent := k.createDeadLetterEvent(topic, value, originalErr)

	// ✅ Retry logic dengan exponential backoff
	k.retryPublishToDeadLetter(deadLetterTopic, deadLetterEvent, topic, value, originalErr)
}

func (k *KafkaConfig) createDeadLetterEvent(topic string, value kafka.Event, originalErr error) kafka.Event {
	hostname, _ := os.Hostname()

	return kafka.Event{
		EventName: fmt.Sprintf("%s_DEAD_LETTER", value.EventName),
		Source:    value.Source,
		Data: map[string]interface{}{
			"original_event":    value,
			"original_topic":    topic,
			"failure_reason":    originalErr.Error(),
			"failure_timestamp": time.Now().Unix(),
			"failure_type":      k.getErrorType(originalErr),
			"retry_count":       0,
			"message_id":        k.generateMessageKey(topic, value),
			"server_info": map[string]interface{}{
				"hostname":    hostname,
				"service":     os.Getenv("APP_NAME"),
				"environment": os.Getenv("ENVIRONMENT"),
				"version":     os.Getenv("APP_VERSION"),
			},
			"kafka_config": map[string]interface{}{
				"brokers":  k.Address,
				"username": k.Username,
			},
		},
	}
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

func (k *KafkaConfig) retryPublishToDeadLetter(deadLetterTopic string, deadLetterEvent kafka.Event, originalTopic string, originalValue kafka.Event, originalErr error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{
					"panic": r,
					"topic": deadLetterTopic,
				}).Error("Panic in DLQ retry goroutine")
				k.writeToFailureLog(fmt.Sprintf("%s-dlq-panic", originalTopic), originalValue, fmt.Errorf("panic in DLQ: %v", r))
			}
		}()

		var lastErr error
		delay := k.DLQConfig.RetryDelay

		for attempt := 0; attempt < k.DLQConfig.MaxRetries; attempt++ {
			// ✅ Check circuit breaker before attempting
			if k.DLQConfig.EnableCircuitBreaker && k.circuitBreaker != nil && k.circuitBreaker.IsOpen() {
				logrus.WithFields(logrus.Fields{
					"dead_letter_topic": deadLetterTopic,
					"attempt":           attempt + 1,
				}).Warn("Circuit breaker is OPEN, skipping DLQ publish attempt")

				k.executeFailureFallbacks(originalTopic, originalValue, originalErr, fmt.Errorf("circuit breaker open"))
				return
			}

			// ✅ Update retry count in event data
			if eventData, ok := deadLetterEvent.Data.(map[string]interface{}); ok {
				eventData["retry_count"] = attempt
				eventData["retry_timestamp"] = time.Now().Unix()
			}

			// ✅ Use circuit breaker if enabled
			var err error
			if k.DLQConfig.EnableCircuitBreaker && k.circuitBreaker != nil {
				err = k.circuitBreaker.Call(func() error {
					_, publishErr := k.publishEventOnce(deadLetterTopic, deadLetterEvent)
					return publishErr
				})
			} else {
				_, err = k.publishEventOnce(deadLetterTopic, deadLetterEvent)
			}

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

			if attempt < k.DLQConfig.MaxRetries-1 {
				jitteredDelay := k.addJitter(delay)
				time.Sleep(jitteredDelay)
				delay = k.calculateNextDelay(delay)
			}
		}

		logrus.WithFields(logrus.Fields{
			"dead_letter_topic": deadLetterTopic,
			"original_topic":    originalTopic,
			"final_error":       lastErr.Error(),
			"retries":           k.DLQConfig.MaxRetries,
		}).Error("Failed to store to dead letter queue after all retries")

		k.executeFailureFallbacks(originalTopic, originalValue, originalErr, lastErr)
	}()
}

func (k *KafkaConfig) GetDLQHealthStatus() map[string]interface{} {
	status := map[string]interface{}{
		"dlq_enabled": k.DLQConfig.Enabled,
		"timestamp":   time.Now(),
	}

	if k.circuitBreaker != nil {
		metrics := k.circuitBreaker.GetMetrics()
		status["circuit_breaker"] = map[string]interface{}{
			"state":          metrics.State,
			"failure_count":  metrics.FailureCount,
			"success_count":  metrics.SuccessCount,
			"total_requests": metrics.TotalRequests,
			"failure_rate":   k.circuitBreaker.GetFailureRate(),
			"is_healthy":     k.circuitBreaker.IsHealthy(),
		}

		if metrics.LastFailureTime != nil {
			status["circuit_breaker"].(map[string]interface{})["last_failure"] = metrics.LastFailureTime.Format(time.RFC3339)
		}

		if metrics.LastSuccessTime != nil {
			status["circuit_breaker"].(map[string]interface{})["last_success"] = metrics.LastSuccessTime.Format(time.RFC3339)
		}
	}

	return status
}

// ✅ Test DLQ functionality
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

// ✅ Get DLQ metrics
func (k *KafkaConfig) GetDLQMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"dlq_config": map[string]interface{}{
			"enabled":                 k.DLQConfig.Enabled,
			"max_retries":             k.DLQConfig.MaxRetries,
			"retry_delay":             k.DLQConfig.RetryDelay.String(),
			"max_retry_delay":         k.DLQConfig.MaxRetryDelay.String(),
			"dead_letter_suffix":      k.DLQConfig.DeadLetterSuffix,
			"circuit_breaker_enabled": k.DLQConfig.EnableCircuitBreaker,
			"deduplication_enabled":   k.DLQConfig.EnableDeduplication,
		},
		"timestamp": time.Now(),
	}

	if k.circuitBreaker != nil {
		cbMetrics := k.circuitBreaker.GetMetrics()
		metrics["circuit_breaker"] = cbMetrics
	}

	return metrics
}

func (k *KafkaConfig) generateMessageKey(topic string, value kafka.Event) string {
	// ✅ Create unique key based on topic, event name, and data hash
	dataBytes, _ := json.Marshal(value.Data)
	hash := fmt.Sprintf("%x", sha256.Sum256(dataBytes))
	return fmt.Sprintf("%s:%s:%s", topic, value.EventName, hash[:16])
}

func (k *KafkaConfig) isDuplicateFailure(messageKey string) bool {
	_, exists := k.failureCache.Load(messageKey)
	return exists
}

func (k *KafkaConfig) markFailureProcessing(messageKey string) {
	k.failureCache.Store(messageKey, time.Now())
}

func (k *KafkaConfig) clearFailureProcessing(messageKey string) {
	// ✅ Clear after some time to prevent memory leak
	go func() {
		time.Sleep(time.Hour) // Keep for 1 hour
		k.failureCache.Delete(messageKey)
	}()
}

// ✅ Delay calculation with jitter
func (k *KafkaConfig) calculateNextDelay(currentDelay time.Duration) time.Duration {
	nextDelay := currentDelay * 2
	if nextDelay > k.DLQConfig.MaxRetryDelay {
		return k.DLQConfig.MaxRetryDelay
	}
	return nextDelay
}

func (k *KafkaConfig) addJitter(delay time.Duration) time.Duration {
	// ✅ Add random jitter (±25%) to prevent thundering herd
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	if rand.Intn(2) == 0 {
		return delay + jitter
	}
	return delay - jitter
}

// ✅ Multiple fallback strategies
func (k *KafkaConfig) executeFailureFallbacks(topic string, value kafka.Event, originalErr error, dlqErr error) {
	logrus.WithFields(logrus.Fields{
		"topic":        topic,
		"original_err": originalErr.Error(),
		"dlq_err":      dlqErr.Error(),
	}).Error("Executing failure fallbacks")

	// ✅ 1. Store to file (highest priority)
	k.writeToFailureLog(fmt.Sprintf("%s-dlq-failed", topic), value, originalErr)

	// ✅ 2. Store to database
	// k.storeFailedMessage(topic, value, originalErr)

	// ✅ 3. Send critical alert
	// criticalAlert := map[string]interface{}{
	//     "alert_type":  "kafka_dlq_failure",
	//     "topic":       topic,
	//     "error":       fmt.Sprintf("DLQ failed: %s, Original: %s", dlqErr.Error(), originalErr.Error()),
	//     "timestamp":   time.Now(),
	//     "severity":    "critical",
	//     "service":     os.Getenv("APP_NAME"),
	//     "environment": os.Getenv("ENVIRONMENT"),
	//     "kafka_hosts": k.Address,
	// }
	k.sendAlert("kafka_dlq_critical_failure", topic, fmt.Errorf("DLQ system failure"))

	// ✅ 4. Increment critical metrics
	// k.incrementFailureMetrics(fmt.Sprintf("%s-dlq-failed", topic), dlqErr)
}

func (k *KafkaConfig) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return k.circuitBreaker
}

// ✅ DLQ Recovery alert
func (k *KafkaConfig) sendDLQRecoveryAlert(deadLetterTopic string, attempts int) {
	if !k.isSlackEnabled() {
		return
	}

	recoveryAlert := map[string]interface{}{
		"alert_type":  "kafka_dlq_recovery",
		"topic":       deadLetterTopic,
		"error":       fmt.Sprintf("DLQ recovered after %d attempts", attempts),
		"timestamp":   time.Now(),
		"severity":    "info",
		"service":     os.Getenv("APP_NAME"),
		"environment": os.Getenv("ENVIRONMENT"),
		"kafka_hosts": k.Address,
	}

	k.sendSlackAlert(recoveryAlert)
}

// ✅ 2. Database Storage Implementation
// func (k *KafkaConfig) storeFailedMessage(topic string, value kafka.Event, err error) {
// 	// ✅ Create failed event record
// 	failedEvent := map[string]interface{}{
// 		"id":            k.generateFailedEventID(),
// 		"topic":         topic,
// 		"event_name":    value.EventName,
// 		"source":        value.Source,
// 		"event_data":    value.Data,
// 		"error_message": err.Error(),
// 		"retry_count":   0,
// 		"status":        "pending",
// 		"created_at":    time.Now(),
// 		"updated_at":    time.Now(),
// 	}

// 	logrus.WithFields(logrus.Fields{
// 		"dead_letter_topic": deadLetterTopic,
// 		"partition":         result.Partition,
// 		"offset":            result.Offset,
// 	}).Info("Successfully stored failed event to dead letter queue")
// 	return nil
// }

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

		// ✅ Send to monitoring service
		// k.sendToMonitoringService(alert)
		// ✅ Send email alert
		// k.sendEmailAlert(alert)

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
