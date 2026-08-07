package constants

const (
	Sucursal       = "01"
	CasaMatriz     = "02"
	DepositoBodega = "04"
	PredioOPatio   = "07"
	Otro           = "20"
)

var (
	AllowedEstablishmentTypes = []string{
		CasaMatriz,
		Sucursal,
		DepositoBodega,
		PredioOPatio,
		Otro,
	}
)
