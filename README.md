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

To stop and remove the container:

```sh
docker-compose down
```
