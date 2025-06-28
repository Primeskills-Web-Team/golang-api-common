package kafka

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendToDLQ_DLQDisabled(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			WithDlq: false,
		},
	}

	msg := &sarama.ConsumerMessage{
		Topic: "test-topic",
	}
	err := errors.New("processing error")

	result := kafka.SendToDLQ(msg, err)

	assert.NoError(t, result)
}

func TestSendToDLQ_Success(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			WithDlq:       true,
			Storage:       &storageInterface,
			ConsumerGroup: "test-group",
		},
	}

	now := time.Now()
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 1,
		Offset:    123,
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Timestamp: now,
		Headers: []*sarama.RecordHeader{
			{Key: []byte("header1"), Value: []byte("value1")},
			{Key: []byte("header2"), Value: []byte("value2")},
		},
	}
	processingErr := errors.New("processing failed")

	err := kafka.SendToDLQ(msg, processingErr)

	assert.NoError(t, err)

	// Verify that exactly one message was stored
	assert.Len(t, mockStorage.data, 1)

	// Get the stored message and verify its content
	var storedKey string
	var storedValue []byte
	for k, v := range mockStorage.data {
		storedKey = k
		storedValue = v
		break
	}

	// Verify the key format
	assert.Contains(t, storedKey, "dlq:test-group:")

	// Verify the stored message content
	var dlqMsg DLQMessage
	err = json.Unmarshal(storedValue, &dlqMsg)
	require.NoError(t, err)

	assert.Equal(t, "test-topic", dlqMsg.OriginalTopic)
	assert.Equal(t, int32(1), dlqMsg.Partition)
	assert.Equal(t, int64(123), dlqMsg.Offset)
	assert.Equal(t, []byte("test-key"), dlqMsg.Key)
	assert.Equal(t, []byte("test-value"), dlqMsg.Value)
	assert.Equal(t, "processing failed", dlqMsg.Error)
	assert.WithinDuration(t, now, dlqMsg.Timestamp, time.Second)
	assert.WithinDuration(t, time.Now(), dlqMsg.FailedAt, time.Second)

	// Verify headers
	expectedHeaders := map[string]string{
		"header1": "value1",
		"header2": "value2",
	}
	assert.Equal(t, expectedHeaders, dlqMsg.Headers)
}

func TestSendToDLQ_WithNilHeaders(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			WithDlq:       true,
			Storage:       &storageInterface,
			ConsumerGroup: "test-group",
		},
	}

	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 1,
		Offset:    123,
		Headers: []*sarama.RecordHeader{
			nil, // nil header should be skipped
			{Key: []byte("header1"), Value: []byte("value1")},
		},
	}
	processingErr := errors.New("processing failed")

	err := kafka.SendToDLQ(msg, processingErr)

	assert.NoError(t, err)

	// Get the stored message
	var storedValue []byte
	for _, v := range mockStorage.data {
		storedValue = v
		break
	}

	var dlqMsg DLQMessage
	err = json.Unmarshal(storedValue, &dlqMsg)
	require.NoError(t, err)

	// Should only have the non-nil header
	expectedHeaders := map[string]string{
		"header1": "value1",
	}
	assert.Equal(t, expectedHeaders, dlqMsg.Headers)
}

func TestSendToDLQ_StorageError(t *testing.T) {
	mockStorage := NewMockStorage()
	mockStorage.SetError(errors.New("storage error"))
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			WithDlq:       true,
			Storage:       &storageInterface,
			ConsumerGroup: "test-group",
		},
	}

	msg := &sarama.ConsumerMessage{
		Topic: "test-topic",
	}
	processingErr := errors.New("processing failed")

	err := kafka.SendToDLQ(msg, processingErr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store message in DLQ")
	assert.Contains(t, err.Error(), "storage error")
}

func TestGetDLQMessage_NoStorage(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			Storage: nil,
		},
	}

	msg, err := kafka.GetDLQMessage("test-key")

	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "no storage configured")
}

func TestGetDLQMessage_Success(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	// Prepare test DLQ message
	dlqMsg := DLQMessage{
		OriginalTopic: "test-topic",
		Partition:     1,
		Offset:        123,
		Key:           []byte("test-key"),
		Value:         []byte("test-value"),
		Headers:       map[string]string{"header1": "value1"},
		Error:         "test error",
		FailedAt:      time.Now(),
		Timestamp:     time.Now(),
	}

	msgBytes, err := json.Marshal(dlqMsg)
	require.NoError(t, err)

	mockStorage.data["test-dlq-key"] = msgBytes

	result, err := kafka.GetDLQMessage("test-dlq-key")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, dlqMsg.OriginalTopic, result.OriginalTopic)
	assert.Equal(t, dlqMsg.Partition, result.Partition)
	assert.Equal(t, dlqMsg.Offset, result.Offset)
	assert.Equal(t, dlqMsg.Key, result.Key)
	assert.Equal(t, dlqMsg.Value, result.Value)
	assert.Equal(t, dlqMsg.Headers, result.Headers)
	assert.Equal(t, dlqMsg.Error, result.Error)
}

func TestGetDLQMessage_NotFound(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	result, err := kafka.GetDLQMessage("non-existent-key")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetDLQMessage_StorageError(t *testing.T) {
	mockStorage := NewMockStorage()
	mockStorage.SetError(errors.New("storage error"))
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	result, err := kafka.GetDLQMessage("test-key")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get DLQ message")
}

func TestGetDLQMessage_InvalidJSON(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	// Store invalid JSON
	mockStorage.data["test-key"] = []byte("invalid json")

	result, err := kafka.GetDLQMessage("test-key")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal DLQ message")
}

func TestDeleteDLQMessage_NoStorage(t *testing.T) {
	kafka := &Kafka{
		config: Config{
			Storage: nil,
		},
	}

	err := kafka.DeleteDLQMessage("test-key")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no storage configured")
}

func TestDeleteDLQMessage_Success(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	// Add a message to storage
	mockStorage.data["test-key"] = []byte("test-data")

	err := kafka.DeleteDLQMessage("test-key")

	assert.NoError(t, err)
	assert.Empty(t, mockStorage.data)
}

func TestDeleteDLQMessage_StorageError(t *testing.T) {
	mockStorage := NewMockStorage()
	mockStorage.SetError(errors.New("storage error"))
	var storageInterface storage.Storage = mockStorage

	kafka := &Kafka{
		config: Config{
			Storage: &storageInterface,
		},
	}

	err := kafka.DeleteDLQMessage("test-key")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete DLQ message")
}

func TestDLQMessage_JSONSerialization(t *testing.T) {
	now := time.Now()

	dlqMsg := DLQMessage{
		OriginalTopic: "test-topic",
		Partition:     1,
		Offset:        123,
		Key:           []byte("test-key"),
		Value:         []byte("test-value"),
		Headers:       map[string]string{"header1": "value1"},
		Error:         "test error",
		FailedAt:      now,
		Timestamp:     now,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(dlqMsg)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled DLQMessage
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, dlqMsg.OriginalTopic, unmarshaled.OriginalTopic)
	assert.Equal(t, dlqMsg.Partition, unmarshaled.Partition)
	assert.Equal(t, dlqMsg.Offset, unmarshaled.Offset)
	assert.Equal(t, dlqMsg.Key, unmarshaled.Key)
	assert.Equal(t, dlqMsg.Value, unmarshaled.Value)
	assert.Equal(t, dlqMsg.Headers, unmarshaled.Headers)
	assert.Equal(t, dlqMsg.Error, unmarshaled.Error)
	assert.True(t, dlqMsg.FailedAt.Equal(unmarshaled.FailedAt))
	assert.True(t, dlqMsg.Timestamp.Equal(unmarshaled.Timestamp))
}
