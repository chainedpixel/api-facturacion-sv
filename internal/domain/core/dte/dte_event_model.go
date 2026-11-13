package dte

import (
	"encoding/json"
	"time"
)

// Tipos de evento que se registran con su evidencia de transmision
const (
	EventInvalidation = "INVALIDATION"
	EventContingency  = "CONTINGENCY"
)

// DTEEvent registra un evento transmitido a Hacienda (invalidacion, contingencia). Normativa v2.0, regla 11:
// los eventos de invalidacion y retorno se entregan al receptor como Archivo DTE, por lo que se conserva la
// estructura firmada (SignedDocument), el sello y la respuesta completa de Hacienda.
type DTEEvent struct {
	GenerationCode   string    `json:"generation_code"`
	BranchID         uint      `json:"branch_id"`
	EventType        string    `json:"event_type"`
	RelatedDTEID     *string   `json:"related_dte_id"`
	ReplacementDTEID *string   `json:"replacement_dte_id"`
	JSONData         string    `json:"-"`
	SignedDocument   string    `json:"-"`
	ReceptionStamp   *string   `json:"reception_stamp"`
	ProcessingDate   string    `json:"processing_date"`
	HaciendaResponse string    `json:"-"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// EventArchiveResponse es lo que se devuelve al consumidor por cada evento: sus datos, la respuesta de
// Hacienda y el Archivo DTE listo para entregar al receptor
type EventArchiveResponse struct {
	DTEEvent
	HaciendaResponse json.RawMessage `json:"hacienda_response,omitempty"`
	Archive          json.RawMessage `json:"archivo_dte,omitempty"`
}
