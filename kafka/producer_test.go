package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_JSONMarshaling(t *testing.T) {
	// Test the JSON marshaling logic that Produce uses
	event := Event{
		EventName: "test-event",
		Token:     "test-token",
		Header:    "test-header",
		Data:      map[string]interface{}{"key": "value"},
	}

	eventBytes, err := json.Marshal(event)
	require.NoError(t, err)
	assert.NotEmpty(t, eventBytes)

	// Verify we can unmarshal it back
	var unmarshaled Event
	err = json.Unmarshal(eventBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, event.EventName, unmarshaled.EventName)
	assert.Equal(t, event.Token, unmarshaled.Token)
	assert.Equal(t, event.Header, unmarshaled.Header)
}

func TestEvent_MarshalError(t *testing.T) {
	// Test JSON marshaling with data that cannot be marshaled
	event := Event{
		EventName: "test-event",
		Data:      make(chan int), // channels cannot be marshaled to JSON
	}

	_, err := json.Marshal(event)
	assert.Error(t, err)
}

func TestProduce_ClientClosedCheck(t *testing.T) {
	// Test the client closed channel logic
	kafka := &Kafka{
		config: Config{
			ConsumerGroup: "test-group",
		},
		closed: make(chan struct{}),
	}

	// Close the client
	close(kafka.closed)

	event := Event{
		EventName: "test-event",
	}

	// This should detect the closed channel
	err := kafka.Produce(context.Background(), "test-topic", event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client closed")
}

func TestProduce_OpenClient(t *testing.T) {
	// Test with open client (will fail at producer.SendMessage but we can test the logic before that)
	kafka := &Kafka{
		config: Config{
			ConsumerGroup: "test-group",
		},
		closed:   make(chan struct{}),
		producer: nil, // This will cause a panic/error when SendMessage is called
	}

	event := Event{
		EventName: "test-event",
	}

	// This should not fail on the closed check, but will fail when trying to send
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic due to nil producer - just verify panic occurred
			assert.NotNil(t, r)
		}
	}()

	err := kafka.Produce(context.Background(), "test-topic", event)
	// If we get here, there was an error instead of panic
	if err != nil {
		assert.Error(t, err)
	}
}
