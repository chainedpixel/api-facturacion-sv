package structs

type InvoiceDTEResponse struct {
	Identificacion       *DTEIdentification   `json:"identificacion"`
	Resumen              *InvoiceSummary      `json:"resumen"`
	Emisor               DTEIssuer            `json:"emisor"`
	Receptor             InvoiceReceiver      `json:"receptor"`
	CuerpoDocumento      []InvoiceItem        `json:"cuerpoDocumento"`
	DocumentoRelacionado []DTERelatedDocument `json:"documentoRelacionado"`
	OtrosDocumentos      []DTEOtherDocument   `json:"otrosDocumentos"`
	VentaTercero         *DTEThirdPartySale   `json:"ventaTercero"`
	Apendice             []DTEApendice        `json:"apendice"`
}

type InvoiceSummary struct {
	TotalNoSuj          float64      `json:"totalNoSuj"`
	TotalExenta         float64      `json:"totalExenta"`
	TotalGravada        float64      `json:"totalGravada"`
	SubTotalVentas      float64      `json:"subTotalVentas"`
	DescuNoSuj          float64      `json:"descuNoSuj"`
	DescuExenta         float64      `json:"descuExenta"`
	DescuGravada        float64      `json:"descuGravada"`
	PorcentajeDescuento float64      `json:"porcentajeDescuento"`
	TotalDescu          float64      `json:"totalDescu"`
	Tributos            []DTETax     `json:"tributos"`
	SubTotal            float64      `json:"subTotal"`
	ReteRenta           float64      `json:"reteRenta"`
	IvaRete             float64      `json:"ivaRete"`
	IvaPerci            *float64     `json:"ivaPerci,omitempty"`
	MontoTotalOperacion float64      `json:"montoTotalOperacion"`
	TotalNoGravado      float64      `json:"totalNoGravado"`
	TotalPagar          float64      `json:"totalPagar"`
	TotalLetras         string       `json:"totalLetras"`
	TotalIva            float64      `json:"totalIva"`
	SaldoFavor          float64      `json:"saldoFavor"`
	CondicionOperacion  int          `json:"condicionOperacion"`
	Pagos               []DTEPayment `json:"pagos"`
	NumPagoElectronico  *string      `json:"numPagoElectronico"`
	Observaciones       *string      `json:"observaciones"`
}

type InvoiceReceiver struct {
	Nombre        *string     `json:"nombre"`
	TipoDocumento *string     `json:"tipoDocumento"`
	NumDocumento  *string     `json:"numDocumento"`
	NRC           *string     `json:"nrc"`
	CodActividad  *string     `json:"codActividad"`
	DescActividad *string     `json:"descActividad"`
	Direccion     *DTEAddress `json:"direccion"`
	Telefono      *string     `json:"telefono"`
	Correo        *string     `json:"correo"`
}

type InvoiceItem struct {
	NumItem         int      `json:"numItem"`
	TipoItem        int      `json:"tipoItem"`
	NumeroDocumento *string  `json:"numeroDocumento"`
	Codigo          *string  `json:"codigo"`
	CodTributo      *string  `json:"codTributo"`
	Descripcion     string   `json:"descripcion"`
	Cantidad        float64  `json:"cantidad"`
	UniMedida       int      `json:"uniMedida"`
	PrecioUni       float64  `json:"precioUni"`
	MontoDescu      float64  `json:"montoDescu"`
	VentaNoSuj      float64  `json:"ventaNoSuj"`
	VentaExenta     float64  `json:"ventaExenta"`
	VentaGravada    float64  `json:"ventaGravada"`
	Tributos        []string `json:"tributos"`
	PSV             float64  `json:"psv"`
	NoGravado       float64  `json:"noGravado"`
	IvaItem         float64  `json:"ivaItem"`
}
