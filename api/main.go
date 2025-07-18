package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/bcrypt"
)

// Global variables
var db *sql.DB
var kafkaWriter *kafka.Writer
var clickhouseConn clickhouse.Conn
var cassandraSession *gocql.Session

// Structs
type ClickHouseLog struct {
	LogID     string `json:"log_id"`
	EventName string `json:"event_name"`
	Timestamp int64  `json:"timestamp"`
}

type CassandraLog struct {
	ProjectID string            `json:"project_id"`
	LogID     string            `json:"log_id"`
	EventName string            `json:"event_name"`
	Timestamp int64             `json:"timestamp"`
	Payload   map[string]string `json:"payload"`
}

// Database initialization functions
func initDB() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Default connection string for local CockroachDB
		connStr = "postgresql://root@localhost:26257/log?sslmode=disable"
	}
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		println("database is not connected yet")
		return err
	}
	return db.Ping()
}

func initKafka() error {
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		return fmt.Errorf("KAFKA_BROKERS environment variable not set")
	}

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(kafkaBrokers, ",")...),
		Topic:    "logs",
		Balancer: &kafka.LeastBytes{},
	}
	return nil
}

func initClickHouse() error {
	// These values should ideally come from environment variables or config
	addr := os.Getenv("CLICKHOUSE_ADDR")
	if addr == "" {
		addr = "localhost:9000"
	}
	username := os.Getenv("CLICKHOUSE_USER")
	if username == "" {
		username = "default"
	}
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	database := os.Getenv("CLICKHOUSE_DB")
	if database == "" {
		database = "default"
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return err
	}
	clickhouseConn = conn
	return nil
}

func initCassandra() error {
	hosts := os.Getenv("CASSANDRA_HOSTS")
	if hosts == "" {
		return fmt.Errorf("CASSANDRA_HOSTS environment variable not set")
	}
	keyspace := os.Getenv("CASSANDRA_KEYSPACE")
	if keyspace == "" {
		return fmt.Errorf("CASSANDRA_KEYSPACE environment variable not set")
	}
	username := os.Getenv("CASSANDRA_USER")
	password := os.Getenv("CASSANDRA_PASSWORD")

	cluster := gocql.NewCluster(strings.Split(hosts, ",")...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	if username != "" || password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: username,
			Password: password,
		}
	}
	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}
	cassandraSession = session
	return nil
}

// Main function
func main() {
	if err := initDB(); err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
	fmt.Println("Connected to database successfully!")

	if err := initKafka(); err != nil {
		panic("Failed to connect to Kafka: " + err.Error())
	}
	fmt.Println("Connected to Kafka successfully!")

	// Initialize ClickHouse
	if err := initClickHouse(); err != nil {
		panic("Failed to connect to ClickHouse: " + err.Error())
	}
	fmt.Println("Connected to ClickHouse successfully!")

	if err := initCassandra(); err != nil {
		panic("Failed to connect to Cassandra: " + err.Error())
	}
	fmt.Println("Connected to Cassandra successfully!")

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/signup", signupHandler)
	// Use strict match for /dashboard and a subrouter for /dashboard/{projectID}
	r.HandleFunc("/dashboard", dashboardHandler).Methods("GET")
	r.HandleFunc("/dashboard/{projectID}", projectHandler).Methods("GET")
	r.HandleFunc("/projects/create", createProjectHandler)
	r.HandleFunc("/api/projects/{projectID}/logs", apiLogHandler).Methods("POST")
	r.HandleFunc("/api/projects/{projectID}/logs", apiProjectLogsHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.HandleFunc("/api/projects/{projectID}/logs/{logID}", apiProjectLogDetailHandler).Methods("GET")
	r.HandleFunc("/projects/{projectID}/logs/{logID}", logDetailsPageHandler).Methods("GET")

	port := ":8080"
	fmt.Printf("Server starting at http://localhost%s ...", port)
	err := http.ListenAndServe(port, r)
	if err != nil {
		fmt.Printf("server failed:%s", err)
	}
}

// Web page handlers
func homeHandler(w http.ResponseWriter, r *http.Request) {
	readmeContent, err := os.ReadFile("README.md")
	if err != nil {
		http.Error(w, "Could not read README.md", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	err = goldmark.Convert(readmeContent, &buf)
	if err != nil {
		http.Error(w, "Could not render markdown", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/home.html"))
	tmpl.Execute(w, map[string]interface{}{
		"ReadmeHTML": template.HTML(buf.String()),
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/login.html"))
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		var hash string
		var userID string
		err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username = $1`, username).Scan(&userID, &hash)
		if err != nil {
			log.Printf("loginHandler: error querying password hash for username '%s': %v", username, err)
			tmpl := template.Must(template.ParseFiles("templates/login.html"))
			tmpl.Execute(w, map[string]string{"Error": "Invalid username or password."})
			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {
			log.Printf("loginHandler: password mismatch for username '%s': %v", username, err)
			tmpl := template.Must(template.ParseFiles("templates/login.html"))
			tmpl.Execute(w, map[string]string{"Error": "Invalid username or password."})
			return
		}
		// Set user_id cookie
		cookie := &http.Cookie{
			Name:     "user_id",
			Value:    userID,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		// On success, redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/signup.html"))
		tmpl.Execute(w, nil)
		return
	}
	if r.Method == http.MethodPost {
		errMsg := ""
		username := r.FormValue("username")
		password := r.FormValue("password")
		repeatPassword := r.FormValue("repeat_password")
		if username == "" || password == "" || repeatPassword == "" {
			errMsg = "All fields are required."
		} else if password != repeatPassword {
			errMsg = "Passwords do not match."
		}
		if errMsg != "" {
			tmpl := template.Must(template.ParseFiles("templates/signup.html"))
			tmpl.Execute(w, map[string]string{"Error": errMsg})
			return
		}
		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("signupHandler: error hashing password for username '%s': %v", username, err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		_, err = db.Exec(`INSERT INTO users (username, password_hash) VALUES ($1, $2)`, username, string(hash))
		if err != nil {
			log.Printf("signupHandler: error inserting user '%s': %v", username, err)
			tmpl := template.Must(template.ParseFiles("templates/signup.html"))
			tmpl.Execute(w, map[string]string{"Error": "Username already exists or DB error."})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID := ""
	if cookie, err := r.Cookie("user_id"); err == nil {
		userID = cookie.Value
	}
	if userID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	rows, err := db.Query(`SELECT id, name, api_key, log_ttl_seconds FROM projects WHERE owner_id = $1`, userID)
	if err != nil {
		log.Printf("dashboardHandler: error querying projects: %v", err)
		http.Error(w, "Could not load projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	projects := []map[string]interface{}{}
	for rows.Next() {
		var id, name, apiKey string
		var logTTL int
		if err := rows.Scan(&id, &name, &apiKey, &logTTL); err != nil {
			log.Printf("dashboardHandler: error scanning project row: %v", err)
			continue
		}
		log.Printf("dashboardHandler: project id: %s", id)
		projects = append(projects, map[string]interface{}{
			"ID":            id,
			"Name":          name,
			"ApiKey":        apiKey,
			"LogTTLSeconds": logTTL,
		})
	}
	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	tmpl.Execute(w, map[string]interface{}{"Projects": projects})
}

func projectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	// Fetch project details
	var name, apiKey string
	var keys []string
	row := db.QueryRow(`SELECT name, api_key FROM projects WHERE id = $1`, projectID)
	err := row.Scan(&name, &apiKey)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	rows, err := db.Query(`SELECT key_name FROM project_searchable_keys WHERE project_id = $1`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err == nil {
				keys = append(keys, key)
			}
		}
	}
	// Pagination
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	offset := (page - 1) * perPage
	// Fetch logs for this project
	logRows, err := db.Query(`SELECT id, event_name, timestamp, payload FROM logs WHERE project_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`, projectID, perPage, offset)
	logs := []map[string]interface{}{}
	if err == nil {
		defer logRows.Close()
		for logRows.Next() {
			var id, eventName, payload string
			var timestamp int64
			if err := logRows.Scan(&id, &eventName, &timestamp, &payload); err == nil {
				logs = append(logs, map[string]interface{}{
					"ID":        id,
					"EventName": eventName,
					"Timestamp": timestamp,
					"Payload":   payload,
				})
			}
		}
	}
	// Count total logs for pagination
	totalLogs := 0
	db.QueryRow(`SELECT count(*) FROM logs WHERE project_id = $1`, projectID).Scan(&totalLogs)
	totalPages := (totalLogs + perPage - 1) / perPage
	loading := r.URL.Query().Get("loading") == "1"
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}
	tmpl := template.Must(template.ParseFiles("templates/project.html"))
	tmpl.Execute(w, map[string]interface{}{
		"ProjectID":      projectID,
		"ProjectName":    name,
		"ApiKey":         apiKey,
		"SearchableKeys": strings.Join(keys, ", "),
		"Logs":           logs,
		"Page":           page,
		"TotalPages":     totalPages,
		"PrevPage":       prevPage,
		"NextPage":       nextPage,
		"Loading":        loading,
	})
}

func createProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie("user_id")
	if err != nil || cookie.Value == "" {
		log.Printf("createProjectHandler: missing or invalid user_id cookie")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	userID := cookie.Value
	projectName := r.FormValue("project_name")
	searchableKeys := r.FormValue("searchable_keys")
	ttl := r.FormValue("ttl")
	if projectName == "" || searchableKeys == "" || ttl == "" {
		log.Printf("createProjectHandler: missing form values")
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	apiKeyBytes := make([]byte, 16)
	_, err = rand.Read(apiKeyBytes)
	if err != nil {
		log.Printf("createProjectHandler: error generating API key: %v", err)
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}
	apiKey := hex.EncodeToString(apiKeyBytes)
	projectID := apiKey // Set project id and api key to the same value
	_, err = db.Exec(
		`INSERT INTO projects (id, name, api_key, log_ttl_seconds, owner_id) VALUES ($1, $2, $3, $4, $5)`,
		projectID, projectName, apiKey, ttl, userID,
	)
	if err != nil {
		log.Printf("createProjectHandler: error inserting project: %v", err)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}
	keys := strings.Split(searchableKeys, ",")
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		_, err := db.Exec(
			`INSERT INTO project_searchable_keys (project_id, key_name) VALUES ($1, $2)`,
			projectID, key,
		)
		if err != nil {
			log.Printf("createProjectHandler: error inserting searchable key '%s': %v", key, err)
		}
	}
	// Update generate-log.sh with the new API key, project ID, and correct API_URL_BASE, then run it for this project
	go func(apiKey, projectID string) {
		content, err := ioutil.ReadFile("generate-log.sh")
		if err != nil {
			log.Printf("generate-log.sh read error: %v", err)
			return
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "API_URL_BASE=") {
				lines[i] = "API_URL_BASE=\"http://localhost:8080/api/projects\""
			}
			if strings.HasPrefix(line, "API_KEY=") {
				lines[i] = "API_KEY=" + apiKey
			}
			if strings.HasPrefix(line, "PROJECT_IDS=(") {
				lines[i] = fmt.Sprintf("PROJECT_IDS=(%s)", projectID)
			}
		}
		newContent := strings.Join(lines, "\n")
		err = ioutil.WriteFile("generate-log.sh", []byte(newContent), 0755)
		if err != nil {
			log.Printf("generate-log.sh write error: %v", err)
			return
		}
		cmd := exec.Command("/bin/bash", "generate-log.sh")
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("generate-log.sh exec error: %v, output: %s", err, string(out))
		} else {
			log.Printf("generate-log.sh executed successfully: %s", string(out))
		}
	}(apiKey, projectID)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func logDetailsPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/log_details.html"))
	tmpl.Execute(w, nil)
}

// API handlers
func apiLogHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	apiKey := r.Header.Get("X-API-KEY")

	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Validate API key against the database
	var dbApiKey string
	err := db.QueryRow(`SELECT api_key FROM projects WHERE id = $1`, projectID).Scan(&dbApiKey)
	if err != nil || dbApiKey != apiKey {
		http.Error(w, "Invalid API key or project", http.StatusUnauthorized)
		return
	}

	// Decode the incoming JSON from the request body
	var incomingLog map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&incomingLog); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Prepare the final message for Kafka, adding the projectID
	messageForKafka := map[string]interface{}{
		"project_id": projectID,
		"payload":    incomingLog,
	}

	// Convert the map to a JSON byte array
	messageBytes, err := json.Marshal(messageForKafka)
	if err != nil {
		http.Error(w, "Failed to serialize log message", http.StatusInternalServerError)
		return
	}

	// Write the message to the Kafka topic
	err = kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Value: messageBytes,
		},
	)

	if err != nil {
		log.Printf("apiLogHandler: error writing to kafka: %v", err)
		http.Error(w, "Failed to submit log", http.StatusInternalServerError)
		return
	}

	// Respond with 202 Accepted
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

func apiProjectLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	ctx := context.Background()

	search := strings.TrimSpace(r.URL.Query().Get("search"))

	var (
		query string
		args  []interface{}
	)
	if search != "" {
		query = `
          SELECT log_id, event_name, timestamp
          FROM logs_index
          WHERE project_id = ?
            AND event_name ILIKE ?
          ORDER BY timestamp DESC
          LIMIT 100
        `
		args = []interface{}{projectID, "%" + search + "%"}
	} else {
		query = `
          SELECT log_id, event_name, timestamp
          FROM logs_index
          WHERE project_id = ?
          ORDER BY timestamp DESC
          LIMIT 100
        `
		args = []interface{}{projectID}
	}

	rows, err := clickhouseConn.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "Failed to query ClickHouse", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []ClickHouseLog{}
	for rows.Next() {
		var l ClickHouseLog
		if err := rows.Scan(&l.LogID, &l.EventName, &l.Timestamp); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Error reading rows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func apiProjectLogDetailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]
	logID := vars["logID"]

	if cassandraSession == nil {
		http.Error(w, "Cassandra not initialized", http.StatusInternalServerError)
		return
	}

	var logData CassandraLog
	query := `SELECT project_id, log_id, event_name, timestamp, payload FROM logs WHERE project_id = ? AND log_id = ? LIMIT 1`
	m := map[string]string{}
	err := cassandraSession.Query(query, projectID, logID).Consistency(gocql.One).Scan(
		&logData.ProjectID,
		&logData.LogID,
		&logData.EventName,
		&logData.Timestamp,
		&m,
	)
	if err != nil {
		http.Error(w, "Log not found", http.StatusNotFound)
		return
	}
	logData.Payload = m
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logData)
}
