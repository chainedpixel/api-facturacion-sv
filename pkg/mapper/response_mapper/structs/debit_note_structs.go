package structs

type DebitNoteDTEResponse struct {
	Identificacion       *DTEIdentification     `json:"identificacion"`
	Emisor               DebitNoteDTEIssuer     `json:"emisor"`
	Receptor             DTEReceiver            `json:"receptor"`
	CuerpoDocumento      []DebitNoteDTEItem     `json:"cuerpoDocumento"`
	Resumen              *DebitNoteDTESummary   `json:"resumen"`
	DocumentoRelacionado []DTERelatedDocument   `json:"documentoRelacionado"`
	VentaTercero         *DTEThirdPartySale     `json:"ventaTercero"`
	Extension            *DebitNoteDTEExtension `json:"extension"`
	Apendice             []DTEApendice          `json:"apendice"`
}

type DebitNoteDTEItem struct {
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
}

type DebitNoteDTESummary struct {
	TotalNoSuj          float64  `json:"totalNoSuj"`
	TotalExenta         float64  `json:"totalExenta"`
	TotalGravada        float64  `json:"totalGravada"`
	SubTotalVentas      float64  `json:"subTotalVentas"`
	DescuNoSuj          float64  `json:"descuNoSuj"`
	DescuExenta         float64  `json:"descuExenta"`
	DescuGravada        float64  `json:"descuGravada"`
	TotalDescu          float64  `json:"totalDescu"`
	Tributos            []DTETax `json:"tributos"`
	SubTotal            float64  `json:"subTotal"`
	IvaRete1            float64  `json:"ivaRete1"`
	IvaPerci1           float64  `json:"ivaPerci1"`
	ReteRenta           float64  `json:"reteRenta"`
	MontoTotalOperacion float64  `json:"montoTotalOperacion"`
	TotalLetras         string   `json:"totalLetras"`
	CondicionOperacion  int      `json:"condicionOperacion"`
	NumPagoElectronico  *string  `json:"numPagoElectronico"`
}

type DebitNoteDTEExtension struct {
	NombreEntrega    string  `json:"nombEntrega"`
	DocumentoEntrega string  `json:"docuEntrega"`
	NombreRecibe     string  `json:"nombRecibe"`
	DocumentoRecibe  string  `json:"docuRecibe"`
	Observacion      *string `json:"observaciones"`
}

type DebitNoteDTEIssuer struct {
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
}
