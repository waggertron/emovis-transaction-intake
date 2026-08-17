package bootstrap

import (
	"net/http"
	"testing"
	"time"
)

func TestLoadConfigRequiresExternalizedAPIKey(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing API key configuration to fail")
	}
}

func TestLoadConfigUsesSafeDefaultsAndExplicitIdentity(t *testing.T) {
	t.Parallel()

	values := map[string]string{"API_KEY": "external-secret", "PARTNER_ID": "partner-west"}
	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.Address != ":8080" || config.APIKey != "external-secret" || config.PartnerID != "partner-west" {
		t.Fatalf("unexpected config: %#v", config)
	}
	if len(config.KafkaBrokers) != 1 || config.KafkaBrokers[0] != "localhost:9092" || config.KafkaTopic != "transaction.review-candidates.v1" || config.KafkaTLS {
		t.Fatalf("unexpected Kafka defaults: %#v", config)
	}
	if config.StoreDriver != "memory" || config.StorePath != ".local/data/transactions.ndjson" {
		t.Fatalf("unexpected storage defaults: %#v", config)
	}
}

func TestLoadConfigRejectsUnknownStoreDriver(t *testing.T) {
	t.Parallel()

	values := map[string]string{"API_KEY": "key", "PARTNER_ID": "partner", "STORE_DRIVER": "mystery"}
	if _, err := LoadConfig(func(name string) string { return values[name] }); err == nil {
		t.Fatal("expected unknown store driver to fail")
	}
}

func TestLoadConfigRequiresAndProtectsPostgresConfiguration(t *testing.T) {
	t.Parallel()
	base := map[string]string{"API_KEY": "key", "PARTNER_ID": "partner", "STORE_DRIVER": "postgres"}
	if _, err := LoadConfig(func(name string) string { return base[name] }); err == nil {
		t.Fatal("expected missing POSTGRES_URL to fail")
	}
	base["POSTGRES_URL"] = "postgres://user:super-secret@database/transactions"
	config, err := LoadConfig(func(name string) string { return base[name] })
	if err != nil {
		t.Fatalf("load PostgreSQL config: %v", err)
	}
	if config.PostgresURL != base["POSTGRES_URL"] || contains(config.String(), "super-secret") || contains(config.String(), "postgres://") {
		t.Fatalf("PostgreSQL configuration leaked: %s", config.String())
	}
}

func TestLoadConfigUsesDynamoDefaultsAndExplicitLocalEndpoint(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"API_KEY": "key", "PARTNER_ID": "partner", "STORE_DRIVER": "dynamodb",
		"DYNAMODB_ENDPOINT": "http://dynamodb-local:8000", "AWS_REGION": "us-east-2", "DYNAMODB_TABLE": "toll-transactions",
	}
	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("load DynamoDB config: %v", err)
	}
	if config.DynamoEndpoint != values["DYNAMODB_ENDPOINT"] || config.DynamoRegion != "us-east-2" || config.DynamoTable != "toll-transactions" {
		t.Fatalf("unexpected DynamoDB config: %#v", config)
	}
	delete(values, "DYNAMODB_TABLE")
	delete(values, "AWS_REGION")
	config, err = LoadConfig(func(name string) string { return values[name] })
	if err != nil || config.DynamoTable != "transaction-intake" || config.DynamoRegion != "us-west-2" {
		t.Fatalf("unexpected DynamoDB defaults: %#v, %v", config, err)
	}
}

func TestLoadConfigParsesKafkaSecurityWithoutLoggingSecrets(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"API_KEY": "external-secret", "PARTNER_ID": "partner-west",
		"KAFKA_BROKERS": "broker-1:9094, broker-2:9094", "KAFKA_TOPIC": "review.v1",
		"KAFKA_TLS": "true", "KAFKA_CA_FILE": "/run/secrets/kafka-ca.pem", "KAFKA_SASL_USERNAME": "user", "KAFKA_SASL_PASSWORD": "password",
	}
	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(config.KafkaBrokers) != 2 || !config.KafkaTLS || config.KafkaCAFile != "/run/secrets/kafka-ca.pem" || config.KafkaSASLUsername != "user" || config.KafkaSASLPassword != "password" {
		t.Fatalf("unexpected Kafka config")
	}
	if config.String() == "" || contains(config.String(), "external-secret") || contains(config.String(), "password") {
		t.Fatalf("config string leaked a secret: %s", config.String())
	}
}

func TestLoadConfigRejectsInvalidBoolean(t *testing.T) {
	t.Parallel()

	values := map[string]string{"API_KEY": "key", "PARTNER_ID": "partner", "KAFKA_TLS": "sometimes"}
	if _, err := LoadConfig(func(name string) string { return values[name] }); err == nil {
		t.Fatal("expected invalid KAFKA_TLS to fail")
	}
}

func contains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}

func TestParseModeAcceptsOnlyDocumentedModes(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"api", "worker", "local"} {
		if mode, err := ParseMode(value); err != nil || string(mode) != value {
			t.Fatalf("parse %q: %q, %v", value, mode, err)
		}
	}
	if _, err := ParseMode("unknown"); err == nil {
		t.Fatal("expected undocumented mode to fail")
	}
}

func TestNewHTTPServerHasExplicitResourceLimits(t *testing.T) {
	t.Parallel()

	server := NewHTTPServer(":0", http.NewServeMux())
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 10*time.Second || server.WriteTimeout != 15*time.Second || server.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected timeouts: %#v", server)
	}
	if server.MaxHeaderBytes != 16*1024 {
		t.Fatalf("unexpected max header bytes: %d", server.MaxHeaderBytes)
	}
}

func TestNewUUIDProducesRFC4122VariantVersionFour(t *testing.T) {
	t.Parallel()

	first, err := NewUUID()
	if err != nil {
		t.Fatalf("new UUID: %v", err)
	}
	second, err := NewUUID()
	if err != nil {
		t.Fatalf("new UUID: %v", err)
	}
	if first == second || len(first) != 36 || first[14] != '4' || (first[19] != '8' && first[19] != '9' && first[19] != 'a' && first[19] != 'b') {
		t.Fatalf("unexpected UUIDs %q and %q", first, second)
	}
}
