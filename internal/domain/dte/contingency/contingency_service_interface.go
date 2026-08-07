package contingency

import "context"

// ContingencyManager interface for managing documents in contingency mode
type ContingencyManager interface {
	StoreDocumentInContingency(ctx context.Context, document interface{}, dteType string, contingencyType int8, reason string) error
	RetransmitPendingDocuments(ctx context.Context) error
}
