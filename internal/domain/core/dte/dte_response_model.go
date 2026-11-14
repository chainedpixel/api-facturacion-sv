package dte

import "encoding/json"

type DTEResponse struct {
	ControlNumber  string                 `json:"control_number"`
	GenerationCode string                 `json:"generation_code"`
	ReceptionStamp *string                `json:"reception_stamp"`
	Transmission   string                 `json:"transmission"`
	Status         string                 `json:"status"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	ProcessingDate string                 `json:"processing_date,omitempty"`
	JSONData       map[string]interface{} `json:"json_data"`
}

type DTEListResponse struct {
	Documents  []DTEModelResponse    `json:"documents"`
	Summary    ListSummary           `json:"summary"`
	Pagination DTEPaginationResponse `json:"pagination"`
}

type ListSummary struct {
	Total         int64 `json:"total"`
	Received      int64 `json:"received"`
	Invalid       int64 `json:"invalid"`
	Rejected      int64 `json:"rejected"`
	Pending       int64 `json:"pending"`
	ByContingency int64 `json:"by_contingency"`
	ByNormal      int64 `json:"by_normal"`
}

type DTEPaginationResponse struct {
	TotalPages int `json:"total_pages"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
}

type DTEModelResponse struct {
	Status           string          `json:"status"`
	TransmissionType string          `json:"transmission_type"`
	Document         json.RawMessage `json:"document"`
}
