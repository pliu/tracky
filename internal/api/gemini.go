package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"tracky/internal/models"

	"google.golang.org/genai"
)

// AnalyzeNotes sends notes and a question to Gemini API and returns the response
func AnalyzeNotes(notes []models.Note, question string, history []models.ChatMessage) (string, error) {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Build context from notes
	var notesContext strings.Builder
	notesContext.WriteString("You are a helpful assistant analyzing a user's personal notes. ")
	notesContext.WriteString("Here are the notes from their notebook:\n\n")

	for _, note := range notes {
		timestamp := note.CreatedAt.Format(time.RFC1123)
		notesContext.WriteString(fmt.Sprintf("--- Note from %s ---\n%s\n\n", timestamp, note.Content))
	}

	// Convert history
	var chatHistory []*genai.Content
	for _, msg := range history {
		role := "user"
		if msg.Role == "model" {
			role = "model"
		}
		chatHistory = append(chatHistory, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: msg.Content},
			},
		})
	}

	// Debug logging
	fmt.Println("=== GEMINI REQUEST DEBUG ===")
	fmt.Println("--- System Instruction ---")
	fmt.Println(notesContext.String())
	fmt.Println("--- Chat History ---")
	for _, msg := range history {
		fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
	}
	fmt.Println("--- Current Question ---")
	fmt.Println(question)
	fmt.Println("============================")

	// Create chat session
	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: notesContext.String()},
			},
		},
	}, chatHistory)
	if err != nil {
		return "", fmt.Errorf("failed to create chat session: %w", err)
	}

	resp, err := chat.SendMessage(ctx, genai.Part{Text: question})
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	// Extract text from the first part
	part := resp.Candidates[0].Content.Parts[0]
	if part.Text != "" {
		return part.Text, nil
	}

	return "", fmt.Errorf("empty response from Gemini")
}
