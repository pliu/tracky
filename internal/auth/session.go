package auth

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// Session store (in-memory for simplicity)
var (
	sessions = make(map[string]int) // token -> user_id
	sessMu   sync.RWMutex
)

func CreateSession(userID int) string {
	token := uuid.New().String()
	sessMu.Lock()
	sessions[token] = userID
	sessMu.Unlock()
	return token
}

func GetUserID(token string) (int, bool) {
	sessMu.RLock()
	defer sessMu.RUnlock()
	id, ok := sessions[token]
	return id, ok
}

func DeleteSession(token string) {
	sessMu.Lock()
	delete(sessions, token)
	sessMu.Unlock()
}

func GetUserIDFromRequest(r *http.Request) (int, error) {
	c, err := r.Cookie("session_token")
	if err != nil {
		return 0, err
	}
	id, ok := GetUserID(c.Value)
	if !ok {
		return 0, fmt.Errorf("invalid session")
	}
	return id, nil
}
