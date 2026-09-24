package services

import (
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/contingency"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/signing"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/processors"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDocumentRequestData_StandardElectronicDTE verifies data extraction for a regular electronic DTE.
func TestGetDocumentRequestData_StandardElectronicDTE(t *testing.T) {
	test.TestMain(t)

	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version":          float64(2),
			"tipoDte":          "01",
			"numeroControl":    "DTE-01-00000000-000000000000042",
			"codigoGeneracion": "12345678-1234-1234-1234-123456789012",
		},
	}

	version, dteType, genCode, seqNum, err := processors.GetDocumentRequestData(doc)
	require.NoError(t, err)
	assert.Equal(t, 2, version)
	assert.Equal(t, "01", dteType)
	assert.Equal(t, "12345678-1234-1234-1234-123456789012", genCode)
	assert.Equal(t, 42, seqNum)
}

// TestGetDocumentRequestData_InvalidationWithNilControlNumber verifies fallback when control number is null.
func TestGetDocumentRequestData_InvalidationWithNilControlNumber(t *testing.T) {
	test.TestMain(t)

	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version":          float64(3),
			"codigoGeneracion": "12345678-1234-1234-1234-123456789012",
		},
		"documento": map[string]interface{}{
			"tipoDte":       "01",
			"numeroControl": nil,
		},
	}

	version, dteType, genCode, seqNum, err := processors.GetDocumentRequestData(doc)
	require.NoError(t, err)
	assert.Equal(t, 3, version)
	assert.Equal(t, "01", dteType)
	assert.Equal(t, "12345678-1234-1234-1234-123456789012", genCode)
	assert.Equal(t, 1, seqNum)
}

// TestGetDocumentRequestData_InvalidationWithPhysicalControlNumber verifies fallback for non-standard control numbers.
func TestGetDocumentRequestData_InvalidationWithPhysicalControlNumber(t *testing.T) {
	test.TestMain(t)

	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version":          float64(3),
			"codigoGeneracion": "12345678-1234-1234-1234-123456789012",
		},
		"documento": map[string]interface{}{
			"tipoDte":       "01",
			"numeroControl": "PHYSICAL-INVOICE-001",
		},
	}

	version, dteType, genCode, seqNum, err := processors.GetDocumentRequestData(doc)
	require.NoError(t, err)
	assert.Equal(t, 3, version)
	assert.Equal(t, "01", dteType)
	assert.Equal(t, "12345678-1234-1234-1234-123456789012", genCode)
	assert.Equal(t, 1, seqNum)
}

// TestGetDocumentRequestData_InvalidationWithStandardControlNumber verifies correlativo extraction in invalidations.
func TestGetDocumentRequestData_InvalidationWithStandardControlNumber(t *testing.T) {
	test.TestMain(t)

	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version":          float64(3),
			"codigoGeneracion": "12345678-1234-1234-1234-123456789012",
		},
		"documento": map[string]interface{}{
			"tipoDte":       "03",
			"numeroControl": "DTE-03-00000000-000000000000100",
		},
	}

	version, dteType, genCode, seqNum, err := processors.GetDocumentRequestData(doc)
	require.NoError(t, err)
	assert.Equal(t, 3, version)
	assert.Equal(t, "03", dteType)
	assert.Equal(t, "12345678-1234-1234-1234-123456789012", genCode)
	assert.Equal(t, 100, seqNum)
}

// TestHaciendaAuthService_TimeoutConfiguration verifies the 8-second HTTP timeout on authentication service.
func TestHaciendaAuthService_TimeoutConfiguration(t *testing.T) {
	test.TestMain(t)

	authSvc := signing.NewHaciendaAuthService(nil, nil)
	require.NotNil(t, authSvc)

	concreteAuth, ok := authSvc.(*signing.HaciendaAuthService)
	require.True(t, ok)
	assert.Equal(t, 8*time.Second, concreteAuth.GetTimeout())
}

// TestContingencyEventService_TimeoutConfiguration verifies the 8-second HTTP timeout on contingency service.
func TestContingencyEventService_TimeoutConfiguration(t *testing.T) {
	test.TestMain(t)

	eventSvc := contingency.NewContingencyEventService(nil, nil, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, eventSvc)
	assert.Equal(t, 8*time.Second, eventSvc.GetHTTPTimeout())
}

// TestMHPaths_NoTrailingSlashes verifies all configured MH paths have trailing slashes trimmed.
func TestMHPaths_NoTrailingSlashes(t *testing.T) {
	test.TestMain(t)

	require.NotNil(t, config.MHPaths)
	config.MHPaths.AuthURL = "https://apitest.dtes.mh.gob.sv/seguridad/auth/"
	config.MHPaths.ReceptionURL = "https://apitest.dtes.mh.gob.sv/fesv/recepciondte/"
	config.MHPaths.LoteReceptionURL = "https://apitest.dtes.mh.gob.sv/fesv/recepcionlote/"
	config.MHPaths.ReceptionConsultURL = "https://apitest.dtes.mh.gob.sv/fesv/recepcion/consultadte/"
	config.MHPaths.LoteReceptionConsultURL = "https://apitest.dtes.mh.gob.sv/fesv/recepcion/consultadtelote/"
	config.MHPaths.ContingencyURL = "https://apitest.dtes.mh.gob.sv/fesv/contingencia/"
	config.MHPaths.NullifyURL = "https://apitest.dtes.mh.gob.sv/fesv/anulardte/"

	config.SanitizeMHPaths(config.MHPaths)

	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/seguridad/auth", config.MHPaths.AuthURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/recepciondte", config.MHPaths.ReceptionURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/recepcionlote", config.MHPaths.LoteReceptionURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/recepcion/consultadte", config.MHPaths.ReceptionConsultURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/recepcion/consultadtelote", config.MHPaths.LoteReceptionConsultURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/contingencia", config.MHPaths.ContingencyURL)
	assert.Equal(t, "https://apitest.dtes.mh.gob.sv/fesv/anulardte", config.MHPaths.NullifyURL)

	assert.False(t, hasTrailingSlash(config.MHPaths.AuthURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.ReceptionURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.LoteReceptionURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.ReceptionConsultURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.LoteReceptionConsultURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.ContingencyURL))
	assert.False(t, hasTrailingSlash(config.MHPaths.NullifyURL))
}

// hasTrailingSlash checks if the given URL string ends with a slash.
func hasTrailingSlash(urlStr string) bool {
	return len(urlStr) > 0 && urlStr[len(urlStr)-1] == '/'
}
