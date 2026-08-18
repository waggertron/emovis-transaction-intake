package kafkaadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

type MessageWriter interface {
	WriteMessages(context.Context, ...kafkago.Message) error
}

type Publisher struct {
	writer MessageWriter
	topic  string
}

func NewPublisher(writer MessageWriter, topic string) *Publisher {
	return &Publisher{writer: writer, topic: topic}
}

func (publisher *Publisher) Publish(ctx context.Context, event app.OutboxEvent) error {
	value, err := json.Marshal(eventEnvelope{
		EventID: event.ID, EventType: event.Type, SchemaVersion: event.SchemaVersion,
		OccurredAt: event.OccurredAt.UTC().Format(time.RFC3339Nano), CorrelationID: event.CorrelationID,
		Source: event.Source, SourceReference: event.SourceReference, TransactionID: event.TransactionID,
		Payload: transactionPayload{
			ID: event.Payload.ID, Source: event.Payload.Source, SourceReference: event.Payload.SourceReference,
			TransactionType: event.Payload.TransactionType, TransactionTimeUTC: event.Payload.TransactionTimeUTC.UTC().Format(time.RFC3339Nano),
			BaseAmount: event.Payload.BaseAmount, Currency: event.Payload.Currency, Plate: event.Payload.Plate,
			TransponderNumber: event.Payload.TransponderNumber,
			Location:          rawObject(event.Payload.LocationRaw, event.Payload.Location), Metadata: rawObject(event.Payload.MetadataRaw, event.Payload.Metadata),
		},
	})
	if err != nil {
		return fmt.Errorf("encode event %s: %w", event.ID, err)
	}
	message := kafkago.Message{
		Topic: publisher.topic,
		Key:   []byte(event.Key),
		Value: value,
		Time:  event.OccurredAt.UTC(),
		Headers: []kafkago.Header{
			{Key: "event-id", Value: []byte(event.ID)},
			{Key: "correlation-id", Value: []byte(event.CorrelationID)},
		},
	}
	if err := publisher.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("publish event %s: %w", event.ID, err)
	}
	return nil
}

type eventEnvelope struct {
	EventID         string             `json:"eventId"`
	EventType       string             `json:"eventType"`
	SchemaVersion   int                `json:"schemaVersion"`
	OccurredAt      string             `json:"occurredAt"`
	CorrelationID   string             `json:"correlationId"`
	Source          string             `json:"source"`
	SourceReference string             `json:"sourceReference"`
	TransactionID   string             `json:"transactionId"`
	Payload         transactionPayload `json:"payload"`
}

type transactionPayload struct {
	ID                 string          `json:"id"`
	Source             string          `json:"source"`
	SourceReference    string          `json:"source_reference"`
	TransactionType    string          `json:"transaction_type"`
	TransactionTimeUTC string          `json:"transaction_time_utc"`
	BaseAmount         string          `json:"base_amount"`
	Currency           string          `json:"currency"`
	Plate              *domain.Plate   `json:"plate,omitempty"`
	TransponderNumber  string          `json:"transponder_number,omitempty"`
	Location           json.RawMessage `json:"location,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

func rawObject(raw json.RawMessage, parsed map[string]any) json.RawMessage {
	if len(raw) > 0 {
		return raw
	}
	if parsed == nil {
		return nil
	}
	payload, _ := json.Marshal(parsed)
	return payload
}
