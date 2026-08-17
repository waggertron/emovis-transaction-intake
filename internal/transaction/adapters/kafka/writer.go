package kafkaadapter

import (
	"crypto/tls"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type WriterConfig struct {
	Brokers      []string
	Topic        string
	TLS          bool
	SASLUsername string
	SASLPassword string
}

func NewWriter(config WriterConfig) (*kafkago.Writer, error) {
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("at least one Kafka broker is required")
	}
	if config.Topic == "" {
		return nil, fmt.Errorf("Kafka topic is required")
	}
	hasUsername := config.SASLUsername != ""
	hasPassword := config.SASLPassword != ""
	if hasUsername != hasPassword {
		return nil, fmt.Errorf("Kafka SASL username and password must be provided together")
	}
	if hasUsername && !config.TLS {
		return nil, fmt.Errorf("Kafka SASL/SCRAM requires TLS")
	}

	transport := &kafkago.Transport{}
	if config.TLS {
		transport.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if hasUsername {
		mechanism, err := scram.Mechanism(scram.SHA512, config.SASLUsername, config.SASLPassword)
		if err != nil {
			return nil, fmt.Errorf("configure Kafka SCRAM: %w", err)
		}
		transport.SASL = mechanism
	}

	return &kafkago.Writer{
		Addr:                   kafkago.TCP(config.Brokers...),
		Balancer:               &kafkago.Hash{},
		RequiredAcks:           kafkago.RequireAll,
		Async:                  false,
		MaxAttempts:            5,
		ReadTimeout:            10 * time.Second,
		WriteTimeout:           10 * time.Second,
		BatchTimeout:           10 * time.Millisecond,
		AllowAutoTopicCreation: false,
		Transport:              transport,
	}, nil
}
