package models

type TransmitResult struct {
	Status         string
	ReceptionStamp *string
	ProcessingDate string
	MessageCode    string
	MessageDesc    string
	Observations   []string
	// SignedDocument es el JWS transmitido (firmaElectronica del Archivo DTE, Anexo I regla 4.1)
	SignedDocument string
	// HaciendaResponse es la respuesta completa de Hacienda en JSON, para consulta y entrega posteriores
	HaciendaResponse string
}
