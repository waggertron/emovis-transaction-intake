package kafkaadapter

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type SecurityConfig struct {
	TLS          bool
	CAFile       string
	SASLUsername string
	SASLPassword string
}

type WriterConfig struct {
	Brokers      []string
	Topic        string
	TLS          bool
	CAFile       string
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
	tlsConfig, mechanism, err := kafkaSecurity(SecurityConfig{
		TLS: config.TLS, CAFile: config.CAFile, SASLUsername: config.SASLUsername, SASLPassword: config.SASLPassword,
	})
	if err != nil {
		return nil, err
	}

	transport := &kafkago.Transport{TLS: tlsConfig, SASL: mechanism}

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

func NewDialer(config SecurityConfig) (*kafkago.Dialer, error) {
	tlsConfig, mechanism, err := kafkaSecurity(config)
	if err != nil {
		return nil, err
	}
	return &kafkago.Dialer{Timeout: 10 * time.Second, TLS: tlsConfig, SASLMechanism: mechanism}, nil
}

func kafkaSecurity(config SecurityConfig) (*tls.Config, sasl.Mechanism, error) {
	hasUsername := config.SASLUsername != ""
	hasPassword := config.SASLPassword != ""
	if hasUsername != hasPassword {
		return nil, nil, fmt.Errorf("Kafka SASL username and password must be provided together")
	}
	if hasUsername && !config.TLS {
		return nil, nil, fmt.Errorf("Kafka SASL/SCRAM requires TLS")
	}
	if config.CAFile != "" && !config.TLS {
		return nil, nil, fmt.Errorf("Kafka CA file requires TLS")
	}

	var tlsConfig *tls.Config
	if config.TLS {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		if config.CAFile != "" {
			certificate, err := os.ReadFile(config.CAFile)
			if err != nil {
				return nil, nil, fmt.Errorf("read Kafka CA file: %w", err)
			}
			roots, err := x509.SystemCertPool()
			if err != nil || roots == nil {
				roots = x509.NewCertPool()
			}
			if !roots.AppendCertsFromPEM(certificate) {
				return nil, nil, fmt.Errorf("Kafka CA file contains no certificates")
			}
			tlsConfig.RootCAs = roots
		}
	}
	var mechanism sasl.Mechanism
	if hasUsername {
		configured, err := scram.Mechanism(scram.SHA512, config.SASLUsername, config.SASLPassword)
		if err != nil {
			return nil, nil, fmt.Errorf("configure Kafka SCRAM: %w", err)
		}
		mechanism = configured
	}
	return tlsConfig, mechanism, nil
}
