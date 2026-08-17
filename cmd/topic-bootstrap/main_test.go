package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
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

func TestSecureKafkaDialRejectsInvalidTrustAndHonorsContext(t *testing.T) {
	t.Parallel()
	settings := topicSettings{Security: kafkaadapter.SecurityConfig{TLS: true, CAFile: filepath.Join(t.TempDir(), "absent.pem")}}
	if _, err := secureKafkaDial(settings); err == nil {
		t.Fatal("expected invalid CA failure")
	}
	dial, err := secureKafkaDial(topicSettings{})
	if err != nil {
		t.Fatalf("plaintext dialer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dial(ctx, "tcp", "127.0.0.1:1"); err == nil {
		t.Fatal("expected canceled connection")
	}
}

func TestRunProductionCLIRejectsInvalidAndUntrustedConfiguration(t *testing.T) {
	t.Parallel()
	if err := runProductionCLI(func(string) string { return "" }); err == nil {
		t.Fatal("expected missing broker rejection")
	}
	values := map[string]string{
		"KAFKA_BROKERS": "kafka:9094", "KAFKA_TLS": "true", "KAFKA_CA_FILE": filepath.Join(t.TempDir(), "absent.pem"),
	}
	if err := runProductionCLI(func(name string) string { return values[name] }); err == nil {
		t.Fatal("expected untrusted configuration rejection")
	}
}

func TestLoadTopicSettingsParsesTLSAndSCRAM(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"KAFKA_BROKERS": "kafka-secure:9094", "KAFKA_TLS": "true", "KAFKA_CA_FILE": "/run/secrets/ca.pem",
		"KAFKA_SASL_USERNAME": "transaction", "KAFKA_SASL_PASSWORD": "external-password",
	}
	settings, err := loadTopicSettings(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("load secure settings: %v", err)
	}
	if !settings.Security.TLS || settings.Security.CAFile != "/run/secrets/ca.pem" || settings.Security.SASLUsername != "transaction" || settings.Security.SASLPassword != "external-password" {
		t.Fatalf("unexpected security settings: %#v", settings.Security)
	}
}

type fakeKafkaConnection struct {
	controller    kafkago.Broker
	controllerErr error
	createErr     error
	closed        bool
}

func (connection *fakeKafkaConnection) Controller() (kafkago.Broker, error) {
	return connection.controller, connection.controllerErr
}
func (connection *fakeKafkaConnection) CreateTopics(...kafkago.TopicConfig) error {
	return connection.createErr
}
func (connection *fakeKafkaConnection) Close() error {
	connection.closed = true
	return nil
}

func TestRunTopicBootstrapDiscoversControllerAndEnsuresTopic(t *testing.T) {
	t.Parallel()
	seed := &fakeKafkaConnection{controller: kafkago.Broker{Host: "controller", Port: 9093}}
	admin := &fakeKafkaConnection{}
	addresses := []string{}
	dial := func(_ context.Context, _, address string) (kafkaConnection, error) {
		addresses = append(addresses, address)
		if len(addresses) == 1 {
			return seed, nil
		}
		return admin, nil
	}
	settings := topicSettings{Broker: "broker:9092", Topic: validTopicConfig()}
	if err := runTopicBootstrap(context.Background(), settings, dial); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(addresses) != 2 || addresses[1] != "controller:9093" || !seed.closed || !admin.closed {
		t.Fatalf("unexpected lifecycle: %#v seedClosed=%t adminClosed=%t", addresses, seed.closed, admin.closed)
	}
}

func TestRunTopicBootstrapReportsConnectionAndKafkaFailures(t *testing.T) {
	t.Parallel()
	want := errors.New("unavailable")
	settings := topicSettings{Broker: "broker:9092", Topic: validTopicConfig()}
	if err := runTopicBootstrap(context.Background(), settings, func(context.Context, string, string) (kafkaConnection, error) { return nil, want }); !errors.Is(err, want) {
		t.Fatalf("expected seed error, got %v", err)
	}
	seed := &fakeKafkaConnection{controllerErr: want}
	if err := runTopicBootstrap(context.Background(), settings, func(context.Context, string, string) (kafkaConnection, error) { return seed, nil }); !errors.Is(err, want) || !seed.closed {
		t.Fatalf("expected controller error and close, got %v", err)
	}
	seed = &fakeKafkaConnection{controller: kafkago.Broker{Host: "controller", Port: 9093}}
	calls := 0
	if err := runTopicBootstrap(context.Background(), settings, func(context.Context, string, string) (kafkaConnection, error) {
		calls++
		if calls == 1 {
			return seed, nil
		}
		return nil, want
	}); !errors.Is(err, want) {
		t.Fatalf("expected admin connection error, got %v", err)
	}
	admin := &fakeKafkaConnection{createErr: want}
	calls = 0
	if err := runTopicBootstrap(context.Background(), settings, func(context.Context, string, string) (kafkaConnection, error) {
		calls++
		if calls == 1 {
			return seed, nil
		}
		return admin, nil
	}); !errors.Is(err, want) || !admin.closed {
		t.Fatalf("expected topic error and close, got %v", err)
	}
}

func TestExecuteLoadsConfigurationAndRunsBootstrap(t *testing.T) {
	t.Parallel()
	seed := &fakeKafkaConnection{controller: kafkago.Broker{Host: "controller", Port: 9093}}
	admin := &fakeKafkaConnection{}
	calls := 0
	settings, err := execute(context.Background(), func(name string) string {
		if name == "KAFKA_BROKERS" {
			return "broker:9092"
		}
		return ""
	}, func(context.Context, string, string) (kafkaConnection, error) {
		calls++
		if calls == 1 {
			return seed, nil
		}
		return admin, nil
	})
	if err != nil || settings.Broker != "broker:9092" || calls != 2 {
		t.Fatalf("execute result %#v calls=%d err=%v", settings, calls, err)
	}
	if _, err := execute(context.Background(), func(string) string { return "" }, nil); err == nil {
		t.Fatal("expected invalid settings")
	}
}

func TestRunCLIAndExitCode(t *testing.T) {
	t.Parallel()
	seed := &fakeKafkaConnection{controller: kafkago.Broker{Host: "controller", Port: 9093}}
	admin := &fakeKafkaConnection{}
	calls := 0
	err := runCLI(func(name string) string {
		if name == "KAFKA_BROKERS" {
			return "broker:9092"
		}
		return ""
	}, func(context.Context, string, string) (kafkaConnection, error) {
		calls++
		if calls == 1 {
			return seed, nil
		}
		return admin, nil
	})
	if err != nil || exitCode(nil) != 0 || exitCode(errors.New("failed")) != 1 {
		t.Fatalf("unexpected CLI result: %v", err)
	}
}

func validTopicConfig() kafkaadapter.TopicConfig {
	return kafkaadapter.TopicConfig{Name: "events", Partitions: 3, ReplicationFactor: 1, Retention: time.Hour}
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
