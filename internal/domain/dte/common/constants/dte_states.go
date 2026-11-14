package constants

const (
	DocumentReceived = "RECEIVED"
	DocumentRejected = "REJECTED"
	DocumentInvalid  = "INVALIDATED"
	DocumentPending  = "PENDING"
	DocumentExpired  = "EXPIRED"
)

// DebitNoteInvalid is the balance transaction type recorded when a debit note is invalidated; it subtracts
// from the balance the debit note had added.
const DebitNoteInvalid = "DEBIT_NOTE_INVALID"

const (
	TransmissionContingency = "CONTINGENCY"
	TransmissionNormal      = "NORMAL"
)

const (
	PhysicalDocument   = 1
	ElectronicDocument = 2
)

var (
	ValidReceiverDocumentStates = map[string]bool{
		DocumentReceived: true,
		DocumentPending:  true,
		DocumentRejected: true,
		DocumentInvalid:  true,
		DocumentExpired:  true,
	}

	ValidTransmissionTypes = map[string]bool{
		TransmissionContingency: true,
		TransmissionNormal:      true,
	}
)
