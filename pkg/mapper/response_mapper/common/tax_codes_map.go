package common

// MapTaxCodes maps the tax codes
func MapTaxCodes(taxes []string) {
	codes := make([]string, len(taxes))
	for i, tax := range taxes {
		codes[i] = tax
	}
}
