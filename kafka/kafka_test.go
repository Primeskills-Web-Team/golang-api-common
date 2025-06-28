package kafka

import (
	"testing"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ConfigError(t *testing.T) {
	config := Config{
		WithDlq: true,
		Storage: nil, // This should cause a config error
	}

	client, err := New(config)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to set config")
}

func TestNew_DefaultConfig(t *testing.T) {
	// Test with no config provided - should use defaults
	client, err := New()

	// We expect an error because no Kafka broker is running
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNew_ValidConfig(t *testing.T) {
	config := Config{
		Brokers: []string{"localhost:9092"},
		AppName: "test-group",
	}

	// This should fail with connection error, but we can verify the error type
	client, err := New(config)

	// We expect an error because no Kafka broker is running
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to create sync producer")
}

func TestKafka_CloseNilComponents(t *testing.T) {
	kafka := &Kafka{
		config: DefaultConfig,
		closed: make(chan struct{}),
	}

	err := kafka.Close()

	assert.NoError(t, err)

	// Verify that the closed channel is closed
	select {
	case <-kafka.closed:
		// Channel is closed, this is expected
	default:
		t.Error("closed channel should be closed")
	}
}

func TestKafka_InitialState(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	config := Config{
		Brokers: []string{"test:9092"},
		AppName: "test-group",
		WithDlq: true,
		Storage: &storageInterface,
	}

	// We can test the config validation
	validatedConfig, err := setConfig(config)
	require.NoError(t, err)

	assert.Equal(t, config.Brokers, validatedConfig.Brokers)
	assert.Equal(t, config.AppName, validatedConfig.AppName)
	assert.True(t, validatedConfig.WithDlq)
	assert.NotNil(t, validatedConfig.Storage)
}

func TestKafka_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config without DLQ",
			config: Config{
				Brokers: []string{"localhost:9092"},
				AppName: "test-group",
				WithDlq: false,
			},
			expectError: false,
		},
		{
			name: "Valid config with DLQ and storage",
			config: Config{
				Brokers: []string{"localhost:9092"},
				AppName: "test-group",
				WithDlq: true,
				Storage: func() *storage.Storage { s := NewMockStorage(); var iface storage.Storage = s; return &iface }(),
			},
			expectError: false,
		},
		{
			name: "Invalid config - DLQ without storage",
			config: Config{
				Brokers: []string{"localhost:9092"},
				AppName: "test-group",
				WithDlq: true,
				Storage: nil,
			},
			expectError: true,
			errorMsg:    "DLQ is enabled but no storage configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := setConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
