package config

import (
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/IBM/sarama"
    "github.com/sirupsen/logrus"
)

type KafkaConfig struct {
    Username      string
    Password      string
    SlackWebhookURL string // Optional, for alerting
    Address       []string
    producer      sarama.SyncProducer
    mu            sync.Mutex
    Config        KafkaConfigOptions
}

// ✅ Configuration options
type KafkaConfigOptions struct {
    // Producer settings
    RequiredAcks          sarama.RequiredAcks
    Retry                 RetryConfig
    Compression           sarama.CompressionCodec
    FlushFrequency        time.Duration
    FlushMessages         int
    FlushBytes            int
    
    // Connection settings
    DialTimeout           time.Duration
    ReadTimeout           time.Duration
    WriteTimeout          time.Duration
    KeepAlive            time.Duration
    
    // Security settings
    EnableTLS            bool
    EnableSASL           bool
    SASLMechanism        string
    
    // Metadata settings
    MetadataRetryMax     int
    MetadataRetryBackoff time.Duration
    MetadataRefreshFreq  time.Duration
}

// ✅ Default configuration
func DefaultKafkaConfigOptions() KafkaConfigOptions {
    return KafkaConfigOptions{
        RequiredAcks:          sarama.WaitForAll,
        Retry:                 DefaultRetryConfig(),
        Compression:           sarama.CompressionSnappy,
        FlushFrequency:        100 * time.Millisecond,
        FlushMessages:         100,
        FlushBytes:            1024 * 1024, // 1MB
        DialTimeout:           30 * time.Second,
        ReadTimeout:           30 * time.Second,
        WriteTimeout:          30 * time.Second,
        KeepAlive:            30 * time.Second,
        EnableTLS:            false,
        EnableSASL:           false,
        SASLMechanism:        "PLAIN",
        MetadataRetryMax:     3,
        MetadataRetryBackoff: 250 * time.Millisecond,
        MetadataRefreshFreq:  10 * time.Minute,
    }
}

// ✅ Enhanced createConfig function
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

func NewKafkaConfig(username string, password string, address []string) *KafkaConfig {
    return &KafkaConfig{
        Username: username,
        Password: password,
        Address:  address,
        Config:   DefaultKafkaConfigOptions(),
    }
}

func NewKafkaConfigWithOptions(username, password string, address []string, options KafkaConfigOptions) *KafkaConfig {
    return &KafkaConfig{
        Username: username,
        Password: password,
        Address:  address,
        Config:   options,
    }
}

func (k *KafkaConfig) getOrCreateProducer() (sarama.SyncProducer, error) {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    if k.producer == nil {
        producer, err := sarama.NewSyncProducer(k.Address, createConfig(k))
        if err != nil {
            return nil, fmt.Errorf("unable to create kafka producer: %w", err)
        }
        k.producer = producer
        logrus.Info("Created new Kafka producer")
    }
    return k.producer, nil
}

func (k *KafkaConfig) Close() error {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    if k.producer != nil {
        err := k.producer.Close()
        k.producer = nil
        logrus.Info("Closed Kafka producer")
        return err
    }
    return nil
}

func (k *KafkaConfig) resetProducer() {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    if k.producer != nil {
        k.producer.Close()
        k.producer = nil
        logrus.Warn("Reset Kafka producer due to error")
    }
}

func (k *KafkaConfig) Validate() error {
    if len(k.Address) == 0 {
        return fmt.Errorf("kafka address cannot be empty")
    }
    
    for i, addr := range k.Address {
        if strings.TrimSpace(addr) == "" {
            return fmt.Errorf("kafka address at index %d cannot be empty", i)
        }
    }
    
    if k.Config.Retry.MaxRetries < 0 {
        return fmt.Errorf("max retries cannot be negative")
    }
    
    if k.Config.Retry.RetryInterval < 0 {
        return fmt.Errorf("retry interval cannot be negative")
    }
    
    if k.Config.FlushFrequency < 0 {
        return fmt.Errorf("flush frequency cannot be negative")
    }
    
    return nil
}

func (k *KafkaConfig) GetInfo() map[string]interface{} {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    return map[string]interface{}{
        "addresses":           k.Address,
        "username":           k.Username,
        "has_password":       k.Password != "",
        "producer_connected": k.producer != nil,
        "config": map[string]interface{}{
            "required_acks":    k.Config.RequiredAcks,
            "compression":      k.Config.Compression.String(),
            "flush_frequency":  k.Config.FlushFrequency.String(),
            "flush_messages":   k.Config.FlushMessages,
            "flush_bytes":      k.Config.FlushBytes,
            "enable_tls":       k.Config.EnableTLS,
            "enable_sasl":      k.Config.EnableSASL,
            "sasl_mechanism":   k.Config.SASLMechanism,
        },
    }
}

func (k *KafkaConfig) UpdateConfig(options KafkaConfigOptions) error {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    // Close existing producer if config changes
    if k.producer != nil {
        if err := k.producer.Close(); err != nil {
            logrus.WithError(err).Warn("Failed to close existing producer during config update")
        }
        k.producer = nil
    }
    
    k.Config = options
    
    // Validate new configuration
    if err := k.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }
    
    logrus.Info("Kafka configuration updated successfully")
    return nil
}

func (k *KafkaConfig) ForceReconnect() error {
    k.mu.Lock()
    defer k.mu.Unlock()
    
    if k.producer != nil {
        if err := k.producer.Close(); err != nil {
            logrus.WithError(err).Warn("Failed to close producer during force reconnect")
        }
        k.producer = nil
    }
    
    logrus.Info("Forced Kafka producer reconnection")
    return nil
}

func isConnectionError(err error) bool {
    if err == nil {
        return false
    }
    
    errStr := err.Error()
    return strings.Contains(errStr, "connection refused") ||
           strings.Contains(errStr, "no such host") ||
           strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "broken pipe") ||
           strings.Contains(errStr, "connection reset")
}