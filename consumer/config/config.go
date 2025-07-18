package config

//TODO: make environment variable file .env for your variables 

import(
	"fmt"
	"os"
	"strings"
)

type ClickhouseConfig struct{
	Host string
	Port string
	Database string
	Username string
	Password string
}

type CassandraConfig struct{
	Hosts []string
	Keyspace string
	Username string
	Password string
}

type KafkaConfig struct{
	Brokers []string
	Topic string
	GroupID string
}

type Config struct{
	Clickhouse ClickhouseConfig
	Cassandra CassandraConfig
	Kafka KafkaConfig
}

func Load()(*Config,error){
	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")

	if len(kafkaBrokers) == 0 || kafkaBrokers[0] == "" {
		return nil, fmt.Errorf("required environment variable KAFKA_BROKERS is not set")
	}

	cassandraHosts := strings.Split(os.Getenv("CASSANDRA_HOSTS"), ",")
	if len(cassandraHosts) == 0 || cassandraHosts[0] == "" {
		return nil, fmt.Errorf("required environment variable CASSANDRA_HOSTS is not set")
	}

		cfg := &Config{
		Clickhouse: ClickhouseConfig{
			Host:     os.Getenv("CLICKHOUSE_HOST"),
			Port:     os.Getenv("CLICKHOUSE_PORT"),
			Database: os.Getenv("CLICKHOUSE_DATABASE"),
			Username: os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},
		Cassandra: CassandraConfig{
			Hosts:    cassandraHosts,
			Keyspace: os.Getenv("CASSANDRA_KEYSPACE"),
			Username: os.Getenv("CASSANDRA_USER"),
			Password: os.Getenv("CASSANDRA_PASSWORD"),
		},
		Kafka: KafkaConfig{
			Brokers: kafkaBrokers,
			Topic:   "logs",
			GroupID: "log-processors",
		},
	}

	if cfg.Clickhouse.Host == "" {
		return nil, fmt.Errorf("required environment variable CLICKHOUSE_HOST is not set")
	}

	return cfg,nil

}