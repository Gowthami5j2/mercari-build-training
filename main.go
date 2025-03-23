package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Item represents an item in the database
type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

// Initialize the database
func initDB() {
	var err error

	// Use environment variable for DB path (Docker compatibility)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/db/mercari.sqlite3" // Docker-compatible path
	}

	// Ensure directory exists
	err = os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create DB directory: %v", err)
	}

	// Open SQLite database
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create tables
	createTables()
}

// Create tables if they don't exist
func createTables() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category_id INTEGER NOT NULL,
			image_name TEXT NOT NULL,
			FOREIGN KEY(category_id) REFERENCES categories(id)
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

// Add an item to the database
func addItem(c *gin.Context) {
	name := c.PostForm("name")
	category := c.PostForm("category")

	if name == "" || category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and category are required"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}

	// Save the image in the Docker volume directory
	imagePath := "/app/images/" + file.Filename
	if err := c.SaveUploadedFile(file, imagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	// Handle category
	var categoryID int
	err = db.QueryRow("SELECT id FROM categories WHERE name = ?", category).Scan(&categoryID)
	if err == sql.ErrNoRows {
		// Insert new category if it doesn't exist
		result, err := db.Exec("INSERT INTO categories (name) VALUES (?)", category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert category"})
			return
		}
		categoryID64, _ := result.LastInsertId()
		categoryID = int(categoryID64)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Insert item
	_, err = db.Exec("INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)", name, categoryID, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert item"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Item added successfully"})
}

// Get an item by ID
func getItem(c *gin.Context) {
	id := c.Param("id")
	var item Item
	err := db.QueryRow(`
		SELECT items.id, items.name, categories.name, items.image_name 
		FROM items 
		JOIN categories ON items.category_id = categories.id 
		WHERE items.id = ?`, id).Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Search for items by keyword
func searchItems(c *gin.Context) {
	keyword := c.Query("keyword")
	keyword = strings.TrimSpace(keyword)

	rows, err := db.Query(`
		SELECT items.id, items.name, categories.name, items.image_name 
		FROM items 
		JOIN categories ON items.category_id = categories.id 
		WHERE items.name LIKE ?`, "%"+keyword+"%")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse item"})
			return
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func main() {
	initDB()

	r := gin.Default()

	// Enable CORS for frontend access
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Add "/" route to avoid 404
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to the Mercari API. Use /items, /items/:id, or /search to access data.",
		})
	})

	// Routes
	r.POST("/items", addItem)
	r.GET("/items/:id", getItem)
	r.GET("/search", searchItems)

	// Get port from environment variables (Docker flexibility)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	fmt.Println("Server running on port " + port)
	r.Run(":" + port)
}
