package config

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	kafkaproducer "github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/producer"
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
	}

	return nil, fmt.Errorf("failed to publish after %d attempts, last error: %w", retryConfig.MaxRetries+1, lastErr)
}

// publishEventOnce performs single publish attempt
func (k *KafkaConfig) publishEventOnce(topic string, value kafka.Event) (*PublishResult, error) {
	syncProducer, err := sarama.NewSyncProducer(k.Address, createConfig(k))
	if err != nil {
		return nil, fmt.Errorf("unable to create kafka producer: %w", err)
	}
	defer func() {
		if closeErr := syncProducer.Close(); closeErr != nil {
			logrus.Errorf("Unable to close kafka producer: %v", closeErr)
		}
	}()

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
		// Here you can add additional error handling like:
		// - Store to dead letter queue
		// - Send to monitoring system
		// - Store to database for manual retry
		k.handlePublishFailure(topic, value, err)
		return
	}

	logrus.Infof("Successfully published event to topic %s, partition %d, offset %d", 
		topic, result.Partition, result.Offset)
}

// handlePublishFailure handles final publish failures
func (k *KafkaConfig) handlePublishFailure(topic string, value kafka.Event, err error) {
	// Example implementations:
	
	// 1. Store to dead letter queue
	deadLetterTopic := fmt.Sprintf("%s-dead-letter", topic)
	logrus.Warnf("Storing failed message to dead letter topic: %s", deadLetterTopic)
	
	// 2. Store to database for manual processing
	// k.storeFailedMessage(topic, value, err)
	
	// 3. Send to monitoring/alerting system
	// k.sendAlert("kafka_publish_failed", topic, err)
	
	// 4. Write to file for later processing
	// k.writeToFailureLog(topic, value, err)
}

// Batch publish method for better performance
func (k *KafkaConfig) PublishEventsBatch(ctx context.Context, topic string, events []kafka.Event) ([]*PublishResult, error) {
	syncProducer, err := sarama.NewSyncProducer(k.Address, createConfig(k))
	if err != nil {
		return nil, fmt.Errorf("unable to create kafka producer: %w", err)
	}
	defer func() {
		if closeErr := syncProducer.Close(); closeErr != nil {
			logrus.Errorf("Unable to close kafka producer: %v", closeErr)
		}
	}()

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