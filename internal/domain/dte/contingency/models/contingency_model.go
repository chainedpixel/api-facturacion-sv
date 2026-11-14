package models

// ContingencyEvent represents the contingency event structure sent to Hacienda.
type ContingencyEvent struct {
	Identification ContingencyIdentification `json:"identificacion"`
	Issuer         ContingencyIssuer         `json:"emisor"`
	DTEDetails     []DTEDetail               `json:"detalleDTE"`
	Reason         ContingencyReason         `json:"motivo"`
}

// ContingencyIdentification contains transmission identification metadata.
type ContingencyIdentification struct {
	Version          int    `json:"version"`
	Ambient          string `json:"ambiente"`
	GenerationCode   string `json:"codigoGeneracion"`
	TransmissionDate string `json:"fTransmision"`
	TransmissionTime string `json:"hTransmision"`
}

// ContingencyIssuer contains contingency emitter information.
type ContingencyIssuer struct {
	NIT                  string  `json:"nit"`
	Name                 string  `json:"nombre"`
	ResponsibleName      string  `json:"nombreResponsable"`
	ResponsibleDocType   string  `json:"tipoDocResponsable"`
	ResponsibleDocNumber string  `json:"numeroDocResponsable"`
	EstablishmentType    string  `json:"tipoEstablecimiento"`
	Phone                string  `json:"telefono"`
	Email                string  `json:"correo"`
	EstablishmentCodeMH  *string `json:"codEstableMH"`
	POSCode              *string `json:"codPuntoVentaMH"`
}

// DTEDetail represents a single DTE included in the contingency.
type DTEDetail struct {
	ItemNumber     int    `json:"noItem"`
	GenerationCode string `json:"codigoGeneracion"`
	DocumentType   string `json:"tipoDoc"`
}

// ContingencyReason defines timeframe and reason for the contingency.
type ContingencyReason struct {
	StartDate         string  `json:"fInicio"`
	EndDate           string  `json:"fFin"`
	StartTime         string  `json:"hInicio"`
	EndTime           string  `json:"hFin"`
	ContingencyType   int8    `json:"tipoContingencia"`
	ContingencyReason *string `json:"motivoContingencia"`
}

// HaciendaContingencyRequest wraps the transmission request payload for Hacienda.
type HaciendaContingencyRequest struct {
	NIT      string `json:"nit"`
	Document string `json:"documento"`
}
