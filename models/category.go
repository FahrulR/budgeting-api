package models

import "time"

type CategoryList struct {
	Categories []Category `json:"categories"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	Total      int32      `json:"total"`
}

type Category struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserId      string    `json:"user_id"`
}

type UpsertCategoryRequest struct {
	Data []Category `json:"data"`
}
