package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Notebook struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Note struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	NotebookID int       `json:"notebook_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}
