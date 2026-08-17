package kafkaadapter

import (
	"context"
	"encoding/json"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
	"testing"
	"time"
)

type writer struct{ m kafkago.Message }

func (w *writer) WriteMessages(_ context.Context, m ...kafkago.Message) error { w.m = m[0]; return nil }
func TestPublisherWritesNewPayload(t *testing.T) {
	w := &writer{}
	p := NewPublisher(w, "topic")
	e := app.OutboxEvent{ID: "evt", Type: "type", SchemaVersion: 1, OccurredAt: time.Now(), Source: "s", SourceReference: "r", TransactionID: "id", Key: "s:r", Payload: domain.Transaction{ID: "id", Source: "s", SourceReference: "r", TransactionType: "toll", TransactionTimeUTC: time.Now(), BaseAmount: "1.25", TransponderNumber: "tag"}}
	if err := p.Publish(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(w.m.Value, &v); err != nil {
		t.Fatal(err)
	}
	if v["source"] != "s" {
		t.Fatalf("payload=%v", v)
	}
}

type failingWriter struct{}

func (failingWriter) WriteMessages(context.Context, ...kafkago.Message) error {
	return context.Canceled
}
func TestPublisherPropagatesWriterFailure(t *testing.T) {
	p := NewPublisher(failingWriter{}, "topic")
	if err := p.Publish(context.Background(), app.OutboxEvent{ID: "e"}); err == nil {
		t.Fatal("expected publish failure")
	}
}
