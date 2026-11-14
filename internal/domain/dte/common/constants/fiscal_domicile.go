package constants

// CAT-032 Domicilio Fiscal
const (
	Domiciled = iota + 1
	NonDomiciled
)

var (
	ValidFiscalDomiciles = map[int]bool{
		Domiciled:    true,
		NonDomiciled: true,
	}
)
