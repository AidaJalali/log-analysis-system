
# Log Analysis System

This project is a Go application for real-time log analysis, using Kafka for message queuing, **ClickHouse for log storage**, and CockroachDB for metadata.

---

## Requirements

* Go `1.18` or newer
* Docker and Docker Compose

---

## Setup & Running the System

This project uses a `docker-compose.yml` file to run all required services. The file is configured with the following service versions:

* **CockroachDB**: `v24.1.4`
* **ZooKeeper**: `3.9.2`
* **Kafka**: `3.8.0`
* **ClickHouse**: `25.6-alpine`

**1. Start all services:**

Run the following command from the root of the project:
```sh
docker compose up -d
````

**2. Create the Kafka `logs` Topic:**

Once the containers are running, execute the command below to create the necessary Kafka topic.

```sh
docker exec -it kafka kafka-topics.sh --create --topic logs --bootstrap-server localhost:9092
```

**3. Install Go Dependencies:**

```sh
go get "[github.com/ClickHouse/clickhouse-go/v2](https://github.com/ClickHouse/clickhouse-go/v2)"
```

**4. Run the Go Application:**

```sh
go run main.go
```

-----

## Service Endpoints

  * **CockroachDB UI**: `http://localhost:26257` (The UI is on the same port as SQL in recent versions)
  * **CockroachDB SQL**: `localhost:26257`
  * **Kafka Broker**: `localhost:9092`
  * **ClickHouse HTTP**: `http://localhost:8123`
  * **ClickHouse Native Client**: `localhost:9000`

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

1.  Connect to the ClickHouse container's client:
    ```sh
    docker exec -it clickhouse-server clickhouse-client --user user --password password
    ```
2.  Create the table for storing the log index in the `default_db`:
    ```sql
-- Create a new database specifically for log data
CREATE DATABASE IF NOT EXISTS log_data;

-- Switch to the new database
USE log_data;

-- Create the table
CREATE TABLE logs_index (
    project_id UUID,
    log_id UUID,
    event_name String,
    timestamp DateTime,
    searchable_key String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, event_name, timestamp);

-----

## Stopping the System

To stop and remove all containers, networks, and volumes:

```sh
docker compose down --volumes
```

```
```