package dynamodb

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

type fakeClient struct {
	item       map[string]types.AttributeValue
	getInput   *awssdk.GetItemInput
	writeInput *awssdk.TransactWriteItemsInput
	getErr     error
	writeErr   error
}

func (client *fakeClient) GetItem(_ context.Context, input *awssdk.GetItemInput, _ ...func(*awssdk.Options)) (*awssdk.GetItemOutput, error) {
	client.getInput = input
	return &awssdk.GetItemOutput{Item: client.item}, client.getErr
}

func (client *fakeClient) TransactWriteItems(_ context.Context, input *awssdk.TransactWriteItemsInput, _ ...func(*awssdk.Options)) (*awssdk.TransactWriteItemsOutput, error) {
	client.writeInput = input
	return &awssdk.TransactWriteItemsOutput{}, client.writeErr
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
