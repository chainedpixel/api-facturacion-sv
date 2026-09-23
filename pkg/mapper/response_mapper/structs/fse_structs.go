package structs

// FSEDTEResponse response structure for FSE according to the Ministry of Finance format
type FSEDTEResponse struct {
	Identificacion  DTEIdentification  `json:"identificacion"`
	Emisor          FSEIssuer          `json:"emisor"`
	Receptor        FSESubjectExcluded `json:"receptor"`
	CuerpoDocumento []FSEItemResponse  `json:"cuerpoDocumento"`
	Resumen         FSESummaryResponse `json:"resumen"`
	Apendice        *[]DTEApendice     `json:"apendice"`
}

// FSEIssuer specific structure for FSE issuer according to schema (excludes tipoEstablecimiento and nombreComercial)
type FSEIssuer struct {
	NIT             string     `json:"nit"`
	NRC             string     `json:"nrc"`
	Nombre          string     `json:"nombre"`
	CodActividad    *string    `json:"codActividad"`
	DescActividad   *string    `json:"descActividad"`
	Direccion       DTEAddress `json:"direccion"`
	Telefono        string     `json:"telefono"`
	Correo          string     `json:"correo"`
	CodEstableMH    *string    `json:"codEstableMH"`
	CodEstable      *string    `json:"codEstable"`
	CodPuntoVentaMH *string    `json:"codPuntoVentaMH"`
	CodPuntoVenta   *string    `json:"codPuntoVenta"`
}

// FSESubjectExcluded structure for the excluded subject according to the FSE schema
type FSESubjectExcluded struct {
	TipoDocumento string      `json:"tipoDocumento"`
	NumDocumento  string      `json:"numDocumento"`
	Nombre        string      `json:"nombre"`
	CodActividad  *string     `json:"codActividad"`
	DescActividad *string     `json:"descActividad"`
	Direccion     *DTEAddress `json:"direccion"`
	Telefono      *string     `json:"telefono"`
	Correo        *string     `json:"correo"`
}

// FSEItemResponse structure for FSE items according to the MH schema
type FSEItemResponse struct {
	NumItem     int     `json:"numItem"`
	TipoItem    int     `json:"tipoItem"`
	Cantidad    float64 `json:"cantidad"`
	Codigo      *string `json:"codigo"`
	UniMedida   int     `json:"uniMedida"`
	Descripcion string  `json:"descripcion"`
	PrecioUni   float64 `json:"precioUni"`
	MontoDescu  float64 `json:"montoDescu"`
	Compra      float64 `json:"compra"`
}

// FSESummaryResponse structure for FSE summary according to the MH schema
type FSESummaryResponse struct {
	TotalCompra        float64       `json:"totalCompra"`
	Descu              float64       `json:"descu"`
	TotalDescu         float64       `json:"totalDescu"`
	SubTotal           float64       `json:"subTotal"`
	ReteRenta          float64       `json:"reteRenta"`
	TotalPagar         float64       `json:"totalPagar"`
	TotalLetras        string        `json:"totalLetras"`
	CondicionOperacion int           `json:"condicionOperacion"`
	Pagos              *[]DTEPayment `json:"pagos"`
	Observaciones      *string       `json:"observaciones"`
}
