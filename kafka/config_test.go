package kafka

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStorage implements the storage interface for testing
type MockStorage struct {
	data map[string][]byte
	err  error
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockStorage) Get(key string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	data, exists := m.data[key]
	if !exists {
		return nil, nil
	}
	return data, nil
}

func (m *MockStorage) Set(key string, val []byte, exp time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = val
	return nil
}

func (m *MockStorage) Delete(key string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.data, key)
	return nil
}

func (m *MockStorage) Reset() error {
	if m.err != nil {
		return m.err
	}
	m.data = make(map[string][]byte)
	return nil
}

func (m *MockStorage) Close() error {
	return m.err
}

func (m *MockStorage) SetError(err error) {
	m.err = err
}

func TestDefaultSaramaConfig(t *testing.T) {
	config := defaultSaramaConfig()

	assert.NotNil(t, config)
	assert.True(t, config.Producer.Return.Successes)
	assert.True(t, config.Producer.Return.Errors)
	assert.True(t, config.Consumer.Return.Errors)
	assert.Equal(t, sarama.V2_8_0_0, config.Version)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig

	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
	assert.Equal(t, 10*time.Second, config.ProducerTimeout)
	assert.Equal(t, "default-group", config.ConsumerGroup)
	assert.Equal(t, 10*time.Second, config.ConsumerTimeout)
	assert.NotNil(t, config.SaramaConfig)
	assert.False(t, config.WithDlq)
	assert.Nil(t, config.Storage)
}

func TestSetConfig_NoConfig(t *testing.T) {
	config, err := setConfig()

	require.NoError(t, err)
	assert.Equal(t, DefaultConfig.Brokers, config.Brokers)
	assert.Equal(t, DefaultConfig.ProducerTimeout, config.ProducerTimeout)
	assert.Equal(t, DefaultConfig.ConsumerGroup, config.ConsumerGroup)
	assert.Equal(t, DefaultConfig.ConsumerTimeout, config.ConsumerTimeout)
	assert.NotNil(t, config.SaramaConfig)
	assert.False(t, config.WithDlq)
	assert.Nil(t, config.Storage)
}

func TestSetConfig_EmptyConfig(t *testing.T) {
	config, err := setConfig(Config{})

	require.NoError(t, err)
	assert.Equal(t, DefaultConfig.Brokers, config.Brokers)
	assert.Equal(t, DefaultConfig.ProducerTimeout, config.ProducerTimeout)
	assert.Equal(t, DefaultConfig.ConsumerGroup, config.ConsumerGroup)
	assert.Equal(t, DefaultConfig.ConsumerTimeout, config.ConsumerTimeout)
	assert.NotNil(t, config.SaramaConfig)
}

func TestSetConfig_PartialConfig(t *testing.T) {
	inputConfig := Config{
		Brokers:         []string{"custom:9092"},
		ProducerTimeout: 5 * time.Second,
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.Equal(t, []string{"custom:9092"}, config.Brokers)
	assert.Equal(t, 5*time.Second, config.ProducerTimeout)
	assert.Equal(t, DefaultConfig.ConsumerGroup, config.ConsumerGroup)
	assert.Equal(t, DefaultConfig.ConsumerTimeout, config.ConsumerTimeout)
	assert.NotNil(t, config.SaramaConfig)
}

func TestSetConfig_CustomSaramaConfig(t *testing.T) {
	customSarama := sarama.NewConfig()
	customSarama.Producer.RequiredAcks = sarama.WaitForAll

	inputConfig := Config{
		SaramaConfig: customSarama,
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.Equal(t, customSarama, config.SaramaConfig)
	assert.Equal(t, sarama.WaitForAll, config.SaramaConfig.Producer.RequiredAcks)
}

func TestSetConfig_DLQEnabled_WithStorage(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage

	inputConfig := Config{
		WithDlq: true,
		Storage: &storageInterface,
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.True(t, config.WithDlq)
	assert.NotNil(t, config.Storage)
}

func TestSetConfig_DLQEnabled_WithoutStorage(t *testing.T) {
	inputConfig := Config{
		WithDlq: true,
		Storage: nil,
	}

	config, err := setConfig(inputConfig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ is enabled but no storage configured")
	assert.Equal(t, Config{}, config)
}

func TestSetConfig_DLQDisabled_WithoutStorage(t *testing.T) {
	inputConfig := Config{
		WithDlq: false,
		Storage: nil,
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.False(t, config.WithDlq)
	assert.Nil(t, config.Storage)
}

func TestSetConfig_AllFields(t *testing.T) {
	mockStorage := NewMockStorage()
	var storageInterface storage.Storage = mockStorage
	customSarama := sarama.NewConfig()

	inputConfig := Config{
		Brokers:         []string{"broker1:9092", "broker2:9092"},
		ProducerTimeout: 15 * time.Second,
		ConsumerGroup:   "test-group",
		ConsumerTimeout: 20 * time.Second,
		SaramaConfig:    customSarama,
		WithDlq:         true,
		Storage:         &storageInterface,
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.Equal(t, inputConfig.Brokers, config.Brokers)
	assert.Equal(t, inputConfig.ProducerTimeout, config.ProducerTimeout)
	assert.Equal(t, inputConfig.ConsumerGroup, config.ConsumerGroup)
	assert.Equal(t, inputConfig.ConsumerTimeout, config.ConsumerTimeout)
	assert.Equal(t, inputConfig.SaramaConfig, config.SaramaConfig)
	assert.Equal(t, inputConfig.WithDlq, config.WithDlq)
	assert.Equal(t, inputConfig.Storage, config.Storage)
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test that the sarama config has proper defaults
	saramaConfig := defaultSaramaConfig()
	assert.NotNil(t, saramaConfig)

	// Test that DefaultConfig is properly initialized
	assert.NotEmpty(t, DefaultConfig.Brokers)
	assert.Greater(t, DefaultConfig.ProducerTimeout, time.Duration(0))
	assert.NotEmpty(t, DefaultConfig.ConsumerGroup)
	assert.Greater(t, DefaultConfig.ConsumerTimeout, time.Duration(0))
}

func TestSetConfig_ZeroValues(t *testing.T) {
	inputConfig := Config{
		Brokers:         []string{}, // Empty slice should be replaced with default
		ProducerTimeout: 0,          // Zero should be replaced with default
		ConsumerGroup:   "",         // Empty string should be replaced with default
		ConsumerTimeout: 0,          // Zero should be replaced with default
		SaramaConfig:    nil,        // Nil should be replaced with default
	}

	config, err := setConfig(inputConfig)

	require.NoError(t, err)
	assert.Equal(t, DefaultConfig.Brokers, config.Brokers)
	assert.Equal(t, DefaultConfig.ProducerTimeout, config.ProducerTimeout)
	assert.Equal(t, DefaultConfig.ConsumerGroup, config.ConsumerGroup)
	assert.Equal(t, DefaultConfig.ConsumerTimeout, config.ConsumerTimeout)
	assert.NotNil(t, config.SaramaConfig)
}
