package main

import (
	"testing"
	"time"
)

func TestLoadTopicSettingsUsesPlannedDefaults(t *testing.T) {
	t.Parallel()

	settings, err := loadTopicSettings(func(name string) string {
		if name == "KAFKA_BROKERS" {
			return "kafka:9092"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.Broker != "kafka:9092" || settings.Topic.Name != "transaction.review-candidates.v1" || settings.Topic.Partitions != 3 || settings.Topic.ReplicationFactor != 1 || settings.Topic.Retention != 7*24*time.Hour {
		t.Fatalf("unexpected defaults: %#v", settings)
	}
}

func TestLoadTopicSettingsRejectsMissingOrInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := loadTopicSettings(func(string) string { return "" }); err == nil {
		t.Fatal("expected missing broker to fail")
	}
	for name, value := range map[string]string{
		"KAFKA_TOPIC_PARTITIONS":  "zero",
		"KAFKA_TOPIC_REPLICATION": "0",
		"KAFKA_TOPIC_RETENTION":   "forever",
	} {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := loadTopicSettings(func(key string) string {
				if key == "KAFKA_BROKERS" {
					return "kafka:9092"
				}
				if key == name {
					return value
				}
				return ""
			})
			if err == nil {
				t.Fatal("expected invalid setting to fail")
			}
		})
	}
}
