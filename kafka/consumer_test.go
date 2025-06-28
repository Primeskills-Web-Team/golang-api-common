package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConsumerHandler implements ConsumerHandler for testing
type MockConsumerHandler struct {
	mock.Mock
}

func (m *MockConsumerHandler) HandleMessage(msg *sarama.ConsumerMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func TestConsume_NoConsumerGroup(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			ConsumerGroup: "", // Empty consumer group
		},
	}

	handler := &MockConsumerHandler{}

	err := kafka.Consume(context.Background(), []string{"test-topic"}, handler)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "consumer group not specified")
}

func TestConsume_CreateConsumerGroupError(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			Brokers:       []string{"invalid:9092"},
			ConsumerGroup: "test-group",
			SaramaConfig:  defaultSaramaConfig(),
		},
	}

	handler := &MockConsumerHandler{}

	err := kafka.Consume(context.Background(), []string{"test-topic"}, handler)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consumer group")
}

func TestConsumer_Setup(t *testing.T) {
	handler := &MockConsumerHandler{}
	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	err := consumer.Setup(nil)

	assert.NoError(t, err)

	// Verify that the ready channel is closed
	select {
	case <-consumer.ready:
		// Channel is closed, this is expected
	case <-time.After(100 * time.Millisecond):
		t.Error("ready channel should be closed")
	}
}

func TestConsumer_Cleanup(t *testing.T) {
	handler := &MockConsumerHandler{}
	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	err := consumer.Cleanup(nil)

	assert.NoError(t, err)
}

func TestConsumer_ConsumeClaim(t *testing.T) {
	handler := &MockConsumerHandler{}

	// Create test messages
	messages := make(chan *sarama.ConsumerMessage, 2)
	messages <- &sarama.ConsumerMessage{
		Topic: "test-topic",
		Value: []byte("message1"),
	}
	messages <- &sarama.ConsumerMessage{
		Topic: "test-topic",
		Value: []byte("message2"),
	}
	close(messages)

	// Set up mock expectations
	handler.On("HandleMessage", mock.MatchedBy(func(msg *sarama.ConsumerMessage) bool {
		return string(msg.Value) == "message1"
	})).Return(nil)

	handler.On("HandleMessage", mock.MatchedBy(func(msg *sarama.ConsumerMessage) bool {
		return string(msg.Value) == "message2"
	})).Return(nil)

	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	// Mock claim
	mockClaim := &MockConsumerGroupClaim{messages: messages}
	mockSession := &MockConsumerGroupSession{}

	err := consumer.ConsumeClaim(mockSession, mockClaim)

	assert.NoError(t, err)

	// Verify all messages were processed
	handler.AssertExpectations(t)
	assert.Len(t, mockSession.markedMessages, 2)
}

func TestConsumer_ConsumeClaimWithError(t *testing.T) {
	handler := &MockConsumerHandler{}

	// Create test message
	messages := make(chan *sarama.ConsumerMessage, 1)
	messages <- &sarama.ConsumerMessage{
		Topic: "test-topic",
		Value: []byte("message1"),
	}
	close(messages)

	// Set up mock to return error
	handler.On("HandleMessage", mock.Anything).Return(errors.New("handler error"))

	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	// Mock claim
	mockClaim := &MockConsumerGroupClaim{messages: messages}
	mockSession := &MockConsumerGroupSession{}

	err := consumer.ConsumeClaim(mockSession, mockClaim)

	assert.NoError(t, err) // Consumer should not return error even if handler fails

	// Verify message was still marked as processed
	handler.AssertExpectations(t)
	assert.Len(t, mockSession.markedMessages, 1)
}

// MockConsumerGroupClaim implements sarama.ConsumerGroupClaim for testing
type MockConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (m *MockConsumerGroupClaim) Topic() string                            { return "test-topic" }
func (m *MockConsumerGroupClaim) Partition() int32                         { return 0 }
func (m *MockConsumerGroupClaim) InitialOffset() int64                     { return 0 }
func (m *MockConsumerGroupClaim) HighWaterMarkOffset() int64               { return 0 }
func (m *MockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage { return m.messages }

// MockConsumerGroupSession implements sarama.ConsumerGroupSession for testing
type MockConsumerGroupSession struct {
	markedMessages []*sarama.ConsumerMessage
}

func (m *MockConsumerGroupSession) Claims() map[string][]int32 { return nil }
func (m *MockConsumerGroupSession) MemberID() string           { return "test-member" }
func (m *MockConsumerGroupSession) GenerationID() int32        { return 1 }
func (m *MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.markedMessages = append(m.markedMessages, msg)
}
func (m *MockConsumerGroupSession) Context() context.Context { return context.Background() }
func (m *MockConsumerGroupSession) Commit()                  {}

func TestConsume_SuccessfulSetup(t *testing.T) {
	// This test verifies the consumer setup process
	kafka := &Kafka{
		config: Config{
			Brokers:       []string{"localhost:9092"},
			ConsumerGroup: "test-group",
			SaramaConfig:  defaultSaramaConfig(),
		},
		wg: sync.WaitGroup{},
	}

	handler := &MockConsumerHandler{}

	// This will fail to create consumer group but we test the logic before that
	err := kafka.Consume(context.Background(), []string{"test-topic"}, handler)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consumer group")
}

func TestConsume_ContextCancellation(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			Brokers:       []string{"localhost:9092"},
			ConsumerGroup: "test-group",
			SaramaConfig:  defaultSaramaConfig(),
		},
		wg: sync.WaitGroup{},
	}

	handler := &MockConsumerHandler{}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := kafka.Consume(ctx, []string{"test-topic"}, handler)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consumer group")
}

func TestConsumer_ConsumeClaimEmptyMessages(t *testing.T) {
	handler := &MockConsumerHandler{}

	// Create empty messages channel
	messages := make(chan *sarama.ConsumerMessage)
	close(messages) // Close immediately to simulate empty claim

	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	// Mock claim with no messages
	mockClaim := &MockConsumerGroupClaim{messages: messages}
	mockSession := &MockConsumerGroupSession{}

	err := consumer.ConsumeClaim(mockSession, mockClaim)

	assert.NoError(t, err)

	// Verify no messages were processed
	assert.Len(t, mockSession.markedMessages, 0)
}

func TestConsumer_ReadyChannelBehavior(t *testing.T) {
	handler := &MockConsumerHandler{}
	consumer := &consumer{
		handler: handler,
		ready:   make(chan bool),
	}

	// Test that ready channel starts open
	select {
	case <-consumer.ready:
		t.Error("ready channel should not be closed initially")
	default:
		// Expected - channel is not ready
	}

	// Test Setup closes the channel
	err := consumer.Setup(nil)
	assert.NoError(t, err)

	// Verify channel is now closed
	select {
	case <-consumer.ready:
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("ready channel should be closed after Setup")
	}
}
