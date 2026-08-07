package constants

const (
	TransmisionNormal = iota + 1
	TransmisionContingencia
)

var (
	AllowedTransmisionTypes = []int{
		TransmisionNormal,
		TransmisionContingencia,
	}
)
