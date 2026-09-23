package mappers_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInvoiceV2ResponseMapping_HaciendaSchemaCompliance validates Invoice v2 JSON serialization rules
func TestInvoiceV2ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	builder := fixtures.NewDTEBuilder()
	invoiceDoc, err := builder.BuildElectronicInvoice()
	require.NoError(t, err)

	id, ok := invoiceDoc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(2))

	mhInvoice := response_mapper.ToMHInvoice(invoiceDoc)
	require.NotNil(t, mhInvoice)

	jsonData, err := json.Marshal(mhInvoice)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	err = json.Unmarshal(jsonData, &rawMap)
	require.NoError(t, err)

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), ident["version"])
	assert.Equal(t, constants.FacturaElectronica, ident["tipoDte"])

	items, ok := rawMap["cuerpoDocumento"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	firstItem := items[0].(map[string]interface{})
	_, hasIvaItem := firstItem["ivaItem"]
	assert.True(t, hasIvaItem)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasIvaRete := resumen["ivaRete"]
	assert.True(t, hasIvaRete)
	_, hasIvaRete1 := resumen["ivaRete1"]
	assert.False(t, hasIvaRete1)

	_, hasObservaciones := resumen["observaciones"]
	assert.True(t, hasObservaciones)
	assert.Nil(t, resumen["observaciones"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)
}

// TestInvoiceV2WithObservations validates observations mapping and extension fallback
func TestInvoiceV2WithObservations(t *testing.T) {
	test.TestMain(t)

	builder := fixtures.NewDTEBuilder()
	invoiceDoc, err := builder.BuildElectronicInvoice()
	require.NoError(t, err)

	obs := "Entrega en sucursal central"
	err = invoiceDoc.GetSummary().SetObservations(&obs)
	require.NoError(t, err)

	mhInvoice := response_mapper.ToMHInvoice(invoiceDoc)
	require.NotNil(t, mhInvoice)
	require.NotNil(t, mhInvoice.Resumen.Observaciones)
	assert.Equal(t, obs, *mhInvoice.Resumen.Observaciones)

	jsonData, err := json.Marshal(mhInvoice)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))
	resumen := rawMap["resumen"].(map[string]interface{})
	assert.Equal(t, obs, resumen["observaciones"])
	assert.NotContains(t, rawMap, "extension")
}

// TestCCFV4ResponseMapping_HaciendaSchemaCompliance validates CCF v4 JSON serialization rules
func TestCCFV4ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	builder := fixtures.NewDTEBuilder()
	ccfDoc, err := builder.BuildCreditFiscalDocument()
	require.NoError(t, err)

	id, ok := ccfDoc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(4))

	mhCCF := response_mapper.ToMHCreditFiscalInvoice(ccfDoc)
	require.NotNil(t, mhCCF)

	jsonData, err := json.Marshal(mhCCF)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	err = json.Unmarshal(jsonData, &rawMap)
	require.NoError(t, err)

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), ident["version"])
	assert.Equal(t, constants.CCFElectronico, ident["tipoDte"])

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasIvaRete := resumen["ivaRete"]
	assert.True(t, hasIvaRete)
	_, hasIvaRete1 := resumen["ivaRete1"]
	assert.False(t, hasIvaRete1)
	_, hasIvaPerci1 := resumen["ivaPerci1"]
	assert.False(t, hasIvaPerci1)

	_, hasObservaciones := resumen["observaciones"]
	assert.True(t, hasObservaciones)

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)
}

// TestCCFV4WithObservations validates CCF observations mapping in summary
func TestCCFV4WithObservations(t *testing.T) {
	test.TestMain(t)

	builder := fixtures.NewDTEBuilder()
	ccfDoc, err := builder.BuildCreditFiscalDocument()
	require.NoError(t, err)

	obs := "Condiciones especiales CCF"
	err = ccfDoc.GetSummary().SetObservations(&obs)
	require.NoError(t, err)

	mhCCF := response_mapper.ToMHCreditFiscalInvoice(ccfDoc)
	require.NotNil(t, mhCCF)
	require.NotNil(t, mhCCF.Resumen.Observaciones)
	assert.Equal(t, obs, *mhCCF.Resumen.Observaciones)

	jsonData, err := json.Marshal(mhCCF)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))
	resumen := rawMap["resumen"].(map[string]interface{})
	assert.Equal(t, obs, resumen["observaciones"])
	assert.NotContains(t, rawMap, "extension")
}

// TestObservationsLengthConstraint validates observations maximum length validation
func TestObservationsLengthConstraint(t *testing.T) {
	test.TestMain(t)

	summary := &models.Summary{}
	validObs := strings.Repeat("A", 3000)
	err := summary.SetObservations(&validObs)
	assert.NoError(t, err)
	assert.Equal(t, validObs, *summary.GetObservations())

	invalidObs := strings.Repeat("B", 3001)
	err = summary.SetObservations(&invalidObs)
	assert.Error(t, err)
}

// TestRequestMapperObservationMapping validates Request to Domain mapping for observations
func TestRequestMapperObservationMapping(t *testing.T) {
	test.TestMain(t)

	req := fixtures.CreateDefaultInvoiceRequest()
	obs := "Nota de pedido especial 123"
	req.Summary.Observations = &obs

	mapper := request_mapper.NewInvoiceMapper()
	issuer := fixtures.CreateDefaultIssuer()

	domainData, err := mapper.MapToInvoiceData(req, issuer)
	require.NoError(t, err)
	require.NotNil(t, domainData)
	require.NotNil(t, domainData.InvoiceSummary.GetObservations())
	assert.Equal(t, obs, *domainData.InvoiceSummary.GetObservations())
	assert.Equal(t, 2, domainData.InputDataCommon.Identification.GetVersion())
}

// TestCCFRequestMapperObservationMapping validates CCF Request to Domain mapping for observations
func TestCCFRequestMapperObservationMapping(t *testing.T) {
	test.TestMain(t)

	req := fixtures.CreateDefaultCreditFiscalRequest()
	obs := "Pago a 30 dias plazo"
	req.Summary.Observations = &obs

	mapper := request_mapper.NewCCFMapper()
	issuer := fixtures.CreateDefaultIssuer()

	domainData, err := mapper.MapToCCFData(req, issuer)
	require.NoError(t, err)
	require.NotNil(t, domainData)
	require.NotNil(t, domainData.CreditSummary.GetObservations())
	assert.Equal(t, obs, *domainData.CreditSummary.GetObservations())
	assert.Equal(t, 4, domainData.InputDataCommon.Identification.GetVersion())
}
