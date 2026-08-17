package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
)

type topicSettings struct {
	Broker string
	Topic  kafkaadapter.TopicConfig
}

type kafkaConnection interface {
	kafkaadapter.TopicAdmin
	Controller() (kafkago.Broker, error)
	Close() error
}

type kafkaDial func(context.Context, string, string) (kafkaConnection, error)

func loadTopicSettings(lookup func(string) string) (topicSettings, error) {
	broker := strings.TrimSpace(strings.Split(lookup("KAFKA_BROKERS"), ",")[0])
	if broker == "" {
		return topicSettings{}, fmt.Errorf("KAFKA_BROKERS is required")
	}
	name := lookup("KAFKA_TOPIC")
	if name == "" {
		name = "transaction.review-candidates.v1"
	}
	partitions, err := positiveInteger(lookup("KAFKA_TOPIC_PARTITIONS"), 3)
	if err != nil {
		return topicSettings{}, fmt.Errorf("KAFKA_TOPIC_PARTITIONS: %w", err)
	}
	replication, err := positiveInteger(lookup("KAFKA_TOPIC_REPLICATION"), 1)
	if err != nil {
		return topicSettings{}, fmt.Errorf("KAFKA_TOPIC_REPLICATION: %w", err)
	}
	retention := 7 * 24 * time.Hour
	if raw := lookup("KAFKA_TOPIC_RETENTION"); raw != "" {
		retention, err = time.ParseDuration(raw)
		if err != nil || retention <= 0 {
			return topicSettings{}, fmt.Errorf("KAFKA_TOPIC_RETENTION must be a positive duration")
		}
	}
	return topicSettings{Broker: broker, Topic: kafkaadapter.TopicConfig{
		Name: name, Partitions: partitions, ReplicationFactor: replication, Retention: retention,
	}}, nil
}

func positiveInteger(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return value, nil
}

func runTopicBootstrap(ctx context.Context, settings topicSettings, dial kafkaDial) error {
	seed, err := dial(ctx, "tcp", settings.Broker)
	if err != nil {
		return fmt.Errorf("connect to Kafka: %w", err)
	}
	controller, err := seed.Controller()
	_ = seed.Close()
	if err != nil {
		return fmt.Errorf("discover Kafka controller: %w", err)
	}
	admin, err := dial(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("connect to Kafka controller: %w", err)
	}
	defer admin.Close()
	if err := kafkaadapter.EnsureTopic(admin, settings.Topic); err != nil {
		return fmt.Errorf("ensure Kafka topic: %w", err)
	}
	return nil
}

func execute(ctx context.Context, lookup func(string) string, dial kafkaDial) (topicSettings, error) {
	settings, err := loadTopicSettings(lookup)
	if err != nil {
		return topicSettings{}, err
	}
	if err := runTopicBootstrap(ctx, settings, dial); err != nil {
		return topicSettings{}, err
	}
	return settings, nil
}

func runCLI(lookup func(string) string, dial kafkaDial) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	settings, err := execute(ctx, lookup, dial)
	if err != nil {
		return err
	}
	slog.Info("Kafka topic ready", "topic", settings.Topic.Name)
	return nil
}

func realKafkaDial(ctx context.Context, network, address string) (kafkaConnection, error) {
	dialer := &kafkago.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, address)
}

func exitCode(err error) int {
	if err != nil {
		slog.Error("bootstrap Kafka topic", "error", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(exitCode(runCLI(os.Getenv, realKafkaDial)))
}
