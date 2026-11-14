package models

type HaciendaRequest struct {
	Ambient        string      `json:"ambiente"`
	SendID         int         `json:"idEnvio"`
	Version        int         `json:"version"`
	Document       interface{} `json:"documento"`
	DTEType        string      `json:"tipoDte,omitempty"`
	GenerationCode string      `json:"codigoGeneracion,omitempty"`
	URL            string      `json:"-"`
}
