package dte

// DTEDetails representa los detalles de un documento tributario electrónico
type DTEDetails struct {
	ID             string  `json:"id,omitempty"`
	DTEType        string  `json:"dte_type"`
	ControlNumber  string  `json:"control_number"`
	ReceptionStamp *string `json:"reception_stamp,omitempty"`
	Transmission   string  `json:"transmission"`
	Status         string  `json:"status"`
	JSONData       string  `json:"json_data"`
}
