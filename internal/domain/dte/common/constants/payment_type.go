package constants

const (
	BilletesMonedas = "01"
	TarjetaDebito   = "02"
	TarjetaCredito  = "03"
	Cheque          = "04"
	TransBancaria   = "05"
	TarjetaPrePago  = "06"
	Vales           = "07"
	CriptoMoneda    = "08"
	PagosElect      = "09"
	GiftCard        = "10"
	NotaAbono       = "11"
	OtraFormaPago   = "12"
	ContoPrepago    = "13"
	AplicaARete     = "14"
	NoAplica        = "99"
)

var (
	AllowedPaymentTypes = []string{
		BilletesMonedas,
		TarjetaDebito,
		TarjetaCredito,
		Cheque,
		TransBancaria,
		TarjetaPrePago,
		Vales,
		CriptoMoneda,
		PagosElect,
		GiftCard,
		NotaAbono,
		OtraFormaPago,
		ContoPrepago,
		AplicaARete,
		NoAplica,
	}
)
