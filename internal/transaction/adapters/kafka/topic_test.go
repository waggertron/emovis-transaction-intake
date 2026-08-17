package kafkaadapter

import (
	"errors"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type fakeTopicAdmin struct {
	configs []kafkago.TopicConfig
	err     error
}

func (admin *fakeTopicAdmin) CreateTopics(configs ...kafkago.TopicConfig) error {
	admin.configs = append(admin.configs, configs...)
	return admin.err
}

func TestEnsureTopicCreatesConfiguredTopic(t *testing.T) {
	t.Parallel()

	admin := &fakeTopicAdmin{}
	err := EnsureTopic(admin, TopicConfig{Name: "transaction.review-candidates.v1", Partitions: 3, ReplicationFactor: 1, Retention: 7 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("ensure topic: %v", err)
	}
	if len(admin.configs) != 1 {
		t.Fatalf("expected one topic config, got %d", len(admin.configs))
	}
	config := admin.configs[0]
	if config.Topic != "transaction.review-candidates.v1" || config.NumPartitions != 3 || config.ReplicationFactor != 1 || len(config.ConfigEntries) != 1 || config.ConfigEntries[0].ConfigName != "retention.ms" || config.ConfigEntries[0].ConfigValue != "604800000" {
		t.Fatalf("unexpected topic config: %#v", config)
	}
}

func TestEnsureTopicIsIdempotentAndRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	admin := &fakeTopicAdmin{err: kafkago.TopicAlreadyExists}
	if err := EnsureTopic(admin, TopicConfig{Name: "topic", Partitions: 1, ReplicationFactor: 1, Retention: time.Hour}); err != nil {
		t.Fatalf("existing topic should succeed: %v", err)
	}
	if err := EnsureTopic(&fakeTopicAdmin{}, TopicConfig{}); err == nil {
		t.Fatal("expected invalid config to fail")
	}
	otherErr := errors.New("controller unavailable")
	if err := EnsureTopic(&fakeTopicAdmin{err: otherErr}, TopicConfig{Name: "topic", Partitions: 1, ReplicationFactor: 1, Retention: time.Hour}); !errors.Is(err, otherErr) {
		t.Fatalf("expected wrapped admin error, got %v", err)
	}
}
