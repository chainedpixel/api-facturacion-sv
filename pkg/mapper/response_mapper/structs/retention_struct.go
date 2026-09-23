package structs

type RetentionDTEResponse struct {
	Identificacion  *DTEIdentification `json:"identificacion"`
	Resumen         *RetentionSummary  `json:"resumen"`
	Emisor          RetentionIssuer    `json:"emisor"`
	Receptor        DTEReceiver        `json:"receptor"`
	CuerpoDocumento []RetentionItem    `json:"cuerpoDocumento"`
	Apendice        []DTEApendice      `json:"apendice"`
}

type RetentionSummary struct {
	TotalSujRetencion float64 `json:"totalSujetoRetencion"`
	TotalIva          float64 `json:"totalIva"`
	TotalIvaRetenido  float64 `json:"totalIvaRetenido"`
	TotalLetras       string  `json:"totalLetras"`
	Observaciones     *string `json:"observaciones"`
}

type RetentionItem struct {
	NumItem            int     `json:"numItem"`
	TipoDTE            string  `json:"tipoDte"`
	TipoGeneracion     int     `json:"tipoGeneracion"`
	NumDoc             string  `json:"numeroDocumento"`
	FechaEmision       string  `json:"fechaEmision"`
	MontoSujetoGravado float64 `json:"montoSujetoGrav"`
	CodigoRetencionMH  string  `json:"codigoRetencionMH"`
	IvaRetenido        float64 `json:"ivaRetenido"`
	Descripcion        string  `json:"descripcion"`
}

type RetentionIssuer struct {
	NIT             string     `json:"nit"`
	NRC             string     `json:"nrc"`
	Nombre          string     `json:"nombre"`
	CodActividad    string     `json:"codActividad"`
	DescActividad   string     `json:"descActividad"`
	NombreComercial *string    `json:"nombreComercial"`
	Direccion       DTEAddress `json:"direccion"`
	CodEstable      *string    `json:"codEstable"`
	CodPuntoVenta   *string    `json:"codPuntoVenta"`
	Telefono        string     `json:"telefono"`
	Correo          string     `json:"correo"`
}
