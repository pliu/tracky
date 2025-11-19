package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"tracky/internal/store"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func getNotesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, err := request.RequireString("username")
	if err != nil {
		return mcp.NewToolResultError("username is required"), nil
	}
	startDateStr, err := request.RequireString("start_date")
	if err != nil {
		return mcp.NewToolResultError("start_date is required"), nil
	}
	endDateStr, err := request.RequireString("end_date")
	if err != nil {
		return mcp.NewToolResultError("end_date is required"), nil
	}

	start, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start_date: %v", err)), nil
	}
	end, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid end_date: %v", err)), nil
	}

	// Look up user ID
	userID, err := store.GetUserID(username)
	if err == sql.ErrNoRows {
		return mcp.NewToolResultError("user not found"), nil
	} else if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("database error: %v", err)), nil
	}

	// Query notes
	notes, err := store.GetNotesByTimeRange(userID, start, end)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("database error: %v", err)), nil
	}

	if len(notes) == 0 {
		return mcp.NewToolResultText("No notes found for this time range."), nil
	}

	var noteStrings []string
	for _, n := range notes {
		noteStrings = append(noteStrings, fmt.Sprintf("[%s] %s", n.CreatedAt.Format(time.RFC3339), n.Content))
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d notes:\n%s", len(notes), strings.Join(noteStrings, "\n"))), nil
}

func NewServer() *server.StreamableHTTPServer {
	// Initialize MCP Server
	mcpServer := server.NewMCPServer("Tracky", "1.0.0")

	// Define tool
	tool := mcp.NewTool("get_notes",
		mcp.WithDescription("Retrieve notes for a user within a specific time range."),
		mcp.WithString("username", mcp.Required(), mcp.Description("The username to fetch notes for")),
		mcp.WithString("start_date", mcp.Required(), mcp.Description("Start of the time range (RFC3339), e.g. 2023-01-01T00:00:00Z")),
		mcp.WithString("end_date", mcp.Required(), mcp.Description("End of the time range (RFC3339), e.g. 2023-12-31T23:59:59Z")),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)

	mcpServer.AddTool(tool, getNotesHandler)

	// Create SSE server
	return server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))
}
