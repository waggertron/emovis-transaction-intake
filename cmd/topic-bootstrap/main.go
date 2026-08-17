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

func main() {
	settings, err := loadTopicSettings(os.Getenv)
	if err != nil {
		slog.Error("invalid topic settings", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dialer := &kafkago.Dialer{Timeout: 10 * time.Second}
	seed, err := dialer.DialContext(ctx, "tcp", settings.Broker)
	if err != nil {
		slog.Error("connect to Kafka", "error", err)
		os.Exit(1)
	}
	controller, err := seed.Controller()
	_ = seed.Close()
	if err != nil {
		slog.Error("discover Kafka controller", "error", err)
		os.Exit(1)
	}
	admin, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		slog.Error("connect to Kafka controller", "error", err)
		os.Exit(1)
	}
	defer admin.Close()
	if err := kafkaadapter.EnsureTopic(admin, settings.Topic); err != nil {
		slog.Error("ensure Kafka topic", "error", err)
		os.Exit(1)
	}
	slog.Info("Kafka topic ready", "topic", settings.Topic.Name)
}
