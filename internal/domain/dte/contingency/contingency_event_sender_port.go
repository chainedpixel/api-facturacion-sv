package contingency

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// ContingencyEventSender interface for sending contingency events
type ContingencyEventSender interface {
	PrepareAndSendContingencyEvent(ctx context.Context, docs []dte.ContingencyDocument) error
}
