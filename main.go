package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/yuin/goldmark"
)

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/signup", signupHandler)

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
	// TODO: Handle POST for login
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/signup.html"))
		tmpl.Execute(w, nil)
		return
	}
	// TODO: Handle POST for signup
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
