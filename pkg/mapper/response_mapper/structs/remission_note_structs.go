package structs

// MHRemissionNote represents the complete structure of a Remission Note for the Ministry of Finance
type MHRemissionNote struct {
	Identification   *DTEIdentification       `json:"identificacion"`
	RelatedDocuments []DTERelatedDocument     `json:"documentoRelacionado"`
	Issuer           DTEIssuer                `json:"emisor"`
	Receiver         *MHRemissionNoteReceiver `json:"receptor"`
	ThirdPartySale   *DTEThirdPartySale       `json:"ventaTercero"`
	DocumentBody     []*MHRemissionNoteItem   `json:"cuerpoDocumento"`
	Summary          *MHRemissionNoteSummary  `json:"resumen"`
	Appendix         []DTEApendice            `json:"apendice"`
}

// MHRemissionNoteReceiver represents the specific receiver for Remission Notes
type MHRemissionNoteReceiver struct {
	DocumentType   string      `json:"tipoDocumento"`
	DocumentNumber string      `json:"numDocumento"`
	NRC            *string     `json:"nrc"`
	Name           string      `json:"nombre"`
	ActivityCode   *string     `json:"codActividad"`
	ActivityDesc   *string     `json:"descActividad"`
	CommercialName *string     `json:"nombreComercial"`
	Address        *DTEAddress `json:"direccion"`
	Phone          *string     `json:"telefono"`
	Email          string      `json:"correo"`
	BienTitulo     string      `json:"bienTitulo"`
}

// MHRemissionNoteItem represents an item of a Remission Note for the MH
type MHRemissionNoteItem struct {
	ItemNumber     int      `json:"numItem"`
	ItemType       int      `json:"tipoItem"`
	DocumentNumber *string  `json:"numeroDocumento"`
	Code           *string  `json:"codigo"`
	TributeCode    *string  `json:"codTributo"`
	Description    string   `json:"descripcion"`
	Quantity       float64  `json:"cantidad"`
	UnitMeasure    int      `json:"uniMedida"`
	UnitPrice      float64  `json:"precioUni"`
	DiscountAmount float64  `json:"montoDescu"`
	NonSubjectSale float64  `json:"ventaNoSuj"`
	ExemptSale     float64  `json:"ventaExenta"`
	TaxedSale      float64  `json:"ventaGravada"`
	Tributes       []string `json:"tributos"`
}

// MHRemissionNoteSummary represents the summary of a Remission Note for the MH
type MHRemissionNoteSummary struct {
	NonSubjectTotal    float64  `json:"totalNoSuj"`
	ExemptTotal        float64  `json:"totalExenta"`
	TaxedTotal         float64  `json:"totalGravada"`
	SubtotalSales      float64  `json:"subTotalVentas"`
	NonSubjectDiscount float64  `json:"descuNoSuj"`
	ExemptDiscount     float64  `json:"descuExenta"`
	TaxedDiscount      float64  `json:"descuGravada"`
	DiscountPercent    *float64 `json:"porcentajeDescuento"`
	TotalDiscount      float64  `json:"totalDescu"`
	Tributes           []DTETax `json:"tributos"`
	Subtotal           float64  `json:"subTotal"`
	TotalAmount        float64  `json:"montoTotalOperacion"`
	AmountInWords      string   `json:"totalLetras"`
	Observaciones      *string  `json:"observaciones"`
}
