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
	ID              string
	Type            string
	SchemaVersion   int
	OccurredAt      time.Time
	CorrelationID   string
	Source          string
	SourceReference string
	TransactionID   string
	Key             string
	Payload         domain.Transaction
}

type Acceptance struct {
	Transaction domain.Transaction
	Fingerprint string
	Event       OutboxEvent
}

type StoreOutcome struct {
	Kind          StoreOutcomeKind
	TransactionID string
	EventID       string
}

type IntakeStore interface {
	Accept(context.Context, Acceptance) (StoreOutcome, error)
}

type AcceptCommand struct {
	Transaction   domain.Transaction
	CorrelationID string
}

type AcceptResult struct {
	Kind    ResultKind
	ID      string
	EventID string
}

type IntakeService struct {
	store        IntakeStore
	now          func() time.Time
	newID        func() string
	allowedTypes map[string]struct{}
}

func NewIntakeService(store IntakeStore, now func() time.Time, newID func() string, typeSets ...map[string]struct{}) *IntakeService {
	allowedTypes := map[string]struct{}{"toll": {}}
	if len(typeSets) > 0 && typeSets[0] != nil {
		allowedTypes = typeSets[0]
	}
	return &IntakeService{store: store, now: now, newID: newID, allowedTypes: allowedTypes}
}

func (service *IntakeService) Accept(ctx context.Context, command AcceptCommand) (AcceptResult, error) {
	if err := command.Transaction.Validate(service.allowedTypes); err != nil {
		return AcceptResult{}, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}
	fingerprint, err := command.Transaction.Fingerprint(service.allowedTypes)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	eventID := service.newID()
	transaction := command.Transaction
	if transaction.ID == "" {
		transaction.ID = service.newID()
	}
	outcome, err := service.store.Accept(ctx, Acceptance{
		Transaction: transaction,
		Fingerprint: fingerprint,
		Event: OutboxEvent{
			ID:              eventID,
			Type:            ReviewCandidateEventType,
			SchemaVersion:   1,
			OccurredAt:      service.now().UTC(),
			CorrelationID:   command.CorrelationID,
			Source:          transaction.Source,
			SourceReference: transaction.SourceReference,
			TransactionID:   transaction.ID,
			Key:             transaction.Source + ":" + transaction.SourceReference,
			Payload:         transaction,
		},
	})
	if err != nil {
		return AcceptResult{}, fmt.Errorf("accept transaction: %w", err)
	}

	result := AcceptResult{ID: transaction.ID, EventID: outcome.EventID}
	switch outcome.Kind {
	case StoreAccepted:
		result.Kind = Accepted
		return result, nil
	case StoreReplay:
		if outcome.TransactionID == "" {
			return AcceptResult{}, fmt.Errorf("replay outcome is missing transaction ID")
		}
		result.Kind = Replayed
		result.ID = outcome.TransactionID
		return result, nil
	case StoreConflict:
		return AcceptResult{}, ErrConflict
	default:
		return AcceptResult{}, fmt.Errorf("unknown store outcome %q", outcome.Kind)
	}
}
