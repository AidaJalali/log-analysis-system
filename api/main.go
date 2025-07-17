package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"database/sql"

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
        fmt.Printf("Failed to connect to database: %v", err)
        os.Exit(1)
    }
    fmt.Println("Connected to database successfully!")


	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/dashboard", dashboardHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	port := ":8080"
	fmt.Printf("Server starting at http://localhost%s ...", port)
	err := http.ListenAndServe(port, nil)
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
		err := db.QueryRow(`SELECT password_hash FROM users WHERE username = $1`, username).Scan(&hash)
		if err != nil {
			tmpl := template.Must(template.ParseFiles("templates/login.html"))
			tmpl.Execute(w, map[string]string{"Error": "Invalid username or password."})
			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {
			tmpl := template.Must(template.ParseFiles("templates/login.html"))
			tmpl.Execute(w, map[string]string{"Error": "Invalid username or password."})
			return
		}
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
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		_, err = db.Exec(`INSERT INTO users (username, password_hash) VALUES ($1, $2)`, username, string(hash))
		if err != nil {
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
	username := "testuser" // TODO: Replace with session-based user
	rows, err := db.Query(`SELECT name, api_key, log_ttl_seconds FROM projects WHERE owner_id = (SELECT id FROM users WHERE username = $1)`, username)
	if err != nil {
		http.Error(w, "Could not load projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	projects := []map[string]interface{}{}
	for rows.Next() {
		var name, apiKey string
		var logTTL int
		if err := rows.Scan(&name, &apiKey, &logTTL); err != nil {
			continue
		}
		projects = append(projects, map[string]interface{}{
			"Name": name,
			"ApiKey": apiKey,
			"LogTTLSeconds": logTTL,
		})
	}
	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	tmpl.Execute(w, map[string]interface{}{"Projects": projects})
}