package structs

type CommonDTEDocument struct {
	Identificacion       *DTEIdentification   `json:"identificacion"`
	Emisor               DTEIssuer            `json:"emisor"`
	Receptor             DTEReceiver          `json:"receptor"`
	Resumen              *DTESummary          `json:"resumen"`
	DocumentoRelacionado []DTERelatedDocument `json:"documentoRelacionado"`
	OtrosDocumentos      []DTEOtherDocument   `json:"otrosDocumentos"`
	VentaTercero         *DTEThirdPartySale   `json:"ventaTercero"`
	CuerpoDocumento      []DTEItem            `json:"cuerpoDocumento"`
	Extension            *DTEExtension        `json:"extension"`
	Apendice             []DTEApendice        `json:"apendice"`
}

// DTEIdentification maps the "identificacion" section of the JSON Schema
type DTEIdentification struct {
	Version          int     `json:"version"`
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"`
	NumeroControl    string  `json:"numeroControl"`
	CodigoGeneracion string  `json:"codigoGeneracion"`
	TipoModelo       int     `json:"tipoModelo"`
	TipoOperacion    int     `json:"tipoOperacion"`
	TipoContingencia *int    `json:"tipoContingencia"`
	MotivoContin     *string `json:"motivoContin"`
	FecEmi           string  `json:"fecEmi"`
	HorEmi           string  `json:"horEmi"`
	TipoMoneda       string  `json:"tipoMoneda"`
}

// DTEIssuer maps the "emisor" section of the JSON Schema
type DTEIssuer struct {
	NIT                 string     `json:"nit,omitempty"`
	NRC                 string     `json:"nrc"`
	Nombre              string     `json:"nombre"`
	CodActividad        string     `json:"codActividad"`
	DescActividad       string     `json:"descActividad"`
	TipoEstablecimiento string     `json:"tipoEstablecimiento"`
	Direccion           DTEAddress `json:"direccion"`
	Telefono            string     `json:"telefono"`
	Correo              string     `json:"correo"`
	NombreComercial     *string    `json:"nombreComercial"`
	CodEstableMH        *string    `json:"codEstableMH"`
	CodEstable          *string    `json:"codEstable"`
	CodPuntoVentaMH     *string    `json:"codPuntoVentaMH"`
	CodPuntoVenta       *string    `json:"codPuntoVenta"`
}

// DTEReceiver maps the "receptor" section of the JSON Schema
type DTEReceiver struct {
	Nombre          *string     `json:"nombre"`
	TipoDocumento   *string     `json:"tipoDocumento,omitempty"`
	NumDocumento    *string     `json:"numDocumento,omitempty"`
	NRC             *string     `json:"nrc"`
	NIT             *string     `json:"nit,omitempty"`
	CodActividad    *string     `json:"codActividad"`
	DescActividad   *string     `json:"descActividad"`
	Direccion       *DTEAddress `json:"direccion"`
	Telefono        *string     `json:"telefono"`
	Correo          *string     `json:"correo"`
	NombreComercial *string     `json:"nombreComercial,omitempty"`
}

// DTEAddress maps the address according to the Schema
type DTEAddress struct {
	Departamento string `json:"departamento"`
	Municipio    string `json:"municipio"`
	Complemento  string `json:"complemento"`
}

// DTEItem maps an item of the document body
type DTEItem struct {
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
	IvaItem         float64  `json:"ivaItem,omitempty"`
}

// DTESummary maps the summary (all fields required according to schema)
type DTESummary struct {
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
	IvaRete1            float64      `json:"ivaRete1"`
	IvaPerci1           *float64     `json:"ivaPerci1,omitempty"`
	ReteRenta           float64      `json:"reteRenta"`
	MontoTotalOperacion float64      `json:"montoTotalOperacion"`
	TotalNoGravado      float64      `json:"totalNoGravado"`
	TotalPagar          float64      `json:"totalPagar"`
	TotalLetras         string       `json:"totalLetras"`
	TotalIva            float64      `json:"totalIva,omitempty"`
	SaldoFavor          float64      `json:"saldoFavor"`
	CondicionOperacion  int          `json:"condicionOperacion"`
	Pagos               []DTEPayment `json:"pagos"`
	NumPagoElectronico  *string      `json:"numPagoElectronico"`
}

// DTETax maps a tax
type DTETax struct {
	Codigo      string  `json:"codigo"`
	Descripcion string  `json:"descripcion"`
	Valor       float64 `json:"valor"`
}

// DTEPayment maps a payment
type DTEPayment struct {
	Codigo     string  `json:"codigo"`
	MontoPago  float64 `json:"montoPago"`
	Referencia *string `json:"referencia"`
	Plazo      *string `json:"plazo"`
	Periodo    *int    `json:"periodo"`
}

type DTEExtension struct {
	NombreEntrega    string  `json:"nombEntrega"`
	DocumentoEntrega string  `json:"docuEntrega"`
	NombreRecibe     string  `json:"nombRecibe"`
	DocumentoRecibe  string  `json:"docuRecibe"`
	Observacion      *string `json:"observaciones"`
	PlacaVehiculo    *string `json:"placaVehiculo"`
}

type DTEApendice struct {
	Campo    string `json:"campo"`
	Etiqueta string `json:"etiqueta"`
	Valor    string `json:"valor"`
}

type DTERelatedDocument struct {
	TipoDocumento   string `json:"tipoDocumento"`
	TipoGeneracion  int    `json:"tipoGeneracion"`
	NumeroDocumento string `json:"numeroDocumento"`
	FechaEmision    string `json:"fechaEmision"`
}

type DTEOtherDocument struct {
	CodDocAsociado int        `json:"codDocAsociado"`
	Description    *string    `json:"descDocumento"`
	Detail         *string    `json:"detalleDocumento"`
	Doctor         *DTEDoctor `json:"medico"`
}

type DTEDoctor struct {
	Nombre            string  `json:"nombre"`
	NIT               *string `json:"nit"`
	DocIdentificacion *string `json:"docIdentificacion"`
	TipoServicio      int     `json:"tipoServicio"`
}

type DTEThirdPartySale struct {
	NIT    string `json:"nit"`
	Nombre string `json:"nombre"`
}
