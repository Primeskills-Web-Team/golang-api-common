package consumer

import (
	"context"
	"encoding/json"
	"os"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/sirupsen/logrus"
)

// KafkaConsumer hold sarama consumer group
type KafkaConsumer struct {
	ConsumerGroup sarama.ConsumerGroup
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type ConsumerGroupHandler struct {
	handler func(value kafka.Event)
	ready   chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			logrus.Infof("New Message from kafka, topic: %s, partition: %d, offset: %d, message: %v",
				message.Topic, message.Partition, message.Offset, string(message.Value))

			var event kafka.Event
			if err := json.Unmarshal(message.Value, &event); err != nil {
				logrus.Errorf("Failed to unmarshal message: %v", err)
				// Mark message as processed even if unmarshaling fails
				session.MarkMessage(message, "")
				continue
			}

			// Handle the message
			func() {
				defer func() {
					if r := recover(); r != nil {
						logrus.Errorf("Panic in message handler: %v", r)
					}
				}()
				h.handler(event)
			}()

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// Consume function to consume message from apache kafka using consumer groups
func (c *KafkaConsumer) Consume(topics []string, signals chan os.Signal, handler func(value kafka.Event)) {
	// Create consumer group handler
	consumerGroupHandler := &ConsumerGroupHandler{
		handler: handler,
		ready:   make(chan bool),
	}

	// Create context for consumer group
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming in a goroutine
	go func() {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := c.ConsumerGroup.Consume(ctx, topics, consumerGroupHandler); err != nil {
				logrus.Errorf("Error from consumer: %v", err)
			}
			// Check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumerGroupHandler.ready = make(chan bool)
		}
	}()

	// Wait for the consumer to be ready
	<-consumerGroupHandler.ready
	logrus.Infof("Kafka consumer group is consuming topics: %v", topics)

	// Wait for interrupt signal to gracefully shutdown the consumer
	<-signals
	logrus.Info("Shutting down gracefully...")

	// Cancel context to stop consumer
	cancel()

	// Close consumer group
	if err := c.ConsumerGroup.Close(); err != nil {
		logrus.Errorf("Error closing consumer group: %v", err)
	}
}
