package services

import (
	"encoding/json"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContingencyEvent_JSONSerialization verifies that ContingencyEvent serializes
// and deserializes to/from JSON with the expected field names.
func TestContingencyEvent_JSONSerialization(t *testing.T) {
	test.TestMain(t)

	reasonText := "Internet service failure"
	event := models.ContingencyEvent{
		Identification: models.ContingencyIdentification{
			Version:          4,
			Ambient:          "00",
			GenerationCode:   "ABCD1234-0000-0000-0000-ABCD12340000",
			TransmissionDate: "2026-03-09",
			TransmissionTime: "10:00:00",
		},
		Issuer: models.ContingencyIssuer{
			NIT:                  "0614-010101-000-0",
			Name:                 "Test Company",
			ResponsibleName:      "John Doe",
			ResponsibleDocType:   "13",
			ResponsibleDocNumber: "00000000-0",
			EstablishmentType:    "01",
			Phone:                "22222222",
			Email:                "test@example.com",
		},
		DTEDetails: []models.DTEDetail{
			{ItemNumber: 1, GenerationCode: "ABCD1234-0000-0000-0000-ABCD12340000", DocumentType: "01"},
		},
		Reason: models.ContingencyReason{
			StartDate:         "2026-03-09",
			EndDate:           "2026-03-09",
			StartTime:         "09:00:00",
			EndTime:           "10:00:00",
			ContingencyType:   3,
			ContingencyReason: &reasonText,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	// Verify JSON field names match Hacienda spec
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Contains(t, raw, "identificacion")
	assert.Contains(t, raw, "emisor")
	assert.Contains(t, raw, "detalleDTE")
	assert.Contains(t, raw, "motivo")

	// Round-trip
	var decoded models.ContingencyEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, event.Identification.Version, decoded.Identification.Version)
	assert.Equal(t, event.Identification.Ambient, decoded.Identification.Ambient)
	assert.Equal(t, event.Identification.GenerationCode, decoded.Identification.GenerationCode)
	assert.Equal(t, event.Issuer.NIT, decoded.Issuer.NIT)
	assert.Len(t, decoded.DTEDetails, 1)
	assert.Equal(t, event.DTEDetails[0].ItemNumber, decoded.DTEDetails[0].ItemNumber)
	assert.Equal(t, event.Reason.ContingencyType, decoded.Reason.ContingencyType)
}

// TestContingencyIdentification_Fields tests the identification sub-struct.
func TestContingencyIdentification_Fields(t *testing.T) {
	test.TestMain(t)

	id := models.ContingencyIdentification{
		Version:          1,
		Ambient:          "01",
		GenerationCode:   "AAAA0000-BBBB-CCCC-DDDD-EEEE11112222",
		TransmissionDate: "2026-01-15",
		TransmissionTime: "14:30:00",
	}

	data, err := json.Marshal(id)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, float64(1), raw["version"])
	assert.Equal(t, "01", raw["ambiente"])
	assert.Equal(t, "AAAA0000-BBBB-CCCC-DDDD-EEEE11112222", raw["codigoGeneracion"])
	assert.Equal(t, "2026-01-15", raw["fTransmision"])
	assert.Equal(t, "14:30:00", raw["hTransmision"])
}

// TestContingencyIssuer_OptionalFields verifies that nil pointer fields are omitted.
func TestContingencyIssuer_OptionalFields(t *testing.T) {
	test.TestMain(t)

	issuer := models.ContingencyIssuer{
		NIT:                  "0614-010101-000-0",
		Name:                 "Company",
		ResponsibleName:      "Jane",
		ResponsibleDocType:   "13",
		ResponsibleDocNumber: "99999999-9",
		EstablishmentType:    "02",
		Phone:                "22334455",
		Email:                "jane@example.com",
		EstablishmentCodeMH:  nil,
		POSCode:              nil,
	}

	data, err := json.Marshal(issuer)
	require.NoError(t, err)

	// Nil pointer fields should be present as null (they are *string)
	assert.Contains(t, string(data), "codEstableMH")
	assert.Contains(t, string(data), "codPuntoVentaMH")

	// With values
	code := "MH-001"
	pos := "POS-001"
	issuer.EstablishmentCodeMH = &code
	issuer.POSCode = &pos

	data2, err := json.Marshal(issuer)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data2, &raw))
	assert.Equal(t, "MH-001", raw["codEstableMH"])
	assert.Equal(t, "POS-001", raw["codPuntoVentaMH"])
}

// TestDTEDetail_JSONFields verifies the DTE detail struct field names.
func TestDTEDetail_JSONFields(t *testing.T) {
	test.TestMain(t)

	detail := models.DTEDetail{
		ItemNumber:     5,
		GenerationCode: "FFFF0000-1111-2222-3333-444455556666",
		DocumentType:   "03",
	}

	data, err := json.Marshal(detail)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, float64(5), raw["noItem"])
	assert.Equal(t, "FFFF0000-1111-2222-3333-444455556666", raw["codigoGeneracion"])
	assert.Equal(t, "03", raw["tipoDoc"])
}

// TestContingencyReason_JSONFields verifies the reason struct field names.
func TestContingencyReason_JSONFields(t *testing.T) {
	test.TestMain(t)

	reasonDesc := "System connection failure"
	reason := models.ContingencyReason{
		StartDate:         "2026-03-01",
		EndDate:           "2026-03-09",
		StartTime:         "08:00:00",
		EndTime:           "17:00:00",
		ContingencyType:   2,
		ContingencyReason: &reasonDesc,
	}

	data, err := json.Marshal(reason)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "2026-03-01", raw["fInicio"])
	assert.Equal(t, "2026-03-09", raw["fFin"])
	assert.Equal(t, "08:00:00", raw["hInicio"])
	assert.Equal(t, "17:00:00", raw["hFin"])
	assert.Equal(t, float64(2), raw["tipoContingencia"])
	assert.Equal(t, "System connection failure", raw["motivoContingencia"])
}

// TestContingencyReason_NilReason verifies null serialization on ContingencyReason field.
func TestContingencyReason_NilReason(t *testing.T) {
	test.TestMain(t)

	reason := models.ContingencyReason{
		StartDate:         "2026-03-01",
		EndDate:           "2026-03-09",
		StartTime:         "08:00:00",
		EndTime:           "17:00:00",
		ContingencyType:   1,
		ContingencyReason: nil,
	}

	data, err := json.Marshal(reason)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "motivoContingencia")
	assert.Nil(t, raw["motivoContingencia"])
}

// TestHaciendaContingencyRequest_JSONFields verifies the Hacienda request wrapper.
func TestHaciendaContingencyRequest_JSONFields(t *testing.T) {
	test.TestMain(t)

	req := models.HaciendaContingencyRequest{
		NIT:      "0614-010101-000-0",
		Document: `{"identificacion":{"version":4}}`,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "0614-010101-000-0", raw["nit"])
	assert.Equal(t, `{"identificacion":{"version":4}}`, raw["documento"])
}
