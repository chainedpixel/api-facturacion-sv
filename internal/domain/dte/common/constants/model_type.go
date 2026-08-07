package constants

const (
	ModeloFacturacionPrevio = iota + 1
	ModeloFacturacionDiferido
)

var (
	AllowedModeloFacturacion = []int{
		ModeloFacturacionPrevio,
		ModeloFacturacionDiferido,
	}
)
