package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type fakeSecretsManager struct {
	output *secretsmanager.GetSecretValueOutput
	err    error
	input  *secretsmanager.GetSecretValueInput
}

func (fake *fakeSecretsManager) GetSecretValue(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	fake.input = input
	return fake.output, fake.err
}

func TestAWSProviderLoadsJSONSecret(t *testing.T) {
	value := `{"API_KEY":"external-value","PARTNER_ID":"partner-west"}`
	client := &fakeSecretsManager{output: &secretsmanager.GetSecretValueOutput{SecretString: &value}}
	values, err := NewAWSProvider(client, "arn:private:secret").Load(context.Background())
	if err != nil || values["API_KEY"] != "external-value" || client.input == nil || *client.input.SecretId != "arn:private:secret" {
		t.Fatalf("unexpected load result: values=%v err=%v input=%v", values, err, client.input)
	}
}

func TestAWSProviderLoadsBinaryJSONSecret(t *testing.T) {
	client := &fakeSecretsManager{output: &secretsmanager.GetSecretValueOutput{SecretBinary: []byte(`{"API_KEY":"binary-value"}`)}}
	values, err := NewAWSProvider(client, "secret-id").Load(context.Background())
	if err != nil || values["API_KEY"] != "binary-value" {
		t.Fatalf("unexpected binary load: %v %v", values, err)
	}
}

func TestAWSProviderClassifiesNotFoundWithoutIdentifierLeak(t *testing.T) {
	client := &fakeSecretsManager{err: &types.ResourceNotFoundException{Message: stringPointer("secret arn:private:secret missing")}}
	_, err := NewAWSProvider(client, "arn:private:secret").Load(context.Background())
	if !errors.Is(err, ErrSecretNotFound) || strings.Contains(err.Error(), "arn:private:secret") {
		t.Fatalf("expected redacted classified error, got %v", err)
	}
}

func TestAWSProviderRejectsInvalidResponsesAndCancellation(t *testing.T) {
	tests := []struct {
		name   string
		client *fakeSecretsManager
	}{
		{name: "empty", client: &fakeSecretsManager{output: &secretsmanager.GetSecretValueOutput{}}},
		{name: "malformed", client: &fakeSecretsManager{output: &secretsmanager.GetSecretValueOutput{SecretString: stringPointer(`{"API_KEY":`)}}},
		{name: "provider error", client: &fakeSecretsManager{err: errors.New("secret-value-must-not-leak")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewAWSProvider(test.client, "secret-id").Load(context.Background())
			if err == nil || strings.Contains(err.Error(), "secret-value-must-not-leak") {
				t.Fatalf("expected redacted failure, got %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewAWSProvider(&fakeSecretsManager{}, "secret-id").Load(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func stringPointer(value string) *string { return &value }
