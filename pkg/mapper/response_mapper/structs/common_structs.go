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

// DTEAdjustmentIdentification maps the "identificacion" section of NCE and NDE (v4), que agregan
// el campo requerido "fusion" (Fusiones y otros). Va null salvo operaciones por fusion.
type DTEAdjustmentIdentification struct {
	*DTEIdentification
	Fusion *string `json:"fusion"`
}

// DTEIssuer maps the "emisor" section of the JSON Schema
type DTEIssuer struct {
	NIT             string     `json:"nit"`
	NRC             string     `json:"nrc"`
	Nombre          string     `json:"nombre"`
	CodActividad    string     `json:"codActividad"`
	DescActividad   string     `json:"descActividad"`
	Direccion       DTEAddress `json:"direccion"`
	Telefono        string     `json:"telefono"`
	Correo          string     `json:"correo"`
	NombreComercial *string    `json:"nombreComercial"`
	// Normativa v2.0 (Anexo IV): solo se informan los codigos asignados por el contribuyente.
	// tipoEstablecimiento, codEstableMH y codPuntoVentaMH ya no existen en los schemas de DTE.
	CodEstable    *string `json:"codEstable"`
	CodPuntoVenta *string `json:"codPuntoVenta"`
}

// DTEReceiverBase contiene los campos del "receptor" comunes a CCFE, NCE, NDE y CRE.
// Ningun campo lleva omitempty: los schemas los marcan como requeridos y se envian null cuando no aplican.
type DTEReceiverBase struct {
	Nombre          *string     `json:"nombre"`
	NRC             *string     `json:"nrc"`
	CodActividad    *string     `json:"codActividad"`
	DescActividad   *string     `json:"descActividad"`
	NombreComercial *string     `json:"nombreComercial"`
	Direccion       *DTEAddress `json:"direccion"`
	Telefono        *string     `json:"telefono"`
	Correo          *string     `json:"correo"`
}

// DTEReceiver maps the "receptor" section of CCFE (fe-ccf-v4): se identifica por NIT
type DTEReceiver struct {
	NIT *string `json:"nit"`
	DTEReceiverBase
}

// DTEDocumentReceiver maps the "receptor" section of NCE, NDE (v4) y CRE (v2): se identifica por
// tipo y numero de documento. Normativa v2.0 (Anexo VI): se elimino "NIT (Receptor)" para NCE y NDE.
type DTEDocumentReceiver struct {
	TipoDocumento *string `json:"tipoDocumento"`
	NumDocumento  *string `json:"numDocumento"`
	DTEReceiverBase
}

// DTEAddress maps the address according to the Schema
type DTEAddress struct {
	Departamento string `json:"departamento"`
	Municipio    string `json:"municipio"`
	Distrito     string `json:"distrito"`
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
}

// DTESummary maps the CCF summary (all fields required according to schema fe-ccf-v4).
// Normativa v2.0 (Anexo VI): se elimino "Retencion renta" para CCFE; el schema tampoco define totalIva.
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
	IvaRete             float64      `json:"ivaRete"`
	IvaPerci            float64      `json:"ivaPerci"`
	MontoTotalOperacion float64      `json:"montoTotalOperacion"`
	TotalNoGravado      float64      `json:"totalNoGravado"`
	TotalPagar          float64      `json:"totalPagar"`
	TotalLetras         string       `json:"totalLetras"`
	SaldoFavor          float64      `json:"saldoFavor"`
	CondicionOperacion  int          `json:"condicionOperacion"`
	Pagos               []DTEPayment `json:"pagos"`
	NumPagoElectronico  *string      `json:"numPagoElectronico"`
	Observaciones       *string      `json:"observaciones"`
}

// DTETax maps a tax
type DTETax struct {
	Codigo      string  `json:"codigo"`
	Descripcion string  `json:"descripcion"`
	Valor       float64 `json:"valor"`
}

// DTEPayment maps a payment
type DTEPayment struct {
	Codigo     *string `json:"codigo"`
	MontoPago  float64 `json:"montoPago"`
	Referencia *string `json:"referencia"`
	Plazo      *string `json:"plazo"`
	Periodo    *int    `json:"periodo"`
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
	NIT            string `json:"nit"`
	Nombre         string `json:"nombre"`
	CodDomiciliado *int   `json:"codDomiciliado"` // CAT-032 Domicilio Fiscal
}
