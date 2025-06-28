package kafka

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConsumerGroup implements sarama.ConsumerGroup for testing
type MockConsumerGroup struct {
	mock.Mock
	closed bool
}

func (m *MockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	args := m.Called(ctx, topics, handler)

	// Simulate the consumer group behavior
	if !m.closed {
		// Call Setup
		if err := handler.Setup(nil); err != nil {
			return err
		}

		// Simulate context cancellation check
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Call Cleanup
		return handler.Cleanup(nil)
	}

	return args.Error(0)
}

func (m *MockConsumerGroup) Errors() <-chan error {
	return make(<-chan error)
}

func (m *MockConsumerGroup) Close() error {
	m.closed = true
	args := m.Called()
	return args.Error(0)
}

func (m *MockConsumerGroup) Pause(partitions map[string][]int32) {
	m.Called(partitions)
}

func (m *MockConsumerGroup) Resume(partitions map[string][]int32) {
	m.Called(partitions)
}

func (m *MockConsumerGroup) PauseAll() {
	m.Called()
}

func (m *MockConsumerGroup) ResumeAll() {
	m.Called()
}

func TestConsume_FullFlow(t *testing.T) {
	// Create a kafka instance with wait group
	kafka := &Kafka{
		config: Config{
			Brokers:       []string{"localhost:9092"},
			ConsumerGroup: "test-group",
			SaramaConfig:  defaultSaramaConfig(),
		},
		wg: sync.WaitGroup{},
	}

	handler := &MockConsumerHandler{}

	// Mock the consumer group creation by modifying the consumer creation logic
	// This test will fail at the actual consumer group creation, but we can
	// verify that the initial validation passes

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := kafka.Consume(ctx, []string{"test-topic"}, handler)

	// Should fail due to no broker, but validates the consumer group check
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consumer group")
}

func TestConsume_ContextCancellationDuringConsume(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			Brokers:       []string{"localhost:9092"},
			ConsumerGroup: "test-group",
			SaramaConfig:  defaultSaramaConfig(),
		},
		wg: sync.WaitGroup{},
	}

	handler := &MockConsumerHandler{}

	// Create context that cancels quickly
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to test cancellation handling
	cancel()

	err := kafka.Consume(ctx, []string{"test-topic"}, handler)

	// Should fail due to connection error
	assert.Error(t, err)
}

func TestKafka_CloseWithComponents(t *testing.T) {
	// Create a kafka instance with mock components to test Close method
	mockProducer := &MockProducer{}
	mockConsumerGroup := &MockConsumerGroup{}

	kafka := &Kafka{
		config:        DefaultConfig,
		producer:      mockProducer,
		consumerGroup: mockConsumerGroup,
		closed:        make(chan struct{}),
		wg:            sync.WaitGroup{},
	}

	// Set up expectations
	mockProducer.On("Close").Return(nil)
	mockConsumerGroup.On("Close").Return(nil)

	err := kafka.Close()

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
	mockConsumerGroup.AssertExpectations(t)

	// Verify closed channel
	select {
	case <-kafka.closed:
		// Expected
	default:
		t.Error("closed channel should be closed")
	}
}

func TestKafka_CloseWithErrors(t *testing.T) {
	mockProducer := &MockProducer{}
	mockConsumerGroup := &MockConsumerGroup{}

	kafka := &Kafka{
		config:        DefaultConfig,
		producer:      mockProducer,
		consumerGroup: mockConsumerGroup,
		closed:        make(chan struct{}),
		wg:            sync.WaitGroup{},
	}

	// Set up expectations with errors
	mockProducer.On("Close").Return(assert.AnError)
	mockConsumerGroup.On("Close").Return(assert.AnError)

	err := kafka.Close()

	// Close should still succeed even if components fail to close
	assert.NoError(t, err)
}

func TestKafka_CloseWithWaitGroup(t *testing.T) {
	kafka := &Kafka{
		config: DefaultConfig,
		closed: make(chan struct{}),
		wg:     sync.WaitGroup{},
	}

	// Add something to wait group to test the Wait() call
	kafka.wg.Add(1)

	// Start a goroutine that will finish quickly
	go func() {
		time.Sleep(10 * time.Millisecond)
		kafka.wg.Done()
	}()

	err := kafka.Close()

	assert.NoError(t, err)
}

// MockProducer for testing Close functionality
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	args := m.Called(msg)
	return args.Get(0).(int32), args.Get(1).(int64), args.Error(2)
}

func (m *MockProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	args := m.Called(msgs)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProducer) BeginTxn() error {
	return sarama.ErrNotConnected
}

func (m *MockProducer) CommitTxn() error {
	return sarama.ErrNotConnected
}

func (m *MockProducer) AbortTxn() error {
	return sarama.ErrNotConnected
}

func (m *MockProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, groupInstanceId *string) error {
	return sarama.ErrNotConnected
}

func (m *MockProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return sarama.ErrNotConnected
}

func (m *MockProducer) IsTransactional() bool {
	return false
}

func (m *MockProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagUninitialized
}

func TestProduce_CompleteFlow(t *testing.T) {
	mockProducer := &MockProducer{}

	kafka := &Kafka{
		config: Config{
			ConsumerGroup: "test-group",
		},
		producer: mockProducer,
		closed:   make(chan struct{}),
	}

	event := Event{
		EventName: "complete-test",
		Token:     "token",
		Header:    "header",
		Data:      map[string]interface{}{"test": true},
	}

	// Set up mock expectation
	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).Return(int32(0), int64(1), nil)

	err := kafka.Produce(context.Background(), "test-topic", event)

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestProduce_ProducerSendError(t *testing.T) {
	mockProducer := &MockProducer{}

	kafka := &Kafka{
		config: Config{
			ConsumerGroup: "test-group",
		},
		producer: mockProducer,
		closed:   make(chan struct{}),
	}

	event := Event{
		EventName: "test-event",
	}

	// Set up mock to return error
	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).Return(int32(0), int64(0), assert.AnError)

	err := kafka.Produce(context.Background(), "test-topic", event)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestSendToDLQ_MissingStorage(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			WithDlq: false, // DLQ disabled, so storage check won't run
			Storage: nil,
		},
	}

	msg := &sarama.ConsumerMessage{
		Topic: "test-topic",
	}

	// This should return early since DLQ is disabled
	err := kafka.SendToDLQ(msg, assert.AnError)

	// Should not error when DLQ is disabled
	assert.NoError(t, err)
}

func TestDLQMessage_EmptyHeaders(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			WithDlq:       true,
			Storage:       &storageInterface,
			ConsumerGroup: "test-group",
		},
	}

	// Test with no headers
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    0,
		Headers:   nil, // No headers
	}

	err := kafka.SendToDLQ(msg, assert.AnError)
	assert.NoError(t, err)

	// Verify message was stored
	assert.Len(t, mockStorage.data, 1)
}
