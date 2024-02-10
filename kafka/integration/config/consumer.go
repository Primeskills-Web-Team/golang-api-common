package config

import (
	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/consumer"
	"github.com/sirupsen/logrus"
	"os"
)

func (k *KafkaConfig) AddConsumerListener(topics []string, handler func(value kafka.Event)) {
	consumers, err := sarama.NewConsumer(k.Address, createConfig(k))
	if err != nil {
		logrus.Errorf("Error create kakfa consumer got error %v", err)
	}
	defer func() {
		if err := consumers.Close(); err != nil {
			logrus.Fatal(err)
			return
		}
	}()

	kafkaConsumer := &consumer.KafkaConsumer{
		Consumer: consumers,
	}

	signals := make(chan os.Signal, 1)
	kafkaConsumer.Consume(topics, signals, handler)
}
