package repository

import (
	"context"
	"encoding/json"
	"os"
)

type Item struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImagePath string `json:"image_path"`
}

type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAll(ctx context.Context) ([]Item, error)
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
	err = decoder.Decode(&items)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	items = append(items, *item)

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(i.fileName, data, 0666)
	return err
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
	err = decoder.Decode(&items)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return items, nil
}
