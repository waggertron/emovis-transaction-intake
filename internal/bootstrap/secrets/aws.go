package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretsManagerClient interface {
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type AWSProvider struct {
	client   SecretsManagerClient
	secretID string
}

func NewAWSProvider(client SecretsManagerClient, secretID string) *AWSProvider {
	return &AWSProvider{client: client, secretID: secretID}
}

func (provider *AWSProvider) Load(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load AWS secret: %w", err)
	}
	output, err := provider.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &provider.secretID})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("load AWS secret: %w", ErrSecretNotFound)
		}
		return nil, fmt.Errorf("load AWS secret: provider unavailable")
	}
	if output == nil {
		return nil, fmt.Errorf("load AWS secret: empty response")
	}
	payload := output.SecretBinary
	if output.SecretString != nil {
		payload = []byte(*output.SecretString)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("load AWS secret: empty response")
	}
	if len(payload) > MaxFileBytes {
		return nil, fmt.Errorf("load AWS secret: payload exceeds %d bytes", MaxFileBytes)
	}
	values := map[string]string{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("load AWS secret: invalid JSON")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("load AWS secret: no values")
	}
	for key, value := range values {
		if strings.TrimSpace(key) == "" || value == "" {
			return nil, fmt.Errorf("load AWS secret: empty key or value")
		}
	}
	return values, nil
}
