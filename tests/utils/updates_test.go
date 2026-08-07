package utils

import (
	"encoding/json"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseIdentificationJSON returns a minimal DTE JSON string with "identificacion" and "apendice".
func baseIdentificationJSON() string {
	return `{
		"identificacion": {
			"tipoDte": "01",
			"numeroControl": "DTE-01-00000001-000000000000001",
			"codigoGeneracion": "AAAA0000-1111-2222-3333-AAAA00001111",
			"tipoModelo": 1,
			"tipoOperacion": 1,
			"tipoContingencia": null,
			"motivoContin": null
		},
		"emisor": {"nit": "0614-010101-000-0"},
		"apendice": null
	}`
}

// TestUpdateContingencyIdentification_FromString tests updating from a JSON string.
// Values parsed from JSON become float64; values set directly keep their Go types.
func TestUpdateContingencyIdentification_FromString(t *testing.T) {
	docStr := baseIdentificationJSON()
	contiType := int8(constants.FallaServicioInternet)
	reason := "Internet outage"

	result, err := utils.UpdateContingencyIdentification(docStr, &contiType, &reason)
	require.NoError(t, err)

	identification, ok := result["identificacion"].(map[string]interface{})
	require.True(t, ok)

	// Constants assigned directly → int type
	assert.Equal(t, constants.ModeloFacturacionDiferido, identification["tipoModelo"])
	assert.Equal(t, constants.TransmisionContingencia, identification["tipoOperacion"])
	// *contiType assigned directly → int8 type
	assert.Equal(t, contiType, identification["tipoContingencia"])
	assert.Equal(t, reason, identification["motivoContin"])
}

// TestUpdateContingencyIdentification_FromBytes tests updating from a []byte.
func TestUpdateContingencyIdentification_FromBytes(t *testing.T) {
	docBytes := []byte(baseIdentificationJSON())
	contiType := int8(constants.FallaEnergiaElectrica)
	reason := "Power failure"

	result, err := utils.UpdateContingencyIdentification(docBytes, &contiType, &reason)
	require.NoError(t, err)

	identification, ok := result["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, contiType, identification["tipoContingencia"])
	assert.Equal(t, reason, identification["motivoContin"])
}

// TestUpdateContingencyIdentification_FromObject tests updating from a Go struct (map).
// When the function re-marshals the Go object to JSON and back, existing values become float64,
// but the contingency fields that are set directly keep their Go native types.
func TestUpdateContingencyIdentification_FromObject(t *testing.T) {
	doc := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"tipoDte":       "01",
			"tipoModelo":    1,
			"tipoOperacion": 1,
		},
	}

	contiType := int8(constants.NoDisponibilidadMH)
	reason := "MH down"

	result, err := utils.UpdateContingencyIdentification(doc, &contiType, &reason)
	require.NoError(t, err)

	identification, ok := result["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, contiType, identification["tipoContingencia"])
	assert.Equal(t, reason, identification["motivoContin"])
}

// TestUpdateContingencyIdentification_InvalidJSONString tests malformed JSON string.
func TestUpdateContingencyIdentification_InvalidJSONString(t *testing.T) {
	contiType := int8(1)
	reason := "test"
	_, err := utils.UpdateContingencyIdentification("{bad json}", &contiType, &reason)
	assert.Error(t, err)
}

// TestUpdateContingencyIdentification_InvalidJSONBytes tests malformed JSON bytes.
func TestUpdateContingencyIdentification_InvalidJSONBytes(t *testing.T) {
	contiType := int8(1)
	reason := "test"
	_, err := utils.UpdateContingencyIdentification([]byte("{bad}"), &contiType, &reason)
	assert.Error(t, err)
}

// TestSetReceptionStampIntoAppendix_NilAppendix tests adding stamp when appendix is null.
func TestSetReceptionStampIntoAppendix_NilAppendix(t *testing.T) {
	docStr := `{"identificacion":{"tipoDte":"01","numeroControl":"DTE-01-00000001-000000000000001","codigoGeneracion":"BBBB0000-1111-2222-3333-BBBB00001111"},"apendice":null}`
	stamp := "SELLO-RECEPCION-12345"

	result, err := utils.SetReceptionStampIntoAppendix(docStr, &stamp)
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &doc))

	appendix, ok := doc["apendice"].([]interface{})
	require.True(t, ok)
	require.Len(t, appendix, 1)

	entry := appendix[0].(map[string]interface{})
	assert.Equal(t, "Sello de recepción", entry["Etiqueta"])
	assert.Equal(t, stamp, entry["Valor"])
}

// TestSetReceptionStampIntoAppendix_ExistingAppendix tests adding stamp to existing appendix.
func TestSetReceptionStampIntoAppendix_ExistingAppendix(t *testing.T) {
	docStr := `{
		"identificacion":{"tipoDte":"01","numeroControl":"DTE-01-00000001-000000000000001","codigoGeneracion":"CCCC0000-1111-2222-3333-CCCC00001111"},
		"apendice": [
			{"Campo": "existing_field", "Etiqueta": "Existing", "Valor": "existing_value"}
		]
	}`
	stamp := "NEW-STAMP-9999"

	result, err := utils.SetReceptionStampIntoAppendix(docStr, &stamp)
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &doc))

	appendix, ok := doc["apendice"].([]interface{})
	require.True(t, ok)
	assert.Len(t, appendix, 2, "existing entry should be preserved and new one appended")

	last := appendix[1].(map[string]interface{})
	assert.Equal(t, "Sello de recepción", last["Etiqueta"])
	assert.Equal(t, stamp, last["Valor"])
}

// TestSetReceptionStampIntoAppendix_InvalidDTEType tests that missing DTE type returns error.
func TestSetReceptionStampIntoAppendix_InvalidDTEType(t *testing.T) {
	// No "identificacion.tipoDte" → DTEType will be empty → error
	docStr := `{"otro_campo": "valor"}`
	stamp := "STAMP"

	_, err := utils.SetReceptionStampIntoAppendix(docStr, &stamp)
	assert.Error(t, err)
}
