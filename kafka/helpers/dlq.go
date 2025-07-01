package helpers

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/Primeskills-Web-Team/golang-api-common/app/config"
)

// DLQHelper contains methods for Dead Letter Queue operations
type DLQHelper struct {
    failureCache sync.Map
	AppConfig    config.AppConfig
}

func (d *DLQHelper) CreateDeadLetterEvent(topic string, value kafka.Event, originalErr error, kafkaAddresses []string, kafkaUsername string) kafka.Event {
	hostname, _ := os.Hostname()

	return kafka.Event{
		EventName: fmt.Sprintf("%s_DEAD_LETTER", value.EventName),
		Source:    value.Source,
		Data: map[string]interface{}{
			"original_event":    value,
			"original_topic":    topic,
			"failure_reason":    originalErr.Error(),
			"failure_timestamp": time.Now().Unix(),
			"failure_type":      GetErrorType(originalErr),
			"retry_count":       0,
			"message_id":        GenerateMessageKey(topic, value),
			"server_info": map[string]interface{}{
				"hostname":    hostname,
				"service":     d.AppConfig.AppName,
				"environment": d.AppConfig.Environment,
				"version":     d.AppConfig.Version,
			},
			"kafka_config": map[string]interface{}{
				"brokers":  kafkaAddresses,
				"username": kafkaUsername,
			},
		},
	}
}

// NewDLQHelper creates a new DLQ helper instance
func NewDLQHelper() *DLQHelper {
    return &DLQHelper{
        failureCache: sync.Map{},
    }
}

// IsDuplicateFailure checks if message has already failed
func (d *DLQHelper) IsDuplicateFailure(messageKey string) bool {
    _, exists := d.failureCache.Load(messageKey)
    return exists
}

// MarkFailureProcessing marks message as being processed
func (d *DLQHelper) MarkFailureProcessing(messageKey string) {
    d.failureCache.Store(messageKey, time.Now())
}

// ClearFailureProcessing clears failure processing mark after delay
func (d *DLQHelper) ClearFailureProcessing(messageKey string) {
    go func() {
        time.Sleep(time.Hour) // Keep for 1 hour
        d.failureCache.Delete(messageKey)
    }()
}

func (d *DLQHelper) CreateDLQRecoveryAlert(deadLetterTopic string, attempts int, kafkaAddresses []string) map[string]interface{} {
	return map[string]interface{}{
		"alert_type":  "kafka_dlq_recovery",
		"topic":       deadLetterTopic,
		"error":       fmt.Sprintf("DLQ recovered after %d attempts", attempts),
		"timestamp":   time.Now(),
		"severity":    "info",
		"service":     d.AppConfig.AppName,
		"environment": d.AppConfig.Environment,
		"kafka_hosts": kafkaAddresses,
	}
}

func (d *DLQHelper) CreateFailureAlert(
	alertType, topic string,
	err error,
	kafkaAddresses []string,
	event map[string]interface{},
) map[string]interface{} {
	return map[string]interface{}{
		"alert_type":  alertType,
		"topic":       topic,
		"error":       err.Error(),
		"timestamp":   time.Now(),
		"severity":    DetermineAlertSeverity(err),
		"service":     d.AppConfig.AppName,
		"environment": d.AppConfig.Environment,
		"kafka_hosts": kafkaAddresses,
		"event":       event,
	}
}


// LogFailureFallback logs when fallback mechanisms are executed
func LogFailureFallback(topic string, value kafka.Event, originalErr error, dlqErr error, kafkaAddresses []string, kafkaUsername string) {
    fmt.Printf("Executing failure fallbacks for topic: %s, original_err: %v, dlq_err: %v\n", 
        topic, originalErr, dlqErr)
    
    // Write to failure log
    WriteToFailureLog(fmt.Sprintf("%s-dlq-failed", topic), value, originalErr, kafkaAddresses, kafkaUsername)
}