package config

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/command/producer"
	"github.com/sirupsen/logrus"
)

func (k *KafkaConfig) PublishEvent(topic string, value kafka.Event) {
	producers, err := sarama.NewSyncProducer(k.Address, createConfig(k))
	if err != nil {
		logrus.Errorf("Unable to create kafka producer got error %v", err)
		return
	}
	defer func() {
		if err := producers.Close(); err != nil {
			logrus.Errorf("Unable to stop kafka producer: %v", err)
			return
		}
	}()

	logrus.Infof("Success create kafka sync-producer")

	kafka := &producer.KafkaProducer{
		Producer: producers,
	}

	msg, errMarshar := json.Marshal(value)
	if errMarshar != nil {
		logrus.Errorf("Failed marshal value event: %v", errMarshar)
		return
	}

	errSend := kafka.SendMessage(topic, string(msg))
	if errSend != nil {
		logrus.Errorf("Failed send message event: %v", errSend)
	}
}
