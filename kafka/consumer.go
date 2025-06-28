package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

// Consume starts consuming messages from the specified topics using the provided handler.
func (c *Kafka) Consume(ctx context.Context, topics []string, handler ConsumerHandler) error {
	if c.config.AppName == "" {
		return fmt.Errorf("consumer group not specified")
	}

	consumerGroup, err := sarama.NewConsumerGroup(c.config.Brokers, c.config.AppName, c.config.SaramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	c.consumerGroup = consumerGroup

	consumer := &consumer{
		kafka:   c,
		handler: handler,
		ready:   make(chan bool),
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Consumer context cancelled, stopping consumption")
				return
			default:
				if err := c.consumerGroup.Consume(ctx, topics, consumer); err != nil {
					// Don't log errors when context is cancelled (normal shutdown)
					if ctx.Err() == nil {
						log.Error().Err(err).Msg("Consumer error")
					}
				}
				if ctx.Err() != nil {
					return
				}
				consumer.ready = make(chan bool)
			}
		}
	}()

	<-consumer.ready
	return nil
}

// consumer implements sarama.ConsumerGroupHandler.
type consumer struct {
	kafka   *Kafka
	handler ConsumerHandler
	ready   chan bool
}

// Setup is run at the beginning of a new session.
func (c *consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session.
func (c *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a single partition.
func (c *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := c.handler.HandleMessage(message); err != nil {
			log.Error().Err(err).Msg("Handler error")
			// Send to DLQ
			if err := c.kafka.SendToDLQ(message, err); err != nil {
				log.Error().Err(err).Msg("Failed to send to DLQ")
			}
		}
		session.MarkMessage(message, "") // Mark message as processed
	}
	return nil
}
