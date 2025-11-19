package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"tracky/internal/store"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/crypto/bcrypt"
)

func TestGetNotesTool(t *testing.T) {
	// Setup DB
	store.InitDB()
	defer store.DB.Close()

	// Setup user
	username := "mcpuser"
	password := "password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := store.DB.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	var userID int
	err = store.DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	// Insert notes
	// Note 1: 2023-01-01
	t1, _ := time.Parse("2006-01-02", "2023-01-01")
	store.DB.Exec("INSERT INTO notes (user_id, content, created_at) VALUES (?, ?, ?)", userID, "Note 1", t1)

	// Note 2: 2023-06-01
	t2, _ := time.Parse("2006-01-02", "2023-06-01")
	store.DB.Exec("INSERT INTO notes (user_id, content, created_at) VALUES (?, ?, ?)", userID, "Note 2", t2)

	// Note 3: 2023-12-31
	t3, _ := time.Parse("2006-01-02", "2023-12-31")
	store.DB.Exec("INSERT INTO notes (user_id, content, created_at) VALUES (?, ?, ?)", userID, "Note 3", t3)

	// Test Case 1: All year 2023
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"username":   username,
				"start_date": "2023-01-01T00:00:00Z",
				"end_date":   "2023-12-31T23:59:59Z",
			},
		},
	}

	result, err := getNotesHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Result is error: %v", result)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent")
	}
	content := textContent.Text
	if !strings.Contains(content, "Note 1") || !strings.Contains(content, "Note 2") || !strings.Contains(content, "Note 3") {
		t.Errorf("Expected all notes, got: %s", content)
	}

	// Test Case 2: First half 2023
	req = mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"username":   username,
				"start_date": "2023-01-01T00:00:00Z",
				"end_date":   "2023-06-30T23:59:59Z",
			},
		},
	}

	result, err = getNotesHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	textContent, ok = result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent")
	}
	content = textContent.Text
	if !strings.Contains(content, "Note 1") || !strings.Contains(content, "Note 2") {
		t.Errorf("Expected Note 1 and Note 2, got: %s", content)
	}
	if strings.Contains(content, "Note 3") {
		t.Errorf("Did not expect Note 3, got: %s", content)
	}
}
