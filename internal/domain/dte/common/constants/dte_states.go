package constants

const (
	DocumentReceived = "RECEIVED"
	DocumentRejected = "REJECTED"
	DocumentInvalid  = "INVALIDATED"
	DocumentPending  = "PENDING"
	DocumentExpired  = "EXPIRED"
)

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
