package dte

// ReceptionResult es lo que Hacienda devuelve por cada documento procesado en un lote o consulta:
// el sello de recepcion y la fecha y hora de procesamiento (fhProcesamiento)
type ReceptionResult struct {
	ReceptionStamp string
	ProcessingDate string
}
