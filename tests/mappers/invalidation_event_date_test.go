package mappers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	"github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
)

// TestMapToInvalidationData_EventDateAndStamp checks that a CCF event uses its generation date and stamp date.
func TestMapToInvalidationData_EventDateAndStamp(t *testing.T) {
	test.TestMain(t)

	mapper := request_mapper.NewInvalidationMapper()
	issuer := fixtures.CreateDefaultIssuer()
	createdAt := utils.TimeNow().AddDate(0, 0, -3)

	ccf := &dte.DTEDetails{
		ID:             "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
		DTEType:        "03",
		ControlNumber:  "DTE-03-00000000-000000000000001",
		ReceptionStamp: utils.ToStringPointer("2025BBFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
		ProcessingDate: "05/11/2026 16:40:07",
		JSONData: `{
			"identificacion": {"fecEmi": "2026-11-05", "horEmi": "16:40:00"},
			"emisor": {"codActividad": "47712"},
			"receptor": {"nombre": "Empresa", "nit": "06141804941035", "nrc": "1234567"},
			"resumen": {"totalIva": 13.00}
		}`,
	}

	t.Run("CCF from a previous day uses its generation date and time", func(t *testing.T) {
		req := fixtures.CreateDefaultInvalidationRequest()
		req.Reason.Type = 2
		req.ReplacementGenerationCode = nil

		doc, err := mapper.MapToInvalidationData(req, issuer, ccf, createdAt)
		assert.NoError(t, err)

		assert.Equal(t, "2026-11-05", doc.Identification.EmissionDate.GetValue().Format("2006-01-02"))
		assert.Equal(t, "16:40:00", doc.Identification.EmissionTime.GetValue().Format("15:04:05"))
		assert.Equal(t, "2026-11-05", doc.Document.EmissionDate.GetValue().Format("2006-01-02"))
		assert.Equal(t, "2026-11-05 16:40:07", doc.Document.StampDate.Format("2006-01-02 15:04:05"))
		assert.Equal(t, "47712", doc.Document.IssuerActivity)
	})

	t.Run("Document without stamp cannot be invalidated", func(t *testing.T) {
		noStamp := *ccf
		noStamp.ReceptionStamp = nil

		_, err := mapper.MapToInvalidationData(fixtures.CreateDefaultInvalidationRequest(), issuer, &noStamp, time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DocumentWithoutStamp")
	})
}
