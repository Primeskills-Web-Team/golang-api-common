package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/utils"
	"github.com/rs/zerolog/log"
)

// Consume starts consuming messages from the specified topics using the provided handler.
func (c *Kafka) Consume(ctx context.Context, topics []string, handler func(message *sarama.ConsumerMessage) error) error {
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
	handler func(message *sarama.ConsumerMessage) error
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
		// Validate message security first
		if err := c.kafka.validateMessageSecurity(message); err != nil {
			log.Error().Err(err).Msg("Message security validation failed")
			// Send to DLQ for security violations
			if err := c.kafka.SendToDLQ(message, err); err != nil {
				log.Error().Err(err).Msg("Failed to send security violation to DLQ")
			}
			// Send alert to telegram
			if err := c.kafka.SendAlert(message, err); err != nil {
				log.Error().Err(err).Msg("Failed to send alert to telegram")
			}
			session.MarkMessage(message, "") // Mark as processed even if invalid
			continue
		}

		// Process the message with your handler
		if err := c.handler(message); err != nil {
			log.Error().Err(err).Msg("Handler error")
			// Send to DLQ
			if err := c.kafka.SendToDLQ(message, err); err != nil {
				log.Error().Err(err).Msg("Failed to send to DLQ")
			}
			// Send alert to telegram
			if err := c.kafka.SendAlert(message, err); err != nil {
				log.Error().Err(err).Msg("Failed to send alert to telegram")
			}
		}
		log.Info().Str("topic", message.Topic).Int32("partition", message.Partition).Int64("offset", message.Offset).Msg("Processed message")
		session.MarkMessage(message, "") // Mark message as processed
	}
	return nil
}

// validateMessageSecurity validates the JWT token in message headers
func (c *Kafka) validateMessageSecurity(message *sarama.ConsumerMessage) error {
	// Skip validation if JWT is not configured
	if !c.config.Secure {
		return nil
	}

	// Find authorization header
	var authToken string
	for _, header := range message.Headers {
		if string(header.Key) == "authorization" {
			authToken = string(header.Value)
			break
		}
	}

	if authToken == "" {
		return fmt.Errorf("missing authorization token in message header")
	}

	// Validate JWT token
	_, err := utils.ValidateJWT(authToken, c.config.SecretKey)
	if err != nil {
		return err
	}

	return nil
}
