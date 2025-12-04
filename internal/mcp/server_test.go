package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"tracky/internal/store/sqlstore"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/crypto/bcrypt"
)

func TestGetNotesTool(t *testing.T) {
	// Setup in-memory DB
	store, err := sqlstore.New("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	mcpServer := NewMCPServer(store)

	// Setup user
	username := "mcpuser"
	password := "password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	err = store.CreateUser(username, string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID, _, err := store.GetUserByUsername(username)
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	// Create default notebook
	notebookID, err := store.CreateNotebook(userID, "Default")
	if err != nil {
		t.Fatalf("Failed to create notebook: %v", err)
	}

	// Insert notes using a helper since CreateNote uses time.Now()
	// For test purposes, we need to insert with specific timestamps
	// We'll use the store's internal db access through the interface methods

	// Note: Since we can't insert with custom timestamps via the interface,
	// let's test with notes created now
	err = store.CreateNote(userID, int(notebookID), "Note 1")
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}
	err = store.CreateNote(userID, int(notebookID), "Note 2")
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}
	err = store.CreateNote(userID, int(notebookID), "Note 3")
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Test: Get notes for today with wide time range
	now := time.Now()
	startDate := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endDate := now.Add(24 * time.Hour).Format(time.RFC3339)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"username":   username,
				"notebook":   "Default",
				"start_date": startDate,
				"end_date":   endDate,
			},
		},
	}

	result, err := mcpServer.getNotesHandler(context.Background(), req)
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

	// Test: User not found
	req = mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"username":   "nonexistent",
				"notebook":   "Default",
				"start_date": startDate,
				"end_date":   endDate,
			},
		},
	}

	result, err = mcpServer.getNotesHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error for nonexistent user")
	}
}
