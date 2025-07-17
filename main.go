package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"database/sql"

	"crypto/rand"
	"encoding/hex"
	"strings"

	"log"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

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

func main() {

	if err := initDB(); err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
	fmt.Println("Connected to database successfully!")

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/signup", signupHandler)
	r.HandleFunc("/dashboard", dashboardHandler)
	r.HandleFunc("/dashboard/{projectID}", projectHandler)
	r.HandleFunc("/projects/create", createProjectHandler)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	port := ":8080"
	fmt.Printf("Server starting at http://localhost%s ...", port)
	err := http.ListenAndServe(port, r)
	if err != nil {
		fmt.Printf("server failed", err)
	}

}

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
	// TODO: Add project-specific logic and data fetching here
	tmpl := template.Must(template.ParseFiles("templates/project.html"))
	tmpl.Execute(w, map[string]interface{}{"ProjectID": projectID})
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
	var projectID string
	err = db.QueryRow(
		`INSERT INTO projects (name, api_key, log_ttl_seconds, owner_id) VALUES ($1, $2, $3, $4) RETURNING id`,
		projectName, apiKey, ttl, userID,
	).Scan(&projectID)
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
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
