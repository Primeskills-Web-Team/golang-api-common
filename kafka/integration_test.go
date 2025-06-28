package kafka

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDLQIntegration_FullWorkflow(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			WithDlq: true,
			Storage: &storageInterface,
			AppName: "integration-test-group",
		},
	}

	// Simulate a consumer message
	originalMsg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 2,
		Offset:    456,
		Key:       []byte("integration-key"),
		Value:     []byte("integration-value"),
		Timestamp: time.Now(),
		Headers: []*sarama.RecordHeader{
			{Key: []byte("test-header"), Value: []byte("test-value")},
		},
	}

	processingErr := assert.AnError

	// Test sending to DLQ
	err := kafka.SendToDLQ(originalMsg, processingErr)
	require.NoError(t, err)

	// Verify message was stored
	assert.Len(t, mockStorage.data, 1)

	// Get the DLQ key
	var dlqKey string
	for k := range mockStorage.data {
		dlqKey = k
		break
	}

	// Test retrieving the message
	retrievedMsg, err := kafka.GetDLQMessage(dlqKey)
	require.NoError(t, err)
	require.NotNil(t, retrievedMsg)

	// Verify the retrieved message content
	assert.Equal(t, "test-topic", retrievedMsg.OriginalTopic)
	assert.Equal(t, int32(2), retrievedMsg.Partition)
	assert.Equal(t, int64(456), retrievedMsg.Offset)
	assert.Equal(t, []byte("integration-key"), retrievedMsg.Key)
	assert.Equal(t, []byte("integration-value"), retrievedMsg.Value)
	assert.Equal(t, processingErr.Error(), retrievedMsg.Error)

	// Test deleting the message
	err = kafka.DeleteDLQMessage(dlqKey)
	require.NoError(t, err)

	// Verify message was deleted
	assert.Empty(t, mockStorage.data)

	// Verify getting deleted message returns nil
	deletedMsg, err := kafka.GetDLQMessage(dlqKey)
	require.NoError(t, err)
	assert.Nil(t, deletedMsg)
}

func TestConfigValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "Minimal valid config",
			config: Config{
				Brokers: []string{"localhost:9092"},
				AppName: "test",
			},
			valid: true,
		},
		{
			name: "Multiple brokers",
			config: Config{
				Brokers: []string{"broker1:9092", "broker2:9092", "broker3:9092"},
				AppName: "test",
			},
			valid: true,
		},
		{
			name: "Custom timeouts",
			config: Config{
				Brokers:         []string{"localhost:9092"},
				AppName:         "test",
				ProducerTimeout: 30 * time.Second,
				ConsumerTimeout: 60 * time.Second,
			},
			valid: true,
		},
		{
			name: "DLQ without storage (invalid)",
			config: Config{
				Brokers: []string{"localhost:9092"},
				AppName: "test",
				WithDlq: true,
				Storage: nil,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := setConfig(tt.config)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMockStorage_AllOperations(t *testing.T) {
	storage := NewMockStorage()

	// Test Set and Get
	err := storage.Set("key1", []byte("value1"), 0)
	assert.NoError(t, err)

	value, err := storage.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)

	// Test Get non-existent key
	value, err = storage.Get("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, value)

	// Test Delete
	err = storage.Delete("key1")
	assert.NoError(t, err)

	// Verify deletion
	value, err = storage.Get("key1")
	assert.NoError(t, err)
	assert.Nil(t, value)

	// Test Reset
	storage.Set("key1", []byte("value1"), 0)
	storage.Set("key2", []byte("value2"), 0)
	assert.Len(t, storage.data, 2)

	err = storage.Reset()
	assert.NoError(t, err)
	assert.Empty(t, storage.data)

	// Test Close
	err = storage.Close()
	assert.NoError(t, err)

	// Test error conditions
	storage.SetError(assert.AnError)

	err = storage.Set("key", []byte("value"), 0)
	assert.Error(t, err)

	_, err = storage.Get("key")
	assert.Error(t, err)

	err = storage.Delete("key")
	assert.Error(t, err)

	err = storage.Reset()
	assert.Error(t, err)

	err = storage.Close()
	assert.Error(t, err)
}
