package kafka

import (
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

// Kafka wraps Kafka producer and consumer group functionality.
type Kafka struct {
	config        Config
	producer      sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup
	closed        chan struct{}
	wg            sync.WaitGroup
}

// New creates a new Kafka client.
func New(config ...Config) (*Kafka, error) {
	cfg, err := setConfig(config...)
	if err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}

	producer, err := sarama.NewSyncProducer(cfg.Brokers, cfg.SaramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	client := &Kafka{
		config:   cfg,
		producer: producer,
		closed:   make(chan struct{}),
	}

	return client, nil
}

// Close shuts down the Kafka client gracefully.
func (c *Kafka) Close() error {
	close(c.closed)

	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close producer")
		}
	}

	if c.consumerGroup != nil {
		if err := c.consumerGroup.Close(); err != nil {
			// Don't log errors when consumer group is already closed (normal during shutdown)
			if err.Error() != "kafka: tried to use consumer group that was closed" {
				log.Error().Err(err).Msg("Failed to close consumer group")
			}
		}
	}

	c.wg.Wait()
	log.Info().Msg("Kafka client closed")
	return nil
}
