package database

import (
	"log"
	"time"

	"log-analysis-system/consumer/config"
	"github.com/gocql/gocql"
)


type CassandraClient struct{
	Session *gocql.Session
}


// LogPayload represents the full log data for the Cassandra table.
type LogPayload struct {
	ProjectID string
	LogID     string
	EventName string
	Timestamp int64
	Payload   map[string]string
}


func NewCassandraClient(cfg config.CassandraConfig)(*CassandraClient,error){
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.Username,
		Password: cfg.Password,
	}

	session, err := cluster.CreateSession()

	if err != nil {
		return nil, err
	}
	return &CassandraClient{Session: session},nil
}

func (c *CassandraClient) WriteLog(logData LogPayload) error {

	log.Printf("DEBUG: Writing to Cassandra. Data: %+v, Timestamp Type: %T", logData, logData.Timestamp)
	err := c.Session.Query(`
		INSERT INTO logs (project_id, log_id, event_name, timestamp, payload) VALUES (?, ?, ?, ?, ?)
	`,
		logData.ProjectID,
		logData.LogID,
		logData.EventName,
		time.Unix(logData.Timestamp, 0),
		logData.Payload,
	).Exec()

	if err != nil {
		log.Printf("ERROR: Failed to write to Cassandra: %v", err)
		return err
	}
	return nil
}

