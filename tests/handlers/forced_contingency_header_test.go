package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

func withForcedContingencyConfig(t *testing.T, enabled bool, ambient string) {
	t.Helper()
	test.TestMain(t)
	previousEnabled, previousAmbient := config.Server.ForceContingency, config.Server.AmbientCode
	config.Server.ForceContingency = enabled
	config.Server.AmbientCode = ambient
	t.Cleanup(func() {
		config.Server.ForceContingency = previousEnabled
		config.Server.AmbientCode = previousAmbient
	})
}

func forcedContingencyRequest(headerValue *string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice", nil)
	if headerValue != nil {
		req.Header.Set(helpers.ForcedContingencyHeader, *headerValue)
	}
	return req
}

// TestForcedContingencyHeaderSelection checks which header values are honored when forced contingency is enabled in
// the test environment.
func TestForcedContingencyHeaderSelection(t *testing.T) {
	withForcedContingencyConfig(t, true, "00")

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"type 1", "1", 1},
		{"type 5", "5", 5},
		{"type 5 with surrounding spaces", "  5 ", 5},
		{"type 0", "0", 0},
		{"type 2", "2", 0},
		{"type 3", "3", 0},
		{"type 4", "4", 0},
		{"type 6", "6", 0},
		{"negative", "-1", 0},
		{"decimal", "5.0", 0},
		{"text", "abc", 0},
		{"empty", "", 0},
		{"only spaces", "   ", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := helpers.RequestContextWithForcedContingency(forcedContingencyRequest(&tc.header))
			assert.Equal(t, tc.want, utils.ForcedContingencyType(ctx))
		})
	}

	t.Run("missing header", func(t *testing.T) {
		ctx := helpers.RequestContextWithForcedContingency(forcedContingencyRequest(nil))
		assert.Zero(t, utils.ForcedContingencyType(ctx))
	})
}

// TestForcedContingencyHeaderIgnoredOutsideTestEnvironment checks that the header never has effect when forced
// contingency is disabled or the ambient is not the test one, and that the request context is returned untouched.
func TestForcedContingencyHeaderIgnoredOutsideTestEnvironment(t *testing.T) {
	cases := []struct {
		name    string
		enabled bool
		ambient string
	}{
		{"forced contingency disabled in test ambient", false, "00"},
		{"forced contingency enabled in production ambient", true, "01"},
		{"forced contingency disabled in production ambient", false, "01"},
		{"forced contingency enabled with empty ambient", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withForcedContingencyConfig(t, tc.enabled, tc.ambient)
			for _, value := range []string{"1", "5"} {
				req := forcedContingencyRequest(&value)
				ctx := helpers.RequestContextWithForcedContingency(req)
				assert.Zero(t, utils.ForcedContingencyType(ctx), "header %s must be ignored", value)
				assert.Equal(t, req.Context(), ctx)
			}
		})
	}
}

// TestForcedContingencyHeaderWithoutServerConfig checks that a missing server configuration does not panic and the
// header is ignored.
func TestForcedContingencyHeaderWithoutServerConfig(t *testing.T) {
	test.TestMain(t)
	previous := config.Server
	config.Server = nil
	t.Cleanup(func() { config.Server = previous })

	value := "5"
	ctx := helpers.RequestContextWithForcedContingency(forcedContingencyRequest(&value))
	assert.Zero(t, utils.ForcedContingencyType(ctx))
}

// TestForcedContingencyTypeOnPlainContexts checks the default value of the context accessor.
func TestForcedContingencyTypeOnPlainContexts(t *testing.T) {
	assert.Zero(t, utils.ForcedContingencyType(context.Background()))
	assert.Zero(t, utils.ForcedContingencyType(nil))
	assert.Equal(t, 5, utils.ForcedContingencyType(utils.WithForcedContingencyType(context.Background(), 5)))
	assert.Equal(t, 1, utils.ForcedContingencyType(utils.WithForcedContingencyType(context.Background(), 1)))
}
