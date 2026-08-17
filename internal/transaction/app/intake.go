package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

const ReviewCandidateEventType = "transaction.review-candidate"

var (
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrConflict           = errors.New("transaction conflicts with an existing payload")
)

type StoreOutcomeKind string

const (
	StoreAccepted StoreOutcomeKind = "accepted"
	StoreReplay   StoreOutcomeKind = "replay"
	StoreConflict StoreOutcomeKind = "conflict"
)

type ResultKind string

const (
	Accepted ResultKind = "accepted"
	Replayed ResultKind = "replayed"
)

type OutboxEvent struct {
	ID            string
	Type          string
	SchemaVersion int
	OccurredAt    time.Time
	CorrelationID string
	PartnerID     string
	TransactionID string
	Key           string
	Payload       domain.Transaction
}

type Acceptance struct {
	Transaction domain.Transaction
	Fingerprint string
	Event       OutboxEvent
}

type StoreOutcome struct {
	Kind    StoreOutcomeKind
	EventID string
}

type TransactionStore interface {
	Accept(context.Context, Acceptance) (StoreOutcome, error)
}

type AcceptCommand struct {
	Transaction   domain.Transaction
	CorrelationID string
}

type AcceptResult struct {
	Kind          ResultKind
	TransactionID string
	EventID       string
}

type IntakeService struct {
	store TransactionStore
	now   func() time.Time
	newID func() string
}

func NewIntakeService(store TransactionStore, now func() time.Time, newID func() string) *IntakeService {
	return &IntakeService{store: store, now: now, newID: newID}
}

func (service *IntakeService) Accept(ctx context.Context, command AcceptCommand) (AcceptResult, error) {
	if err := command.Transaction.Validate(); err != nil {
		return AcceptResult{}, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}
	fingerprint, err := command.Transaction.Fingerprint()
	if err != nil {
		return AcceptResult{}, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	eventID := service.newID()
	transaction := command.Transaction
	outcome, err := service.store.Accept(ctx, Acceptance{
		Transaction: transaction,
		Fingerprint: fingerprint,
		Event: OutboxEvent{
			ID:            eventID,
			Type:          ReviewCandidateEventType,
			SchemaVersion: 1,
			OccurredAt:    service.now().UTC(),
			CorrelationID: command.CorrelationID,
			PartnerID:     transaction.PartnerID,
			TransactionID: transaction.ID,
			Key:           transaction.PartnerID + ":" + transaction.ID,
			Payload:       transaction,
		},
	})
	if err != nil {
		return AcceptResult{}, fmt.Errorf("accept transaction: %w", err)
	}

	result := AcceptResult{TransactionID: transaction.ID, EventID: outcome.EventID}
	switch outcome.Kind {
	case StoreAccepted:
		result.Kind = Accepted
		return result, nil
	case StoreReplay:
		result.Kind = Replayed
		return result, nil
	case StoreConflict:
		return AcceptResult{}, ErrConflict
	default:
		return AcceptResult{}, fmt.Errorf("unknown store outcome %q", outcome.Kind)
	}
}
