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

	createNotesTable := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	if _, err := DB.Exec(createUsersTable); err != nil {
		log.Fatal(err)
	}
	if _, err := DB.Exec(createNotesTable); err != nil {
		log.Fatal(err)
	}
}

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

func CreateNote(userID int, content string) error {
	_, err := DB.Exec("INSERT INTO notes (user_id, content, created_at) VALUES (?, ?, ?)", userID, content, time.Now())
	return err
}

func GetNotes(userID int) ([]models.Note, error) {
	rows, err := DB.Query("SELECT id, content, created_at FROM notes WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err != nil {
			continue
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func GetNotesByTimeRange(userID int, start, end time.Time) ([]models.Note, error) {
	rows, err := DB.Query("SELECT content, created_at FROM notes WHERE user_id = ? AND created_at >= ? AND created_at <= ? ORDER BY created_at DESC", userID, start, end)
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
