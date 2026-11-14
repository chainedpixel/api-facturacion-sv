package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/calendar"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/fse/fse_models"
	fseStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/fse/validator/strategy"
	invalidationPolicy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/policy"
	invalidationStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
)

func advCodes(err *dte_errors.DTEError) []string {
	if err == nil {
		return nil
	}
	if len(err.BusinessErrors) == 0 {
		return []string{err.GetCode()}
	}
	var codes []string
	for _, nested := range err.BusinessErrors {
		codes = append(codes, advCodes(nested)...)
	}
	return codes
}

func advTranslated(t *testing.T, section, key string, params ...interface{}) string {
	t.Helper()
	msg := config.Translate(section+"."+strings.ToLower(key), params...)
	assert.False(t, strings.HasPrefix(msg, "["), "key %s.%s not translated: %s", section, key, msg)
	assert.NotContains(t, msg, "No se encontró documentación")
	assert.NotContains(t, msg, "unknownerror")
	assert.NotContains(t, msg, "%!")
	assert.NotContains(t, msg, "%d")
	assert.NotContains(t, msg, "%s")
	return msg
}

func advStr(value string) *string { return &value }

// TestAdversarialInvalidationMatrix attacks every DTE type x reason type x replacement presence combination.
func TestAdversarialInvalidationMatrix(t *testing.T) {
	test.TestMain(t)

	dteTypes := []string{"01", "03", "04", "05", "06", "07", "08", "09", "11", "14", "15"}
	for _, dteType := range dteTypes {
		for reasonType := 1; reasonType <= 3; reasonType++ {
			for _, withReplacement := range []bool{false, true} {
				name := fmt.Sprintf("dte%s_type%d_replacement%t", dteType, reasonType, withReplacement)
				t.Run(name, func(t *testing.T) {
					want := ""
					nullOnly := dteType == "05" || dteType == "08"
					switch {
					case reasonType == 2 && withReplacement:
						want = "InvalidInvalidationType2"
					case reasonType == 2:
					case nullOnly && withReplacement:
						want = "ReplacementCodeNotAllowedForType"
					case !nullOnly && !withReplacement:
						want = "InvalidReplacementCode1And3"
					}

					doc, err := fixtures.NewInvalidationBuilder().
						AddIdentification().AddIssuer().AddInvalidatedDocument().
						AddInvalidationReason(reasonType).
						BuildWithoutValidation()
					require.NoError(t, err)
					docType, err := document.NewDTEType(dteType)
					require.NoError(t, err)
					doc.Document.Type = *docType
					doc.Document.ReplacementCode = nil
					if withReplacement {
						doc.Document.ReplacementCode, err = identification.NewGenerationCode()
						require.NoError(t, err)
					}
					if reasonType == 3 && doc.Reason.Reason == nil {
						reasonText := &document.InvalidationReason{Value: "motivo"}
						doc.Reason.Reason = reasonText
					}

					got := (&invalidationStrategy.InvalidationReasonStrategy{Document: doc}).Validate()
					if want == "" {
						assert.Nil(t, got)
						return
					}
					require.NotNil(t, got)
					assert.Equal(t, []string{strings.ToUpper(want)}, advCodes(got))
				})
			}
		}
	}
}

// TestAdversarialInvalidationReasonBoundaries covers a missing reason for type 3 and the policy edge values.
func TestAdversarialInvalidationReasonBoundaries(t *testing.T) {
	test.TestMain(t)

	t.Run("type 3 without reason text is rejected before replacement checks", func(t *testing.T) {
		doc, err := fixtures.NewInvalidationBuilder().AddIdentification().AddIssuer().AddInvalidatedDocument().
			AddInvalidationReason(3).BuildWithoutValidation()
		require.NoError(t, err)
		doc.Reason.Reason = nil
		doc.Document.ReplacementCode = nil
		got := (&invalidationStrategy.InvalidationReasonStrategy{Document: doc}).Validate()
		require.NotNil(t, got)
		assert.Equal(t, []string{"REQUIREDFIELD"}, advCodes(got))
	})

	t.Run("nil reason", func(t *testing.T) {
		doc, err := fixtures.NewInvalidationBuilder().AddIdentification().AddIssuer().AddInvalidatedDocument().
			AddInvalidationReason(1).BuildWithoutValidation()
		require.NoError(t, err)
		doc.Reason = nil
		got := (&invalidationStrategy.InvalidationReasonStrategy{Document: doc}).Validate()
		require.NotNil(t, got)
		assert.Equal(t, []string{"REQUIREDFIELD"}, advCodes(got))
	})

	t.Run("RequiresReplacement for unknown and empty types", func(t *testing.T) {
		assert.True(t, invalidationPolicy.RequiresReplacement(""))
		assert.True(t, invalidationPolicy.RequiresReplacement("99"))
		assert.False(t, invalidationPolicy.RequiresReplacement("05"))
		assert.False(t, invalidationPolicy.RequiresReplacement("08"))
		assert.True(t, invalidationPolicy.RequiresReplacement("09"))
		assert.True(t, invalidationPolicy.RequiresReplacement("06"))
	})
}

// TestAdversarialBatch1KeysTranslate guarantees every key raised by this batch resolves with the exact params.
func TestAdversarialBatch1KeysTranslate(t *testing.T) {
	test.TestMain(t)

	for _, tc := range []struct {
		section string
		key     string
		params  []interface{}
	}{
		{"validation_errors", "InvalidReplacementCode1And3", nil},
		{"validation_errors", "InvalidInvalidationType2", nil},
		{"validation_errors", "ReplacementCodeNotAllowedForType", []interface{}{"05"}},
		{"validation_errors", "UnsupportedTaxCode", []interface{}{"D1"}},
		{"validation_errors", "InvalidFSEItemType", []interface{}{4}},
		{"validation_errors", "InvalidAssociatedDocumentCodeForType", []interface{}{4, "01"}},
		{"validation_errors", "InvalidPaymentTotal", []interface{}{1.0, 2.0}},
		{"validation_errors", "CreditPaymentCodeNotAllowed", []interface{}{"03"}},
		{"validation_errors", "CreditPaymentReferenceNotAllowed", []interface{}{"REF"}},
		{"validation_errors", "CreditPaymentTermRequired", nil},
		{"validation_errors", "CreditPaymentPeriodRequired", nil},
		{"validation_errors", "PaymentCodeRequired", []interface{}{1}},
		{"validation_errors", "UnreferencedRelatedDoc", []interface{}{"DOC"}},
		{"validation_errors", "NoPaymentDueRequiresCashCondition", []interface{}{2}},
		{"validation_errors", "NoPaymentDueRequiresCode99", nil},
		{"validation_errors", "InvalidDebitNoteNotSubjectAmount", []interface{}{10.0, "CCF", 5.0}},
		{"service_errors", "StampedDocumentNotPersisted", []interface{}{"GEN", "STAMP"}},
		{"service_errors", "AdditionalOperationsFailed", []interface{}{"GEN"}},
		{"service_errors", "InvalidInvalidationType2", nil},
	} {
		t.Run(tc.section+"."+tc.key, func(t *testing.T) {
			advTranslated(t, tc.section, tc.key, tc.params...)
		})
	}
}

// TestAdversarialAddMonthsClamped attacks month ends, leap years, zones and unusual month counts.
func TestAdversarialAddMonthsClamped(t *testing.T) {
	sv := time.FixedZone("SV", -6*3600)
	utc := time.UTC
	tokyo := time.FixedZone("JST", 9*3600)

	cases := []struct {
		name   string
		from   time.Time
		months int
		want   string
	}{
		{"31-Jan +1 non-leap", time.Date(2027, 1, 31, 10, 0, 0, 0, utc), 1, "2027-02-28"},
		{"31-Jan +1 leap", time.Date(2028, 1, 31, 10, 0, 0, 0, utc), 1, "2028-02-29"},
		{"29-Feb leap +12", time.Date(2028, 2, 29, 10, 0, 0, 0, utc), 12, "2029-02-28"},
		{"29-Feb leap +48", time.Date(2028, 2, 29, 10, 0, 0, 0, utc), 48, "2032-02-29"},
		{"30-Nov +3", time.Date(2026, 11, 30, 10, 0, 0, 0, utc), 3, "2027-02-28"},
		{"30-Nov +3 leap target", time.Date(2027, 11, 30, 10, 0, 0, 0, utc), 3, "2028-02-29"},
		{"31-Dec +3", time.Date(2026, 12, 31, 10, 0, 0, 0, utc), 3, "2027-03-31"},
		{"31-Dec +2", time.Date(2026, 12, 31, 10, 0, 0, 0, utc), 2, "2027-02-28"},
		{"31-Dec +24 pharma", time.Date(2026, 12, 31, 10, 0, 0, 0, utc), 24, "2028-12-31"},
		{"31-Aug +1", time.Date(2026, 8, 31, 10, 0, 0, 0, utc), 1, "2026-09-30"},
		{"31-Mar -1 leap", time.Date(2028, 3, 31, 10, 0, 0, 0, utc), -1, "2028-02-29"},
		{"31-Mar -1 non-leap", time.Date(2027, 3, 31, 10, 0, 0, 0, utc), -1, "2027-02-28"},
		{"zero months", time.Date(2027, 5, 31, 10, 0, 0, 0, utc), 0, "2027-05-31"},
		{"1-Jan -1", time.Date(2027, 1, 1, 10, 0, 0, 0, utc), -1, "2026-12-01"},
		{"SV zone 31-Jan", time.Date(2027, 1, 31, 23, 59, 59, 0, sv), 1, "2027-02-28"},
		{"Tokyo zone 31-Jan", time.Date(2027, 1, 31, 0, 0, 1, 0, tokyo), 1, "2027-02-28"},
		{"century non-leap 2100", time.Date(2099, 11, 30, 0, 0, 0, 0, utc), 3, "2100-02-28"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calendar.AddMonthsClamped(tc.from, tc.months)
			assert.Equal(t, tc.want, got.Format("2006-01-02"))
			assert.Equal(t, tc.from.Location(), got.Location())
			assert.Equal(t, tc.from.Hour(), got.Hour())
			assert.Equal(t, tc.from.Minute(), got.Minute())
			assert.Equal(t, tc.from.Second(), got.Second())
		})
	}

	t.Run("never lands past the target month", func(t *testing.T) {
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, utc)
		for offset := 0; offset < 800; offset++ {
			from := start.AddDate(0, 0, offset)
			for _, months := range []int{1, 3, 12, 24} {
				got := calendar.AddMonthsClamped(from, months)
				expectedMonth := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, utc).AddDate(0, months, 0)
				require.Equal(t, expectedMonth.Month(), got.Month(), "from %s months %d", from.Format("2006-01-02"), months)
				require.Equal(t, expectedMonth.Year(), got.Year(), "from %s months %d", from.Format("2006-01-02"), months)
			}
		}
	})
}

// TestAdversarialRelatedDocPeriodBoundaries attacks the NC/ND three month window around month ends.
func TestAdversarialRelatedDocPeriodBoundaries(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name       string
		dteType    string
		related    string
		relatedDay string
		generation string
		wantReject bool
	}{
		{"CCF 30-Nov last day 28-Feb", "05", "03", "2026-11-30", "2027-02-28", false},
		{"CCF 30-Nov one day past", "05", "03", "2026-11-30", "2027-03-01", true},
		{"CCF 30-Nov leap last day 29-Feb", "05", "03", "2027-11-30", "2028-02-29", false},
		{"CCF 30-Nov leap one day past", "05", "03", "2027-11-30", "2028-03-01", true},
		{"CCF 31-Dec last day 31-Mar", "06", "03", "2026-12-31", "2027-03-31", false},
		{"CCF 31-Dec one day past", "06", "03", "2026-12-31", "2027-04-01", true},
		{"CCF 31-Jan last day 30-Apr", "05", "03", "2027-01-31", "2027-04-30", false},
		{"CCF 31-Jan one day past", "05", "03", "2027-01-31", "2027-05-01", true},
		{"CCF same day", "05", "03", "2027-01-31", "2027-01-31", false},
		{"CCF generation before related", "05", "03", "2027-01-31", "2026-01-01", false},
		{"CR adjusted by ND", "06", "07", "2026-11-30", "2027-03-01", true},
		{"FE related is not windowed", "05", "01", "2020-01-01", "2027-03-01", false},
		{"FE issuing with CCF related not windowed", "01", "03", "2020-01-01", "2027-03-01", false},
		{"zero date skipped", "05", "03", "", "2027-03-01", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var related time.Time
			if tc.relatedDay != "" {
				related = day(tc.relatedDay)
			}
			got := strategy.ValidateRelatedDocPeriod(tc.dteType, tc.related, related, day(tc.generation))
			if !tc.wantReject {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, []string{"RELATEDDOCOUTOFPERIOD"}, advCodes(got))
			advTranslated(t, "validation_errors", "RelatedDocOutOfPeriod", "03", "2026-01-01", "2026-04-01")
		})
	}

	t.Run("generation in El Salvador zone", func(t *testing.T) {
		sv := time.FixedZone("SV", -6*3600)
		related := time.Date(2026, 11, 30, 0, 0, 0, 0, sv)
		generation := time.Date(2027, 2, 28, 23, 59, 0, 0, sv)
		assert.Nil(t, strategy.ValidateRelatedDocPeriod("05", "03", related, generation))
		generation = time.Date(2027, 3, 1, 0, 0, 0, 0, sv)
		assert.NotNil(t, strategy.ValidateRelatedDocPeriod("05", "03", related, generation))
	})
}

// TestAdversarialItemDiscountAmount attacks the monetary discount rules.
func TestAdversarialItemDiscountAmount(t *testing.T) {
	test.TestMain(t)

	t.Run("SetDiscount rounding and bounds", func(t *testing.T) {
		for _, tc := range []struct {
			in      float64
			want    float64
			wantErr bool
		}{
			{0, 0, false},
			{0.005, 0.005, false},
			{33.335, 33.335, false},
			{0.123456789, 0.12345679, false},
			{100, 100, false},
			{100.01, 100.01, false},
			{99999999999.99, 99999999999.99, false},
			{100000000000, 0, true},
			{-0.005, 0, true},
			{-1, 0, true},
		} {
			var base models.Item
			err := base.SetDiscount(tc.in)
			if tc.wantErr {
				require.Error(t, err, "%v", tc.in)
				continue
			}
			require.NoError(t, err, "%v", tc.in)
			assert.InDelta(t, tc.want, base.GetDiscount(), 1e-9)
		}
	})

	t.Run("mapper accepts discount over 100 and rejects negative", func(t *testing.T) {
		mapped, err := common.MapCommonRequestItem(structs.ItemRequest{Type: 1, Quantity: 1, UnitMeasure: 59, UnitPrice: 500, Discount: 250.5, Description: "x"}, 0)
		require.NoError(t, err)
		assert.Equal(t, 250.5, mapped.GetDiscount())

		_, err = common.MapCommonRequestItem(structs.ItemRequest{Type: 1, Quantity: 1, UnitMeasure: 59, UnitPrice: 500, Discount: -0.01, Description: "x"}, 0)
		require.Error(t, err)
		assert.Equal(t, "InvalidAmount", validationErrorType(t, err))
	})

	t.Run("sale amount with discount equal to line amount is zero", func(t *testing.T) {
		assert.Nil(t, strategy.ValidateItemSaleAmount(1, 50, 2, 100, 0))
		assert.NotNil(t, strategy.ValidateItemSaleAmount(1, 50, 2, 100, 0.02))
	})

	t.Run("sale amount with discount above line amount expects negative", func(t *testing.T) {
		got := strategy.ValidateItemSaleAmount(1, 50, 2, 101, 0)
		require.NotNil(t, got)
		assert.Equal(t, []string{"INVALIDITEMSALEAMOUNT"}, advCodes(got))
	})

	t.Run("sale amount rounding boundaries", func(t *testing.T) {
		assert.Nil(t, strategy.ValidateItemSaleAmount(1, 33.335, 3, 0, 100.005))
		assert.Nil(t, strategy.ValidateItemSaleAmount(1, 10, 3, 0.005, 29.995))
	})

	t.Run("summary percentage stays 0-100", func(t *testing.T) {
		for _, value := range []float64{0, 0.005, 33.335, 100} {
			_, err := financial.NewDiscount(value)
			assert.NoError(t, err, "%v", value)
		}
		for _, value := range []float64{100.01, 101, -0.01, -1} {
			_, err := financial.NewDiscount(value)
			require.Error(t, err, "%v", value)
			assert.Equal(t, "InvalidDiscount", validationErrorType(t, err))
		}
	})
}

// TestAdversarialUnsupportedFuelTaxes attacks the D1/C8 rejection in every entry point.
func TestAdversarialUnsupportedFuelTaxes(t *testing.T) {
	test.TestMain(t)

	t.Run("NewTaxType exact codes", func(t *testing.T) {
		for _, tc := range []struct {
			code string
			want string
		}{
			{"D1", "UnsupportedTaxCode"},
			{"C8", "UnsupportedTaxCode"},
			{"d1", "InvalidTaxType"},
			{"c8", "InvalidTaxType"},
			{" D1", "InvalidTaxType"},
			{"D1 ", "InvalidTaxType"},
			{"", "InvalidTaxType"},
			{"D", "InvalidTaxType"},
			{"A8", "InvalidTaxType"},
		} {
			got, err := financial.NewTaxType(tc.code)
			require.Error(t, err, tc.code)
			assert.Equal(t, tc.want, validationErrorType(t, err), tc.code)
			assert.Empty(t, got.GetValue())
		}
		_, err := financial.NewTaxType("20")
		assert.NoError(t, err)
	})

	t.Run("item taxes and item tax code", func(t *testing.T) {
		for _, code := range []string{"D1", "C8", "d1", "c8"} {
			code := code
			_, err := common.MapCommonRequestItem(structs.ItemRequest{Type: 1, Quantity: 1, UnitMeasure: 59, UnitPrice: 1, Description: "x", Taxes: []string{"20", code}}, 0)
			require.Error(t, err, code)
			_, err = common.MapCommonRequestItem(structs.ItemRequest{Type: 1, Quantity: 1, UnitMeasure: 59, UnitPrice: 1, Description: "x", TaxCode: &code}, 0)
			require.Error(t, err, code)
		}
		_, err := common.MapCommonRequestItem(structs.ItemRequest{Type: 1, Quantity: 1, UnitMeasure: 59, UnitPrice: 1, Description: "x", Taxes: []string{"D1"}}, 0)
		assert.Equal(t, "UnsupportedTaxCode", validationErrorType(t, err))
	})

	t.Run("summary tributes", func(t *testing.T) {
		for _, tc := range []struct {
			code string
			want string
		}{
			{"D1", "UnsupportedTaxCode"},
			{"C8", "UnsupportedTaxCode"},
			{"d1", "InvalidTaxType"},
		} {
			_, err := common.MapCommonRequestSummaryTaxes([]structs.TaxRequest{{Code: "20", Description: "IVA", Value: 1.3}, {Code: tc.code, Description: "FOVIAL", Value: 0.2}})
			require.Error(t, err, tc.code)
			assert.Equal(t, tc.want, validationErrorType(t, err), tc.code)
		}

		_, err := common.MapCommonRequestSummaryTaxes([]structs.TaxRequest{{Code: "D1", Description: "FOVIAL", Value: 0}})
		require.Error(t, err)
		t.Logf("D1 with zero value is rejected with: %v", err)
	})

	t.Run("allowed map no longer contains D1 or C8", func(t *testing.T) {
		assert.False(t, constants.MapAllowedTaxTypes["D1"])
		assert.False(t, constants.MapAllowedTaxTypes["C8"])
		assert.True(t, constants.UnsupportedTaxTypes["D1"])
		assert.True(t, constants.UnsupportedTaxTypes["C8"])
		assert.Len(t, constants.UnsupportedTaxTypes, 2)
	})
}

func advPaymentDocument(dteType string, condition int, total float64, amounts []float64, withTerms bool) *models.DTEDocument {
	payments := make([]interfaces.PaymentType, len(amounts))
	for index, amount := range amounts {
		payment := &models.PaymentType{
			Code:   *financial.NewValidatedPaymentType(constants.TarjetaCredito),
			Amount: *financial.NewValidatedAmount(amount),
		}
		if condition == constants.Credit {
			payment.Code = financial.PaymentType{}
		}
		if withTerms {
			period := 30
			payment.Period = &period
			payment.Term = financial.NewValidatedPaymentTerm("01")
		}
		payments[index] = payment
	}
	return &models.DTEDocument{
		Identification: &models.Identification{DTEType: *document.NewValidatedDTEType(dteType)},
		Summary: &models.Summary{
			OperationCondition: *financial.NewValidatedPaymentCondition(condition),
			TotalToPay:         *financial.NewValidatedAmount(total),
			PaymentTypes:       payments,
		},
	}
}

// TestAdversarialPaymentTotals attacks the credit zero-payment rule.
func TestAdversarialPaymentTotals(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name      string
		dteType   string
		condition int
		total     float64
		amounts   []float64
		terms     bool
		wantCode  string
	}{
		{"credit two zero payments", "01", 2, 100, []float64{0, 0}, true, ""},
		{"credit three zero payments", "03", 2, 100, []float64{0, 0, 0}, true, ""},
		{"credit zero plus partial", "01", 2, 100, []float64{0, 40}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit partial plus zero", "01", 2, 100, []float64{40, 0}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit zero plus full", "01", 2, 100, []float64{0, 100}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit split sums full", "01", 2, 100, []float64{60, 40}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit split over total", "01", 2, 100, []float64{60, 40.02}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit within tolerance 0.01", "01", 2, 100, []float64{99.99}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit just outside tolerance", "01", 2, 100, []float64{99.98}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit tiny non-zero amount", "01", 2, 100, []float64{0.005}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit empty list", "01", 2, 100, nil, true, "INVALIDPAYMENTTOTAL"},
		{"credit negative amounts summing zero", "01", 2, 100, []float64{50, -50}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit negative sum", "01", 2, 100, []float64{-10}, true, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit zero payment zero total", "01", 2, 0, []float64{0}, true, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"cash zero payment", "01", 1, 100, []float64{0}, false, "INVALIDPAYMENTTOTAL"},
		{"cash two zero payments", "01", 1, 100, []float64{0, 0}, false, "INVALIDPAYMENTTOTAL"},
		{"cash full", "01", 1, 100, []float64{100}, false, ""},
		{"condition 3 zero payment", "01", 3, 100, []float64{0}, false, "INVALIDPAYMENTTOTAL"},
		{"condition 3 full", "01", 3, 100, []float64{100}, false, ""},
		{"credit CCF zero", "03", 2, 100, []float64{0}, true, ""},
		{"FSE credit zero", "14", 2, 100, []float64{0}, true, ""},
		{"NC skips validation", "05", 1, 100, []float64{0}, false, ""},
		{"ND skips validation", "06", 2, 100, []float64{7}, false, ""},
		{"credit zero without terms still needs terms", "01", 2, 100, []float64{0}, false, "CREDITPAYMENTTERMREQUIRED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := advPaymentDocument(tc.dteType, tc.condition, tc.total, tc.amounts, tc.terms)
			got := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
		})
	}

	t.Run("nil document and nil summary", func(t *testing.T) {
		assert.Nil(t, (&strategy.PaymentTotalStrategy{}).Validate())
		assert.Nil(t, (&strategy.PaymentTotalStrategy{Document: &models.DTEDocument{}}).Validate())
	})

	t.Run("mapper accepts zero amount and empty code, rejects negative", func(t *testing.T) {
		mapped, err := common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "01", Amount: 0}, {Code: "02", Amount: 0}})
		require.NoError(t, err)
		assert.Len(t, mapped, 2)

		emptyCode, err := common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "", Amount: 5}})
		require.NoError(t, err)
		assert.Equal(t, "", emptyCode[0].GetCode())

		_, err = common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "01", Amount: -1}})
		require.Error(t, err)
		assert.Equal(t, "InvalidAmount", validationErrorType(t, err))

		empty, err := common.MapCommonRequestPaymentsType(nil)
		require.NoError(t, err)
		assert.Empty(t, empty)

		_, err = common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "ZZ", Amount: 1}})
		assert.Error(t, err)
	})
}

// TestAdversarialFSEItemTypes attacks the item type rule for the excluded subject invoice.
func TestAdversarialFSEItemTypes(t *testing.T) {
	test.TestMain(t)

	newItem := func(itemType int) *models.Item {
		return &models.Item{
			Type:        *item.NewValidatedItemType(itemType),
			Description: "Item",
			Quantity:    *item.NewValidatedQuantity(1),
			UnitPrice:   *financial.NewValidatedAmount(10),
		}
	}

	for _, itemType := range []int{1, 2, 3, 4, 5, 0, -1} {
		t.Run(fmt.Sprintf("type %d", itemType), func(t *testing.T) {
			doc := &fse_models.FSEModel{FSEItems: []fse_models.FSEItem{{Item: newItem(itemType), Purchase: *financial.NewValidatedAmount(10)}}}
			got := fseStrategy.NewFSEItemStrategy(doc).Validate()
			if itemType == 4 {
				require.NotNil(t, got)
				assert.Equal(t, []string{"INVALIDFSEITEMTYPE"}, advCodes(got))
				return
			}
			t.Logf("item type %d result: %v", itemType, advCodes(got))
		})
	}

	t.Run("second item of type 4 is rejected", func(t *testing.T) {
		doc := &fse_models.FSEModel{FSEItems: []fse_models.FSEItem{
			{Item: newItem(1), Purchase: *financial.NewValidatedAmount(10)},
			{Item: newItem(4), Purchase: *financial.NewValidatedAmount(10)},
		}}
		got := fseStrategy.NewFSEItemStrategy(doc).Validate()
		require.NotNil(t, got)
		assert.Contains(t, advCodes(got), "INVALIDFSEITEMTYPE")
	})
}

// TestAdversarialOtherDocumentsCodes attacks associated document code 4 for every DTE type.
func TestAdversarialOtherDocumentsCodes(t *testing.T) {
	test.TestMain(t)

	build := func(dteType string, code int) *models.DTEDocument {
		return &models.DTEDocument{
			Identification: &models.Identification{DTEType: *document.NewValidatedDTEType(dteType)},
			OtherDocuments: []interfaces.OtherDocuments{&models.OtherDocument{
				AssociatedCode: *document.NewValidatedAssociatedDocumentCode(code),
				Description:    advStr("Description"),
				Detail:         advStr("Detail"),
				Doctor:         &models.DoctorInfo{},
			}},
		}
	}

	for _, dteType := range []string{"01", "03", "04", "05", "06", "07", "08", "09", "11", "14", "15"} {
		for _, code := range []int{0, 1, 2, 4, 5, -1, 99} {
			t.Run(fmt.Sprintf("dte%s_code%d", dteType, code), func(t *testing.T) {
				got := (&strategy.OtherDocumentsStrategy{Document: build(dteType, code)}).Validate()
				switch {
				case code < 1 || code > 4:
					require.NotNil(t, got)
					assert.Equal(t, "INVALIDASSOCIATEDDOCUMENTCODE", got.GetCode())
				case code == 4 && dteType != "11":
					require.NotNil(t, got)
					assert.Equal(t, "INVALIDASSOCIATEDDOCUMENTCODEFORTYPE", got.GetCode())
				default:
					assert.Nil(t, got)
				}
			})
		}
	}
}
