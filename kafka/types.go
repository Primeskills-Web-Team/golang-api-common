package kafka

import "github.com/IBM/sarama"

// Event represents a Kafka event message.
type Event struct {
	EventName string      `json:"event_name"`
	Token     string      `json:"token"`
	Header    string      `json:"header"`
	Data      interface{} `json:"data"`
}

// ConsumerHandler defines the interface for handling consumed messages.
type ConsumerHandler interface {
	HandleMessage(*sarama.ConsumerMessage) error
}
