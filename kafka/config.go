package kafka

import (
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/storage"
	telegramnotifier "github.com/Primeskills-Web-Team/golang-api-common/v2/telegram_notifier"
)

// Config holds Kafka client configuration.
type Config struct {
	// Default: []string{"localhost:9092"}
	Brokers []string

	// Default: 10 * time.Second
	ProducerTimeout time.Duration

	// Default: "default-app"
	AppName string

	// Default: 10 * time.Second
	ConsumerTimeout time.Duration

	// SaramaConfig holds the Sarama configuration.
	// Default: sarama.NewConfig() with sensible defaults
	// Set this to nil to use the default Sarama configuration.
	SaramaConfig *sarama.Config

	// If true, the client will use a dead letter queue to store messages that cannot be processed.
	// Default: false
	WithDlq bool

	// If WithDlq is true, the client will use the storage to store messages that cannot be processed.
	// Default: nil
	Storage *storage.Storage

	// If true, the client will use the alert to send alerts when there are errors.
	// Default: false
	WithAlert bool

	// If WithAlert is true, the client will use the telegram notifier to send alerts when there are errors.
	// Default: nil
	TelegramNotifier *telegramnotifier.TelegramNotifier
}

// DefaultConfig returns a default Kafka configuration.
var DefaultConfig = Config{
	Brokers:          []string{"localhost:9092"},
	ProducerTimeout:  10 * time.Second,
	AppName:          "default-app",
	ConsumerTimeout:  10 * time.Second,
	SaramaConfig:     defaultSaramaConfig(),
	WithDlq:          false,
	Storage:          nil,
	WithAlert:        false,
	TelegramNotifier: nil,
}

// defaultSaramaConfig returns a default sarama kafka configuration.
func defaultSaramaConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Consumer.Return.Errors = true
	config.Version = sarama.V2_8_0_0 // Adjust based on your Kafka version
	return config
}

// setConfig sets the Kafka client configuration.
func setConfig(config ...Config) (Config, error) {
	if len(config) == 0 {
		return DefaultConfig, nil
	}

	// Override default config with provided configs
	cfg := config[0]

	// Set default values if not provided
	if len(cfg.Brokers) == 0 {
		cfg.Brokers = DefaultConfig.Brokers
	}
	if cfg.ProducerTimeout == 0 {
		cfg.ProducerTimeout = DefaultConfig.ProducerTimeout
	}
	if cfg.AppName == "" {
		cfg.AppName = DefaultConfig.AppName
	}
	if cfg.ConsumerTimeout == 0 {
		cfg.ConsumerTimeout = DefaultConfig.ConsumerTimeout
	}
	if cfg.SaramaConfig == nil {
		cfg.SaramaConfig = defaultSaramaConfig()
	}

	// Validate config
	if cfg.WithDlq && cfg.Storage == nil {
		return Config{}, fmt.Errorf("DLQ is enabled but no storage configured")
	}
	if cfg.WithAlert && cfg.TelegramNotifier == nil {
		return Config{}, fmt.Errorf("alert is enabled but no telegram notifier configured")
	}
	return cfg, nil
}
