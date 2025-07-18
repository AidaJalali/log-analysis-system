package database

import (
	"context"
	"fmt"
	"log"

	"log-analysis-system/consumer/config"
	"github.com/ClickHouse/clickhouse-go/v2"
)


type ClickhouseClient struct {
	Conn clickhouse.Conn
}


// LogIndex represents the data structure for the ClickHouse table.
type LogIndex struct {
	ProjectID string
	LogID string
	EventName string
	Timestamp int64
	SearchableKey string
}

func NewClickHouseClient(cfg config.ClickhouseConfig)(*ClickhouseClient, error){
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	conn,err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})

	if err != nil{
		return nil,err
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil,err
	}
	
	//nil means no error
	return &ClickhouseClient{Conn: conn},nil
}


/*
func (c *CassandraClient) WriteLog(...): This declares a method named WriteLog. It's associated with the CassandraClient struct. The c *CassandraClient part is called the "receiver," meaning this method can be called on any variable that is a pointer to a CassandraClient (e.g., myClient.WriteLog(...)).
*/


func (c *ClickhouseClient) WriteLog(logData LogIndex) error {
	ctx := context.Background()
	err := c.Conn.Exec(ctx, `INSERT INTO logs_index (project_id, log_id, event_name, timestamp, searchable_key) VALUES (?, ?, ?, ?, ?)`,
		logData.ProjectID,
		logData.LogID,
		logData.EventName,
		logData.Timestamp,
		logData.SearchableKey, 
	)
	if err != nil {
		log.Printf("ERROR: Failed to write to ClickHouse: %v", err)
		return err
	}
	return nil
}



