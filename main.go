package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
const maxUploadSize = 10 << 20 // 10MB

// Item struct
type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

// Initializes Database
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./db/mercari.sqlite3")
	if err != nil {
		log.Fatal(err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		image_name TEXT NOT NULL,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	createUploadsDir()
	fmt.Println("Database initialized!")
}

// Creates uploads directory if not exists
func createUploadsDir() {
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		os.Mkdir("uploads", os.ModePerm)
	}
}

// Saves image and returns its hashed filename
func saveImage(file io.Reader) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	hashString := hex.EncodeToString(hash[:])

	imagePath := filepath.Join("uploads", hashString+".jpg")
	err = os.WriteFile(imagePath, data, 0644)
	if err != nil {
		return "", err
	}

	return hashString + ".jpg", nil
}

// Handles fetching items
func getItemsHandler(w http.ResponseWriter, r *http.Request) {
	stmt, err := db.Prepare(`
		SELECT items.id, items.name, categories.name, items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}

// Handles searching items
func searchItemsHandler(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "Missing keyword parameter", http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare(`
		SELECT items.name, categories.name 
		FROM items
		JOIN categories ON items.category_id = categories.id
		WHERE items.name LIKE ?
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query("%" + keyword + "%")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}

// Handles adding new items
func postItemsHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http.Error(w, "File size too large", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	categoryID := r.FormValue("category_id")
	categoryIDInt, err := strconv.Atoi(categoryID)
	if err != nil {
		http.Error(w, "Invalid category_id", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	hashFilename, err := saveImage(file)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)", name, categoryIDInt, hashFilename)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tx.Commit()

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Item added successfully!")
}

func main() {
	initDB()
	http.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getItemsHandler(w, r)
		case "POST":
			postItemsHandler(w, r)
		default:
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/search", searchItemsHandler)

	fmt.Println("Server is running on port 9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
