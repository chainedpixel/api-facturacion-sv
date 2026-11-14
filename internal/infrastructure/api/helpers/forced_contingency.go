package helpers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

// ForcedContingencyHeader is the request header that selects the contingency type simulated by the forced
// contingency mode of the test environment.
const ForcedContingencyHeader = "X-Force-Contingency-Type"

// RequestContextWithForcedContingency returns the request context carrying the contingency type asked in
// ForcedContingencyHeader. The header is honored only when forced contingency is enabled in the test environment
// (ambient 00) and only for types 1 and 5; otherwise the request context is returned unchanged.
func RequestContextWithForcedContingency(r *http.Request) context.Context {
	ctx := r.Context()
	if config.Server == nil || !config.Server.ForceContingency || config.Server.AmbientCode != "00" {
		return ctx
	}

	contingencyType, err := strconv.Atoi(strings.TrimSpace(r.Header.Get(ForcedContingencyHeader)))
	if err != nil {
		return ctx
	}
	if contingencyType != constants.NoDisponibilidadMH && contingencyType != constants.OtroMotivo {
		return ctx
	}

	return utils.WithForcedContingencyType(ctx, contingencyType)
}
