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
	getItems     []map[string]types.AttributeValue
	getInput     *awssdk.GetItemInput
	writeInput   *awssdk.TransactWriteItemsInput
	queryInput   *awssdk.QueryInput
	queryItems   []map[string]types.AttributeValue
	queryOutputs []*awssdk.QueryOutput
	queryInputs  []*awssdk.QueryInput
	updateInputs []*awssdk.UpdateItemInput
	updateErrors []error
	getErr       error
	writeErr     error
	queryErr     error
}

func (client *fakeClient) GetItem(_ context.Context, input *awssdk.GetItemInput, _ ...func(*awssdk.Options)) (*awssdk.GetItemOutput, error) {
	client.getInput = input
	if len(client.getItems) > 0 {
		item := client.getItems[0]
		client.getItems = client.getItems[1:]
		return &awssdk.GetItemOutput{Item: item}, client.getErr
	}
	return &awssdk.GetItemOutput{Item: client.item}, client.getErr
}

func (client *fakeClient) TransactWriteItems(_ context.Context, input *awssdk.TransactWriteItemsInput, _ ...func(*awssdk.Options)) (*awssdk.TransactWriteItemsOutput, error) {
	client.writeInput = input
	return &awssdk.TransactWriteItemsOutput{}, client.writeErr
}

func (client *fakeClient) Query(_ context.Context, input *awssdk.QueryInput, _ ...func(*awssdk.Options)) (*awssdk.QueryOutput, error) {
	client.queryInput = input
	client.queryInputs = append(client.queryInputs, input)
	if len(client.queryOutputs) > 0 {
		output := client.queryOutputs[0]
		client.queryOutputs = client.queryOutputs[1:]
		return output, client.queryErr
	}
	return &awssdk.QueryOutput{Items: client.queryItems}, client.queryErr
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
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", Source: "partner-west", SourceReference: "source-ref",
		TransactionType: "toll", TransactionTimeUTC: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC),
		BaseAmount: "7.25", Currency: "USD", TransponderNumber: "tag",
		LocationRaw: json.RawMessage(`{ "lane" : 9007199254740993 }`), MetadataRaw: json.RawMessage(`{ "rate" : 12.50 }`),
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1, OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC),
		Source: transaction.Source, SourceReference: transaction.SourceReference, TransactionID: transaction.ID, Key: transaction.Source + ":" + transaction.SourceReference, Payload: transaction,
	}}
}

func TestStoreAcceptsWithConditionalTransactionalWrite(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	result, err := NewStore(client, "transactions").Accept(context.Background(), dynamoAcceptance())
	if err != nil || result.Kind != app.StoreAccepted || result.EventID != "evt-1" || result.TransactionID != dynamoAcceptance().Transaction.ID {
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
	transactionItem := client.writeInput.TransactItems[0].Put.Item
	if string(transactionItem["location_raw"].(*types.AttributeValueMemberB).Value) != string(dynamoAcceptance().Transaction.LocationRaw) || string(transactionItem["metadata_raw"].(*types.AttributeValueMemberB).Value) != string(dynamoAcceptance().Transaction.MetadataRaw) {
		t.Fatalf("transaction raw audit fields missing: %#v", transactionItem)
	}
	if string(outbox["location_raw"].(*types.AttributeValueMemberB).Value) != string(dynamoAcceptance().Event.Payload.LocationRaw) || string(outbox["metadata_raw"].(*types.AttributeValueMemberB).Value) != string(dynamoAcceptance().Event.Payload.MetadataRaw) {
		t.Fatalf("outbox raw audit fields missing: %#v", outbox)
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
			"fingerprint":    &types.AttributeValueMemberS{Value: test.fingerprint},
			"event_id":       &types.AttributeValueMemberS{Value: "evt-original"},
			"transaction_id": &types.AttributeValueMemberS{Value: "transaction-original"},
		}}
		result, err := NewStore(client, "transactions").Accept(context.Background(), acceptance)
		if err != nil || result.Kind != test.want || client.writeInput != nil {
			t.Fatalf("unexpected identity result %#v, %v", result, err)
		}
		if test.want == app.StoreReplay && (result.EventID != "evt-original" || result.TransactionID != "transaction-original") {
			t.Fatalf("expected original identity, got %#v", result)
		}
	}
}

func TestStoreReclassifiesConditionalAcceptanceRace(t *testing.T) {
	t.Parallel()

	acceptance := dynamoAcceptance()
	client := &fakeClient{
		getItems: []map[string]types.AttributeValue{nil, {
			"fingerprint":    &types.AttributeValueMemberS{Value: acceptance.Fingerprint},
			"event_id":       &types.AttributeValueMemberS{Value: "evt-winner"},
			"transaction_id": &types.AttributeValueMemberS{Value: "transaction-winner"},
		}},
		writeErr: &types.TransactionCanceledException{},
	}
	result, err := NewStore(client, "transactions").Accept(context.Background(), acceptance)
	if err != nil || result.Kind != app.StoreReplay || result.EventID != "evt-winner" || result.TransactionID != "transaction-winner" {
		t.Fatalf("conditional race was not reclassified: %#v, %v", result, err)
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
			"location_raw":  &types.AttributeValueMemberB{Value: dynamoAcceptance().Event.Payload.LocationRaw},
			"metadata_raw":  &types.AttributeValueMemberB{Value: dynamoAcceptance().Event.Payload.MetadataRaw},
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
	if string(events[0].Event.Payload.LocationRaw) != string(dynamoAcceptance().Event.Payload.LocationRaw) || string(events[0].Event.Payload.MetadataRaw) != string(dynamoAcceptance().Event.Payload.MetadataRaw) {
		t.Fatalf("claimed event lost raw bytes: %#v", events[0].Event.Payload)
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

func TestStorePaginatesPastFilteredLeadingPage(t *testing.T) {
	t.Parallel()

	payload, _ := json.Marshal(dynamoAcceptance().Event)
	lastKey := map[string]types.AttributeValue{"dispatch_pk": &types.AttributeValueMemberS{Value: "OUTBOX#PENDING"}, "dispatch_sk": &types.AttributeValueMemberS{Value: "cursor"}}
	client := &fakeClient{queryOutputs: []*awssdk.QueryOutput{
		{LastEvaluatedKey: lastKey},
		{Items: []map[string]types.AttributeValue{{
			"pk": &types.AttributeValueMemberS{Value: "EVENT#evt-1"}, "event_payload": &types.AttributeValueMemberB{Value: payload}, "attempts": &types.AttributeValueMemberN{Value: "0"},
		}}},
	}}
	events, err := NewStore(client, "transactions").ClaimPending(context.Background(), time.Now(), time.Minute, 1)
	if err != nil || len(events) != 1 {
		t.Fatalf("claim beyond filtered page: %#v, %v", events, err)
	}
	if len(client.queryInputs) != 2 || len(client.queryInputs[1].ExclusiveStartKey) == 0 {
		t.Fatalf("expected pagination cursor, got %#v", client.queryInputs)
	}
}

func TestStoreRecordsPublishedRetryAndTerminalOutcomes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	client := &fakeClient{}
	store := NewStore(client, "transactions")
	if err := store.MarkPublished(context.Background(), "evt-1", "claim-1", now); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	if err := store.RecordFailure(context.Background(), app.PublishFailure{
		EventID: "evt-2", ClaimToken: "claim-2", Attempts: 3, RetryAt: now.Add(time.Minute), Reason: "publish_failed",
	}); err != nil {
		t.Fatalf("record retry: %v", err)
	}
	if err := store.RecordFailure(context.Background(), app.PublishFailure{
		EventID: "evt-3", ClaimToken: "claim-3", Attempts: 5, Terminal: true, Reason: "publish_failed",
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

func TestStoreRejectsAWSAndMalformedPersistenceResults(t *testing.T) {
	t.Parallel()
	want := errors.New("unavailable")
	acceptance := dynamoAcceptance()
	if _, err := NewStore(&fakeClient{getErr: want}, "transactions").Accept(context.Background(), acceptance); !errors.Is(err, want) {
		t.Fatalf("expected read error, got %v", err)
	}
	if _, err := NewStore(&fakeClient{item: map[string]types.AttributeValue{"fingerprint": &types.AttributeValueMemberS{Value: "x"}}}, "transactions").Accept(context.Background(), acceptance); err == nil {
		t.Fatal("expected malformed identity")
	}
	if _, err := NewStore(&fakeClient{writeErr: want}, "transactions").Accept(context.Background(), acceptance); !errors.Is(err, want) {
		t.Fatalf("expected write error, got %v", err)
	}
	store := NewStore(&fakeClient{queryErr: want}, "transactions")
	if _, err := store.ClaimPending(context.Background(), time.Now(), time.Second, 1); !errors.Is(err, want) {
		t.Fatalf("expected query error, got %v", err)
	}
	if events, err := store.ClaimPending(context.Background(), time.Now(), time.Second, 0); err != nil || len(events) != 0 {
		t.Fatalf("zero limit: %#v %v", events, err)
	}
}

func TestStoreRejectsMalformedPendingItemsAndOutcomeFailures(t *testing.T) {
	t.Parallel()
	for _, item := range []map[string]types.AttributeValue{
		{"event_payload": &types.AttributeValueMemberB{Value: []byte("{}")}, "attempts": &types.AttributeValueMemberN{Value: "0"}},
		{"pk": &types.AttributeValueMemberS{Value: "EVENT#x"}, "attempts": &types.AttributeValueMemberN{Value: "0"}},
		{"pk": &types.AttributeValueMemberS{Value: "EVENT#x"}, "event_payload": &types.AttributeValueMemberB{Value: []byte("not-json")}, "attempts": &types.AttributeValueMemberN{Value: "0"}},
		{"pk": &types.AttributeValueMemberS{Value: "EVENT#x"}, "event_payload": &types.AttributeValueMemberB{Value: []byte("{}")}, "attempts": &types.AttributeValueMemberN{Value: "not-number"}},
	} {
		client := &fakeClient{queryItems: []map[string]types.AttributeValue{item}}
		if _, err := NewStore(client, "transactions").ClaimPending(context.Background(), time.Now(), time.Second, 1); err == nil {
			t.Fatalf("expected malformed item failure: %#v", item)
		}
	}
	want := errors.New("update failed")
	client := &fakeClient{updateErrors: []error{want}}
	if err := NewStore(client, "transactions").MarkPublished(context.Background(), "evt", "claim", time.Now()); !errors.Is(err, want) {
		t.Fatalf("expected outcome error, got %v", err)
	}
}
