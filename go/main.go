package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Item struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImagePath string `json:"image_path"`
}

type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAll(ctx context.Context) ([]Item, error)
	GetByID(ctx context.Context, id int) (*Item, error)
}

type itemRepository struct {
	fileName string
}

func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	var items []Item

	file, err := os.OpenFile(i.fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&items); err != nil && err != io.EOF {
		return err
	}

	items = append(items, *item)

data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(i.fileName, data, 0666)
}

func (i *itemRepository) GetAll(ctx context.Context) ([]Item, error) {
	var items []Item

	file, err := os.Open(i.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&items); err != nil && err != io.EOF {
		return nil, err
	}

	return items, nil
}

func (i *itemRepository) GetByID(ctx context.Context, id int) (*Item, error) {
	items, err := i.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	if id < 0 || id >= len(items) {
		return nil, fmt.Errorf("item not found")
	}

	return &items[id], nil
}

type Handlers struct {
	itemRepo ItemRepository
}

func (s *Handlers) UploadImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

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
	if err := os.MkdirAll(imageDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	imagePath := filepath.Join(imageDir, imageFilename)
	if err := os.WriteFile(imagePath, fileBytes, 0666); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"image_path": imagePath})
}

func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.itemRepo.Insert(context.Background(), &item); err != nil {
		http.Error(w, "Failed to store item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Item stored successfully!", "image_path": item.ImagePath})
}

func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.itemRepo.GetAll(context.Background())
	if err != nil {
		http.Error(w, "Failed to retrieve items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Handlers) GetItemByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := s.itemRepo.GetByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func main() {
	itemRepo := NewItemRepository()
	h := &Handlers{itemRepo: itemRepo}

	http.HandleFunc("/upload", h.UploadImage)
	http.HandleFunc("/items", h.AddItem)
	http.HandleFunc("/items/all", h.GetItems)
	http.HandleFunc("/items/", h.GetItemByID) 

	fs := http.FileServer(http.Dir("images"))
	http.Handle("/images/", http.StripPrefix("/images/", fs))

	fmt.Println("Server running on port 9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}

