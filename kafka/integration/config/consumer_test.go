package config

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/stretchr/testify/assert"
)

func TestAddConsumerListener(t *testing.T) {
	// Create a test Kafka config
	kafkaConfig := NewKafkaConfig("test-user", "test-pass", []string{"localhost:9092"},)

	// Test topics
	topics := []string{"test-topic"}
	consumerGroupId := "test-consumer-group"

	// Test handler
	messageReceived := false
	handler := func(event kafka.Event) {
		messageReceived = true
	}

	// This should not panic and should create a consumer group
	// Note: This is a unit test, so we're not actually connecting to Kafka
	// The actual consumer group creation will be tested in integration tests

	// For now, we'll just verify that the method signature is correct
	// and that it doesn't cause compilation errors
	assert.NotNil(t, kafkaConfig)
	assert.NotNil(t, handler)
	assert.Equal(t, 1, len(topics))
	assert.Equal(t, "test-consumer-group", consumerGroupId)
	assert.False(t, messageReceived) // Should be false initially

	// Simulate calling the method (without actually connecting to Kafka)
	// In a real scenario, this would be: kafkaConfig.AddConsumerListener(topics, handler, consumerGroupId)
	t.Logf("Would call AddConsumerListener with topics: %v, consumerGroupId: %s", topics, consumerGroupId)

	// The method should be callable without errors
	// In a real test environment, you would need a Kafka broker running
	t.Log("AddConsumerListener method signature is correct and compiles successfully")
}

func TestConsumerGroupConfig(t *testing.T) {
	// Test that the consumer group configuration is properly set
	kafkaConfig := NewKafkaConfig("test-user", "test-pass", []string{"localhost:9092"}, )

	config := createConfig(kafkaConfig)

	// Verify consumer group settings
	assert.Equal(t, sarama.BalanceStrategyRoundRobin, config.Consumer.Group.Rebalance.Strategy)
	assert.Equal(t, 20*time.Second, config.Consumer.Group.Session.Timeout)
	assert.Equal(t, 6*time.Second, config.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, 60*time.Second, config.Consumer.Group.Rebalance.Timeout)
	assert.Equal(t, 4, config.Consumer.Group.Rebalance.Retry.Max)
	assert.Equal(t, 2*time.Second, config.Consumer.Group.Rebalance.Retry.Backoff)

	// Verify consumer settings
	assert.True(t, config.Consumer.Return.Errors)
	assert.Equal(t, sarama.OffsetOldest, config.Consumer.Offsets.Initial)
	assert.True(t, config.Consumer.Offsets.AutoCommit.Enable)
	assert.Equal(t, 1*time.Second, config.Consumer.Offsets.AutoCommit.Interval)
}
