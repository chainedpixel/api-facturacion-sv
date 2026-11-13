package db_models

import "time"

// DTEEvent guarda los eventos transmitidos a Hacienda (invalidacion, contingencia) con su evidencia:
// estructura (json_data), JWS transmitido (signed_document), sello, fecha de procesamiento y la respuesta
// completa de Hacienda. Con signed_document y reception_stamp se arma el Archivo DTE del evento.
type DTEEvent struct {
	GenerationCode   string    `gorm:"column:generation_code;type:varchar(36);primaryKey;not null"`
	BranchID         uint      `gorm:"column:branch_id;type:uint;not null;index:idx_dte_event_branch"`
	EventType        string    `gorm:"column:event_type;type:varchar(20);not null;index:idx_dte_event_type"`
	RelatedDTEID     *string   `gorm:"column:related_dte_id;type:varchar(36);index:idx_dte_event_related"`
	ReplacementDTEID *string   `gorm:"column:replacement_dte_id;type:varchar(36)"`
	JSONData         string    `gorm:"column:json_data;type:json;not null"`
	SignedDocument   *string   `gorm:"column:signed_document;type:longtext"`
	ReceptionStamp   *string   `gorm:"column:reception_stamp;type:varchar(40)"`
	ProcessingDate   *string   `gorm:"column:processing_date;type:varchar(30)"`
	HaciendaResponse *string   `gorm:"column:hacienda_response;type:text"`
	Status           string    `gorm:"column:status;type:varchar(15);not null"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`
}

func (DTEEvent) TableName() string {
	return "dte_events"
}
