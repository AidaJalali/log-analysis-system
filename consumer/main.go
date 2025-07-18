package main

import (
	"encoding/json"
	"log"
	"log-analysis-system/consumer/config"
	"log-analysis-system/consumer/database"
	"log-analysis-system/consumer/kafka"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// HTTP handler for receiving logs directly from API service
type LogHandler struct {
	chClient        *database.ClickhouseClient
	cassandraClient *database.CassandraClient
}

type LogRequest struct {
	ProjectID string            `json:"project_id"`
	EventName string            `json:"event_name"`
	Payload   map[string]string `json:"payload"`
}

func (h *LogHandler) handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var logReq LogRequest
	if err := json.NewDecoder(r.Body).Decode(&logReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process the log (same logic as Kafka consumer)
	logID := uuid.NewString()
	timestamp := time.Now().Unix()

	// Fan out to databases
	go h.writeToCassandra(logReq, logID, timestamp)
	go h.writeToClickHouse(logReq, logID, timestamp)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *LogHandler) writeToCassandra(logData LogRequest, logID string, ts int64) {
	payload := database.LogPayload{
		ProjectID: logData.ProjectID,
		LogID:     logID,
		EventName: logData.EventName,
		Timestamp: ts,
		Payload:   logData.Payload,
	}
	if err := h.cassandraClient.WriteLog(payload); err != nil {
		log.Printf("ERROR: could not write to Cassandra: %v", err)
	}
}

func (h *LogHandler) writeToClickHouse(logData LogRequest, logID string, ts int64) {
	index := database.LogIndex{
		ProjectID: logData.ProjectID,
		LogID:     logID,
		EventName: logData.EventName,
		Timestamp: ts,
	}
	if err := h.chClient.WriteLog(index); err != nil {
		log.Printf("ERROR: could not write to ClickHouse: %v", err)
	}
}

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("Could not load config in consumer: %v", err)
	}

	clickhouseClient, err := database.NewClickHouseClient(cfg.Clickhouse)
	if err != nil {
		log.Fatalf("Could not connect to ClickHouse: %v", err)
	}

	cassandraClient, err := database.NewCassandraClient(cfg.Cassandra)
	if err != nil {
		log.Fatalf("Could not connect to CassandraHouse: %v", err)
	}

	// Start Kafka consumer in background (unchanged - your partner's implementation)
	consumerService := kafka.NewConsumer(cfg.Kafka, clickhouseClient, cassandraClient)
	go func() {
		log.Println("Starting Kafka consumer service...")
		consumerService.Start()
	}()

	// Start HTTP server for direct API communication
	handler := &LogHandler{
		chClient:        clickhouseClient,
		cassandraClient: cassandraClient,
	}

	r := mux.NewRouter()
	r.HandleFunc("/logs", handler.handleLog).Methods("POST")

	port := ":8081" // Different port from API service
	log.Printf("Consumer HTTP server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, r))
}
