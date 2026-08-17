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
		PartnerID: event.PartnerID, TransactionID: event.TransactionID,
		Payload: transactionPayload{
			TransactionID: event.Payload.ID, OccurredAt: event.Payload.OccurredAt.UTC().Format(time.RFC3339Nano),
			AmountMinor: event.Payload.AmountMinor, Currency: event.Payload.Currency,
			AgencyID: event.Payload.AgencyID, PlazaID: event.Payload.PlazaID, LaneID: event.Payload.LaneID,
			VehicleClass: event.Payload.VehicleClass,
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
	EventID       string             `json:"eventId"`
	EventType     string             `json:"eventType"`
	SchemaVersion int                `json:"schemaVersion"`
	OccurredAt    string             `json:"occurredAt"`
	CorrelationID string             `json:"correlationId"`
	PartnerID     string             `json:"partnerId"`
	TransactionID string             `json:"transactionId"`
	Payload       transactionPayload `json:"payload"`
}

type transactionPayload struct {
	TransactionID string              `json:"transactionId"`
	OccurredAt    string              `json:"occurredAt"`
	AmountMinor   int64               `json:"amountMinor"`
	Currency      string              `json:"currency"`
	AgencyID      string              `json:"agencyId"`
	PlazaID       string              `json:"plazaId"`
	LaneID        string              `json:"laneId"`
	VehicleClass  domain.VehicleClass `json:"vehicleClass"`
}
