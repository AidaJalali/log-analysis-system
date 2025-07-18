# HTTP Communication Setup Guide

This guide shows how to set up HTTP communication between the API and Consumer services without changing any existing Kafka implementation.

## 🏗️ Architecture

```
API Service (Port 8080) → HTTP Request → Consumer Service (Port 8081)
                                    ↓
                              ClickHouse + Cassandra
```

## 🚀 Quick Setup

### 1. Start Infrastructure (Docker)
```bash
docker compose up -d
```

### 2. Start Consumer Service
```bash
cd consumer

# Set environment variables
export KAFKA_BROKERS=localhost:9092
export CASSANDRA_HOSTS=localhost:9042
export CASSANDRA_KEYSPACE=log_data
export CLICKHOUSE_HOST=localhost
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=log_data
export CLICKHOUSE_USER=user
export CLICKHOUSE_PASSWORD=password

# Start the service
go run main.go
```

**Consumer will start on port 8081** and provide:
- HTTP endpoint: `POST http://localhost:8081/logs`
- Kafka consumer (unchanged - your partner's implementation)

### 3. Start API Service
```bash
cd api

# Set environment variables
export CONSUMER_URL=http://localhost:8081  # Optional - defaults to this value

# Start the service
go run main.go
```

**API will start on port 8080** and provide:
- Web dashboard: `http://localhost:8080`
- REST API: `POST http://localhost:8080/api/projects/{id}/logs`

## 📡 HTTP Communication

### API → Consumer Request Format
```json
POST http://localhost:8081/logs
Content-Type: application/json

{
  "project_id": "your-project-id",
  "event_name": "user_login",
  "payload": {
    "user_id": "12345",
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  }
}
```

### Consumer Response
```json
{
  "status": "ok"
}
```

## 🔧 Environment Variables

### Consumer Service
- `KAFKA_BROKERS` - Kafka broker addresses (for your partner's implementation)
- `CASSANDRA_HOSTS` - Cassandra host addresses
- `CASSANDRA_KEYSPACE` - Cassandra keyspace name
- `CLICKHOUSE_HOST` - ClickHouse host
- `CLICKHOUSE_PORT` - ClickHouse port
- `CLICKHOUSE_DATABASE` - ClickHouse database name
- `CLICKHOUSE_USER` - ClickHouse username
- `CLICKHOUSE_PASSWORD` - ClickHouse password

### API Service
- `CONSUMER_URL` - Consumer service URL (defaults to `http://localhost:8081`)
- `DATABASE_URL` - CockroachDB connection string (optional)

## 🧪 Testing

### 1. Create a project via web interface
- Go to `http://localhost:8080`
- Sign up/login
- Create a project
- Note the API key

### 2. Send a test log
```bash
curl -X POST http://localhost:8080/api/projects/YOUR_PROJECT_ID/logs \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: YOUR_API_KEY" \
  -d '{
    "event_name": "test_event",
    "timestamp": 1640995200,
    "payload": {
      "test_key": "test_value",
      "number": 123
    }
  }'
```

### 3. Check the logs
- View in web dashboard: `http://localhost:8080/dashboard/YOUR_PROJECT_ID`
- Check ClickHouse and Cassandra databases

## 📊 Port Summary

- **API Service**: `localhost:8080` (Web + REST API)
- **Consumer Service**: `localhost:8081` (HTTP endpoint)
- **CockroachDB**: `localhost:26257` (Metadata)
- **ClickHouse**: `localhost:8123/9000` (Analytics)
- **Cassandra**: `localhost:9042` (Full payload)
- **Kafka**: `localhost:9092` (Your partner's implementation)

## ✅ What's Working

1. **API Service** receives logs via HTTP
2. **API Service** stores basic metadata in CockroachDB (for web display)
3. **API Service** sends log data to Consumer via HTTP
4. **Consumer Service** receives HTTP requests and stores in ClickHouse + Cassandra
5. **Consumer Service** also runs Kafka consumer (unchanged - your partner's code)
6. **Web Dashboard** displays logs from CockroachDB

## 🔍 Troubleshooting

### Consumer not receiving requests?
- Check if consumer is running on port 8081
- Verify `CONSUMER_URL` environment variable
- Check consumer logs for errors

### Database connection issues?
- Ensure Docker containers are running
- Verify environment variables are set correctly
- Check database schemas are created

### API service errors?
- Check if consumer service is accessible
- Verify API key authentication
- Check CockroachDB connection

## 🎯 Benefits

- ✅ **No changes to Kafka implementation** (your partner's code untouched)
- ✅ **Simple HTTP communication** between services
- ✅ **Same data processing logic** as Kafka consumer
- ✅ **Easy to test and debug**
- ✅ **Works alongside existing Kafka setup** 