package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

type Client interface {
	GetItem(context.Context, *awssdk.GetItemInput, ...func(*awssdk.Options)) (*awssdk.GetItemOutput, error)
	TransactWriteItems(context.Context, *awssdk.TransactWriteItemsInput, ...func(*awssdk.Options)) (*awssdk.TransactWriteItemsOutput, error)
}

type Store struct {
	client Client
	table  string
}

func NewStore(client Client, table string) *Store {
	return &Store{client: client, table: table}
}

func (store *Store) Accept(ctx context.Context, acceptance app.Acceptance) (app.StoreOutcome, error) {
	identityKey := "TX#" + acceptance.Transaction.PartnerID + "#" + acceptance.Transaction.ID
	existing, err := store.client.GetItem(ctx, &awssdk.GetItemInput{
		TableName: aws.String(store.table), ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: identityKey},
			"sk": &types.AttributeValueMemberS{Value: "TRANSACTION"},
		},
	})
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("read transaction identity: %w", err)
	}
	if len(existing.Item) > 0 {
		fingerprint, fingerprintOK := existing.Item["fingerprint"].(*types.AttributeValueMemberS)
		eventID, eventOK := existing.Item["event_id"].(*types.AttributeValueMemberS)
		if !fingerprintOK || !eventOK {
			return app.StoreOutcome{}, fmt.Errorf("stored transaction identity is malformed")
		}
		if fingerprint.Value == acceptance.Fingerprint {
			return app.StoreOutcome{Kind: app.StoreReplay, EventID: eventID.Value}, nil
		}
		return app.StoreOutcome{Kind: app.StoreConflict}, nil
	}

	transactionPayload, err := json.Marshal(acceptance.Transaction)
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("encode transaction: %w", err)
	}
	eventPayload, err := json.Marshal(acceptance.Event)
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("encode outbox event: %w", err)
	}
	condition := aws.String("attribute_not_exists(pk)")
	_, err = store.client.TransactWriteItems(ctx, &awssdk.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Put: &types.Put{TableName: aws.String(store.table), ConditionExpression: condition, Item: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: identityKey}, "sk": &types.AttributeValueMemberS{Value: "TRANSACTION"},
			"fingerprint": &types.AttributeValueMemberS{Value: acceptance.Fingerprint}, "event_id": &types.AttributeValueMemberS{Value: acceptance.Event.ID},
			"payload": &types.AttributeValueMemberB{Value: transactionPayload},
		}}},
		{Put: &types.Put{TableName: aws.String(store.table), ConditionExpression: condition, Item: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: "EVENT#" + acceptance.Event.ID}, "sk": &types.AttributeValueMemberS{Value: "OUTBOX"},
			"status": &types.AttributeValueMemberS{Value: "pending"}, "attempts": &types.AttributeValueMemberN{Value: "0"},
			"occurred_at":   &types.AttributeValueMemberS{Value: acceptance.Event.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")},
			"event_payload": &types.AttributeValueMemberB{Value: eventPayload},
		}}},
	}})
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("transact transaction and outbox: %w", err)
	}
	return app.StoreOutcome{Kind: app.StoreAccepted, EventID: acceptance.Event.ID}, nil
}
