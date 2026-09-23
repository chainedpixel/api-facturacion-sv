package mappers_test

import (
	"encoding/json"
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

// TestRetentionV2ResponseMapping_HaciendaSchemaCompliance tests Comprobante de Retención v2 compliance.
func TestRetentionV2ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidRetention()
	require.NoError(t, err)

	id, ok := doc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(2))

	obs := "Retención de prueba 2026"
	doc.RetentionSummary.Observations = &obs

	mhCR := response_mapper.ToMHRetention(doc)
	require.NotNil(t, mhCR)

	jsonData, err := json.Marshal(mhCR)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), ident["version"])
	assert.Equal(t, constants.ComprobanteRetencionElectronico, ident["tipoDte"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)

	emisor, ok := rawMap["emisor"].(map[string]interface{})
	require.True(t, ok)
	_, hasCodEstable := emisor["codEstable"]
	assert.True(t, hasCodEstable)
	_, hasCodPuntoVenta := emisor["codPuntoVenta"]
	assert.True(t, hasCodPuntoVenta)
	_, hasTipoEstablecimiento := emisor["tipoEstablecimiento"]
	assert.False(t, hasTipoEstablecimiento)

	items, ok := rawMap["cuerpoDocumento"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	firstItem := items[0].(map[string]interface{})
	_, hasTipoGeneracion := firstItem["tipoGeneracion"]
	assert.True(t, hasTipoGeneracion)
	_, hasTipoDoc := firstItem["tipoDoc"]
	assert.False(t, hasTipoDoc)
	_, hasNumeroDocumento := firstItem["numeroDocumento"]
	assert.True(t, hasNumeroDocumento)
	_, hasNumDocumento := firstItem["numDocumento"]
	assert.False(t, hasNumDocumento)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasTotalIva := resumen["totalIva"]
	assert.True(t, hasTotalIva)
	_, hasTotalIvaRetenido := resumen["totalIvaRetenido"]
	assert.True(t, hasTotalIvaRetenido)
	_, hasTotalLetras := resumen["totalLetras"]
	assert.True(t, hasTotalLetras)
	_, hasTotalIVARetenidoLetras := resumen["totalIVAretenidoLetras"]
	assert.False(t, hasTotalIVARetenidoLetras)
	assert.Equal(t, obs, resumen["observaciones"])
}

// TestFSEV2ResponseMapping_HaciendaSchemaCompliance tests Factura de Sujeto Excluido v2 compliance.
func TestFSEV2ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidFSE()
	require.NoError(t, err)

	id, ok := doc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(2))

	mhFSE := response_mapper.ToMHFSE(doc)
	require.NotNil(t, mhFSE)

	jsonData, err := json.Marshal(mhFSE)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), ident["version"])
	assert.Equal(t, constants.FacturaSujetoExcluidoElectronica, ident["tipoDte"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)

	_, hasReceptor := rawMap["receptor"]
	assert.True(t, hasReceptor)
	_, hasSujetoExcluido := rawMap["sujetoExcluido"]
	assert.False(t, hasSujetoExcluido)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasIvaRete1 := resumen["ivaRete1"]
	assert.False(t, hasIvaRete1)
	_, hasReteRenta := resumen["reteRenta"]
	assert.True(t, hasReteRenta)
}

// TestRemissionNoteV4ResponseMapping_HaciendaSchemaCompliance tests Nota de Remisión v4 compliance.
func TestRemissionNoteV4ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, _, err := fixtures.BuildValidRemissionNote()
	require.NoError(t, err)

	id, ok := doc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(4))

	obs := "Traslado de mercadería a bodega central"
	require.NoError(t, doc.GetSummary().SetObservations(&obs))

	mhNR := response_mapper.ToMHRemissionNote(doc)
	require.NotNil(t, mhNR)

	jsonData, err := json.Marshal(mhNR)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), ident["version"])
	assert.Equal(t, constants.NotaRemisionElectronica, ident["tipoDte"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, obs, resumen["observaciones"])
}

// TestCreditNoteV4ResponseMapping_HaciendaSchemaCompliance tests Nota de Crédito v4 compliance.
func TestCreditNoteV4ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidCreditNote()
	require.NoError(t, err)

	id, ok := doc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(4))

	obs := "Ajuste de precio según acuerdo comercial"
	require.NoError(t, doc.GetSummary().SetObservations(&obs))

	mhNC := response_mapper.ToMHCreditNote(doc)
	require.NotNil(t, mhNC)

	jsonData, err := json.Marshal(mhNC)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), ident["version"])
	assert.Equal(t, constants.NotaCreditoElectronica, ident["tipoDte"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)

	emisor, ok := rawMap["emisor"].(map[string]interface{})
	require.True(t, ok)
	_, hasTipoEstablecimiento := emisor["tipoEstablecimiento"]
	assert.False(t, hasTipoEstablecimiento)

	items, ok := rawMap["cuerpoDocumento"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	firstItem := items[0].(map[string]interface{})
	_, hasNoGravado := firstItem["noGravado"]
	assert.True(t, hasNoGravado)
	_, hasIvaPerci := firstItem["ivaPerci"]
	assert.True(t, hasIvaPerci)
	_, hasTotalIva := firstItem["totalIva"]
	assert.True(t, hasTotalIva)
	_, hasIvaRete := firstItem["ivaRete"]
	assert.True(t, hasIvaRete)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasSummaryTotalIva := resumen["totalIva"]
	assert.True(t, hasSummaryTotalIva)
	_, hasTotalNoGravado := resumen["totalNoGravado"]
	assert.True(t, hasTotalNoGravado)
	_, hasTotalPagar := resumen["totalPagar"]
	assert.True(t, hasTotalPagar)
	_, hasSummaryIvaPerci := resumen["ivaPerci"]
	assert.True(t, hasSummaryIvaPerci)
	_, hasSummaryIvaRete := resumen["ivaRete"]
	assert.True(t, hasSummaryIvaRete)
	assert.Equal(t, obs, resumen["observaciones"])

	_, hasDescuNoSuj := resumen["descuNoSuj"]
	assert.False(t, hasDescuNoSuj)
	_, hasDescuExenta := resumen["descuExenta"]
	assert.False(t, hasDescuExenta)
	_, hasDescuGravada := resumen["descuGravada"]
	assert.False(t, hasDescuGravada)
	_, hasSubTotal := resumen["subTotal"]
	assert.False(t, hasSubTotal)
	_, hasReteRenta := resumen["reteRenta"]
	assert.False(t, hasReteRenta)
	_, hasIvaRete1 := resumen["ivaRete1"]
	assert.False(t, hasIvaRete1)
	_, hasIvaPerci1 := resumen["ivaPerci1"]
	assert.False(t, hasIvaPerci1)
}

// TestDebitNoteV4ResponseMapping_HaciendaSchemaCompliance tests Nota de Débito v4 compliance.
func TestDebitNoteV4ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidDebitNote()
	require.NoError(t, err)

	id, ok := doc.GetIdentification().(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(4))

	obs := "Intereses por mora período marzo"
	require.NoError(t, doc.GetSummary().SetObservations(&obs))

	mhND := response_mapper.ToMHDebitNote(doc)
	require.NotNil(t, mhND)

	jsonData, err := json.Marshal(mhND)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), ident["version"])
	assert.Equal(t, constants.NotaDebitoElectronica, ident["tipoDte"])

	_, hasExtension := rawMap["extension"]
	assert.False(t, hasExtension)

	emisor, ok := rawMap["emisor"].(map[string]interface{})
	require.True(t, ok)
	_, hasTipoEstablecimiento := emisor["tipoEstablecimiento"]
	assert.False(t, hasTipoEstablecimiento)

	items, ok := rawMap["cuerpoDocumento"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	firstItem := items[0].(map[string]interface{})
	_, hasNoGravado := firstItem["noGravado"]
	assert.True(t, hasNoGravado)
	_, hasIvaPerci := firstItem["ivaPerci"]
	assert.True(t, hasIvaPerci)
	_, hasTotalIva := firstItem["totalIva"]
	assert.True(t, hasTotalIva)
	_, hasIvaRete := firstItem["ivaRete"]
	assert.True(t, hasIvaRete)

	resumen, ok := rawMap["resumen"].(map[string]interface{})
	require.True(t, ok)
	_, hasSummaryTotalIva := resumen["totalIva"]
	assert.True(t, hasSummaryTotalIva)
	_, hasTotalNoGravado := resumen["totalNoGravado"]
	assert.True(t, hasTotalNoGravado)
	_, hasTotalPagar := resumen["totalPagar"]
	assert.True(t, hasTotalPagar)
	_, hasSummaryIvaPerci := resumen["ivaPerci"]
	assert.True(t, hasSummaryIvaPerci)
	_, hasSummaryIvaRete := resumen["ivaRete"]
	assert.True(t, hasSummaryIvaRete)
	assert.Equal(t, obs, resumen["observaciones"])

	_, hasNumPagoElectronico := resumen["numPagoElectronico"]
	assert.True(t, hasNumPagoElectronico)

	_, hasDescuNoSuj := resumen["descuNoSuj"]
	assert.False(t, hasDescuNoSuj)
	_, hasDescuExenta := resumen["descuExenta"]
	assert.False(t, hasDescuExenta)
	_, hasDescuGravada := resumen["descuGravada"]
	assert.False(t, hasDescuGravada)
	_, hasSubTotal := resumen["subTotal"]
	assert.False(t, hasSubTotal)
	_, hasReteRenta := resumen["reteRenta"]
	assert.False(t, hasReteRenta)
	_, hasIvaRete1 := resumen["ivaRete1"]
	assert.False(t, hasIvaRete1)
	_, hasIvaPerci1 := resumen["ivaPerci1"]
	assert.False(t, hasIvaPerci1)
}

// TestPhase3RequestMappers_VersionAndObservations tests request mappers versioning and observations.
func TestPhase3RequestMappers_VersionAndObservations(t *testing.T) {
	test.TestMain(t)
	issuer := fixtures.CreateDefaultIssuer()

	t.Run("RetentionRequestMapper", func(t *testing.T) {
		req := fixtures.CreatePhysicalDocumentsRetentionRequest()
		obs := "Obs Retención"
		req.Summary.Observations = &obs
		mapper := request_mapper.NewRetentionMapper()
		data, err := mapper.MapToRetentionData(req, issuer)
		require.NoError(t, err)
		assert.Equal(t, 2, data.InputDataCommon.Identification.GetVersion())
		assert.Equal(t, &obs, data.RetentionSummary.Observations)
	})

	t.Run("FSERequestMapper", func(t *testing.T) {
		req := fixtures.CreateDefaultFSERequest()
		mapper := request_mapper.NewFSEMapper()
		data, err := mapper.MapToFSEData(req, issuer)
		require.NoError(t, err)
		assert.Equal(t, 2, data.InputDataCommon.Identification.GetVersion())
	})

	t.Run("RemissionNoteRequestMapper", func(t *testing.T) {
		req := fixtures.CreateDefaultRemissionNoteRequest()
		obs := "Obs Remisión"
		req.Summary.Observations = &obs
		mapper := request_mapper.NewRemissionNoteMapper()
		data, err := mapper.MapToRemissionNoteData(req, issuer)
		require.NoError(t, err)
		assert.Equal(t, 4, data.InputDataCommon.Identification.GetVersion())
		assert.Equal(t, &obs, data.RemissionSummary.Observations)
	})

	t.Run("CreditNoteRequestMapper", func(t *testing.T) {
		req := fixtures.CreateDefaultCreditNoteRequest()
		obs := "Obs Nota Crédito"
		req.Summary.Observations = &obs
		mapper := request_mapper.NewCreditNoteMapper()
		data, err := mapper.MapToCreditNoteData(req, issuer)
		require.NoError(t, err)
		assert.Equal(t, 4, data.Identification.GetVersion())
		assert.Equal(t, &obs, data.CreditSummary.Observations)
	})

	t.Run("DebitNoteRequestMapper", func(t *testing.T) {
		req := fixtures.CreateDefaultDebitNoteRequest()
		obs := "Obs Nota Débito"
		req.Summary.Observations = &obs
		mapper := request_mapper.NewDebitNoteMapper()
		data, err := mapper.MapToDebitNoteData(req, issuer)
		require.NoError(t, err)
		assert.Equal(t, 4, data.Identification.GetVersion())
		assert.Equal(t, &obs, data.DebitSummary.Observations)
	})
}
