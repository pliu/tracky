package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tracky/internal/auth"
	"tracky/internal/models"
	"tracky/internal/store/sqlstore"
)

var testHandlers *Handlers

func TestMain(m *testing.M) {
	// Setup - use in-memory SQLite for tests
	store, err := sqlstore.New("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	testHandlers = NewHandlers(store)

	// Run tests
	code := m.Run()

	// Teardown
	store.Close()
	os.Exit(code)
}

// Helper to create request with user ID in context
func requestWithUserID(req *http.Request, userID int) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestSignupAndLogin(t *testing.T) {
	// Signup
	signupBody := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/signup", strings.NewReader(signupBody))
	w := httptest.NewRecorder()
	testHandlers.SignupHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	// Login
	loginBody := `{"username": "testuser", "password": "password123"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(loginBody))
	w = httptest.NewRecorder()
	testHandlers.LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	// Check cookie - should now be auth_token
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "auth_token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected auth_token cookie")
	}
}

func TestNotesFlow(t *testing.T) {
	// Signup
	signupBody := `{"username": "noteuser2", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/signup", strings.NewReader(signupBody))
	w := httptest.NewRecorder()
	testHandlers.SignupHandler(w, req)

	// Get user ID from store
	userID, _, _ := testHandlers.Store.GetUserByUsername("noteuser2")

	// Get notebooks - inject user ID into context
	req = httptest.NewRequest("GET", "/api/notebooks", nil)
	req = requestWithUserID(req, userID)
	w = httptest.NewRecorder()
	testHandlers.NotebooksHandler(w, req)

	var notebooks []models.Notebook
	json.NewDecoder(w.Body).Decode(&notebooks)
	if len(notebooks) == 0 {
		t.Fatal("Expected at least one notebook")
	}
	notebookID := notebooks[0].ID

	// Create Note
	noteBody := `{"content": "This is a test note"}`
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/notes?notebook_id=%d", notebookID), strings.NewReader(noteBody))
	req = requestWithUserID(req, userID)
	w = httptest.NewRecorder()
	testHandlers.NotesHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	// Get Notes
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/notes?notebook_id=%d", notebookID), nil)
	req = requestWithUserID(req, userID)
	w = httptest.NewRecorder()
	testHandlers.NotesHandler(w, req)

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
