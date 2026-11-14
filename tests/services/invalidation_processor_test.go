package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/processors"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

// TestInvalidationProcessor_RequestBody checks that the invalidation body only has ambiente, idEnvio, version and documento.
func TestInvalidationProcessor_RequestBody(t *testing.T) {
	test.TestMain(t)

	numeroControl := "DTE-03-M001P001-000000000000001"
	event := &structs.InvalidationResponse{
		Identificacion: structs.InvalidationIdentification{
			Version:          3,
			Ambiente:         "00",
			CodigoGeneracion: "AB123456-1234-4234-9234-AB1234567890",
			FecEmi:           "2026-11-12",
			HorEmi:           "10:00:00",
		},
		Documento: structs.DocumentResponse{
			TipoDte:          "03",
			CodigoGeneracion: "CD123456-1234-4234-9234-AB1234567890",
			NumeroControl:    &numeroControl,
		},
	}

	req, err := (&processors.InvalidationProcessor{}).ProcessRequest("signed.jws.value", event)
	assert.NoError(t, err)

	body, err := json.Marshal(req)
	assert.NoError(t, err)

	var fields map[string]interface{}
	assert.NoError(t, json.Unmarshal(body, &fields))

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, []string{"ambiente", "idEnvio", "version", "documento"}, keys)
	assert.Equal(t, float64(3), fields["version"])
}
