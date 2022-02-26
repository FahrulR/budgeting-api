package models

import "time"

type IncomeList struct {
	Incomes []Income `json:"incomes"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
	Total   int32    `json:"total"`
}

type Income struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Id          string    `json:"id"`
	UserId      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Date        string    `json:"date"`
	Currency    string    `json:"currency"`
	Amount      float64   `json:"amount"`
}

type IncomeFilter struct {
	MinDate   string `json:"min_date"`
	MaxDate   string `json:"max_date"`
	Income    `json:"income"`
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
}

type UpsertIncomeRequest struct {
	Data []Income `json:"data"`
}

type IncomeReport struct {
	TotalIdr float64 `json:"total_idr"`
	TotalUsd float64 `json:"total_usd"`
}
