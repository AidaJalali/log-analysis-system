package main


import(
	"log"
	"log-analysis-system/consumer/config"
	"log-analysis-system/consumer/database"
	"log-analysis-system/consumer/kafka"
)

func main(){
	cfg, err := config.Load()

	if err != nil{
		log.Fatalf("Could not load config in consumer: %v",err)
	}

	clickhouseClient, err := database.NewClickHouseClient(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("Could not connect to ClickHouse: %v", err)
	}


	cassandraClient, err := database.NewCassandraClient(cfg.Cassandra)
	if err != nil {
		log.Fatalf("Could not connect to CassandraHouse: %v", err)
	}

	consumerService := kafka.NewConsumer(cfg.Kafka, clickhouseClient, cassandraClient) 

	log.Println("Starting Kafka consumer service...")
	consumerService.Start()


}