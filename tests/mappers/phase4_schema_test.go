package mappers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	contingencyModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/batch"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestInvoiceDTE() *dte.DTEDetails {
	return &dte.DTEDetails{
		ID:             "FF54E9DB-79C3-42CE-B432-EC522C97EFB9",
		DTEType:        "01",
		ControlNumber:  "DTE-01-00000000-000000000000001",
		ReceptionStamp: utils.ToStringPointer("2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
		JSONData: `{
			"receptor": {
				"nombre": "Cliente Ejemplo",
				"telefono": "22123456",
				"correo": "cliente@example.com",
				"tipoDocumento": "13",
				"numDocumento": "01234567-8"
			},
			"resumen": {
				"totalIva": 13.00
			}
		}`,
	}
}

// TestInvalidationV3ResponseMapping_HaciendaSchemaCompliance verifies Invalidation v3 schema compliance.
func TestInvalidationV3ResponseMapping_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildInvalidationWithReplacement()
	require.NoError(t, err)

	mhInv := response_mapper.ToMHInvalidation(doc)
	require.NotNil(t, mhInv)

	jsonData, err := json.Marshal(mhInv)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), ident["version"])
	assert.Equal(t, "00", ident["ambiente"])
	_, hasFecEmi := ident["fecEmi"]
	assert.True(t, hasFecEmi)
	_, hasHorEmi := ident["horEmi"]
	assert.True(t, hasHorEmi)
	_, hasFecAnula := ident["fecAnula"]
	assert.False(t, hasFecAnula)
	_, hasHorAnula := ident["horAnula"]
	assert.False(t, hasHorAnula)
	_, hasFusion := ident["fusion"]
	assert.True(t, hasFusion)

	emisor, ok := rawMap["emisor"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "E001", emisor["codEstableMH"])
	assert.Equal(t, "P001", emisor["codPuntoVentaMH"])
	_, hasCodEstable := emisor["codEstable"]
	assert.True(t, hasCodEstable)
	_, hasCodPuntoVenta := emisor["codPuntoVenta"]
	assert.True(t, hasCodPuntoVenta)
	_, hasTipoEstablecimiento := emisor["tipoEstablecimiento"]
	assert.False(t, hasTipoEstablecimiento)
	_, hasNomEstablecimiento := emisor["nomEstablecimiento"]
	assert.False(t, hasNomEstablecimiento)

	documento, ok := rawMap["documento"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, constants.FacturaElectronica, documento["tipoDte"])
	_, hasNumeroControl := documento["numeroControl"]
	assert.True(t, hasNumeroControl)
	_, hasMontoIva := documento["montoIva"]
	assert.False(t, hasMontoIva)
	_, hasSelloRecibido := documento["selloRecibido"]
	assert.True(t, hasSelloRecibido)
	_, hasCodigoGeneracionR := documento["codigoGeneracionR"]
	assert.True(t, hasCodigoGeneracionR)

	motivo, ok := rawMap["motivo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), motivo["tipoAnulacion"])
	assert.Equal(t, "JUAN RESPONSABLE", motivo["nombreResponsable"])
	assert.Equal(t, "ANA SOLICITANTE", motivo["nombreSolicita"])
}

// TestContingencyV4_HaciendaSchemaCompliance verifies Contingency v4 schema compliance.
func TestContingencyV4_HaciendaSchemaCompliance(t *testing.T) {
	test.TestMain(t)

	codEstableMH := "M001"
	codPuntoVentaMH := "P001"
	reasonDesc := "Falla de conectividad a servidores MH"

	event := contingencyModels.ContingencyEvent{
		Identification: contingencyModels.ContingencyIdentification{
			Version:          4,
			Ambient:          "00",
			GenerationCode:   "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
			TransmissionDate: "2026-03-09",
			TransmissionTime: "10:00:00",
		},
		Issuer: contingencyModels.ContingencyIssuer{
			NIT:                  "06140101010000",
			Name:                 "Empresa de Prueba S.A. de C.V.",
			ResponsibleName:      "Carlos Gomez",
			ResponsibleDocType:   "13",
			ResponsibleDocNumber: "01234567-8",
			EstablishmentType:    "01",
			Phone:                "22223333",
			Email:                "contacto@empresa.com",
			EstablishmentCodeMH:  &codEstableMH,
			POSCode:              &codPuntoVentaMH,
		},
		DTEDetails: []contingencyModels.DTEDetail{
			{
				ItemNumber:     1,
				DocumentType:   "01",
				GenerationCode: "B2C3D4E5-F6A7-8901-BCDE-F12345678901",
			},
		},
		Reason: contingencyModels.ContingencyReason{
			StartDate:         "2026-03-09",
			EndDate:           "2026-03-09",
			StartTime:         "08:00:00",
			EndTime:           "10:00:00",
			ContingencyType:   2,
			ContingencyReason: &reasonDesc,
		},
	}

	jsonData, err := json.Marshal(event)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	ident, ok := rawMap["identificacion"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), ident["version"])
	assert.Equal(t, "00", ident["ambiente"])
	assert.Equal(t, "2026-03-09", ident["fTransmision"])
	assert.Equal(t, "10:00:00", ident["hTransmision"])

	emisor, ok := rawMap["emisor"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "06140101010000", emisor["nit"])
	assert.Equal(t, "P001", emisor["codPuntoVentaMH"])
	assert.Equal(t, "M001", emisor["codEstableMH"])
	_, hasOldPOS := emisor["codPuntoVenta"]
	assert.False(t, hasOldPOS)

	detalleDTE, ok := rawMap["detalleDTE"].([]interface{})
	require.True(t, ok)
	require.Len(t, detalleDTE, 1)
	firstDetail := detalleDTE[0].(map[string]interface{})
	assert.Equal(t, float64(1), firstDetail["noItem"])
	assert.Equal(t, "01", firstDetail["tipoDoc"])

	motivo, ok := rawMap["motivo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), motivo["tipoContingencia"])
	assert.Equal(t, reasonDesc, motivo["motivoContingencia"])
}

// TestContingencyV4_NilContingencyReasonSerialization verifies nil reason serializes to JSON null.
func TestContingencyV4_NilContingencyReasonSerialization(t *testing.T) {
	test.TestMain(t)

	reason := contingencyModels.ContingencyReason{
		StartDate:         "2026-03-09",
		EndDate:           "2026-03-09",
		StartTime:         "08:00:00",
		EndTime:           "10:00:00",
		ContingencyType:   1,
		ContingencyReason: nil,
	}

	jsonData, err := json.Marshal(reason)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonData, &rawMap))

	_, hasReason := rawMap["motivoContingencia"]
	assert.True(t, hasReason)
	assert.Nil(t, rawMap["motivoContingencia"])
}

// TestBatchTransmitter_GetDTEVersionResolution tests batch DTE version resolution.
func TestBatchTransmitter_GetDTEVersionResolution(t *testing.T) {
	test.TestMain(t)

	service := &batch.BatchTransmitterService{}

	tests := []struct {
		dteType         string
		expectedVersion int
	}{
		{constants.FacturaElectronica, 2},
		{constants.ComprobanteRetencionElectronico, 2},
		{constants.FacturaSujetoExcluidoElectronica, 2},
		{constants.FacturaExportacionElectronica, 3},
		{constants.CCFElectronico, 4},
		{constants.NotaRemisionElectronica, 4},
		{constants.NotaCreditoElectronica, 4},
		{constants.NotaDebitoElectronica, 4},
		{"99", 2},
	}

	for _, tt := range tests {
		t.Run(tt.dteType, func(t *testing.T) {
			actual := service.GetDTEVersion(tt.dteType)
			assert.Equal(t, tt.expectedVersion, actual)
		})
	}
}

// TestInvalidationRequestMapper_Version3Resolution verifies request mapper assigns version 3.
func TestInvalidationRequestMapper_Version3Resolution(t *testing.T) {
	test.TestMain(t)

	req := fixtures.CreateDefaultInvalidationRequest()
	issuer := fixtures.CreateDefaultIssuer()
	baseDTE := createTestInvoiceDTE()

	mapper := request_mapper.NewInvalidationMapper()
	doc, err := mapper.MapToInvalidationData(req, issuer, baseDTE, time.Now())
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Identification)
	assert.Equal(t, 3, doc.Identification.Version.GetValue())
}
