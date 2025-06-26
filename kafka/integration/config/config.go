package config

import (
	"sync"
	"time"

	"github.com/IBM/sarama"

	"github.com/Primeskills-Web-Team/golang-api-common/kafka/helpers"
)


type PublishResult struct {
	Success   bool
	Error     error
	Partition int32
	Offset    int64
	Timestamp time.Time
}

type RetryConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	BackoffFactor float64
}

type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type DLQConfig struct {
	Enabled             bool          `json:"enabled"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	MaxRetryDelay       time.Duration `json:"max_retry_delay"`
	DeadLetterSuffix    string        `json:"dead_letter_suffix"`
	EnableDeduplication bool          `json:"enable_deduplication"`
}

// DefaultDLQConfig returns the default configuration for DLQ.
func DefaultDLQConfig() DLQConfig {
	return DLQConfig{
		Enabled:             true,
		MaxRetries:          3,
		RetryDelay:          time.Second * 2,
		DeadLetterSuffix:    "-dead-letter",
		EnableDeduplication: true,
	}
}

// createConfig is a function that creates a sarama config
func createConfig(k *KafkaConfig) *sarama.Config {
	config := sarama.NewConfig()

	// Producer configuration
	config.Producer.RequiredAcks = k.Config.RequiredAcks
	config.Producer.Retry.Max = k.Config.Retry.MaxRetries
	config.Producer.Retry.Backoff = k.Config.Retry.RetryInterval
	config.Producer.Compression = k.Config.Compression
	config.Producer.Flush.Frequency = k.Config.FlushFrequency
	config.Producer.Flush.Messages = k.Config.FlushMessages
	config.Producer.Flush.Bytes = k.Config.FlushBytes
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	// Consumer Group configuration
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Group.Session.Timeout = 20 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 6 * time.Second
	config.Consumer.Group.Rebalance.Timeout = 60 * time.Second
	config.Consumer.Group.Rebalance.Retry.Max = 4
	config.Consumer.Group.Rebalance.Retry.Backoff = 2 * time.Second

	// Consumer configuration
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	// Network configuration
	config.Net.DialTimeout = k.Config.DialTimeout
	config.Net.ReadTimeout = k.Config.ReadTimeout
	config.Net.WriteTimeout = k.Config.WriteTimeout
	config.Net.KeepAlive = k.Config.KeepAlive

	// TLS configuration
	if k.Config.EnableTLS {
		config.Net.TLS.Enable = true
	}

	// SASL configuration
	if (k.Config.EnableSASL || k.Username != "") && k.Username != "" && k.Password != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = k.Username
		config.Net.SASL.Password = k.Password

		switch k.Config.SASLMechanism {
		case "PLAIN":
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case "SCRAM-SHA-256":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	// Metadata configuration
	config.Metadata.Retry.Max = k.Config.MetadataRetryMax
	config.Metadata.Retry.Backoff = k.Config.MetadataRetryBackoff
	config.Metadata.RefreshFrequency = k.Config.MetadataRefreshFreq
	config.Metadata.Full = true

	// Version configuration
	config.Version = sarama.V2_6_0_0

	return config
}

func NewKafkaConfig(username string, password string, address []string, slackWebhookURL string) *KafkaConfig {
	config := &KafkaConfig{
		Username:        username,
		Password:        password,
		Address:         address,
		SlackWebhookURL: slackWebhookURL,
		Config:          DefaultKafkaConfigOptions(),
		DLQConfig:       DefaultDLQConfig(),
	}

	// Initialize helpers
	config.InitializeHelpers(slackWebhookURL)

	return config
}

func (k *KafkaConfig) InitializeHelpers(slackWebhookURL string) {
	k.dlqHelper = helpers.NewDLQHelper()
	k.slackHelper = helpers.NewSlackHelper(slackWebhookURL)
}

// GetDLQHelper returns the DLQ helper instance
func (k *KafkaConfig) GetDLQHelper() *helpers.DLQHelper {
	return k.dlqHelper
}

// GetSlackHelper returns the Slack helper instance
func (k *KafkaConfig) GetSlackHelper() *helpers.SlackHelper {
	return k.slackHelper
}

func NewKafkaConfigWithOptions(username, password string, address []string, options KafkaConfigOptions, slackWebhookURL string) *KafkaConfig {
	config := &KafkaConfig{
		Username:        username,
		Password:        password,
		Address:         address,
		SlackWebhookURL: slackWebhookURL,
		Config:          options,
		DLQConfig:       DefaultDLQConfig(),
	}

	// Initialize helpers
	config.InitializeHelpers(slackWebhookURL)

	return config
}

type KafkaConfig struct {
	Username        string
	Password        string
	SlackWebhookURL string
	AlertEnabled    bool // Indicates if alerts are enabled
	Address         []string
	producer        sarama.SyncProducer
	mu              sync.Mutex
	Config          KafkaConfigOptions
	DLQConfig       DLQConfig
	failureCache    sync.Map

	// Helper instances
	dlqHelper   *helpers.DLQHelper
	slackHelper *helpers.SlackHelper

}

type KafkaConfigOptions struct {
	// Producer settings
	RequiredAcks   sarama.RequiredAcks
	Retry          RetryConfig
	Compression    sarama.CompressionCodec
	FlushFrequency time.Duration
	FlushMessages  int
	FlushBytes     int

	// Connection settings
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeepAlive    time.Duration

	// Security settings
	EnableTLS     bool
	EnableSASL    bool
	SASLMechanism string

	// Metadata settings
	MetadataRetryMax     int
	MetadataRetryBackoff time.Duration
	MetadataRefreshFreq  time.Duration
}

// DefaultKafkaConfigOptions is a function that returns the default configuration for Kafka
func DefaultKafkaConfigOptions() KafkaConfigOptions {
	return KafkaConfigOptions{
		RequiredAcks:         sarama.WaitForAll,
		Retry:                DefaultRetryConfig(),
		Compression:          sarama.CompressionSnappy,
		FlushFrequency:       100 * time.Millisecond,
		FlushMessages:        100,
		FlushBytes:           1024 * 1024, // 1MB
		DialTimeout:          30 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		KeepAlive:            30 * time.Second,
		EnableTLS:            false,
		EnableSASL:           false,
		SASLMechanism:        "PLAIN",
		MetadataRetryMax:     3,
		MetadataRetryBackoff: 250 * time.Millisecond,
		MetadataRefreshFreq:  10 * time.Minute,
	}
}
