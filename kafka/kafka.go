package kafka

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

type KafkaService interface {
	ProduceMessage(topic string, message string) error
	ConsumeMessage(ctx context.Context, groupID string, topics []string, handler func(message string) error)
	Ping() error
	Close() error
}

type service struct {
	producer      sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup
}

var (
	producedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_messages_produced_total",
		Help: "The total number of messages produced",
	})
	consumedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_messages_consumed_total",
		Help: "The total number of messages consumed",
	})
	messageProcessingLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kafka_message_processing_seconds",
		Help:    "Time taken to process messages",
		Buckets: prometheus.DefBuckets,
	})
	consumerErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_errors_total",
		Help: "The total number of consumer errors",
	})
	producerErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_producer_errors_total",
		Help: "The total number of producer errors",
	})
)

func (s *service) Ping() error {
	_, _, err := s.producer.SendMessage(&sarama.ProducerMessage{
		Topic: "ping",
		Key:   sarama.StringEncoder("ping"),
		Value: nil,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to ping")
		return err
	}
	return nil
}

func (s *service) Close() error {
	var errs []error

	if s.producer != nil {
		if err := s.producer.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close producer")
			errs = append(errs, err)
		}
	}

	if s.consumerGroup != nil {
		if err := s.consumerGroup.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close consumer group")
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (s *service) ProduceMessage(topic string, message string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	partition, offset, err := s.producer.SendMessage(msg)
	if err != nil {
		producerErrors.Inc()
		log.Error().Err(err).Msg("Failed to produce message")
		return err
	}

	producedMessages.Inc()
	log.Info().Msgf("Message sent to partition %d at offset %d", partition, offset)
	return nil
}

type consumerGroupHandler struct {
	handler func(message string) error
	ctx     context.Context
}

func (c *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (c *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			start := time.Now()
			if err := c.handler(string(msg.Value)); err != nil {
				consumerErrors.Inc()
				log.Error().Err(err).Msg("Handler failed")
			} else {
				consumedMessages.Inc()
				sess.MarkMessage(msg, "")
			}
			messageProcessingLatency.Observe(time.Since(start).Seconds())
		case <-c.ctx.Done():
			log.Info().Msg("Context canceled. Stopping consumer group handler...")
			return nil
		}
	}
}

func (s *service) ConsumeMessage(ctx context.Context, groupID string, topics []string, handler func(message string) error) {
	// Create a new context that will be canceled when the service is closed
	consumerCtx, cancel := context.WithCancel(ctx)

	go func() {
		defer cancel()

		h := &consumerGroupHandler{
			handler: handler,
			ctx:     consumerCtx,
		}

		backoff := time.Second
		maxBackoff := 30 * time.Second

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.consumerGroup.Consume(consumerCtx, topics, h); err != nil {
					consumerErrors.Inc()
					log.Error().Err(err).Msg("Error from consumer")

					// Implement exponential backoff
					time.Sleep(backoff)
					backoff = min(time.Duration(float64(backoff)*1.5), maxBackoff)
				}

				if consumerCtx.Err() != nil {
					return
				}
			}
		}
	}()
}

type Config struct {
	ProducerRequiredAcks      sarama.RequiredAcks
	ProducerRetryMax          int
	ProducerTimeout           time.Duration
	ConsumerSessionTimeout    time.Duration
	ConsumerHeartbeatInterval time.Duration
	ConsumerInitialOffset     int64
	Version                   sarama.KafkaVersion
}

func defaultConfig() *Config {
	return &Config{
		ProducerRequiredAcks:      sarama.WaitForAll,
		ProducerRetryMax:          5,
		ProducerTimeout:           5 * time.Second,
		ConsumerSessionTimeout:    10 * time.Second,
		ConsumerHeartbeatInterval: 3 * time.Second,
		ConsumerInitialOffset:     sarama.OffsetOldest,
		Version:                   sarama.V2_6_0_0,
	}
}

func createSaramaConfig(cfg *Config) *sarama.Config {
	saramaCfg := sarama.NewConfig()

	// Producer configuration
	saramaCfg.Producer.RequiredAcks = cfg.ProducerRequiredAcks
	saramaCfg.Producer.Retry.Max = cfg.ProducerRetryMax
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Timeout = cfg.ProducerTimeout

	// Consumer configuration
	saramaCfg.Consumer.Group.Session.Timeout = cfg.ConsumerSessionTimeout
	saramaCfg.Consumer.Group.Heartbeat.Interval = cfg.ConsumerHeartbeatInterval
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaCfg.Consumer.Offsets.Initial = cfg.ConsumerInitialOffset
	saramaCfg.Consumer.Return.Errors = true

	saramaCfg.Version = cfg.Version
	return saramaCfg
}

func createProducer(brokers []string, cfg *sarama.Config) sarama.SyncProducer {
	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create producer")
	}
	return producer
}

func createConsumerGroup(brokers []string, groupID string, cfg *sarama.Config) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create consumer group")
	}
	return cg
}

func NewService(brokers []string, groupID string, cfg *Config) KafkaService {
	if cfg == nil {
		cfg = defaultConfig()
	}

	saramaCfg := createSaramaConfig(cfg)

	return &service{
		producer:      createProducer(brokers, saramaCfg),
		consumerGroup: createConsumerGroup(brokers, groupID, saramaCfg),
	}
}
