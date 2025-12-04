package store

import (
	"time"

	"tracky/internal/models"
)

// Store defines the interface for all database operations
type Store interface {
	// Users
	CreateUser(username, passwordHash string) error
	GetUserByUsername(username string) (int, string, error)
	GetUserID(username string) (int, error)

	// Notebooks
	CreateNotebook(userID int, name string) (int64, error)
	CreateDefaultNotebook(userID int) (int64, error)
	GetNotebooks(userID int) ([]models.Notebook, error)
	GetNotebookByName(userID int, name string) (int, error)
	DeleteNotebook(notebookID, userID int) error

	// Notes
	CreateNote(userID, notebookID int, content string) error
	GetNotes(userID, notebookID int) ([]models.Note, error)
	GetNotesByTimeRange(userID, notebookID int, start, end time.Time) ([]models.Note, error)
	UpdateNote(noteID, userID int, content string) error
	DeleteNote(noteID, userID int) error

	// Note Images
	CreateNoteImage(noteID int, filename string) (int64, error)
	GetNoteImages(noteID int) ([]models.NoteImage, error)
	GetNoteImageWithOwner(imageID, userID int) (string, error) // Returns filename if user owns image
	DeleteNoteImage(imageID int) (string, error)
	GetNoteImagesByNoteIDs(noteIDs []int) (map[int][]models.NoteImage, error)

	Close() error
}
