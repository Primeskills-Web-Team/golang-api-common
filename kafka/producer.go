package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
)

// Produce sends a message to the specified topic.
func (c *Kafka) Produce(ctx context.Context, topic string, event Event) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("source"),
				Value: []byte(c.config.AppName),
			},
		},
		Value: sarama.ByteEncoder(eventBytes),
	}
	select {
	case <-c.closed:
		return fmt.Errorf("client closed")
	default:
	}
	_, _, err = c.producer.SendMessage(msg)
	return err
}
