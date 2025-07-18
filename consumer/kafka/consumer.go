package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log-analysis-system/consumer/config"
	"log-analysis-system/consumer/database"
)

// IngestedLog is the structure of the log message as it comes from Kafka.
type IngestedLog struct {
	ProjectID string            `json:"project_id"`
	EventName string            `json:"event_name"`
	Payload   map[string]string `json:"payload"`
}

type Consumer struct {
	reader          *kafka.Reader
	chClient        *database.ClickhouseClient
	cassandraClient *database.CassandraClient
}

func NewConsumer(cfg config.KafkaConfig, ch *database.ClickhouseClient, cass *database.CassandraClient) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
		GroupID: cfg.GroupID,
	})
	return &Consumer{
		reader:          reader,
		chClient:        ch,
		cassandraClient: cass,
	}
}

func (c *Consumer) Start() {
	defer c.reader.Close()

	for {
		msg, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("ERROR: could not read message: %v", err)
			continue
		}

		var ingestedLog IngestedLog
		if err := json.Unmarshal(msg.Value, &ingestedLog); err != nil {
			log.Printf("ERROR: could not unmarshal message: %v", err)
			continue
		}

		// Generate a unique ID and timestamp for this log event
		logID := uuid.NewString()
		timestamp := time.Now().Unix()

		// Fan out to databases
		go c.writeToCassandra(ingestedLog, logID, timestamp)
		go c.writeToClickHouse(ingestedLog, logID, timestamp)
	}
}

func (c *Consumer) writeToCassandra(logData IngestedLog, logID string, ts int64) {
	payload := database.LogPayload{
		ProjectID: logData.ProjectID,
		LogID:     logID,
		EventName: logData.EventName,
		Timestamp: ts,
		Payload:   logData.Payload,
	}
	if err := c.cassandraClient.WriteLog(payload); err != nil {
		log.Printf("ERROR: could not write to Cassandra: %v", err)
	}
}

func (c *Consumer) writeToClickHouse(logData IngestedLog, logID string, ts int64) {
	index := database.LogIndex{
		ProjectID: logData.ProjectID,
		LogID:     logID,
		EventName: logData.EventName,
		Timestamp: ts,
	}
	if err := c.chClient.WriteLog(index); err != nil {
		log.Printf("ERROR: could not write to ClickHouse: %v", err)
	}
}