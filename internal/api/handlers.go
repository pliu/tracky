package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tracky/internal/auth"
	"tracky/internal/models"
	"tracky/internal/store"

	"golang.org/x/crypto/bcrypt"
)

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = store.CreateUser(u.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}

	// Get user ID and create default notebook
	userID, _, _ := store.GetUserByUsername(u.Username)
	store.CreateDefaultNotebook(userID)

	w.WriteHeader(http.StatusCreated)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, hash, err := store.GetUserByUsername(u.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(u.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Ensure user has at least one notebook (for existing users)
	notebooks, _ := store.GetNotebooks(id)
	if len(notebooks) == 0 {
		nbID, _ := store.CreateDefaultNotebook(id)
		store.MigrateOrphanedNotes(id, nbID)
	}

	token := auth.CreateSession(id)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	auth.DeleteSession(c.Value)

	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}

func NotebooksHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		notebooks, err := store.GetNotebooks(userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(notebooks)

	case http.MethodPost:
		var nb models.Notebook
		if err := json.NewDecoder(r.Body).Decode(&nb); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		id, err := store.CreateNotebook(userID, nb.Name)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]int64{"id": id})

	case http.MethodDelete:
		notebookID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid notebook ID", http.StatusBadRequest)
			return
		}
		err = store.DeleteNotebook(notebookID, userID)
		if err != nil {
			http.Error(w, "Notebook not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func NotesHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notebookID, err := strconv.Atoi(r.URL.Query().Get("notebook_id"))
	if err != nil && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		http.Error(w, "Invalid notebook ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		notes, err := store.GetNotes(userID, notebookID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var n models.Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		err := store.CreateNote(userID, notebookID, n.Content)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	case http.MethodPut:
		noteID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid note ID", http.StatusBadRequest)
			return
		}
		var n models.Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		err = store.UpdateNote(noteID, userID, n.Content)
		if err != nil {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		noteID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid note ID", http.StatusBadRequest)
			return
		}
		err = store.DeleteNote(noteID, userID)
		if err != nil {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
