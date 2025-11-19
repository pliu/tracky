package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tracky/internal/models"
	"tracky/internal/store"
)

func TestMain(m *testing.M) {
	// Setup
	store.InitDB()
	// Run tests
	code := m.Run()
	// Teardown
	store.DB.Close()
	os.Remove("./tracky.db")
	os.Exit(code)
}

func TestSignupAndLogin(t *testing.T) {
	// Signup
	signupBody := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/signup", strings.NewReader(signupBody))
	w := httptest.NewRecorder()
	SignupHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	// Login
	loginBody := `{"username": "testuser", "password": "password123"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(loginBody))
	w = httptest.NewRecorder()
	LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	// Check cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Expected session cookie")
	}
}

func TestNotesFlow(t *testing.T) {
	// Signup
	signupBody := `{"username": "noteuser", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/signup", strings.NewReader(signupBody))
	w := httptest.NewRecorder()
	SignupHandler(w, req)

	// Login
	loginBody := `{"username": "noteuser", "password": "password123"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(loginBody))
	w = httptest.NewRecorder()
	LoginHandler(w, req)
	cookie := w.Result().Cookies()[0]

	// Create Note
	noteBody := `{"content": "This is a test note"}`
	req = httptest.NewRequest("POST", "/api/notes", strings.NewReader(noteBody))
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	NotesHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	// Get Notes
	req = httptest.NewRequest("GET", "/api/notes", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	NotesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var notes []models.Note
	if err := json.NewDecoder(w.Body).Decode(&notes); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}

	if notes[0].Content != "This is a test note" {
		t.Errorf("Expected note content 'This is a test note', got '%s'", notes[0].Content)
	}
}
