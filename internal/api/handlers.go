package api

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tracky/internal/auth"
	"tracky/internal/models"
	"tracky/internal/store"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"
)

const maxImageDimension = 1920 // Max width or height
const jpegQuality = 85         // JPEG compression quality (1-100)

// compressImage resizes and compresses an image, returning the processed image data
func compressImage(file io.Reader, ext string) (image.Image, string, error) {
	var img image.Image
	var err error

	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		// For unsupported formats (gif, webp), we can't compress - return nil to skip compression
		return nil, ext, nil
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %v", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions if resizing needed
	if width > maxImageDimension || height > maxImageDimension {
		var newWidth, newHeight int
		if width > height {
			newWidth = maxImageDimension
			newHeight = int(float64(height) * float64(maxImageDimension) / float64(width))
		} else {
			newHeight = maxImageDimension
			newWidth = int(float64(width) * float64(maxImageDimension) / float64(height))
		}

		// Resize image
		resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)
		img = resized
	}

	// Always save as JPEG for better compression
	return img, ".jpg", nil
}

// Handlers holds dependencies for API handlers
type Handlers struct {
	Store store.Store
}

// NewHandlers creates a new Handlers instance
func NewHandlers(s store.Store) *Handlers {
	return &Handlers{Store: s}
}

func (h *Handlers) SignupHandler(w http.ResponseWriter, r *http.Request) {
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

	err = h.Store.CreateUser(u.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}

	// Get user ID and create default notebook
	userID, _, _ := h.Store.GetUserByUsername(u.Username)
	h.Store.CreateDefaultNotebook(userID)

	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, hash, err := h.Store.GetUserByUsername(u.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(u.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Ensure user has at least one notebook (for existing users)
	notebooks, _ := h.Store.GetNotebooks(id)
	if len(notebooks) == 0 {
		h.Store.CreateDefaultNotebook(id)
	}

	// Set signed auth cookie
	auth.SetAuthCookie(w, id)

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	auth.ClearAuthCookie(w)
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) NotebooksHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		notebooks, err := h.Store.GetNotebooks(userID)
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
		id, err := h.Store.CreateNotebook(userID, nb.Name)
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
		err = h.Store.DeleteNotebook(notebookID, userID)
		if err != nil {
			http.Error(w, "Notebook not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) NotesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
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
		notes, err := h.Store.GetNotes(userID, notebookID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		// Get images for all notes
		if len(notes) > 0 {
			noteIDs := make([]int, len(notes))
			for i, n := range notes {
				noteIDs[i] = n.ID
			}
			imageMap, _ := h.Store.GetNoteImagesByNoteIDs(noteIDs)
			for i := range notes {
				notes[i].Images = imageMap[notes[i].ID]
			}
		}
		json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var n models.Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		err := h.Store.CreateNote(userID, notebookID, n.Content)
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
		err = h.Store.UpdateNote(noteID, userID, n.Content)
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
		// Delete associated images from filesystem
		images, _ := h.Store.GetNoteImages(noteID)
		for _, img := range images {
			os.Remove(filepath.Join("uploads", img.Filename))
		}
		err = h.Store.DeleteNote(noteID, userID)
		if err != nil {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) ImagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Parse multipart form (max 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		noteID, err := strconv.Atoi(r.FormValue("note_id"))
		if err != nil {
			http.Error(w, "Invalid note ID", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "No image provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			http.Error(w, "Invalid file type", http.StatusBadRequest)
			return
		}

		// Create uploads directory if it doesn't exist
		if err := os.MkdirAll("uploads", 0755); err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Try to compress the image
		img, newExt, err := compressImage(file, ext)
		if err != nil {
			http.Error(w, "Failed to process image", http.StatusBadRequest)
			return
		}

		// Use new extension if compression was applied
		if newExt != "" {
			ext = newExt
		}

		// Generate unique filename
		filename := fmt.Sprintf("%d_%d_%d%s", userID, noteID, time.Now().UnixNano(), ext)
		fpath := filepath.Join("uploads", filename)

		// Save file
		dst, err := os.Create(fpath)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if img != nil {
			// Save compressed image as JPEG
			if err := jpeg.Encode(dst, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}
		} else {
			// For unsupported formats (gif, webp), save as-is
			file.Seek(0, 0) // Reset file position
			if _, err := io.Copy(dst, file); err != nil {
				http.Error(w, "Server error", http.StatusInternalServerError)
				return
			}
		}

		// Save to database
		imageID, err := h.Store.CreateNoteImage(noteID, filename)
		if err != nil {
			os.Remove(fpath)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       imageID,
			"filename": filename,
		})

	case http.MethodDelete:
		imageID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid image ID", http.StatusBadRequest)
			return
		}

		filename, err := h.Store.DeleteNoteImage(imageID)
		if err != nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}

		// Delete file from filesystem
		os.Remove(filepath.Join("uploads", filename))
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	_ = userID // Used for authorization context
}

// ServeImageHandler serves images with ownership check
func (h *Handlers) ServeImageHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract image ID from path: /uploads/{id}
	path := strings.TrimPrefix(r.URL.Path, "/uploads/")
	imageID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Check ownership and get filename
	filename, err := h.Store.GetNoteImageWithOwner(imageID, userID)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filepath.Join("uploads", filename))
}

// AnalysisHandler handles note analysis requests using Gemini
func (h *Handlers) AnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		NotebookID int                  `json:"notebook_id"`
		Question   string               `json:"question"`
		History    []models.ChatMessage `json:"history"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "Question is required", http.StatusBadRequest)
		return
	}

	// Fetch all notes from the notebook
	notes, err := h.Store.GetNotes(userID, req.NotebookID)
	if err != nil {
		http.Error(w, "Failed to fetch notes", http.StatusInternalServerError)
		return
	}

	if len(notes) == 0 {
		json.NewEncoder(w).Encode(map[string]string{
			"answer": "There are no notes in this notebook to analyze.",
		})
		return
	}

	// Call Gemini API
	// Call Gemini API
	answer, err := AnalyzeNotes(notes, req.Question, req.History)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"answer": answer,
	})
}
