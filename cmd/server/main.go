package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"tracky/internal/api"
	"tracky/internal/mcp"
	"tracky/internal/middleware"
	"tracky/internal/store"
)

var version = strconv.FormatInt(time.Now().Unix(), 10)

func main() {
	store.InitDB()
	defer store.DB.Close()

	mux := http.NewServeMux()

	// Serve index.html with cache-busting version
	tmpl := template.Must(template.ParseFiles("./static/index.html"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
			return
		}
		tmpl.Execute(w, map[string]string{"Version": version})
	})

	// Serve other static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	mux.HandleFunc("/api/signup", api.SignupHandler)
	mux.HandleFunc("/api/login", api.LoginHandler)
	mux.HandleFunc("/api/logout", api.LogoutHandler)
	mux.HandleFunc("/api/notebooks", api.NotebooksHandler)
	mux.HandleFunc("/api/notes", api.NotesHandler)

	// Add MCP route
	sseServer := mcp.NewServer()
	mux.Handle("/mcp", sseServer)

	handler := middleware.Logging(mux)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
