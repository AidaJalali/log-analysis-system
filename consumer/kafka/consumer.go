package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log-analysis-system/consumer/config"
	"log-analysis-system/consumer/database"
)

type KafkaMessage struct {
	ProjectID string      `json:"project_id"`
	Payload   IngestedLog `json:"payload"`
}

type IngestedLog struct {
	EventName string                 `json:"event_name"`
	Payload   map[string]interface{} `json:"payload"`
}

type Consumer struct {
	reader          *kafka.Reader
	clickhouseClient *database.ClickhouseClient
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
		clickhouseClient: ch,
		cassandraClient: cass,
	}
}

func (c *Consumer) Start() {
	defer c.reader.Close()

	for {
		msg, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("ERROR: could not read message from kafka: %v", err)
			continue
		}

		var kafkaMsg KafkaMessage
		if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
			log.Printf("ERROR: could not unmarshal kafka message: %v", err)
			continue
		}

		projectID := kafkaMsg.ProjectID
		ingestedLog := kafkaMsg.Payload

		logID := uuid.NewString()
		timestamp := time.Now().Unix()

		go c.writeToCassandra(projectID, ingestedLog, logID, timestamp)
		go c.writeToClickHouse(projectID, ingestedLog, logID, timestamp)
	}
}

func (c *Consumer) writeToCassandra(projectID string, logData IngestedLog, logID string, ts int64) {
	payload := database.LogPayload{
		ProjectID: projectID,
		LogID:     logID,
		EventName: logData.EventName,
		Timestamp: ts,
		Payload:   convertPayload(logData.Payload),
	}
	if err := c.cassandraClient.WriteLog(payload); err != nil {
		log.Printf("ERROR: could not write to Cassandra: %v", err)
	}
}

func (c *Consumer) writeToClickHouse(projectID string, logData IngestedLog, logID string, ts int64) {
	var searchableKey string
	if key, ok := logData.Payload["searchable_key_1"].(string); ok {
		searchableKey = key
	}

	index := database.LogIndex{
		ProjectID:    projectID,
		LogID:        logID,
		EventName:    logData.EventName,
		Timestamp:    ts,
		SearchableKey: searchableKey,
	}
	if err := c.clickhouseClient.WriteLog(index); err != nil {
		log.Printf("ERROR: could not write to ClickHouse: %v", err)
	}
}

func convertPayload(payload map[string]interface{}) map[string]string {
	stringPayload := make(map[string]string)
	for key, value := range payload {
		stringPayload[key] = fmt.Sprintf("%v", value)
	}
	return stringPayload
}