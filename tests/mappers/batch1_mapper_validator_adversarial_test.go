package mappers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/policy"
	invalidationValidator "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/validator"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
)

func flattenDTECodes(err *dte_errors.DTEError) []string {
	if err == nil {
		return nil
	}
	if len(err.BusinessErrors) == 0 {
		return []string{err.GetCode()}
	}
	var codes []string
	for _, nested := range err.BusinessErrors {
		codes = append(codes, flattenDTECodes(nested)...)
	}
	return codes
}

func replacementCodes(codes []string) []string {
	var result []string
	for _, code := range codes {
		switch code {
		case "INVALIDREPLACEMENTCODE1AND3", "INVALIDINVALIDATIONTYPE2", "REPLACEMENTCODENOTALLOWEDFORTYPE":
			result = append(result, code)
		}
	}
	return result
}

// TestAdversarialInvalidationMapperToValidator runs every DTE type x reason type x replacement presence through the request
// validation, the mapper and the rules validator, asserting the single replacement-related code raised.
func TestAdversarialInvalidationMapperToValidator(t *testing.T) {
	test.TestMain(t)

	mapper := request_mapper.NewInvalidationMapper()
	issuer := fixtures.CreateDefaultIssuer()
	now := utils.TimeNow()
	generated := now.Add(-2 * time.Hour)
	deadlines := policy.NewDeadlines(nil)

	for _, dteType := range []string{"01", "03", "04", "05", "06", "07", "08", "09", "11", "14", "15"} {
		for reasonType := 1; reasonType <= 3; reasonType++ {
			for _, withReplacement := range []bool{false, true} {
				t.Run(fmt.Sprintf("dte%s_type%d_replacement%t", dteType, reasonType, withReplacement), func(t *testing.T) {
					req := fixtures.CreateDefaultInvalidationRequest()
					req.Reason.Type = reasonType
					if reasonType == 3 {
						reasonText := "Motivo de la invalidacion"
						req.Reason.Reason = &reasonText
					}
					if !withReplacement {
						req.ReplacementGenerationCode = nil
					}

					base := &dte.DTEDetails{
						ID:             "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
						DTEType:        dteType,
						ControlNumber:  "DTE-" + dteType + "-00000000-000000000000001",
						ReceptionStamp: utils.ToStringPointer("2026BBFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
						ProcessingDate: generated.Format("02/01/2006 15:04:05"),
						JSONData: fmt.Sprintf(`{"identificacion":{"fecEmi":"%s","horEmi":"%s"},"emisor":{"codActividad":"47712"},"receptor":{"nombre":"Empresa","nit":"06141804941035","nrc":"1234567"},"resumen":{"totalIva":13.00}}`,
							generated.Format("2006-01-02"), generated.Format("15:04:05")),
					}

					nullOnly := dteType == "05" || dteType == "08"
					want := ""
					switch {
					case reasonType == 2 && withReplacement:
						want = "INVALIDINVALIDATIONTYPE2"
					case reasonType != 2 && nullOnly && withReplacement:
						want = "REPLACEMENTCODENOTALLOWEDFORTYPE"
					case reasonType != 2 && !nullOnly && !withReplacement:
						want = "INVALIDREPLACEMENTCODE1AND3"
					}

					reqErr := mapper.ValidateInvalidationReRequest(req)
					if reasonType == 2 && withReplacement {
						require.Error(t, reqErr)
						var svcErr *shared_error.ServiceError
						require.ErrorAs(t, reqErr, &svcErr)
						assert.Equal(t, "InvalidInvalidationType2", svcErr.Code)
						assert.False(t, strings.HasPrefix(svcErr.Message, "["), svcErr.Message)
						return
					}
					require.NoError(t, reqErr, "the request level check must not decide the replacement rule for types 1 and 3")

					doc, err := mapper.MapToInvalidationData(req, issuer, base, now)
					require.NoError(t, err)
					assert.Equal(t, dteType, doc.Document.Type.GetValue())
					assert.Equal(t, withReplacement, doc.Document.ReplacementCode != nil)

					got := invalidationValidator.NewInvalidationRulesValidator(doc, deadlines, now).Validate()
					codes := replacementCodes(flattenDTECodes(got))
					if want == "" {
						assert.Empty(t, codes)
						return
					}
					assert.Equal(t, []string{want}, codes)
				})
			}
		}
	}
}

// TestAdversarialCreditPaymentsMapperToStrategy maps a credit invoice request and runs the payment total rules on the result.
func TestAdversarialCreditPaymentsMapperToStrategy(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()
	mapper := request_mapper.NewInvoiceMapper()

	period := 30
	term := "01"
	cases := []struct {
		name      string
		condition int
		amounts   []float64
		wantCode  string
	}{
		{"credit two zero payments", 2, []float64{0, 0}, ""},
		{"credit zero plus partial", 2, []float64{0, 40}, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit full", 2, []float64{100}, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit split full", 2, []float64{30, 70}, "INVALIDCREDITPAYMENTAMOUNT"},
		{"cash zero", 1, []float64{0}, "INVALIDPAYMENTTOTAL"},
		{"cash full", 1, []float64{100}, ""},
		{"condition 3 zero", 3, []float64{0}, "INVALIDPAYMENTTOTAL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := fixtures.CreateDefaultInvoiceRequest()
			req.Summary.OperationCondition = tc.condition
			req.Summary.PaymentTypes = nil
			for _, amount := range tc.amounts {
				payment := fixtures.CreateDefaultInvoiceSummary().PaymentTypes[0]
				payment.Amount = amount
				if tc.condition != 1 {
					payment.Code = "02"
					payment.Period = &period
					payment.Term = &term
				}
				if tc.condition == 2 {
					payment.Code = ""
					payment.Reference = nil
				}
				req.Summary.PaymentTypes = append(req.Summary.PaymentTypes, payment)
			}

			data, err := mapper.MapToInvoiceData(req, issuer)
			require.NoError(t, err)

			doc := &models.DTEDocument{
				Identification: &models.Identification{DTEType: *document.NewValidatedDTEType("01")},
				Summary:        data.InvoiceSummary.Summary,
			}
			got := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
		})
	}

	t.Run("empty payment list maps and is then rejected for credit", func(t *testing.T) {
		req := fixtures.CreateDefaultInvoiceRequest()
		req.Summary.OperationCondition = 2
		req.Summary.PaymentTypes = nil
		data, err := mapper.MapToInvoiceData(req, issuer)
		if err != nil {
			t.Logf("mapper already rejects an empty payment list: %v", err)
			return
		}
		doc := &models.DTEDocument{
			Identification: &models.Identification{DTEType: *document.NewValidatedDTEType("01")},
			Summary:        data.InvoiceSummary.Summary,
		}
		got := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
		require.NotNil(t, got)
		assert.Equal(t, "INVALIDPAYMENTTOTAL", got.GetCode())
	})
}
