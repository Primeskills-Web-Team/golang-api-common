package helpers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
)

// GetEnvOrDefault returns environment variable value or default if not set
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GenerateMessageKey creates unique key based on topic, event name, and data hash
func GenerateMessageKey(topic string, value kafka.Event) string {
	dataBytes, _ := json.Marshal(value.Data)
	hash := fmt.Sprintf("%x", sha256.Sum256(dataBytes))
	return fmt.Sprintf("%s:%s:%s", topic, value.EventName, hash[:16])
}

// GenerateFailedEventID creates unique ID for failed events
func GenerateFailedEventID() string {
	return fmt.Sprintf("failed_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// GetErrorType categorizes error types based on error message
func GetErrorType(err error) string {
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

// DetermineAlertSeverity determines alert severity based on error type
func DetermineAlertSeverity(err error) string {
	errorStr := err.Error()

	// Critical errors
	if strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "no such host") ||
		strings.Contains(errorStr, "network unreachable") {
		return "critical"
	}

	// Warning errors
	if strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "context deadline exceeded") {
		return "warning"
	}

	// Default to info
	return "info"
}

// MaskWebhookURL masks sensitive webhook URL for logging
func MaskWebhookURL(url string) string {
	if len(url) > 30 {
		return url[:30] + "***MASKED***"
	}
	return "***MASKED***"
}

// IsConnectionError checks if error is connection-related
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	connectionErrors := []string{
		"connection refused",
		"no such host",
		"network unreachable",
		"timeout",
		"broken pipe",
		"connection reset",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errorStr, connErr) {
			return true
		}
	}

	return false
}

// WriteToFailureLog writes failed events to file for later processing
func WriteToFailureLog(topic string, value kafka.Event, err error, kafkaAddresses []string, kafkaUsername string) {
	logEntry := map[string]interface{}{
		"id":        GenerateFailedEventID(),
		"timestamp": time.Now().Format(time.RFC3339),
		"topic":     topic,
		"event":     value,
		"error":     err.Error(),
		"kafka_config": map[string]interface{}{
			"addresses": kafkaAddresses,
			"username":  kafkaUsername,
		},
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Note: Import logrus in your main package and pass logger if needed
				fmt.Printf("Panic while writing to failure log: %v\n", r)
			}
		}()

		logDir := "storage/kafka_failures"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create failure log directory: %v\n", err)
			return
		}

		fileName := fmt.Sprintf("%s/kafka_failures_%s_%s.jsonl",
			logDir, topic, time.Now().Format("2006-01-02"))

		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Failed to open failure log file: %v\n", err)
			return
		}
		defer file.Close()

		jsonData, err := json.Marshal(logEntry)
		if err != nil {
			fmt.Printf("Failed to marshal failure log entry: %v\n", err)
			return
		}

		if _, err := file.Write(append(jsonData, '\n')); err != nil {
			fmt.Printf("Failed to write to failure log file: %v\n", err)
		}
	}()
}
