package kafkaadapter

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type TopicAdmin interface {
	CreateTopics(...kafkago.TopicConfig) error
}

type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	Retention         time.Duration
}

func EnsureTopic(admin TopicAdmin, config TopicConfig) error {
	if config.Name == "" || config.Partitions <= 0 || config.ReplicationFactor <= 0 || config.Retention <= 0 {
		return fmt.Errorf("topic name, partitions, replication factor, and retention must be configured")
	}
	err := admin.CreateTopics(kafkago.TopicConfig{
		Topic: config.Name, NumPartitions: config.Partitions, ReplicationFactor: config.ReplicationFactor,
		ConfigEntries: []kafkago.ConfigEntry{{
			ConfigName: "retention.ms", ConfigValue: strconv.FormatInt(config.Retention.Milliseconds(), 10),
		}},
	})
	if errors.Is(err, kafkago.TopicAlreadyExists) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create Kafka topic %q: %w", config.Name, err)
	}
	return nil
}
