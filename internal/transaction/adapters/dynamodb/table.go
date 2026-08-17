package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type AdminClient interface {
	DescribeTable(context.Context, *awssdk.DescribeTableInput, ...func(*awssdk.Options)) (*awssdk.DescribeTableOutput, error)
	CreateTable(context.Context, *awssdk.CreateTableInput, ...func(*awssdk.Options)) (*awssdk.CreateTableOutput, error)
}

func EnsureTable(ctx context.Context, client AdminClient, table string) error {
	return ensureTable(ctx, client, table, func(ctx context.Context) error {
		timer := time.NewTimer(250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	})
}

func ensureTable(ctx context.Context, client AdminClient, table string, wait func(context.Context) error) error {
	if table == "" {
		return fmt.Errorf("DynamoDB table name is required")
	}
	output, err := client.DescribeTable(ctx, &awssdk.DescribeTableInput{TableName: aws.String(table)})
	if err == nil && output != nil && output.Table != nil && output.Table.TableStatus == types.TableStatusActive {
		return nil
	}
	var missing *types.ResourceNotFoundException
	if err != nil && !errors.As(err, &missing) {
		return fmt.Errorf("describe DynamoDB table: %w", err)
	}
	if errors.As(err, &missing) {
		_, err = client.CreateTable(ctx, tableInput(table))
		var alreadyCreating *types.ResourceInUseException
		if err != nil && !errors.As(err, &alreadyCreating) {
			return fmt.Errorf("create DynamoDB table: %w", err)
		}
	}
	for {
		output, err = client.DescribeTable(ctx, &awssdk.DescribeTableInput{TableName: aws.String(table)})
		if err != nil {
			return fmt.Errorf("wait for DynamoDB table: %w", err)
		}
		if output != nil && output.Table != nil && output.Table.TableStatus == types.TableStatusActive {
			return nil
		}
		if err := wait(ctx); err != nil {
			return fmt.Errorf("wait for DynamoDB table: %w", err)
		}
	}
}

func tableInput(table string) *awssdk.CreateTableInput {
	attributes := []types.AttributeDefinition{
		{AttributeName: aws.String("pk"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("sk"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("dispatch_pk"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("dispatch_sk"), AttributeType: types.ScalarAttributeTypeS},
	}
	return &awssdk.CreateTableInput{
		TableName: aws.String(table), BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: attributes,
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{
			IndexName: aws.String("outbox-dispatch"),
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String("dispatch_pk"), KeyType: types.KeyTypeHash},
				{AttributeName: aws.String("dispatch_sk"), KeyType: types.KeyTypeRange},
			},
			Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
		}},
	}
}
