package dte_documents

import "time"

type ReservedSequence struct {
	ID                    uint
	BranchID              uint
	DTEType               string
	SequenceNumber        uint
	Year                  int
	Status                string
	DocumentID            *string
	ReservedAt            time.Time
	ConfirmedAt           *time.Time
	ReleasedAt            *time.Time
	ExpiresAt             *time.Time
	RejectionReason       *string
	HaciendaCode          *string
	IsContingency         bool
	ContingencyDocumentID *string
}

const (
	ReservationStatusReserved  = "RESERVED"
	ReservationStatusConfirmed = "CONFIRMED"
	ReservationStatusReleased  = "RELEASED"
)
