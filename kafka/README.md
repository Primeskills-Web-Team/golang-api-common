# Kafka Service

A robust Kafka client service implementation for Go applications that provides both producer and consumer functionality with monitoring capabilities.

## Features

- Synchronous message production
- Consumer group message consumption
- Prometheus metrics integration
- Configurable producer and consumer settings
- Graceful shutdown handling
- Error tracking and monitoring
- Exponential backoff for consumer reconnection

## Installation

```bash
go get github.com/Primeskills-Web-Team/golang-api-common/v2/kafka
```

## Usage

### Basic Usage

```go
import "github.com/Primeskills-Web-Team/golang-api-common/v2/kafka"

func main() {
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
    ctx := context.Background()
    kafkaService.ConsumeMessage(ctx, groupID, []string{"example-topic"}, func(message string) error {
        // Handle the message
        log.Info().Msgf("Received message: %s", message)
        return nil
    })
}
```

### Custom Configuration

```go
cfg := &kafka.Config{
    ProducerRequiredAcks:      sarama.WaitForAll,
    ProducerRetryMax:          5,
    ProducerTimeout:           5 * time.Second,
    ConsumerSessionTimeout:    10 * time.Second,
    ConsumerHeartbeatInterval: 3 * time.Second,
    ConsumerInitialOffset:     sarama.OffsetOldest,
    Version:                   sarama.V2_6_0_0,
}

kafkaService := kafka.NewService(brokers, groupID, cfg)
```

## Metrics

The service exposes the following Prometheus metrics:

- `kafka_messages_produced_total`: Total number of messages produced
- `kafka_messages_consumed_total`: Total number of messages consumed
- `kafka_message_processing_seconds`: Message processing latency histogram
- `kafka_consumer_errors_total`: Total number of consumer errors
- `kafka_producer_errors_total`: Total number of producer errors

## Configuration Options

| Option                    | Description                            | Default      |
| ------------------------- | -------------------------------------- | ------------ |
| ProducerRequiredAcks      | Required acknowledgments for producer  | WaitForAll   |
| ProducerRetryMax          | Maximum number of retries for producer | 5            |
| ProducerTimeout           | Producer timeout duration              | 5s           |
| ConsumerSessionTimeout    | Consumer group session timeout         | 10s          |
| ConsumerHeartbeatInterval | Consumer group heartbeat interval      | 3s           |
| ConsumerInitialOffset     | Initial offset for consumer            | OffsetOldest |
| Version                   | Kafka protocol version                 | V2_6_0_0     |

## Error Handling

The service implements:

- Exponential backoff for consumer reconnection
- Error tracking through Prometheus metrics
- Graceful shutdown handling
- Nil checks for producer and consumer

## Best Practices

1. Always call `Close()` when shutting down the service
2. Use context cancellation for graceful consumer shutdown
3. Implement proper error handling in message handlers
4. Monitor the provided Prometheus metrics
5. Configure appropriate timeouts and retry settings for your use case
