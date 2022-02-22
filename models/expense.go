package models

import "time"

type ExpenseList struct {
	Expenses []Expense `json:"expenses"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	Total    int32     `json:"total"`
}

type ExpenseFilter struct {
	MinDate   string `json:"min_date"`
	MaxDate   string `json:"max_date"`
	Expense   `json:"expense"`
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
}

type Expense struct {
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Id                  string    `json:"id"`
	UserId              string    `json:"user_id"`
	CategoryId          string    `json:"category_id"`
	CategoryName        string    `json:"category_name"`
	CategoryDescription string    `json:"category_description"`
	ProductId           string    `json:"product_id"`
	ProductName         string    `json:"product_name"`
	ProductDescription  string    `json:"product_description"`
	Date                string    `json:"date"`
	Currency            string    `json:"currency"`
	Amount              float64   `json:"amount"`
}

type ExpenseReport struct {
	ReportsIdr []CategoryTotalReport `json:"reports_idr"`
	ReportsUsd []CategoryTotalReport `json:"reports_usd"`
	TotalIdr   float64               `json:"total_idr"`
	TotalUsd   float64               `json:"total_usd"`
}

type CategoryTotalReport struct {
	Id    string  `json:"id"`
	Name  string  `json:"name"`
	Total float64 `json:"total"`
}

type UpsertExpenseRequest struct {
	Data []Expense `json:"data"`
}
