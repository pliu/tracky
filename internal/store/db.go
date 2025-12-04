package store

import (
	"database/sql"
	"log"
	"time"

	"tracky/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./tracky.db")
	if err != nil {
		log.Fatal(err)
	}

	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL
	);`

	createNotebooksTable := `
	CREATE TABLE IF NOT EXISTS notebooks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	createNotesTable := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		notebook_id INTEGER,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(notebook_id) REFERENCES notebooks(id)
	);`

	if _, err := DB.Exec(createUsersTable); err != nil {
		log.Fatal(err)
	}
	if _, err := DB.Exec(createNotebooksTable); err != nil {
		log.Fatal(err)
	}
	if _, err := DB.Exec(createNotesTable); err != nil {
		log.Fatal(err)
	}

	// Add notebook_id column if it doesn't exist (migration for existing DBs)
	DB.Exec("ALTER TABLE notes ADD COLUMN notebook_id INTEGER")
}

// User functions
func CreateUser(username, passwordHash string) error {
	_, err := DB.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	return err
}

func GetUserByUsername(username string) (int, string, error) {
	var id int
	var hash string
	err := DB.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
	return id, hash, err
}

func GetUserID(username string) (int, error) {
	var id int
	err := DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id)
	return id, err
}

// Notebook functions
func CreateNotebook(userID int, name string) (int64, error) {
	result, err := DB.Exec("INSERT INTO notebooks (user_id, name, created_at) VALUES (?, ?, ?)", userID, name, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func CreateDefaultNotebook(userID int) (int64, error) {
	return CreateNotebook(userID, "Default")
}

func GetNotebooks(userID int) ([]models.Notebook, error) {
	rows, err := DB.Query("SELECT id, name, created_at FROM notebooks WHERE user_id = ? ORDER BY created_at ASC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notebooks []models.Notebook
	for rows.Next() {
		var nb models.Notebook
		nb.UserID = userID
		if err := rows.Scan(&nb.ID, &nb.Name, &nb.CreatedAt); err != nil {
			continue
		}
		notebooks = append(notebooks, nb)
	}
	return notebooks, nil
}

func GetNotebookByName(userID int, name string) (int, error) {
	var id int
	err := DB.QueryRow("SELECT id FROM notebooks WHERE user_id = ? AND name = ?", userID, name).Scan(&id)
	return id, err
}

func DeleteNotebook(notebookID, userID int) error {
	result, err := DB.Exec("DELETE FROM notebooks WHERE id = ? AND user_id = ?", notebookID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	// Also delete notes in this notebook
	DB.Exec("DELETE FROM notes WHERE notebook_id = ?", notebookID)
	return nil
}

// Note functions
func CreateNote(userID, notebookID int, content string) error {
	_, err := DB.Exec("INSERT INTO notes (user_id, notebook_id, content, created_at) VALUES (?, ?, ?, ?)", userID, notebookID, content, time.Now())
	return err
}

func GetNotes(userID, notebookID int) ([]models.Note, error) {
	rows, err := DB.Query("SELECT id, content, created_at FROM notes WHERE user_id = ? AND notebook_id = ? ORDER BY created_at DESC", userID, notebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		n.UserID = userID
		n.NotebookID = notebookID
		if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err != nil {
			continue
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func GetNotesByTimeRange(userID, notebookID int, start, end time.Time) ([]models.Note, error) {
	rows, err := DB.Query("SELECT content, created_at FROM notes WHERE user_id = ? AND notebook_id = ? AND created_at >= ? AND created_at <= ? ORDER BY created_at DESC", userID, notebookID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.Content, &n.CreatedAt); err != nil {
			continue
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func UpdateNote(noteID, userID int, content string) error {
	result, err := DB.Exec("UPDATE notes SET content = ? WHERE id = ? AND user_id = ?", content, noteID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func DeleteNote(noteID, userID int) error {
	result, err := DB.Exec("DELETE FROM notes WHERE id = ? AND user_id = ?", noteID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Migration: Assign orphaned notes to default notebook
func MigrateOrphanedNotes(userID int, defaultNotebookID int64) error {
	_, err := DB.Exec("UPDATE notes SET notebook_id = ? WHERE user_id = ? AND notebook_id IS NULL", defaultNotebookID, userID)
	return err
}
