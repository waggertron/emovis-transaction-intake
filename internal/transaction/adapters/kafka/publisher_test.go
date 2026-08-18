package kafkaadapter

import (
	"context"
	"encoding/json"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
	"strings"
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

func TestPublisherEmitsNumberSafePassthroughFromRawAuditFields(t *testing.T) {
	w := &writer{}
	event := app.OutboxEvent{ID: "evt-raw", Type: "transaction.review-candidate", SchemaVersion: 1, OccurredAt: time.Now().UTC(), Source: "s", SourceReference: "r", TransactionID: "id", Key: "s:r", Payload: domain.Transaction{
		ID: "id", Source: "s", SourceReference: "r", TransactionType: "toll", TransactionTimeUTC: time.Now().UTC(), BaseAmount: "12.50", Currency: "USD", TransponderNumber: "tag",
		LocationRaw: json.RawMessage(`{"lane":9007199254740993}`), MetadataRaw: json.RawMessage(`{"rate":12.50}`),
	}}
	if err := NewPublisher(w, "topic").Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(w.m.Value)))
	decoder.UseNumber()
	var envelope map[string]any
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	payload := envelope["payload"].(map[string]any)
	if payload["location"].(map[string]any)["lane"] != json.Number("9007199254740993") {
		t.Fatalf("location=%#v", payload["location"])
	}
	if payload["metadata"].(map[string]any)["rate"] != json.Number("12.50") {
		t.Fatalf("metadata=%#v", payload["metadata"])
	}
}

func TestRawObjectFallsBackToParsedValues(t *testing.T) {
	if got := string(rawObject(nil, map[string]any{"lane": json.Number("9007199254740993")})); got != `{"lane":9007199254740993}` {
		t.Fatalf("raw object fallback=%s", got)
	}
	if rawObject(nil, nil) != nil {
		t.Fatal("absent object must remain absent")
	}
}
