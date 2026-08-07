package hacienda_error

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
)

type HTTPResponseError struct {
	StatusCode int
	Body       []byte
	URL        string
	Method     string
}

func (e *HTTPResponseError) Error() string {
	return fmt.Sprintf("HTTP %d: %s %s - %s",
		e.StatusCode, e.Method, e.URL, string(e.Body))
}

type HaciendaResponseError struct {
	Version        int      `json:"version"`
	Ambient        string   `json:"ambiente"`
	VersionApp     int      `json:"versionApp"`
	Status         string   `json:"estado"`
	StatusCode     int      `json:"codigoEstado"`
	Code           string   `json:"codigoMsg"`
	Description    string   `json:"descripcionMsg"`
	Classification string   `json:"clasificaMsg"`
	Observations   []string `json:"observaciones"`
	ProcessedAt    string   `json:"fhProcesamiento"`
}

func NewHaciendaError(resp *models.HaciendaResponse, httpCode int) *HaciendaResponseError {
	return &HaciendaResponseError{
		Version:        resp.Version,
		Ambient:        resp.Ambient,
		VersionApp:     resp.VersionApp,
		Status:         resp.Status,
		Code:           resp.MessageCode,
		Description:    resp.DescriptionMessage,
		Classification: resp.ClassifyMessage,
		ProcessedAt:    resp.ProcessingDate,
		Observations:   resp.Observations,
		StatusCode:     httpCode,
	}
}

func (e *HaciendaResponseError) Error() string {
	return fmt.Sprintf("Hacienda said: [%s] %s", e.Code, e.Description)
}

type NetworkError struct {
	OriginalError error
	Operation     string
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("Network error during %s: %v", e.Operation, e.OriginalError)
}
