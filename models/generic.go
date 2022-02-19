package models

type RowError struct {
	Message string `json:"message"`
	Row     int    `json:"row"`
}

type BatchDeleteRequest struct {
	Data []string `json:"data"`
}

type RowResponseError struct {
	Message string     `json:"message"`
	Detail  []RowError `json:"detail"`
}
