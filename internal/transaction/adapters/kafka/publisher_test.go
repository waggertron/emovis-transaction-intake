package kafkaadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

type fakeWriter struct {
	messages []kafkago.Message
	err      error
}

func (writer *fakeWriter) WriteMessages(_ context.Context, messages ...kafkago.Message) error {
	writer.messages = append(writer.messages, messages...)
	return writer.err
}

func kafkaTestEvent() app.OutboxEvent {
	return app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1,
		OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC), CorrelationID: "req-1",
		PartnerID: "partner-west", TransactionID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01",
		Key: "partner-west:018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01",
		Payload: domain.Transaction{
			ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", PartnerID: "partner-west",
			OccurredAt: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC), AmountMinor: 725,
			Currency: "USD", AgencyID: "agency-17", PlazaID: "plaza-4", LaneID: "lane-2",
			VehicleClass: domain.VehicleClassCar,
		},
	}
}

func TestPublisherWritesVersionedMessageWithStableKey(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	publisher := NewPublisher(writer, "transaction.review-candidates.v1")
	if err := publisher.Publish(context.Background(), kafkaTestEvent()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(writer.messages))
	}
	message := writer.messages[0]
	if message.Topic != "transaction.review-candidates.v1" || string(message.Key) != kafkaTestEvent().Key {
		t.Fatalf("unexpected destination: %#v", message)
	}
	var envelope map[string]any
	if err := json.Unmarshal(message.Value, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope["eventId"] != "evt-1" || envelope["eventType"] != app.ReviewCandidateEventType || envelope["schemaVersion"] != float64(1) {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
	payload, ok := envelope["payload"].(map[string]any)
	if !ok || payload["transactionId"] != kafkaTestEvent().TransactionID || payload["amountMinor"] != float64(725) {
		t.Fatalf("unexpected payload: %#v", envelope["payload"])
	}
	for _, forbidden := range []string{"apiKey", "authorization", "xApiKey"} {
		if _, found := envelope[forbidden]; found {
			t.Fatalf("unsafe field %q in envelope", forbidden)
		}
	}
}

func TestPublisherWrapsWriterFailure(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("broker unavailable")
	publisher := NewPublisher(&fakeWriter{err: writeErr}, "transaction.review-candidates.v1")
	if err := publisher.Publish(context.Background(), kafkaTestEvent()); !errors.Is(err, writeErr) {
		t.Fatalf("expected wrapped writer error, got %v", err)
	}
}
