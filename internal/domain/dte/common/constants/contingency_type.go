package constants

const (
	NoDisponibilidadMH = iota + 1
	FallaConexionSistema
	FallaServicioInternet
	FallaEnergiaElectrica
	OtroMotivo
)

var (
	AllowedContingencyTypes = []int{
		FallaConexionSistema,
		FallaServicioInternet,
		FallaEnergiaElectrica,
		NoDisponibilidadMH,
		OtroMotivo,
	}

	ContingencyReasons = map[int8]string{
		FallaConexionSistema:  "Error de conexión con sistemas internos",
		FallaServicioInternet: "Falla en el servicio de internet",
		FallaEnergiaElectrica: "Interrupción del servicio eléctrico",
		NoDisponibilidadMH:    "Servicio del Ministerio de Hacienda no disponible",
		OtroMotivo:            "Error en el proceso de emisión del documento",
	}
)

func GetContingencyReason(contingencyType int8) string {
	if reason, exists := ContingencyReasons[contingencyType]; exists {
		return reason
	}
	return ContingencyReasons[5]
}
