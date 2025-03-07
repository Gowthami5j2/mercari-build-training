package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"my-go-app/internal/repository"
)

type Handlers struct {
	ItemRepo repository.ItemRepository
}

func NewHandlers(repo repository.ItemRepository) *Handlers {
	return &Handlers{ItemRepo: repo}
}

func (h *Handlers) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	hash := sha256.Sum256(fileBytes)
	hashStr := hex.EncodeToString(hash[:])
	imageFilename := hashStr + ".jpg"

	imageDir := "images"
	if _, err := os.Stat(imageDir); os.IsNotExist(err) {
		os.Mkdir(imageDir, os.ModePerm)
	}

	imagePath := filepath.Join(imageDir, imageFilename)
	err = os.WriteFile(imagePath, fileBytes, 0666)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"image_path": imagePath})
}

func (h *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var item repository.Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.ItemRepo.Insert(context.Background(), &item)
	if err != nil {
		http.Error(w, "Failed to store item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Item stored successfully!", "image_path": item.ImagePath})
}

func (h *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.ItemRepo.GetAll(context.Background())
	if err != nil {
		http.Error(w, "Failed to retrieve items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
