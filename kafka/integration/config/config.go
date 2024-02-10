package config

import (
	"github.com/IBM/sarama"
	"time"
)

type KafkaConfig struct {
	Username string
	Password string
	Address  []string
}

func createConfig(config *KafkaConfig) *sarama.Config {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Net.WriteTimeout = 5 * time.Second
	kafkaConfig.Producer.Retry.Max = 0

	if config.Username != "" {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.Username
		kafkaConfig.Net.SASL.Password = config.Password
	}
	return kafkaConfig
}

func NewKafkaConfig(username string, password string, address []string) *KafkaConfig {
	return &KafkaConfig{
		Username: username,
		Password: password,
		Address:  address,
	}
}
