package kafka

import (
	"context"
	"errors"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/twmb/franz-go/pkg/kgo"
)

var ErrMessageEmpty = errors.New("message empty")

type (
	Message struct {
		Topic       string
		Key         []byte
		Body        []byte
		ContentType string
		EventID     string
		TaskType    entity.TaskType
	}
)
type Producer interface {
	Produce(ctx context.Context, message ...*Message) error
	Close()
}
type kafkaProducer struct {
	client *kgo.Client
}

func NewProducer(client *kgo.Client) *kafkaProducer {
	k := &kafkaProducer{client: client}
	return k
}

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

func (p *kafkaProducer) Close() {
	p.client.Close()
}

type PoolConfiger func() (*kgo.Client, error)
type KafkaProducerPool struct {
	producers []Producer
}

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

func (p *KafkaProducerPool) Producers() []Producer {
	return p.producers
}

func (p *KafkaProducerPool) Close() {
	for _, producer := range p.producers {
		producer.Close()
	}
}
