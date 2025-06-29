package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/utils"
	"github.com/rs/zerolog/log"
)

// Produce sends a message to the specified topic.
func (c *Kafka) Produce(ctx context.Context, topic string, value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	headers := []sarama.RecordHeader{
		{
			Key:   []byte("source"),
			Value: []byte(c.config.AppName),
		},
	}

	// Add JWT token header if JWT is configured
	if c.config.Secure {
		token, err := utils.GenerateJWT(utils.JWTConfig{
			SecretKey:     c.config.SecretKey,
			TokenDuration: c.config.TokenDuration,
			AppName:       c.config.AppName,
		})
		if err != nil {
			return fmt.Errorf("failed to generate JWT token: %w", err)
		}

		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("authorization"),
			Value: []byte(token),
		})
	}

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Headers: headers,
		Value:   sarama.ByteEncoder(valueBytes),
	}

	select {
	case <-c.closed:
		return fmt.Errorf("client closed")
	default:
	}

	par, off, err := c.producer.SendMessage(msg)
	if err != nil {
		return err
	}
	log.Info().Str("topic", topic).Int32("partition", par).Int64("offset", off).Msg("Produced message")
	return nil
}
