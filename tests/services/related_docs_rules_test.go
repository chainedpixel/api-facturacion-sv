package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func day(value string) time.Time {
	parsed, _ := time.Parse("2006-01-02", value)
	return parsed
}

// TestRelatedDocRules covers the allowed related document types and generation windows.
func TestRelatedDocRules(t *testing.T) {
	test.TestMain(t)

	t.Run("Remission note only relates FE, CCF or FEXE", func(t *testing.T) {
		assert.Nil(t, strategy.ValidateRelatedDocType(constants.NotaRemisionElectronica, constants.FacturaElectronica))
		assert.Nil(t, strategy.ValidateRelatedDocType(constants.NotaRemisionElectronica, constants.CCFElectronico))
		err := strategy.ValidateRelatedDocType(constants.NotaRemisionElectronica, constants.NotaCreditoElectronica)
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "InvalidRelatedDocDTEType")
		}
	})

	t.Run("FE and CCF relate remission notes, CL and DCL only", func(t *testing.T) {
		for _, related := range []string{constants.NotaRemisionElectronica, constants.ComprobanteLiquidacionElectronico, constants.DocContableLiquidacionElectronico} {
			assert.True(t, constants.ValidInvoiceDTETypesRelateDoc[related], "FE should accept %s", related)
			assert.True(t, constants.ValidCCFDTETypesRelateDoc[related], "CCF should accept %s", related)
		}
		for _, related := range []string{constants.FacturaElectronica, constants.CCFElectronico, constants.NotaCreditoElectronica} {
			assert.False(t, constants.ValidInvoiceDTETypesRelateDoc[related], "FE should reject %s", related)
			assert.False(t, constants.ValidCCFDTETypesRelateDoc[related], "CCF should reject %s", related)
		}
	})

	t.Run("FE relating a remission note within its period plus three days", func(t *testing.T) {
		nr := day("2026-11-20")
		assert.Nil(t, strategy.ValidateRelatedDocPeriod(constants.FacturaElectronica, constants.NotaRemisionElectronica, nr, day("2026-12-03")))
		err := strategy.ValidateRelatedDocPeriod(constants.FacturaElectronica, constants.NotaRemisionElectronica, nr, day("2026-12-04"))
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "RelatedDocOutOfPeriod")
		}
	})

	t.Run("CCF relating a DCL within its period", func(t *testing.T) {
		dcl := day("2026-11-20")
		assert.Nil(t, strategy.ValidateRelatedDocPeriod(constants.CCFElectronico, constants.DocContableLiquidacionElectronico, dcl, day("2026-11-30")))
		assert.NotNil(t, strategy.ValidateRelatedDocPeriod(constants.CCFElectronico, constants.DocContableLiquidacionElectronico, dcl, day("2026-12-01")))
	})

	t.Run("Credit note over a CCF within and after three months", func(t *testing.T) {
		ccf := day("2026-08-12")
		assert.Nil(t, strategy.ValidateRelatedDocPeriod(constants.NotaCreditoElectronica, constants.CCFElectronico, ccf, day("2026-11-12")))
		err := strategy.ValidateRelatedDocPeriod(constants.NotaCreditoElectronica, constants.CCFElectronico, ccf, day("2026-11-13"))
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "RelatedDocOutOfPeriod")
		}
		assert.NotNil(t, strategy.ValidateRelatedDocPeriod(constants.NotaDebitoElectronica, constants.ComprobanteRetencionElectronico, ccf, day("2026-11-13")))
	})

	t.Run("Unknown related date is not validated", func(t *testing.T) {
		assert.Nil(t, strategy.ValidateRelatedDocPeriod(constants.NotaCreditoElectronica, constants.CCFElectronico, time.Time{}, day("2026-11-13")))
	})
}

// TestRelatedDocPeriod_ClampedWindow checks the credit note window ends on the last day of a shorter month.
func TestRelatedDocPeriod_ClampedWindow(t *testing.T) {
	test.TestMain(t)

	ccf := day("2026-11-30")
	assert.Nil(t, strategy.ValidateRelatedDocPeriod(constants.NotaCreditoElectronica, constants.CCFElectronico, ccf, day("2027-02-28")))
	err := strategy.ValidateRelatedDocPeriod(constants.NotaCreditoElectronica, constants.CCFElectronico, ccf, day("2027-03-01"))
	if assert.NotNil(t, err) {
		test.AssertErrorCode(t, err, "RelatedDocOutOfPeriod")
	}
}
