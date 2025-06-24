package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/kafka"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	loggerInstance := logger.New(logger.Config{
		Level:         "info",
		IsDevelopment: true,
	})
	if err := loggerInstance.Setup(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup logger")
	}

	// Define Kafka configuration
	brokers := []string{"localhost:9092"}
	groupID := "example-consumer-group"

	// Create Kafka service with default configuration
	kafkaService := kafka.NewService(brokers, groupID, nil)
	defer kafkaService.Close()

	// Produce a message
	err := kafkaService.ProduceMessage("example-topic", "Hello, Kafka!")
	if err != nil {
		log.Error().Err(err).Msg("Failed to produce message")
	}

	// Consume messages
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaService.ConsumeMessage(ctx, groupID, []string{"example-topic"}, failedHandler)

	// Wait for a termination signal (e.g., Ctrl+C)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	println("Shutting down gracefully...")
}

func failedHandler(message string) error {
	log.Error().Msgf("Failed to handle message: %s", message)
	return errors.New("failed to handle message")
}
