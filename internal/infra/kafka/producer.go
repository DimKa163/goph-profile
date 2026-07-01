package kafka

import (
	"context"
	"time"

	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type ProducerOption func(p *kafkaProducer)

type kafkaProducer struct {
	client    *kgo.Client
	topicName string
}

func NewProducer(client *kgo.Client, opt ...ProducerOption) *kafkaProducer {
	k := &kafkaProducer{client: client}
	for _, o := range opt {
		o(k)
	}
	return k
}

func (p *kafkaProducer) Write(ctx context.Context, key []byte, value []byte, headers ...Header) error {
	rh := make([]kgo.RecordHeader, len(headers)+1)
	for i, h := range headers {
		rh[i] = kgo.RecordHeader{
			Key:   h.Key,
			Value: []byte(h.Value),
		}
	}
	rh[len(headers)] = kgo.RecordHeader{
		Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339)),
	}
	record := &kgo.Record{
		Topic:   p.topicName,
		Key:     key,
		Value:   value,
		Headers: rh,
	}
	p.client.Produce(ctx, record, func(record *kgo.Record, err error) {
		log := logging.Logger(ctx)
		if err != nil {
			log.Error("failed to produce message", zap.Error(err))
		} else {
			log.Debug("produced message", zap.String("topic", record.Topic),
				zap.Any("header", record.Headers),
				zap.Int32("partition", record.Partition),
				zap.String("key", string(record.Key)),
			)
		}
	})
	return p.client.Flush(ctx)
}

func Topic(topicName string) ProducerOption {
	return func(p *kafkaProducer) {
		p.topicName = topicName
	}
}
