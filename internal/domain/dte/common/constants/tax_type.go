package constants

const (
	TaxIVA            = "20"
	TaxIVAExport      = "C3"
	TaxTourism        = "59"
	TaxTourismAirport = "71"
	TaxFOVIAL         = "D1"
	TaxCOTRANS        = "C8"
	TaxSpecialOther   = "D5"

	TaxIvaAmount            = 0.13
	TaxIVAExportAmount      = 0.0
	TaxTourismAmount        = 0.05
	TaxTourismAirportAmount = 7.0
	TaxFOVIALAmount         = 0.20
	TaxCOTRANSAmount        = 0.10
)

var (
	AllowedTaxTypes = []string{
		TaxIVA,
		TaxIVAExport,
		TaxTourism,
		TaxTourismAirport,
		TaxFOVIAL,
		TaxCOTRANS,
		TaxSpecialOther,
	}

	MapAllowedTaxTypes = map[string]bool{
		TaxIVA:            true,
		TaxIVAExport:      true,
		TaxTourism:        true,
		TaxTourismAirport: true,
		TaxFOVIAL:         true,
		TaxCOTRANS:        true,
		TaxSpecialOther:   true,
	}
)
