package kafka

import (
	"context"
	"errors"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ErrMessageEmpty is returned when a Kafka message is empty.
var ErrMessageEmpty = errors.New("message empty")

type (
	// Message defines message.
	Message struct {
		// Topic stores the topic value.
		Topic string
		// Key stores the key value.
		Key []byte
		// Body stores the body value.
		Body []byte
		// ContentType stores the content type value.
		ContentType string
		// EventID stores the event i d value.
		EventID string
		// TaskType stores the task type value.
		TaskType entity.TaskType
	}
)

// Producer describes Kafka producer operations.
type Producer interface {
	// Produce publishes a Kafka message.
	Produce(ctx context.Context, message ...*Message) error
	// Close releases resources.
	Close()
}
type kafkaProducer struct {
	client *kgo.Client
}

// NewProducer creates a Kafka producer.
func NewProducer(client *kgo.Client) *kafkaProducer {
	k := &kafkaProducer{client: client}
	return k
}

// Produce publishes a Kafka message.
func (p *kafkaProducer) Produce(ctx context.Context, messages ...*Message) error {
	if len(messages) == 0 {
		return ErrMessageEmpty
	}
	records := make([]*kgo.Record, len(messages))
	for i, message := range messages {
		records[i] = &kgo.Record{
			Topic: message.Topic,
			Key:   message.Key,
			Value: message.Body,
			Headers: []kgo.RecordHeader{
				{
					Key:   EventTypeHeaderKey,
					Value: []byte(message.TaskType.String()),
				},
				{
					Key:   EventIDHeaderKey,
					Value: []byte(message.EventID),
				},
				{
					Key:   ContentTypeHeaderKey,
					Value: []byte(message.ContentType),
				},
			},
		}
	}
	if err := p.client.ProduceSync(ctx, records...).FirstErr(); err != nil {
		return err
	}
	return nil
}

// Close releases resources.
func (p *kafkaProducer) Close() {
	p.client.Close()
}

// PoolConfiger creates Kafka producer clients for a pool.
type PoolConfiger func() (*kgo.Client, error)

// KafkaProducerPool manages a set of Kafka producers.
type KafkaProducerPool struct {
	producers []Producer
}

// NewKafkaProducerPool creates a pool of Kafka producers.
func NewKafkaProducerPool(count int, configer PoolConfiger) *KafkaProducerPool {
	producers := make([]Producer, count)
	for i := 0; i < count; i++ {
		c, err := configer()
		if err != nil {
			panic(err)
		}
		producers[i] = NewProducer(c)
	}
	return &KafkaProducerPool{producers}
}

// Producers returns the producers in the pool.
func (p *KafkaProducerPool) Producers() []Producer {
	return p.producers
}

// Close releases resources.
func (p *KafkaProducerPool) Close() {
	for _, producer := range p.producers {
		producer.Close()
	}
}
