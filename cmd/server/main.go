package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"tracky/internal/api"
	"tracky/internal/middleware"
	"tracky/internal/store/sqlstore"
)

var version = strconv.FormatInt(time.Now().Unix(), 10)

func main() {
	// Determine database type from environment (default SQLite)
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite3"
	}
	dbConnStr := os.Getenv("DB_CONN")
	if dbConnStr == "" {
		dbConnStr = "./tracky.db"
	}

	// Initialize store
	store, err := sqlstore.New(dbDriver, dbConnStr)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Create handlers
	handlers := api.NewHandlers(store)

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

	mux.HandleFunc("/api/signup", handlers.SignupHandler)
	mux.HandleFunc("/api/login", handlers.LoginHandler)
	mux.HandleFunc("/api/logout", handlers.LogoutHandler)
	mux.HandleFunc("/api/notebooks", handlers.NotebooksHandler)
	mux.HandleFunc("/api/notes", handlers.NotesHandler)
	mux.HandleFunc("/api/images", handlers.ImagesHandler)

	// Serve uploaded images with authentication
	mux.HandleFunc("/uploads/", handlers.ServeImageHandler)

	// Apply middleware: Logging -> Auth
	handler := middleware.Logging(middleware.Auth(mux))

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
