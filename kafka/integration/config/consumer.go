package config

import (
	"os"

	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/consumer"
	"github.com/sirupsen/logrus"
)

func (k *KafkaConfig) AddConsumerListener(topics []string, handler func(value kafka.Event), consumerGroupId string) {
	consumerGroup, err := sarama.NewConsumerGroup(k.Address, consumerGroupId, createConfig(k))
	if err != nil {
		logrus.Errorf("Error create kafka consumer group got error %v", err)
		return
	}

	kafkaConsumer := &consumer.KafkaConsumer{
		ConsumerGroup: consumerGroup,
	}

	signals := make(chan os.Signal, 1)
	kafkaConsumer.Consume(topics, signals, handler)
}
