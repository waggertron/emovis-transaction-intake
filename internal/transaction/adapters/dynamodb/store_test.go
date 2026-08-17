package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

type fakeClient struct {
	item         map[string]types.AttributeValue
	getInput     *awssdk.GetItemInput
	writeInput   *awssdk.TransactWriteItemsInput
	queryInput   *awssdk.QueryInput
	queryItems   []map[string]types.AttributeValue
	updateInputs []*awssdk.UpdateItemInput
	updateErrors []error
	getErr       error
	writeErr     error
}

func (client *fakeClient) GetItem(_ context.Context, input *awssdk.GetItemInput, _ ...func(*awssdk.Options)) (*awssdk.GetItemOutput, error) {
	client.getInput = input
	return &awssdk.GetItemOutput{Item: client.item}, client.getErr
}

func (client *fakeClient) TransactWriteItems(_ context.Context, input *awssdk.TransactWriteItemsInput, _ ...func(*awssdk.Options)) (*awssdk.TransactWriteItemsOutput, error) {
	client.writeInput = input
	return &awssdk.TransactWriteItemsOutput{}, client.writeErr
}

func (client *fakeClient) Query(_ context.Context, input *awssdk.QueryInput, _ ...func(*awssdk.Options)) (*awssdk.QueryOutput, error) {
	client.queryInput = input
	return &awssdk.QueryOutput{Items: client.queryItems}, nil
}

func (client *fakeClient) UpdateItem(_ context.Context, input *awssdk.UpdateItemInput, _ ...func(*awssdk.Options)) (*awssdk.UpdateItemOutput, error) {
	client.updateInputs = append(client.updateInputs, input)
	if len(client.updateErrors) == 0 {
		return &awssdk.UpdateItemOutput{}, nil
	}
	err := client.updateErrors[0]
	client.updateErrors = client.updateErrors[1:]
	return &awssdk.UpdateItemOutput{}, err
}

func dynamoAcceptance() app.Acceptance {
	transaction := domain.Transaction{
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", PartnerID: "partner-west",
		OccurredAt: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC), AmountMinor: 725,
		Currency: "USD", AgencyID: "agency-17", PlazaID: "plaza-4", LaneID: "lane-2", VehicleClass: domain.VehicleClassCar,
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1, OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC),
		PartnerID: transaction.PartnerID, TransactionID: transaction.ID, Key: transaction.PartnerID + ":" + transaction.ID, Payload: transaction,
	}}
}

func TestStoreAcceptsWithConditionalTransactionalWrite(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	result, err := NewStore(client, "transactions").Accept(context.Background(), dynamoAcceptance())
	if err != nil || result.Kind != app.StoreAccepted || result.EventID != "evt-1" {
		t.Fatalf("unexpected accept %#v, %v", result, err)
	}
	if client.getInput == nil || client.getInput.ConsistentRead == nil || !*client.getInput.ConsistentRead || client.writeInput == nil || len(client.writeInput.TransactItems) != 2 {
		t.Fatalf("unexpected DynamoDB calls: %#v / %#v", client.getInput, client.writeInput)
	}
	for _, item := range client.writeInput.TransactItems {
		if item.Put == nil || item.Put.ConditionExpression == nil || *item.Put.ConditionExpression != "attribute_not_exists(pk)" {
			t.Fatalf("transactional put is not conditional: %#v", item)
		}
	}
	outbox := client.writeInput.TransactItems[1].Put.Item
	if outbox["dispatch_pk"].(*types.AttributeValueMemberS).Value != "OUTBOX#PENDING" || outbox["dispatch_sk"] == nil {
		t.Fatalf("outbox is not indexed for dispatch: %#v", outbox)
	}
}

func TestStoreReturnsReplayOrConflictFromExistingIdentity(t *testing.T) {
	t.Parallel()

	acceptance := dynamoAcceptance()
	for _, test := range []struct {
		fingerprint string
		want        app.StoreOutcomeKind
	}{
		{fingerprint: acceptance.Fingerprint, want: app.StoreReplay},
		{fingerprint: "different", want: app.StoreConflict},
	} {
		client := &fakeClient{item: map[string]types.AttributeValue{
			"fingerprint": &types.AttributeValueMemberS{Value: test.fingerprint},
			"event_id":    &types.AttributeValueMemberS{Value: "evt-original"},
		}}
		result, err := NewStore(client, "transactions").Accept(context.Background(), acceptance)
		if err != nil || result.Kind != test.want || client.writeInput != nil {
			t.Fatalf("unexpected identity result %#v, %v", result, err)
		}
		if test.want == app.StoreReplay && result.EventID != "evt-original" {
			t.Fatalf("expected original event, got %q", result.EventID)
		}
	}
}

func TestStoreClaimsDueOutboxEventsWithConditionalLease(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(dynamoAcceptance().Event)
	client := &fakeClient{queryItems: []map[string]types.AttributeValue{
		{
			"pk":            &types.AttributeValueMemberS{Value: "EVENT#evt-1"},
			"event_payload": &types.AttributeValueMemberB{Value: payload},
			"attempts":      &types.AttributeValueMemberN{Value: "2"},
		},
		{
			"pk":            &types.AttributeValueMemberS{Value: "EVENT#evt-raced"},
			"event_payload": &types.AttributeValueMemberB{Value: payload},
			"attempts":      &types.AttributeValueMemberN{Value: "0"},
		},
	}, updateErrors: []error{nil, &types.ConditionalCheckFailedException{Message: aws.String("already leased")}}}

	events, err := NewStore(client, "transactions").ClaimPending(context.Background(), now, 30*time.Second, 10)
	if err != nil || len(events) != 1 || events[0].Event.ID != "evt-1" || events[0].Attempts != 2 {
		t.Fatalf("unexpected claim %#v, %v", events, err)
	}
	if client.queryInput == nil || aws.ToString(client.queryInput.IndexName) != "outbox-dispatch" || aws.ToInt32(client.queryInput.Limit) != 10 {
		t.Fatalf("unexpected dispatch query: %#v", client.queryInput)
	}
	if len(client.updateInputs) != 2 {
		t.Fatalf("expected a conditional lease attempt per candidate, got %d", len(client.updateInputs))
	}
	lease := client.updateInputs[0]
	if !strings.Contains(aws.ToString(lease.ConditionExpression), "lease_until") || !strings.Contains(aws.ToString(lease.UpdateExpression), "lease_until") {
		t.Fatalf("lease update is not conditional: %#v", lease)
	}
}

func TestStoreRecordsPublishedRetryAndTerminalOutcomes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	client := &fakeClient{}
	store := NewStore(client, "transactions")
	if err := store.MarkPublished(context.Background(), "evt-1", now); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	if err := store.RecordFailure(context.Background(), app.PublishFailure{
		EventID: "evt-2", Attempts: 3, RetryAt: now.Add(time.Minute), Reason: "publish_failed",
	}); err != nil {
		t.Fatalf("record retry: %v", err)
	}
	if err := store.RecordFailure(context.Background(), app.PublishFailure{
		EventID: "evt-3", Attempts: 5, Terminal: true, Reason: "publish_failed",
	}); err != nil {
		t.Fatalf("record terminal failure: %v", err)
	}
	if len(client.updateInputs) != 3 {
		t.Fatalf("expected three outcome updates, got %d", len(client.updateInputs))
	}
	if update := aws.ToString(client.updateInputs[0].UpdateExpression); !strings.Contains(update, "published_at") || !strings.Contains(update, "REMOVE ") || !strings.Contains(update, "dispatch_pk") {
		t.Fatalf("published update leaves event dispatchable: %s", update)
	}
	if update := aws.ToString(client.updateInputs[1].UpdateExpression); !strings.Contains(update, "retry_at") || strings.Contains(update, "dispatch_pk") {
		t.Fatalf("retry update is incorrect: %s", update)
	}
	if update := aws.ToString(client.updateInputs[2].UpdateExpression); !strings.Contains(update, "failed") || !strings.Contains(update, "REMOVE ") || !strings.Contains(update, "dispatch_pk") {
		t.Fatalf("terminal update leaves event dispatchable: %s", update)
	}
}

func TestStorePropagatesUnexpectedLeaseAndOutcomeErrors(t *testing.T) {
	t.Parallel()

	payload, _ := json.Marshal(dynamoAcceptance().Event)
	client := &fakeClient{
		queryItems: []map[string]types.AttributeValue{{
			"pk": &types.AttributeValueMemberS{Value: "EVENT#evt-1"}, "event_payload": &types.AttributeValueMemberB{Value: payload}, "attempts": &types.AttributeValueMemberN{Value: "0"},
		}},
		updateErrors: []error{errors.New("unavailable")},
	}
	if _, err := NewStore(client, "transactions").ClaimPending(context.Background(), time.Now(), time.Second, 1); err == nil {
		t.Fatal("expected unexpected lease failure")
	}
}
