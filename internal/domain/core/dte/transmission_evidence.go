package dte

// TransmissionEvidence agrupa lo que Hacienda devuelve al recibir un DTE o evento y lo que se transmitio.
// Con ella se arma el Archivo DTE que se entrega al receptor (Anexo I, regla 4.1): estructura firmada,
// firmaElectronica (SignedDocument) y selloRecibido (ReceptionStamp).
type TransmissionEvidence struct {
	ReceptionStamp   *string
	SignedDocument   string
	ProcessingDate   string
	HaciendaResponse string
}
