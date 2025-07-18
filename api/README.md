# Log Analysis System

This project is a Go application for log analysis.

## Getting Started

**Run the application:**
   ```sh
   go run main.go
   ```

## Project Structure
- `go.mod` - Go module definition
- `main.go` - Application entry point (to be created)

## Requirements
- Go 1.18 or newer

---


### cockraoch docker image tag 
cockroachdb/cockroach:latest-v23.2



###Cockrocach DB models

commands:
docker exec -it bb4e807cc011 cockroach sql --insecure
cockroach sql --insecure

create database log;
use log;

```sql
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
CREATE TABLE user_projects (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, project_id)
);
CREATE TABLE project_searchable_keys (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key_name STRING NOT NULL,
    PRIMARY KEY (project_id, key_name)
);


Feel free to update this README as the project evolves.

## Running CockroachDB with Docker Compose

This project includes a `docker-compose.yml` file to run CockroachDB for development and testing.

To start CockroachDB:

```sh
docker-compose up -d
```

- The database will be available on port 26257 (SQL) and 8080 (web UI).
- Data is stored in a 500MB tmpfs volume for persistence during container runtime.

## Running Kafka and Zookeeper

    The configuration of Kafka is available in docker-compose file.

  **Create the `logs` Topic:**
    Once the container is running, you need to create the Kafka topic that the application will use.

    ```sh
    # Replace 'kafka-container-name' with your actual container name from 'docker ps'
    docker exec -it kafka-container-name kafka-topics.sh --create --topic logs --bootstrap-server localhost:9092
    ```

      * The Kafka broker will be available on port `9092`.


To stop and remove the container:

```sh
docker-compose down
```
