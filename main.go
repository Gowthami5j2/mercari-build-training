package main

import (
	"database/sql"
	//"encoding/json"
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
	dbPath := "./db/mercari.sqlite3"

	// Create the database directory if it doesn't exist
	os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create tables if they don't exist
	createTables()
}

// Create tables
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
		log.Fatal("Failed to create tables:", err)
	}
}

// Add an item
func addItem(c *gin.Context) {
	name := c.PostForm("name")
	category := c.PostForm("category")

	// Validate required fields
	if name == "" || category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and category are required"})
		return
	}

	// Handle image upload
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}

	// Save the image
	imagePath := "./images/" + file.Filename
	c.SaveUploadedFile(file, imagePath)

	// Check if category exists, otherwise insert it
	var categoryID int
	err = db.QueryRow("SELECT id FROM categories WHERE name = ?", category).Scan(&categoryID)
	if err == sql.ErrNoRows {
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

	// Insert item into database
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

// Search items
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
		rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func main() {
	initDB()
	r := gin.Default()

	// Enable CORS
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

	// Routes
	r.POST("/items", addItem)
	r.GET("/items/:id", getItem)
	r.GET("/search", searchItems)

	// Run server
	port := "8080"
	fmt.Println("Server running on port " + port)
	r.Run(":" + port)
}
