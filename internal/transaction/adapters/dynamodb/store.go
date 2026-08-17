package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

type Client interface {
	GetItem(context.Context, *awssdk.GetItemInput, ...func(*awssdk.Options)) (*awssdk.GetItemOutput, error)
	TransactWriteItems(context.Context, *awssdk.TransactWriteItemsInput, ...func(*awssdk.Options)) (*awssdk.TransactWriteItemsOutput, error)
	Query(context.Context, *awssdk.QueryInput, ...func(*awssdk.Options)) (*awssdk.QueryOutput, error)
	UpdateItem(context.Context, *awssdk.UpdateItemInput, ...func(*awssdk.Options)) (*awssdk.UpdateItemOutput, error)
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
			"dispatch_pk":   &types.AttributeValueMemberS{Value: "OUTBOX#PENDING"},
			"dispatch_sk":   &types.AttributeValueMemberS{Value: timestamp(acceptance.Event.OccurredAt) + "#" + acceptance.Event.ID},
			"occurred_at":   &types.AttributeValueMemberS{Value: acceptance.Event.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")},
			"event_payload": &types.AttributeValueMemberB{Value: eventPayload},
		}}},
	}})
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("transact transaction and outbox: %w", err)
	}
	return app.StoreOutcome{Kind: app.StoreAccepted, EventID: acceptance.Event.ID}, nil
}

func (store *Store) ClaimPending(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	result, err := store.client.Query(ctx, &awssdk.QueryInput{
		TableName:              aws.String(store.table),
		IndexName:              aws.String("outbox-dispatch"),
		KeyConditionExpression: aws.String("dispatch_pk = :pending"),
		FilterExpression:       aws.String("#status = :status AND (attribute_not_exists(retry_at) OR retry_at <= :now) AND (attribute_not_exists(lease_until) OR lease_until <= :now)"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pending": &types.AttributeValueMemberS{Value: "OUTBOX#PENDING"},
			":status":  &types.AttributeValueMemberS{Value: "pending"},
			":now":     &types.AttributeValueMemberS{Value: timestamp(now)},
		},
		Limit: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("query pending outbox events: %w", err)
	}

	claimed := make([]app.PendingEvent, 0, len(result.Items))
	for _, item := range result.Items {
		pk, ok := item["pk"].(*types.AttributeValueMemberS)
		if !ok {
			return nil, fmt.Errorf("pending outbox event has malformed key")
		}
		_, err := store.client.UpdateItem(ctx, &awssdk.UpdateItemInput{
			TableName: aws.String(store.table),
			Key: map[string]types.AttributeValue{
				"pk": &types.AttributeValueMemberS{Value: pk.Value},
				"sk": &types.AttributeValueMemberS{Value: "OUTBOX"},
			},
			ConditionExpression: aws.String("#status = :pending AND (attribute_not_exists(lease_until) OR lease_until <= :now)"),
			UpdateExpression:    aws.String("SET lease_until = :lease_until"),
			ExpressionAttributeNames: map[string]string{
				"#status": "status",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pending":     &types.AttributeValueMemberS{Value: "pending"},
				":now":         &types.AttributeValueMemberS{Value: timestamp(now)},
				":lease_until": &types.AttributeValueMemberS{Value: timestamp(now.Add(lease))},
			},
		})
		if err != nil {
			var raced *types.ConditionalCheckFailedException
			if errors.As(err, &raced) {
				continue
			}
			return nil, fmt.Errorf("lease outbox event %q: %w", pk.Value, err)
		}
		pending, err := decodePending(item)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, pending)
	}
	return claimed, nil
}

func (store *Store) MarkPublished(ctx context.Context, eventID string, at time.Time) error {
	return store.updateOutcome(ctx, eventID,
		"SET #status = :published, published_at = :published_at REMOVE lease_until, retry_at, dispatch_pk, dispatch_sk",
		map[string]types.AttributeValue{
			":published":    &types.AttributeValueMemberS{Value: "published"},
			":published_at": &types.AttributeValueMemberS{Value: timestamp(at)},
		})
}

func (store *Store) RecordFailure(ctx context.Context, failure app.PublishFailure) error {
	values := map[string]types.AttributeValue{
		":attempts": &types.AttributeValueMemberN{Value: strconv.Itoa(failure.Attempts)},
		":reason":   &types.AttributeValueMemberS{Value: failure.Reason},
	}
	if failure.Terminal {
		values[":failed"] = &types.AttributeValueMemberS{Value: "failed"}
		return store.updateOutcome(ctx, failure.EventID,
			"SET #status = :failed, attempts = :attempts, last_error = :reason REMOVE lease_until, retry_at, dispatch_pk, dispatch_sk", values)
	}
	values[":pending"] = &types.AttributeValueMemberS{Value: "pending"}
	values[":retry_at"] = &types.AttributeValueMemberS{Value: timestamp(failure.RetryAt)}
	return store.updateOutcome(ctx, failure.EventID,
		"SET #status = :pending, attempts = :attempts, retry_at = :retry_at, last_error = :reason REMOVE lease_until", values)
}

func (store *Store) updateOutcome(ctx context.Context, eventID, update string, values map[string]types.AttributeValue) error {
	_, err := store.client.UpdateItem(ctx, &awssdk.UpdateItemInput{
		TableName: aws.String(store.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: "EVENT#" + eventID},
			"sk": &types.AttributeValueMemberS{Value: "OUTBOX"},
		},
		ConditionExpression:       aws.String("attribute_exists(pk)"),
		UpdateExpression:          aws.String(update),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: values,
	})
	if err != nil {
		return fmt.Errorf("update outbox event %q: %w", eventID, err)
	}
	return nil
}

func decodePending(item map[string]types.AttributeValue) (app.PendingEvent, error) {
	payload, payloadOK := item["event_payload"].(*types.AttributeValueMemberB)
	attemptsValue, attemptsOK := item["attempts"].(*types.AttributeValueMemberN)
	if !payloadOK || !attemptsOK {
		return app.PendingEvent{}, fmt.Errorf("pending outbox event is malformed")
	}
	var pending app.PendingEvent
	if err := json.Unmarshal(payload.Value, &pending.Event); err != nil {
		return app.PendingEvent{}, fmt.Errorf("decode pending outbox event: %w", err)
	}
	attempts, err := strconv.Atoi(attemptsValue.Value)
	if err != nil {
		return app.PendingEvent{}, fmt.Errorf("decode pending outbox attempts: %w", err)
	}
	pending.Attempts = attempts
	return pending, nil
}

func timestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
