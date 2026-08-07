package constants

const (
	Producto = iota + 1
	Servicio
	ProductoYServicio
	Impuesto
)

var (
	AllowedItemTypes = []int{
		Producto,
		Servicio,
		ProductoYServicio,
		Impuesto,
	}
)
