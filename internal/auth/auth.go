package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Context key for user ID
type contextKey string

const UserIDKey contextKey = "userID"

// Secret key for signing cookies (should be set via environment variable in production)
var cookieSecret = []byte(getSecretKey())

func getSecretKey() string {
	key := os.Getenv("COOKIE_SECRET")
	if key == "" {
		// Default key for development (DO NOT use in production)
		key = "tracky-dev-secret-key-change-in-prod"
	}
	return key
}

// CreateSignedCookie creates a signed cookie value containing the user ID and expiration
func CreateSignedCookie(userID int) string {
	// Cookie format: userID.expiration.signature
	expiration := time.Now().Add(7 * 24 * time.Hour).Unix() // 7 days
	data := fmt.Sprintf("%d.%d", userID, expiration)
	signature := signData(data)
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s.%s", data, signature)))
}

// ValidateSignedCookie validates the cookie and returns the user ID if valid
func ValidateSignedCookie(cookieValue string) (int, error) {
	decoded, err := base64.URLEncoding.DecodeString(cookieValue)
	if err != nil {
		return 0, fmt.Errorf("invalid cookie encoding")
	}

	parts := strings.Split(string(decoded), ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid cookie format")
	}

	userIDStr, expirationStr, signature := parts[0], parts[1], parts[2]

	// Verify signature
	data := fmt.Sprintf("%s.%s", userIDStr, expirationStr)
	if !verifySignature(data, signature) {
		return 0, fmt.Errorf("invalid signature")
	}

	// Check expiration
	expiration, err := strconv.ParseInt(expirationStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid expiration")
	}
	if time.Now().Unix() > expiration {
		return 0, fmt.Errorf("cookie expired")
	}

	// Parse user ID
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID")
	}

	return userID, nil
}

func signData(data string) string {
	h := hmac.New(sha256.New, cookieSecret)
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func verifySignature(data, signature string) bool {
	expectedSig := signData(data)
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// SetAuthCookie sets the signed auth cookie on the response
func SetAuthCookie(w http.ResponseWriter, userID int) {
	cookieValue := CreateSignedCookie(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

// ClearAuthCookie clears the auth cookie
func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
