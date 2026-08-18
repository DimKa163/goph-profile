package kafka

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

func EnsureTopic(ctx context.Context, broker string, topicName string,
	partitions int32) error {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(broker),
	)
	if err != nil {
		return err
	}
	defer cl.Close()

	admin := kadm.NewClient(cl)
	_, err = admin.CreateTopic(
		ctx,
		partitions,
		1,
		nil,
		topicName,
	)
	if errors.Is(err, kerr.TopicAlreadyExists) {
		return nil
	}
	return err
}
