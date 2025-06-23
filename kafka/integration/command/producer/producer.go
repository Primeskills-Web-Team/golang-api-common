package producer

import (
	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
)

type KafkaProducer struct {
	Producer sarama.SyncProducer
}

// SendMessage existing method - untuk backward compatibility
func (p *KafkaProducer) SendMessage(topic, msg string) error {
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}

	partition, offset, err := p.Producer.SendMessage(kafkaMsg)
	if err != nil {
		logrus.Errorf("Send message error: %v", err)
		return err
	}

	logrus.Infof("Send message success, Topic %v, Partition %v, Offset %d", topic, partition, offset)
	return nil
}

// SendMessageSync method baru yang mengembalikan partition dan offset
func (p *KafkaProducer) SendMessageSync(topic, msg string) (int32, int64, error) {
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}

	partition, offset, err := p.Producer.SendMessage(kafkaMsg)
	if err != nil {
		logrus.Errorf("Send message error: %v", err)
		return 0, 0, err
	}

	logrus.Infof("Send message success, Topic %v, Partition %v, Offset %d", topic, partition, offset)
	return partition, offset, nil
}

// SendMessageWithKey sends message dengan key untuk partitioning
func (p *KafkaProducer) SendMessageWithKey(topic, key, msg string) (int32, int64, error) {
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(msg),
	}

	partition, offset, err := p.Producer.SendMessage(kafkaMsg)
	if err != nil {
		logrus.Errorf("Send message with key error: %v", err)
		return 0, 0, err
	}

	logrus.Infof("Send message with key success, Topic %v, Key %v, Partition %v, Offset %d", topic, key, partition, offset)
	return partition, offset, nil
}

// SendMessageWithHeaders sends message dengan custom headers
func (p *KafkaProducer) SendMessageWithHeaders(topic, msg string, headers map[string]string) (int32, int64, error) {
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}

	// Add headers if provided
	if headers != nil && len(headers) > 0 {
		kafkaMsg.Headers = make([]sarama.RecordHeader, 0, len(headers))
		for key, value := range headers {
			kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
				Key:   []byte(key),
				Value: []byte(value),
			})
		}
	}

	partition, offset, err := p.Producer.SendMessage(kafkaMsg)
	if err != nil {
		logrus.Errorf("Send message with headers error: %v", err)
		return 0, 0, err
	}

	logrus.Infof("Send message with headers success, Topic %v, Partition %v, Offset %d", topic, partition, offset)
	return partition, offset, nil
}