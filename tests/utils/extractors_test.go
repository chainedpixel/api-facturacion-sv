package utils

import (
	"encoding/json"
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dteJSONWithIdentification returns a minimal DTE JSON string.
func dteJSONWithIdentification(tipoDte, numeroControl, codigoGeneracion, nit string) string {
	return `{"identificacion":{"tipoDte":"` + tipoDte + `","numeroControl":"` + numeroControl + `","codigoGeneracion":"` + codigoGeneracion + `"},"emisor":{"nit":"` + nit + `"}}`
}

// TestExtractAuxiliarIdentification_FromObject tests extraction from a Go struct.
func TestExtractAuxiliarIdentification_FromObject(t *testing.T) {
	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"tipoDte":          "01",
			"numeroControl":    "DTE-01-00000001-000000000000001",
			"codigoGeneracion": "AAAA1111-2222-3333-4444-BBBB55556666",
		},
		"emisor": map[string]interface{}{
			"nit": "0614-010101-000-0",
		},
	}

	result, err := utils.ExtractAuxiliarIdentification(doc)
	require.NoError(t, err)

	assert.Equal(t, "01", result.Identification.DTEType)
	assert.Equal(t, "DTE-01-00000001-000000000000001", result.Identification.ControlNumber)
	assert.Equal(t, "AAAA1111-2222-3333-4444-BBBB55556666", result.Identification.GenerationCode)
	assert.Equal(t, "0614-010101-000-0", result.Issuer.NIT)
}

// TestExtractAuxiliarIdentification_PartialFields tests that missing fields result in zero values.
func TestExtractAuxiliarIdentification_PartialFields(t *testing.T) {
	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"tipoDte": "03",
		},
	}

	result, err := utils.ExtractAuxiliarIdentification(doc)
	require.NoError(t, err)

	assert.Equal(t, "03", result.Identification.DTEType)
	assert.Empty(t, result.Identification.ControlNumber)
	assert.Empty(t, result.Identification.GenerationCode)
	assert.Empty(t, result.Issuer.NIT)
}

// TestExtractAuxiliarIdentification_UnmarshalError tests that invalid input returns an error.
func TestExtractAuxiliarIdentification_UnmarshalError(t *testing.T) {
	// A channel cannot be marshalled to JSON
	_, err := utils.ExtractAuxiliarIdentification(make(chan int))
	assert.Error(t, err)
}

// TestExtractAuxiliarIdentificationFromStringJSON_Valid tests parsing from a JSON string.
func TestExtractAuxiliarIdentificationFromStringJSON_Valid(t *testing.T) {
	raw := dteJSONWithIdentification("01", "DTE-01-00000001-000000000000001", "CCCC2222-0000-0000-0000-CCCC22220000", "0614-999999-000-0")

	result, err := utils.ExtractAuxiliarIdentificationFromStringJSON(raw)
	require.NoError(t, err)

	assert.Equal(t, "01", result.Identification.DTEType)
	assert.Equal(t, "DTE-01-00000001-000000000000001", result.Identification.ControlNumber)
	assert.Equal(t, "CCCC2222-0000-0000-0000-CCCC22220000", result.Identification.GenerationCode)
	assert.Equal(t, "0614-999999-000-0", result.Issuer.NIT)
}

// TestExtractAuxiliarIdentificationFromStringJSON_InvalidJSON tests malformed JSON.
func TestExtractAuxiliarIdentificationFromStringJSON_InvalidJSON(t *testing.T) {
	_, err := utils.ExtractAuxiliarIdentificationFromStringJSON("{not valid json}")
	assert.Error(t, err)
}

// TestExtractSummaryTotalAmounts_FromObject tests summary extraction from a struct.
func TestExtractSummaryTotalAmounts_FromObject(t *testing.T) {
	doc := map[string]interface{}{
		"resumen": map[string]interface{}{
			"totalGravada": 100.50,
			"totalExenta":  25.00,
			"totalNoSuj":   10.75,
		},
	}

	result, err := utils.ExtractSummaryTotalAmounts(doc)
	require.NoError(t, err)

	assert.InDelta(t, 100.50, result.Summary.TotalTaxed, 0.001)
	assert.InDelta(t, 25.00, result.Summary.TotalExempt, 0.001)
	assert.InDelta(t, 10.75, result.Summary.TotalNotSubject, 0.001)
}

// TestExtractSummaryTotalAmountsFromStringJSON_Valid tests summary from JSON string.
func TestExtractSummaryTotalAmountsFromStringJSON_Valid(t *testing.T) {
	raw := `{"resumen":{"totalGravada":50.0,"totalExenta":0.0,"totalNoSuj":5.0}}`

	result, err := utils.ExtractSummaryTotalAmountsFromStringJSON(raw)
	require.NoError(t, err)

	assert.InDelta(t, 50.0, result.Summary.TotalTaxed, 0.001)
	assert.InDelta(t, 0.0, result.Summary.TotalExempt, 0.001)
	assert.InDelta(t, 5.0, result.Summary.TotalNotSubject, 0.001)
}

// TestExtractSummaryTotalAmountsFromStringJSON_Invalid tests malformed JSON.
func TestExtractSummaryTotalAmountsFromStringJSON_Invalid(t *testing.T) {
	_, err := utils.ExtractSummaryTotalAmountsFromStringJSON("{bad json}")
	assert.Error(t, err)
}

// TestExtractRelatedDocAndItemsFromStringJSON_Valid tests related docs extraction.
func TestExtractRelatedDocAndItemsFromStringJSON_Valid(t *testing.T) {
	raw := `{
		"documentoRelacionado": [
			{"tipoGeneracion": 1, "numeroDocumento": "DOC-001"},
			{"tipoGeneracion": 2, "numeroDocumento": "DOC-002"}
		],
		"cuerpoDocumento": [
			{"ventaGravada": 100.0, "ventaExenta": 0.0, "ventaNoSuj": 0.0, "numeroDocumento": "DOC-001"}
		]
	}`

	result := utils.ExtractRelatedDocAndItemsFromStringJSON(raw)

	require.Len(t, result.RelatedDocs, 2)
	assert.Equal(t, 1, result.RelatedDocs[0].GenerationType)
	assert.Equal(t, "DOC-001", result.RelatedDocs[0].DocumentNumber)
	require.Len(t, result.Items, 1)
	assert.InDelta(t, 100.0, result.Items[0].TaxedAmount, 0.001)
}

// TestExtractRelatedDocAndItemsFromStringJSON_Empty tests empty/missing fields.
func TestExtractRelatedDocAndItemsFromStringJSON_Empty(t *testing.T) {
	raw := `{}`

	result := utils.ExtractRelatedDocAndItemsFromStringJSON(raw)

	assert.Empty(t, result.RelatedDocs)
	assert.Empty(t, result.Items)
}

// TestExtractDTEReceiverFromString_Valid tests receiver extraction.
func TestExtractDTEReceiverFromString_Valid(t *testing.T) {
	raw := `{"receptor":{"nit":"0614-123456-789-0"}}`

	result, err := utils.ExtractDTEReceiverFromString(raw)
	require.NoError(t, err)

	assert.Equal(t, "0614-123456-789-0", result.Receiver.NIT)
}

// TestExtractDTEReceiverFromString_Invalid tests malformed JSON.
func TestExtractDTEReceiverFromString_Invalid(t *testing.T) {
	_, err := utils.ExtractDTEReceiverFromString("{bad json}")
	assert.Error(t, err)
}

// TestExtractAuxiliarIdentification_RawJSON tests extraction from json.RawMessage.
func TestExtractAuxiliarIdentification_RawJSON(t *testing.T) {
	raw := json.RawMessage(`{"identificacion":{"tipoDte":"14","numeroControl":"DTE-14-CCCAAAAA-000000000000001","codigoGeneracion":"DDDD0000-EEEE-FFFF-0000-DDDD0000EEEE"},"emisor":{"nit":"0614-AABBCC-001-0"}}`)

	result, err := utils.ExtractAuxiliarIdentification(raw)
	require.NoError(t, err)

	assert.Equal(t, "14", result.Identification.DTEType)
	assert.Equal(t, "DDDD0000-EEEE-FFFF-0000-DDDD0000EEEE", result.Identification.GenerationCode)
}
