package constants

const (
	DocumentoEmisor = iota + 1
	DocumentoReceptor
	DocumentoMedico
	DocumentoTransporte
)

var (
	AllowedAssociatedDocumentCodes = []int{
		DocumentoEmisor,
		DocumentoReceptor,
		DocumentoMedico,
		DocumentoTransporte,
	}
)
