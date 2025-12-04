package sqlstore

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"tracky/internal/models"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DBType represents the type of database
type DBType string

const (
	SQLite   DBType = "sqlite3"
	Postgres DBType = "postgres"
)

// SQLStore implements the Store interface for SQL databases
type SQLStore struct {
	db     *sql.DB
	dbType DBType
}

// New creates a new SQLStore with the given driver and connection string
func New(driver, connStr string) (*SQLStore, error) {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	store := &SQLStore{
		db:     db,
		dbType: DBType(driver),
	}

	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

// rebind converts ? placeholders to $1, $2, etc. for PostgreSQL
func (s *SQLStore) rebind(query string) string {
	if s.dbType == SQLite {
		return query
	}
	// Convert ? to $1, $2, etc. for PostgreSQL
	var result strings.Builder
	argNum := 1
	for _, c := range query {
		if c == '?' {
			result.WriteString(fmt.Sprintf("$%d", argNum))
			argNum++
		} else {
			result.WriteRune(c)
		}
	}
	return result.String()
}

func (s *SQLStore) initSchema() error {
	var createUsersTable, createNotebooksTable, createNotesTable, createNoteImagesTable string

	if s.dbType == Postgres {
		createUsersTable = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		);`

		createNotebooksTable = `
		CREATE TABLE IF NOT EXISTS notebooks (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`

		createNotesTable = `
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			notebook_id INTEGER REFERENCES notebooks(id),
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`

		createNoteImagesTable = `
		CREATE TABLE IF NOT EXISTS note_images (
			id SERIAL PRIMARY KEY,
			note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			filename TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`
	} else {
		createUsersTable = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		);`

		createNotebooksTable = `
		CREATE TABLE IF NOT EXISTS notebooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`

		createNotesTable = `
		CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			notebook_id INTEGER,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id),
			FOREIGN KEY(notebook_id) REFERENCES notebooks(id)
		);`

		createNoteImagesTable = `
		CREATE TABLE IF NOT EXISTS note_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(note_id) REFERENCES notes(id) ON DELETE CASCADE
		);`
	}

	for _, stmt := range []string{createUsersTable, createNotebooksTable, createNotesTable, createNoteImagesTable} {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}

	// SQLite migration for existing DBs
	if s.dbType == SQLite {
		s.db.Exec("ALTER TABLE notes ADD COLUMN notebook_id INTEGER")
	}

	return nil
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

// User functions
func (s *SQLStore) CreateUser(username, passwordHash string) error {
	_, err := s.db.Exec(s.rebind("INSERT INTO users (username, password_hash) VALUES (?, ?)"), username, passwordHash)
	return err
}

func (s *SQLStore) GetUserByUsername(username string) (int, string, error) {
	var id int
	var hash string
	err := s.db.QueryRow(s.rebind("SELECT id, password_hash FROM users WHERE username = ?"), username).Scan(&id, &hash)
	return id, hash, err
}

func (s *SQLStore) GetUserID(username string) (int, error) {
	var id int
	err := s.db.QueryRow(s.rebind("SELECT id FROM users WHERE username = ?"), username).Scan(&id)
	return id, err
}

// Notebook functions
func (s *SQLStore) CreateNotebook(userID int, name string) (int64, error) {
	if s.dbType == Postgres {
		var id int64
		err := s.db.QueryRow(s.rebind("INSERT INTO notebooks (user_id, name, created_at) VALUES (?, ?, ?) RETURNING id"), userID, name, time.Now()).Scan(&id)
		return id, err
	}
	result, err := s.db.Exec(s.rebind("INSERT INTO notebooks (user_id, name, created_at) VALUES (?, ?, ?)"), userID, name, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLStore) CreateDefaultNotebook(userID int) (int64, error) {
	return s.CreateNotebook(userID, "Default")
}

func (s *SQLStore) GetNotebooks(userID int) ([]models.Notebook, error) {
	rows, err := s.db.Query(s.rebind("SELECT id, name, created_at FROM notebooks WHERE user_id = ? ORDER BY created_at ASC"), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notebooks []models.Notebook
	for rows.Next() {
		var nb models.Notebook
		nb.UserID = userID
		if err := rows.Scan(&nb.ID, &nb.Name, &nb.CreatedAt); err != nil {
			log.Printf("Error scanning notebook: %v", err)
			continue
		}
		notebooks = append(notebooks, nb)
	}
	return notebooks, nil
}

func (s *SQLStore) GetNotebookByName(userID int, name string) (int, error) {
	var id int
	err := s.db.QueryRow(s.rebind("SELECT id FROM notebooks WHERE user_id = ? AND name = ?"), userID, name).Scan(&id)
	return id, err
}

func (s *SQLStore) DeleteNotebook(notebookID, userID int) error {
	result, err := s.db.Exec(s.rebind("DELETE FROM notebooks WHERE id = ? AND user_id = ?"), notebookID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	// Also delete notes in this notebook
	s.db.Exec(s.rebind("DELETE FROM notes WHERE notebook_id = ?"), notebookID)
	return nil
}

// Note functions
func (s *SQLStore) CreateNote(userID, notebookID int, content string) error {
	_, err := s.db.Exec(s.rebind("INSERT INTO notes (user_id, notebook_id, content, created_at) VALUES (?, ?, ?, ?)"), userID, notebookID, content, time.Now())
	return err
}

func (s *SQLStore) GetNotes(userID, notebookID int) ([]models.Note, error) {
	rows, err := s.db.Query(s.rebind("SELECT id, content, created_at FROM notes WHERE user_id = ? AND notebook_id = ? ORDER BY created_at DESC"), userID, notebookID)
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

func (s *SQLStore) GetNotesByTimeRange(userID, notebookID int, start, end time.Time) ([]models.Note, error) {
	rows, err := s.db.Query(s.rebind("SELECT content, created_at FROM notes WHERE user_id = ? AND notebook_id = ? AND created_at >= ? AND created_at <= ? ORDER BY created_at DESC"), userID, notebookID, start, end)
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

func (s *SQLStore) UpdateNote(noteID, userID int, content string) error {
	result, err := s.db.Exec(s.rebind("UPDATE notes SET content = ? WHERE id = ? AND user_id = ?"), content, noteID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLStore) DeleteNote(noteID, userID int) error {
	result, err := s.db.Exec(s.rebind("DELETE FROM notes WHERE id = ? AND user_id = ?"), noteID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Note Image functions
func (s *SQLStore) CreateNoteImage(noteID int, filename string) (int64, error) {
	if s.dbType == Postgres {
		var id int64
		err := s.db.QueryRow(s.rebind("INSERT INTO note_images (note_id, filename, created_at) VALUES (?, ?, ?) RETURNING id"), noteID, filename, time.Now()).Scan(&id)
		return id, err
	}
	result, err := s.db.Exec(s.rebind("INSERT INTO note_images (note_id, filename, created_at) VALUES (?, ?, ?)"), noteID, filename, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLStore) GetNoteImages(noteID int) ([]models.NoteImage, error) {
	rows, err := s.db.Query(s.rebind("SELECT id, filename, created_at FROM note_images WHERE note_id = ? ORDER BY created_at ASC"), noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.NoteImage
	for rows.Next() {
		var img models.NoteImage
		img.NoteID = noteID
		if err := rows.Scan(&img.ID, &img.Filename, &img.CreatedAt); err != nil {
			continue
		}
		images = append(images, img)
	}
	return images, nil
}

func (s *SQLStore) DeleteNoteImage(imageID int) (string, error) {
	var filename string
	err := s.db.QueryRow(s.rebind("SELECT filename FROM note_images WHERE id = ?"), imageID).Scan(&filename)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(s.rebind("DELETE FROM note_images WHERE id = ?"), imageID)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func (s *SQLStore) GetNoteImageWithOwner(imageID, userID int) (string, error) {
	var filename string
	query := `SELECT ni.filename FROM note_images ni 
	          JOIN notes n ON ni.note_id = n.id 
	          WHERE ni.id = ? AND n.user_id = ?`
	err := s.db.QueryRow(s.rebind(query), imageID, userID).Scan(&filename)
	return filename, err
}

func (s *SQLStore) GetNoteImagesByNoteIDs(noteIDs []int) (map[int][]models.NoteImage, error) {
	if len(noteIDs) == 0 {
		return make(map[int][]models.NoteImage), nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(noteIDs))
	args := make([]interface{}, len(noteIDs))
	for i, id := range noteIDs {
		if s.dbType == Postgres {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		} else {
			placeholders[i] = "?"
		}
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id, note_id, filename, created_at FROM note_images WHERE note_id IN (%s) ORDER BY created_at ASC", strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]models.NoteImage)
	for rows.Next() {
		var img models.NoteImage
		if err := rows.Scan(&img.ID, &img.NoteID, &img.Filename, &img.CreatedAt); err != nil {
			continue
		}
		result[img.NoteID] = append(result[img.NoteID], img)
	}
	return result, nil
}
