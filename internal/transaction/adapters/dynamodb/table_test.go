package dynamodb

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeAdminClient struct {
	describeOutputs []*awssdk.DescribeTableOutput
	describeErrors  []error
	createInput     *awssdk.CreateTableInput
	createErr       error
}

func (client *fakeAdminClient) DescribeTable(_ context.Context, _ *awssdk.DescribeTableInput, _ ...func(*awssdk.Options)) (*awssdk.DescribeTableOutput, error) {
	var output *awssdk.DescribeTableOutput
	if len(client.describeOutputs) > 0 {
		output, client.describeOutputs = client.describeOutputs[0], client.describeOutputs[1:]
	}
	var err error
	if len(client.describeErrors) > 0 {
		err, client.describeErrors = client.describeErrors[0], client.describeErrors[1:]
	}
	return output, err
}

func (client *fakeAdminClient) CreateTable(_ context.Context, input *awssdk.CreateTableInput, _ ...func(*awssdk.Options)) (*awssdk.CreateTableOutput, error) {
	client.createInput = input
	return &awssdk.CreateTableOutput{}, client.createErr
}

func TestEnsureTableCreatesKeysAndDispatchIndexThenWaitsForActive(t *testing.T) {
	t.Parallel()
	missing := &types.ResourceNotFoundException{Message: aws.String("missing")}
	client := &fakeAdminClient{
		describeErrors:  []error{missing, nil, nil},
		describeOutputs: []*awssdk.DescribeTableOutput{nil, {Table: &types.TableDescription{TableStatus: types.TableStatusCreating}}, {Table: &types.TableDescription{TableStatus: types.TableStatusActive}}},
	}
	waits := 0
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { waits++; return nil }); err != nil {
		t.Fatalf("ensure table: %v", err)
	}
	if client.createInput == nil || len(client.createInput.KeySchema) != 2 || len(client.createInput.GlobalSecondaryIndexes) != 1 || waits != 1 {
		t.Fatalf("unexpected table bootstrap: %#v waits=%d", client.createInput, waits)
	}
	index := client.createInput.GlobalSecondaryIndexes[0]
	if aws.ToString(index.IndexName) != "outbox-dispatch" || index.Projection == nil || index.Projection.ProjectionType != types.ProjectionTypeAll {
		t.Fatalf("unexpected dispatch index: %#v", index)
	}
}

func TestEnsureTableIsIdempotentAndPropagatesFailures(t *testing.T) {
	t.Parallel()
	active := &awssdk.DescribeTableOutput{Table: &types.TableDescription{TableStatus: types.TableStatusActive}}
	client := &fakeAdminClient{describeOutputs: []*awssdk.DescribeTableOutput{active}}
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { return nil }); err != nil || client.createInput != nil {
		t.Fatalf("idempotent ensure: %v %#v", err, client.createInput)
	}
	if err := ensureTable(context.Background(), client, "", func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected empty table failure")
	}
	want := errors.New("unavailable")
	client = &fakeAdminClient{describeErrors: []error{want}}
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { return nil }); !errors.Is(err, want) {
		t.Fatalf("describe error: %v", err)
	}
	client = &fakeAdminClient{describeErrors: []error{&types.ResourceNotFoundException{}}, createErr: want}
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { return nil }); !errors.Is(err, want) {
		t.Fatalf("create error: %v", err)
	}
	client = &fakeAdminClient{describeErrors: []error{&types.ResourceNotFoundException{}, nil}, describeOutputs: []*awssdk.DescribeTableOutput{nil, {Table: &types.TableDescription{TableStatus: types.TableStatusCreating}}}}
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { return want }); !errors.Is(err, want) {
		t.Fatalf("wait error: %v", err)
	}
}

func TestEnsureTableWaitsWhenAnotherProcessCreatesTheTable(t *testing.T) {
	t.Parallel()
	client := &fakeAdminClient{
		describeErrors:  []error{&types.ResourceNotFoundException{}, nil},
		describeOutputs: []*awssdk.DescribeTableOutput{nil, {Table: &types.TableDescription{TableStatus: types.TableStatusActive}}},
		createErr:       &types.ResourceInUseException{Message: aws.String("already creating")},
	}
	if err := ensureTable(context.Background(), client, "transactions", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("concurrent bootstrap: %v", err)
	}
}
