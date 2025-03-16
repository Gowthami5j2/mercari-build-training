package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var router *gin.Engine

// Setup before running tests
func setup() {
	gin.SetMode(gin.TestMode)
	initDB()
	router = gin.Default()
	router.POST("/items", addItem)
	router.GET("/items/:id", getItem)
	router.GET("/search", searchItems)
}

func TestAddItem(t *testing.T) {
	setup()

	// Mock request with form data
	body := bytes.NewBufferString("name=TestItem&category=TestCategory")
	req, _ := http.NewRequest("POST", "/items", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code) // Since image is required
}

func TestGetItem_NotFound(t *testing.T) {
	setup()

	req, _ := http.NewRequest("GET", "/items/9999", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestSearchItems_EmptyResult(t *testing.T) {
	setup()

	req, _ := http.NewRequest("GET", "/search?keyword=nonexistent", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var response map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &response)

	assert.Equal(t, http.StatusOK, resp.Code)

	// Check if "items" exists in response and is an array
	items, exists := response["items"]
	if !exists || items == nil {
		items = []interface{}{} // Assign empty array if nil
	}

	assert.Equal(t, 0, len(items.([]interface{})))
}


func TestSearchItems(t *testing.T) {
	setup()

	req, _ := http.NewRequest("GET", "/search?keyword=Test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}
