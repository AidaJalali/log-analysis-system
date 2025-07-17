package database

import (
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

