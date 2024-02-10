package config

import (
	"github.com/Primeskills-Web-Team/golang-api-common/kafka"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestPublishMessage(t *testing.T) {
	conf := NewKafkaConfig("", "", []string{"petraverse.id:9092"})
	conf.PublishEvent("test_topic", kafka.Event{
		EventName: "TEST",
		Source:    "TESTING",
		Data:      nil,
	})
}

func TestConsumerMessage(t *testing.T) {
	conf := NewKafkaConfig("", "", []string{"petraverse.id:9092"})
	conf.AddConsumerListener([]string{"test_topic"}, func(value kafka.Event) {
		logrus.Infoln("retrieve value", value.EventName)
	})
}
