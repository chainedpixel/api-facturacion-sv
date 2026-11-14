package utils

import "context"

type forcedContingencyTypeKey struct{}

// WithForcedContingencyType returns a context that asks the forced contingency simulation for the given contingency type.
func WithForcedContingencyType(ctx context.Context, contingencyType int) context.Context {
	return context.WithValue(ctx, forcedContingencyTypeKey{}, contingencyType)
}

// ForcedContingencyType returns the contingency type requested for the forced contingency simulation, or 0 when none
// was requested.
func ForcedContingencyType(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	contingencyType, _ := ctx.Value(forcedContingencyTypeKey{}).(int)
	return contingencyType
}
