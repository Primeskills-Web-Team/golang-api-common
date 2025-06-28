package kafka

import (
	"encoding/json"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_JSONSerialization(t *testing.T) {
	event := Event{
		EventName: "test-event",
		Token:     "test-token",
		Header:    "test-header",
		Data:      map[string]interface{}{"key": "value", "number": 42},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(event)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, event.EventName, unmarshaled.EventName)
	assert.Equal(t, event.Token, unmarshaled.Token)
	assert.Equal(t, event.Header, unmarshaled.Header)

	// Handle the fact that JSON unmarshaling converts numbers to float64
	expectedData := map[string]interface{}{"key": "value", "number": float64(42)}
	assert.Equal(t, expectedData, unmarshaled.Data)
}

func TestEvent_EmptyEvent(t *testing.T) {
	event := Event{}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(event)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify empty values
	assert.Equal(t, "", unmarshaled.EventName)
	assert.Equal(t, "", unmarshaled.Token)
	assert.Equal(t, "", unmarshaled.Header)
	assert.Nil(t, unmarshaled.Data)
}

func TestEvent_WithComplexData(t *testing.T) {
	complexData := map[string]interface{}{
		"string":  "value",
		"number":  42,
		"boolean": true,
		"array":   []interface{}{1, 2, 3},
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}

	event := Event{
		EventName: "complex-event",
		Token:     "token",
		Header:    "header",
		Data:      complexData,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(event)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify complex data structure
	assert.Equal(t, event.EventName, unmarshaled.EventName)
	assert.Equal(t, event.Token, unmarshaled.Token)
	assert.Equal(t, event.Header, unmarshaled.Header)

	// Verify complex data (JSON unmarshaling converts numbers to float64)
	dataMap := unmarshaled.Data.(map[string]interface{})
	assert.Equal(t, "value", dataMap["string"])
	assert.Equal(t, float64(42), dataMap["number"]) // JSON numbers become float64
	assert.Equal(t, true, dataMap["boolean"])

	// Verify array
	array := dataMap["array"].([]interface{})
	assert.Len(t, array, 3)
	assert.Equal(t, float64(1), array[0])
	assert.Equal(t, float64(2), array[1])
	assert.Equal(t, float64(3), array[2])

	// Verify nested object
	nested := dataMap["nested"].(map[string]interface{})
	assert.Equal(t, "value", nested["inner"])
}

func TestConsumerHandler_Interface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var handler ConsumerHandler = &MockConsumerHandler{}

	// Create a test message
	msg := &sarama.ConsumerMessage{
		Topic: "test-topic",
		Value: []byte("test-value"),
	}

	// This should compile and work
	mockHandler := handler.(*MockConsumerHandler)
	mockHandler.On("HandleMessage", msg).Return(nil)

	err := handler.HandleMessage(msg)
	assert.NoError(t, err)

	mockHandler.AssertExpectations(t)
}

func TestEvent_JSONTags(t *testing.T) {
	// Verify that JSON tags are working correctly
	event := Event{
		EventName: "test-event",
		Token:     "test-token",
		Header:    "test-header",
		Data:      "test-data",
	}

	jsonBytes, err := json.Marshal(event)
	require.NoError(t, err)

	// Parse as raw JSON to verify field names
	var rawJSON map[string]interface{}
	err = json.Unmarshal(jsonBytes, &rawJSON)
	require.NoError(t, err)

	// Verify JSON field names match the tags
	assert.Contains(t, rawJSON, "event_name")
	assert.Contains(t, rawJSON, "token")
	assert.Contains(t, rawJSON, "header")
	assert.Contains(t, rawJSON, "data")

	assert.Equal(t, "test-event", rawJSON["event_name"])
	assert.Equal(t, "test-token", rawJSON["token"])
	assert.Equal(t, "test-header", rawJSON["header"])
	assert.Equal(t, "test-data", rawJSON["data"])
}
