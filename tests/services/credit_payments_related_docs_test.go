package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	ccfStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	responseCommon "github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type creditPaymentSpec struct {
	code      string
	amount    float64
	reference string
	term      *string
	period    *int
}

func creditTestPaymentDocument(dteType string, condition int, total float64, specs ...creditPaymentSpec) *models.DTEDocument {
	payments := make([]interfaces.PaymentType, len(specs))
	for index, spec := range specs {
		payment := &models.PaymentType{
			Code:      *financial.NewValidatedPaymentType(spec.code),
			Amount:    *financial.NewValidatedAmount(spec.amount),
			Reference: spec.reference,
			Period:    spec.period,
		}
		if spec.term != nil {
			payment.Term = financial.NewValidatedPaymentTerm(*spec.term)
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

func creditTestInt(value int) *int { return &value }

func creditTestStr(value string) *string { return &value }

// TestCreditPaymentContract covers every rule of a credit sale payment and the unchanged cash and other rules.
func TestCreditPaymentContract(t *testing.T) {
	test.TestMain(t)

	valid := creditPaymentSpec{term: creditTestStr("01"), period: creditTestInt(30)}
	with := func(change func(*creditPaymentSpec)) creditPaymentSpec {
		spec := valid
		change(&spec)
		return spec
	}

	cases := []struct {
		name      string
		dteType   string
		condition int
		total     float64
		payments  []creditPaymentSpec
		wantCode  string
	}{
		{"FE credit valid", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{valid}, ""},
		{"CCF credit valid", constants.CCFElectronico, 2, 100, []creditPaymentSpec{valid}, ""},
		{"FSE credit valid", constants.FacturaSujetoExcluidoElectronica, 2, 100, []creditPaymentSpec{valid}, ""},
		{"credit two valid payments", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{valid, valid}, ""},
		{"credit terms 02 and 03 accepted", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{
			with(func(s *creditPaymentSpec) { s.term = creditTestStr("02") }),
			with(func(s *creditPaymentSpec) { s.term = creditTestStr("03") }),
		}, ""},
		{"credit code 03 rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.code = "03" })}, "CREDITPAYMENTCODENOTALLOWED"},
		{"credit code 01 rejected in CCF", constants.CCFElectronico, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.code = "01" })}, "CREDITPAYMENTCODENOTALLOWED"},
		{"credit code 99 rejected in FSE", constants.FacturaSujetoExcluidoElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.code = "99" })}, "CREDITPAYMENTCODENOTALLOWED"},
		{"credit second payment with code rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{valid, with(func(s *creditPaymentSpec) { s.code = "05" })}, "CREDITPAYMENTCODENOTALLOWED"},
		{"credit non-zero amount rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.amount = 100 })}, "INVALIDCREDITPAYMENTAMOUNT"},
		{"credit reference rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.reference = "REF-1" })}, "CREDITPAYMENTREFERENCENOTALLOWED"},
		{"credit reference rejected in FSE", constants.FacturaSujetoExcluidoElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.reference = "X" })}, "CREDITPAYMENTREFERENCENOTALLOWED"},
		{"credit without term rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.term = nil })}, "CREDITPAYMENTTERMREQUIRED"},
		{"credit without period rejected", constants.CCFElectronico, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.period = nil })}, "CREDITPAYMENTPERIODREQUIRED"},
		{"credit period zero rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.period = creditTestInt(0) })}, "CREDITPAYMENTPERIODREQUIRED"},
		{"credit negative period rejected", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.period = creditTestInt(-5) })}, "CREDITPAYMENTPERIODREQUIRED"},
		{"credit without term nor period reports term first", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.term = nil; s.period = nil })}, "CREDITPAYMENTTERMREQUIRED"},
		{"credit code reported before amount", constants.FacturaElectronica, 2, 100, []creditPaymentSpec{with(func(s *creditPaymentSpec) { s.code = "03"; s.amount = 100 })}, "CREDITPAYMENTCODENOTALLOWED"},
		{"credit empty payment list keeps total check", constants.FacturaElectronica, 2, 100, nil, "INVALIDPAYMENTTOTAL"},
		{"cash with code and full amount", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{code: "01", amount: 100}}, ""},
		{"cash without code rejected", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{amount: 100}}, "PAYMENTCODEREQUIRED"},
		{"cash second payment without code rejected", constants.CCFElectronico, 1, 100, []creditPaymentSpec{{code: "01", amount: 50}, {amount: 50}}, "PAYMENTCODEREQUIRED"},
		{"condition 3 without code rejected", constants.FacturaSujetoExcluidoElectronica, 3, 100, []creditPaymentSpec{{amount: 100}}, "PAYMENTCODEREQUIRED"},
		{"condition 3 with code and full amount", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "02", amount: 100}}, ""},
		{"cash payments not summing total rejected", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{code: "01", amount: 60}}, "INVALIDPAYMENTTOTAL"},
		{"NC skips payment rules", constants.NotaCreditoElectronica, 2, 100, []creditPaymentSpec{{code: "03", amount: 5}}, ""},
		{"NR skips payment rules", constants.NotaRemisionElectronica, 1, 100, []creditPaymentSpec{{amount: 0}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := creditTestPaymentDocument(tc.dteType, tc.condition, tc.total, tc.payments...)
			got := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
			assert.NotContains(t, got.Message, "No se encontr")
			assert.NotContains(t, got.Message, "%!")
		})
	}
}

// TestCreditPaymentMapperRoundTrip checks that a null code survives the request mapper and is serialized as null.
func TestCreditPaymentMapperRoundTrip(t *testing.T) {
	test.TestMain(t)

	period := 30
	mapped, err := common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Amount: 0, Term: creditTestStr("01"), Period: &period}})
	require.NoError(t, err)
	require.Len(t, mapped, 1)
	assert.Equal(t, "", mapped[0].GetCode())
	assert.Equal(t, "", mapped[0].GetReference())

	response := responseCommon.MapCommonResponsePayments(mapped)
	require.Len(t, response, 1)
	assert.Nil(t, response[0].Codigo)
	assert.Nil(t, response[0].Referencia)
	require.NotNil(t, response[0].Plazo)
	assert.Equal(t, "01", *response[0].Plazo)
	require.NotNil(t, response[0].Periodo)
	assert.Equal(t, 30, *response[0].Periodo)

	cash, err := common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "01", Amount: 10}})
	require.NoError(t, err)
	cashResponse := responseCommon.MapCommonResponsePayments(cash)
	require.NotNil(t, cashResponse[0].Codigo)
	assert.Equal(t, "01", *cashResponse[0].Codigo)

	_, err = common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Code: "XX", Amount: 0}})
	assert.Error(t, err)

	_, err = common.MapCommonRequestPaymentsType([]structs.PaymentRequest{{Amount: 0, Term: creditTestStr("09"), Period: &period}})
	assert.Error(t, err)
}

func relatedDocsTestDocs(numbers ...string) []interfaces.RelatedDocument {
	if numbers == nil {
		return nil
	}
	docs := make([]interfaces.RelatedDocument, 0, len(numbers))
	for _, number := range numbers {
		doc := &models.RelatedDocument{DocumentNumber: number}
		_ = doc.SetDocumentType(constants.NotaRemisionElectronica)
		_ = doc.SetGenerationType(constants.ElectronicDocument)
		docs = append(docs, doc)
	}
	return docs
}

func relatedDocsTestCCF(docs []interfaces.RelatedDocument, refs ...*string) *ccf_models.CreditFiscalDocument {
	items := make([]ccf_models.CreditItem, 0, len(refs))
	for index, ref := range refs {
		items = append(items, ccf_models.CreditItem{Item: &models.Item{
			Number:     item.ItemNumber{Value: index + 1},
			RelatedDoc: ref,
		}})
	}
	return &ccf_models.CreditFiscalDocument{
		DTEDocument: &models.DTEDocument{RelatedDocuments: docs},
		CreditItems: items,
	}
}

// TestItemRelatedDocsRule covers the shared strict rule that links items and related documents, directly and through
// the CCF strategy.
func TestItemRelatedDocsRule(t *testing.T) {
	test.TestMain(t)

	a, b := "DOC-A", "DOC-B"
	cases := []struct {
		name     string
		docs     []interfaces.RelatedDocument
		refs     []*string
		wantCode string
	}{
		{"no section and no refs", nil, []*string{nil, nil}, ""},
		{"empty section and no refs", []interfaces.RelatedDocument{}, []*string{nil}, ""},
		{"no section and no items", nil, nil, ""},
		{"loose item with section rejected", relatedDocsTestDocs(a), []*string{&a, nil}, "MISSINGITEMRELATEDDOC"},
		{"one doc referenced by every item", relatedDocsTestDocs(a), []*string{&a, &a}, ""},
		{"two docs each referenced", relatedDocsTestDocs(a, b), []*string{&b, &a, &a}, ""},
		{"first item loose with two docs rejected", relatedDocsTestDocs(a, b), []*string{nil, &a, &b}, "MISSINGITEMRELATEDDOC"},
		{"items without refs with section rejected", relatedDocsTestDocs(a), []*string{nil, nil}, "MISSINGITEMRELATEDDOC"},
		{"second doc not referenced", relatedDocsTestDocs(a, b), []*string{&a, &a}, "UNREFERENCEDRELATEDDOC"},
		{"doc without items", relatedDocsTestDocs(a), nil, "UNREFERENCEDRELATEDDOC"},
		{"ref without section", nil, []*string{&a}, "INVALIDITEMRELATEDDOC"},
		{"ref with empty section", []interfaces.RelatedDocument{}, []*string{nil, &a}, "INVALIDITEMRELATEDDOC"},
		{"ref outside section", relatedDocsTestDocs(a), []*string{&a, &b}, "INVALIDITEMRELATEDDOC"},
		{"empty string ref", relatedDocsTestDocs(a), []*string{&a, creditTestStr("")}, "INVALIDITEMRELATEDDOC"},
		{"case sensitive ref", relatedDocsTestDocs(a), []*string{creditTestStr("doc-a")}, "INVALIDITEMRELATEDDOC"},
		{"unknown ref reported before unreferenced doc", relatedDocsTestDocs(a), []*string{&b}, "INVALIDITEMRELATEDDOC"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ccf := relatedDocsTestCCF(tc.docs, tc.refs...)
			got := strategy.ValidateItemRelatedDocs(tc.docs, ccf.CreditItems)
			viaCCF := (&ccfStrategy.CCFRelatedDocStrategy{Document: ccf}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				assert.Nil(t, viaCCF)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
			require.NotNil(t, viaCCF)
			assert.Equal(t, tc.wantCode, viaCCF.GetCode())
			assert.NotContains(t, got.Message, "No se encontr")
			assert.NotContains(t, got.Message, "%!")
		})
	}
}

// TestCCFRelatedDocStrategyKeepsTypeAndLimitRules checks that the CCF related document type and limit rules still apply.
func TestCCFRelatedDocStrategyKeepsTypeAndLimitRules(t *testing.T) {
	test.TestMain(t)

	a := "DOC-A"
	docs := relatedDocsTestDocs(a)
	_ = docs[0].(*models.RelatedDocument).SetDocumentType(constants.FacturaElectronica)
	err := (&ccfStrategy.CCFRelatedDocStrategy{Document: relatedDocsTestCCF(docs, &a)}).Validate()
	require.NotNil(t, err)
	assert.Equal(t, "INVALIDRELATEDDOCDTETYPE", err.GetCode())

	numbers := make([]string, 51)
	refs := make([]*string, 51)
	for index := range numbers {
		numbers[index] = "DOC-" + string(rune('A'+index%26)) + string(rune('A'+index/26))
		refs[index] = &numbers[index]
	}
	err = (&ccfStrategy.CCFRelatedDocStrategy{Document: relatedDocsTestCCF(relatedDocsTestDocs(numbers...), refs...)}).Validate()
	require.NotNil(t, err)
	assert.Equal(t, "EXCEEDEDRELATEDDOCSLIMIT", err.GetCode())

	assert.Nil(t, (&ccfStrategy.CCFRelatedDocStrategy{Document: relatedDocsTestCCF(relatedDocsTestDocs(numbers[:50]...), refs[:50]...)}).Validate())
}
