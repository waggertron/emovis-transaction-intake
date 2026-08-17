package kafkaadapter

import (
	"crypto/tls"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestNewWriterRejectsUnsafeOrIncompleteConfig(t *testing.T) {
	t.Parallel()

	for _, config := range []WriterConfig{
		{},
		{Brokers: []string{"kafka:9092"}},
		{Brokers: []string{"kafka:9092"}, Topic: "topic", SASLUsername: "user", SASLPassword: "password"},
		{Brokers: []string{"kafka:9092"}, Topic: "topic", TLS: true, SASLUsername: "user"},
	} {
		if _, err := NewWriter(config); err == nil {
			t.Fatalf("expected invalid config to fail: %#v", config)
		}
	}
}

func TestNewWriterBuildsBoundedSecureKafkaClient(t *testing.T) {
	t.Parallel()

	writer, err := NewWriter(WriterConfig{
		Brokers: []string{"broker-1:9094", "broker-2:9094"}, Topic: "transaction.review-candidates.v1",
		TLS: true, SASLUsername: "user", SASLPassword: "external-secret",
	})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	if writer.Topic != "" || writer.RequiredAcks != kafkago.RequireAll || writer.Async || writer.MaxAttempts != 5 {
		t.Fatalf("unexpected writer reliability config: %#v", writer)
	}
	if writer.WriteTimeout != 10*time.Second || writer.ReadTimeout != 10*time.Second || writer.AllowAutoTopicCreation {
		t.Fatalf("unexpected writer bounds: %#v", writer)
	}
	if _, ok := writer.Balancer.(*kafkago.Hash); !ok {
		t.Fatalf("expected hash balancer, got %T", writer.Balancer)
	}
	transport, ok := writer.Transport.(*kafkago.Transport)
	if !ok || transport.TLS == nil || transport.TLS.MinVersion != tls.VersionTLS12 || transport.SASL == nil {
		t.Fatalf("unexpected secure transport: %#v", writer.Transport)
	}
}
