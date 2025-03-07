package main

import (
	"fmt"
	"log"
	"net/http"

	"my-go-app/internal/handlers"
	"my-go-app/internal/repository"
)

func main() {
	itemRepo := repository.NewItemRepository()
	h := handlers.NewHandlers(itemRepo)

	http.HandleFunc("/upload", h.UploadImage)
	http.HandleFunc("/items", h.AddItem)
	http.HandleFunc("/items/all", h.GetItems)

	fs := http.FileServer(http.Dir("images"))
	http.Handle("/images/", http.StripPrefix("/images/", fs))

	fmt.Println("Server running on port 9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
