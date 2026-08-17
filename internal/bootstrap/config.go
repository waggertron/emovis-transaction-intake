package bootstrap

import (
	"fmt"
	"strconv"
	"strings"
)

type Mode string

const (
	ModeAPI    Mode = "api"
	ModeWorker Mode = "worker"
	ModeLocal  Mode = "local"
)

type Config struct {
	Address           string
	PartnerID         string
	APIKey            string
	KafkaBrokers      []string
	KafkaTopic        string
	KafkaTLS          bool
	KafkaSASLUsername string
	KafkaSASLPassword string
	StoreDriver       string
	StorePath         string
}

func LoadConfig(lookup func(string) string) (Config, error) {
	config := Config{
		Address:           lookup("HTTP_ADDRESS"),
		PartnerID:         lookup("PARTNER_ID"),
		APIKey:            lookup("API_KEY"),
		KafkaTopic:        lookup("KAFKA_TOPIC"),
		KafkaSASLUsername: lookup("KAFKA_SASL_USERNAME"),
		KafkaSASLPassword: lookup("KAFKA_SASL_PASSWORD"),
		StoreDriver:       lookup("STORE_DRIVER"),
		StorePath:         lookup("STORE_PATH"),
	}
	if config.Address == "" {
		config.Address = ":8080"
	}
	if config.PartnerID == "" {
		return Config{}, fmt.Errorf("PARTNER_ID is required")
	}
	if config.APIKey == "" {
		return Config{}, fmt.Errorf("API_KEY is required")
	}
	brokers := lookup("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	for _, broker := range strings.Split(brokers, ",") {
		if broker = strings.TrimSpace(broker); broker != "" {
			config.KafkaBrokers = append(config.KafkaBrokers, broker)
		}
	}
	if len(config.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}
	if config.KafkaTopic == "" {
		config.KafkaTopic = "transaction.review-candidates.v1"
	}
	if config.StoreDriver == "" {
		config.StoreDriver = "memory"
	}
	switch config.StoreDriver {
	case "memory", "ndjson", "dynamodb", "postgres":
	default:
		return Config{}, fmt.Errorf("STORE_DRIVER must be memory, ndjson, dynamodb, or postgres")
	}
	if config.StorePath == "" {
		config.StorePath = ".local/data/transactions.ndjson"
	}
	if rawTLS := lookup("KAFKA_TLS"); rawTLS != "" {
		value, err := strconv.ParseBool(rawTLS)
		if err != nil {
			return Config{}, fmt.Errorf("KAFKA_TLS must be a boolean: %w", err)
		}
		config.KafkaTLS = value
	}
	return config, nil
}

func (config Config) String() string {
	return fmt.Sprintf("address=%s partner=%s store_driver=%s kafka_brokers=%s kafka_topic=%s kafka_tls=%t kafka_sasl=%t",
		config.Address, config.PartnerID, config.StoreDriver, strings.Join(config.KafkaBrokers, ","), config.KafkaTopic,
		config.KafkaTLS, config.KafkaSASLUsername != "")
}

func ParseMode(value string) (Mode, error) {
	mode := Mode(value)
	switch mode {
	case ModeAPI, ModeWorker, ModeLocal:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", value)
	}
}
