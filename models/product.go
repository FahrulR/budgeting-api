package models

import "time"

type ProductList struct {
	Products []Product `json:"products"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	Total    int32     `json:"total"`
}

type Product struct {
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Id                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	UserId              string    `json:"user_id"`
	CategoryId          string    `json:"category_id,omitempty"`
	CategoryName        string    `json:"category_name"`
	CategoryDescription string    `json:"category_description"`
}

type UpsertProductRequest struct {
	Data []Product `json:"data"`
}
