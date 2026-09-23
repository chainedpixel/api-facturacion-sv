package structs

// InvalidationResponse represents the complete DTE invalidation event payload for Hacienda.
type InvalidationResponse struct {
	Identificacion InvalidationIdentification `json:"identificacion"`
	Emisor         InvalidationIssuer         `json:"emisor"`
	Documento      DocumentResponse           `json:"documento"`
	Motivo         ReasonResponse             `json:"motivo"`
}

// DocumentResponse represents the invalidated DTE reference data.
type DocumentResponse struct {
	TipoDte           string  `json:"tipoDte"`
	CodigoGeneracion  string  `json:"codigoGeneracion"`
	SelloRecibido     string  `json:"selloRecibido"`
	NumeroControl     *string `json:"numeroControl"`
	FecEmi            string  `json:"fecEmi"`
	CodigoGeneracionR *string `json:"codigoGeneracionR"`
	TipoDocumento     *string `json:"tipoDocumento"`
	NumDocumento      *string `json:"numDocumento"`
	Nombre            *string `json:"nombre"`
	Telefono          *string `json:"telefono"`
	Correo            *string `json:"correo"`
}

// ReasonResponse represents the invalidation reason and responsible parties.
type ReasonResponse struct {
	TipoAnulacion     int     `json:"tipoAnulacion"`
	MotivoAnulacion   *string `json:"motivoAnulacion"`
	NombreResponsable string  `json:"nombreResponsable"`
	TipDocResponsable string  `json:"tipDocResponsable"`
	NumDocResponsable string  `json:"numDocResponsable"`
	NombreSolicita    string  `json:"nombreSolicita"`
	TipDocSolicita    string  `json:"tipDocSolicita"`
	NumDocSolicita    string  `json:"numDocSolicita"`
}

// InvalidationIdentification represents identification metadata for an invalidation event.
type InvalidationIdentification struct {
	Version          int     `json:"version"`
	Ambiente         string  `json:"ambiente"`
	CodigoGeneracion string  `json:"codigoGeneracion"`
	FecEmi           string  `json:"fecEmi"`
	HorEmi           string  `json:"horEmi"`
	Fusion           *string `json:"fusion"`
}

// InvalidationIssuer represents issuer data for an invalidation event.
type InvalidationIssuer struct {
	NIT             string  `json:"nit"`
	Nombre          string  `json:"nombre"`
	CodEstableMH    string  `json:"codEstableMH"`
	CodEstable      *string `json:"codEstable"`
	CodPuntoVentaMH string  `json:"codPuntoVentaMH"`
	CodPuntoVenta   *string `json:"codPuntoVenta"`
	Telefono        string  `json:"telefono"`
	Correo          string  `json:"correo"`
}
