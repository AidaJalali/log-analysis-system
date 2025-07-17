package kafka

import (
	"log-analysis-system/consumer/database"
	"log-analysis-system/consumer/kafka"
)

// IngestedLog is the structure of the log message as it comes from Kafka
type IngestedLog struct {
	ProjectID string            `json:"project_id"`
	EventName string            `json:"event_name"`
	Payload   map[string]string `json:"payload"`
}

type Consumer struct {
	reader	*kafka.reader
	chClinet *database.ClickhouseClient
	cassandraClient *database.CassandraClient
}
