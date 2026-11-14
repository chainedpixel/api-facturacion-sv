package contingency

import "context"

// ContingencyManager interface for managing documents in contingency mode
type ContingencyManager interface {
	// StoreDocumentInContingency signs the deferred document once, stores it for retransmission and returns it so it
	// can be delivered to the consumer exactly as it will be transmitted.
	StoreDocumentInContingency(ctx context.Context, document interface{}, dteType string, contingencyType int8, reason string) (*DeferredDocument, error)
	RetransmitPendingDocuments(ctx context.Context) error
}
