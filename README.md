# Log Analysis System 

This project is a Go application for real-time log analysis, using Kafka for message queuing, Cassandra for log storage, and CockroachDB for metadata.

## Requirements

* Go `1.18` or newer
* Docker and Docker Compose

## Setup & Running the System

This project uses a `docker-compose.yml` file to run all required services. The file is configured with the following service versions:

* **CockroachDB**: `latest-v23.2`
* **ZooKeeper**: `3.9.2`
* **Kafka**: `3.8.0`
* **Cassandra**: `4.1.6`

**1. Start all services:**

Run the following command from the root of the project:
```sh
docker-compose up -d
````

**2. Create the Kafka `logs` Topic:**

Once the containers are running, execute the command below to create the necessary Kafka topic.

```sh
docker exec -it kafka kafka-topics.sh --create --topic logs --bootstrap-server localhost:9092
```

**3. Run the Go Application:**

```sh
go run main.go
```

-----

## Service Endpoints

  * **CockroachDB UI**: `http://localhost:8080`
  * **CockroachDB SQL**: `localhost:26257`
  * **Kafka Broker**: `localhost:9092`
  * **Cassandra CQLSH**: `localhost:9042`

-----

## Database Schemas & Setup

### CockroachDB Setup

1.  Connect to the database:
    ```sh
    cockroach sql --insecure --host=localhost:26257
    ```
2.  Create the database and tables:
    ```sql
    CREATE DATABASE log;
    USE log;

    CREATE TABLE users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        username STRING UNIQUE NOT NULL,
        password_hash STRING NOT NULL
    );

    CREATE TABLE projects (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        name STRING NOT NULL,
        api_key STRING UNIQUE NOT NULL,
        log_ttl_seconds INT NOT NULL,
        owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
    );
    ```

### Cassandra Setup

1.  Connect to the Cassandra container:
    ```sh
    docker exec -it cassandra cqlsh
    ```
2.  Create the keyspace and tables for storing logs (example):
    ```sql
    CREATE KEYSPACE log_data WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

    USE log_data;

    CREATE TABLE logs (
        project_id uuid,
        timestamp timeuuid,
        level text,
        message text,
        PRIMARY KEY (project_id, timestamp)
    ) WITH CLUSTERING ORDER BY (timestamp DESC);
    ```

-----

## Stopping the System

To stop and remove all containers, networks, and volumes:

```sh
docker-compose down
```

```

Don't forget to run these commands

go get "github.com/ClickHouse/clickhouse-go/v2"
go get "github.com/gocql/gocql"
```
