

# Log Analysis System

This project is a Go application for real-time log analysis, using Kafka for message queuing, **ClickHouse** and **Cassandra** for durable log storage, and **CockroachDB** for metadata.

-----

## Requirements

  * Go `1.18` or newer
  * Docker and Docker Compose

-----

## Setup & Running the System

This project uses `docker-compose.yml` to run all services. For reproducibility, it's configured with the following specific versions:

  * **CockroachDB**: `v24.1.4`
  * **ZooKeeper**: `3.9.2`
  * **Kafka**: `3.8.0`
  * **Cassandra**: `4.1`
  * **ClickHouse**: `25.6-alpine`

Follow these steps to get the entire system running.

**1. Start All Services**

Run the following command from the root of the project to start all databases and Kafka in the background.

```sh
docker compose up -d
```

**2. Create the Kafka `logs` Topic**

Once the containers are running, execute this command to create the necessary Kafka topic.

```sh
docker exec -it kafka kafka-topics.sh --create --topic logs --bootstrap-server localhost:9092
```

**3. Set Up Database Schemas**

You need to connect to each database to create the required tables. Follow the instructions in the **Database Schemas & Setup** section below.

**4. Set Environment Variables**

Before running the application, you must set environment variables so the Go program knows how to connect to the other services.





**5. Run the Go Application**

Finally, run the main application.

```sh
go run main.go
```

-----

## Service Endpoints

  * **CockroachDB SQL**: `localhost:26257`
  * **Kafka Broker**: `localhost:9092`
  * **Cassandra CQL**: `localhost:9042`
  * **ClickHouse HTTP**: `http://localhost:8123`
  * **ClickHouse Native**: `localhost:9000`
  * **KAFKA_TOPIC**: `logs`
  * **CLICKHOUSE_HOST**: `localhost`
  * **CLICKHOUSE_PORT**: `9000`
  * **CLICKHOUSE_DATABASE**: `default`
  * **CLICKHOUSE_USER**: `user`
  * **CLICKHOUSE_PASSWORD**: `password`
  * **CASSANDRA_KEYSPACE**: `log_system`
  * **CASSANDRA_HOSTS**: `127.0.0.1:9042`
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

### ClickHouse Setup

Connect to the ClickHouse and create tables:
    ```sh
    docker exec -it click_house /usr/bin/clickhouse-client -q "CREATE TABLE IF NOT EXISTS default.logs_index (project_id UUID, log_id UUID, event_name String, timestamp DateTime, searchable_key_1 String) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (project_id, event_name, timestamp);"
    ```

### Cassandra Setup

Connect to the Cassandra client and create required tables:
    ```sh
    docker exec -it cassandra cqlsh -e "CREATE KEYSPACE log_system WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};"
    docker exec -it cassandra cqlsh -e "CREATE TABLE log_system.logs (project_id uuid, log_id uuid, event_name text, timestamp timestamp, payload map<text, text>, PRIMARY KEY (project_id, log_id));"

    ```
-----

## Stopping the System

To stop and remove all containers, networks, and volumes:

```sh
docker compose down --volumes
```