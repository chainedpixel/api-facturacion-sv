package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/validator/strategy"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
)

type adversarialRelDoc struct {
	docType string
	genType int
	number  string
}

func adversarialInvoice(relDocs []adversarialRelDoc, itemRefs []*string) *invoice_models.ElectronicInvoice {
	var related []interfaces.RelatedDocument
	if relDocs != nil {
		related = make([]interfaces.RelatedDocument, 0, len(relDocs))
	}
	for _, rd := range relDocs {
		doc := &models.RelatedDocument{}
		_ = doc.SetDocumentType(rd.docType)
		_ = doc.SetGenerationType(rd.genType)
		doc.DocumentNumber = rd.number
		_ = doc.SetEmissionDate(utils.TimeNow())
		related = append(related, doc)
	}

	items := make([]invoice_models.InvoiceItem, 0, len(itemRefs))
	for i, ref := range itemRefs {
		items = append(items, invoice_models.InvoiceItem{Item: &models.Item{
			Number:     item.ItemNumber{Value: i + 1},
			RelatedDoc: ref,
		}})
	}

	return &invoice_models.ElectronicInvoice{
		DTEDocument:  &models.DTEDocument{RelatedDocuments: related},
		InvoiceItems: items,
	}
}

func adversarialStr(value string) *string { return &value }

// TestInvoiceRelatedDocStrategyAdversarial attacks the related document rules of the electronic invoice.
func TestInvoiceRelatedDocStrategyAdversarial(t *testing.T) {
	test.TestMain(t)

	const num = "001BEDAD-93F3-4F49-85D9-1E3618425F6B"
	elec := constants.ElectronicDocument
	nr := constants.NotaRemisionElectronica
	s := adversarialStr

	cases := []struct {
		name      string
		relDocs   []adversarialRelDoc
		refs      []*string
		errorCode string
	}{
		{"nil related docs ignores items without ref", nil, []*string{nil}, ""},
		{"empty related docs ignores items without ref", []adversarialRelDoc{}, []*string{nil}, ""},
		{"valid single doc and refs", []adversarialRelDoc{{nr, elec, num}}, []*string{s(num), s(num)}, ""},
		{"type 08 accepted", []adversarialRelDoc{{constants.ComprobanteLiquidacionElectronico, elec, num}}, []*string{s(num)}, ""},
		{"type 09 accepted", []adversarialRelDoc{{constants.DocContableLiquidacionElectronico, elec, num}}, []*string{s(num)}, ""},
		{"physical generation type accepted", []adversarialRelDoc{{nr, constants.PhysicalDocument, "PHYS-0001"}}, []*string{s("PHYS-0001")}, ""},
		{"duplicate document numbers accepted", []adversarialRelDoc{{nr, elec, num}, {nr, elec, num}}, []*string{s(num)}, ""},
		{"two docs items reference each", []adversarialRelDoc{{nr, elec, num}, {constants.ComprobanteLiquidacionElectronico, elec, "OTHER-1"}}, []*string{s(num), s("OTHER-1")}, ""},
		{"type 03 rejected", []adversarialRelDoc{{constants.CCFElectronico, elec, num}}, []*string{s(num)}, "INVALIDRELATEDDOCDTETYPE"},
		{"type 01 rejected", []adversarialRelDoc{{constants.FacturaElectronica, elec, num}}, []*string{s(num)}, "INVALIDRELATEDDOCDTETYPE"},
		{"type 05 rejected", []adversarialRelDoc{{constants.NotaCreditoElectronica, elec, num}}, []*string{s(num)}, "INVALIDRELATEDDOCDTETYPE"},
		{"empty type rejected", []adversarialRelDoc{{"", elec, num}}, []*string{s(num)}, "INVALIDRELATEDDOCDTETYPE"},
		{"mixed valid first invalid second rejected", []adversarialRelDoc{{nr, elec, num}, {constants.CCFElectronico, elec, "X"}}, []*string{s(num)}, "INVALIDRELATEDDOCDTETYPE"},
		{"invalid type wins over missing item ref", []adversarialRelDoc{{constants.CCFElectronico, elec, num}}, []*string{nil}, "INVALIDRELATEDDOCDTETYPE"},
		{"item without ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{nil}, "MISSINGITEMRELATEDDOC"},
		{"second item without ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s(num), nil}, "MISSINGITEMRELATEDDOC"},
		{"first item without ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{nil, s(num)}, "MISSINGITEMRELATEDDOC"},
		{"second of two docs unreferenced", []adversarialRelDoc{{nr, elec, num}, {nr, elec, "OTHER-1"}}, []*string{s(num), s(num)}, "UNREFERENCEDRELATEDDOC"},
		{"nil related docs with item ref rejected", nil, []*string{s(num)}, "INVALIDITEMRELATEDDOC"},
		{"empty related docs with item ref rejected", []adversarialRelDoc{}, []*string{nil, s(num)}, "INVALIDITEMRELATEDDOC"},
		{"unknown ref wins over unreferenced doc", []adversarialRelDoc{{nr, elec, num}}, []*string{s("UNKNOWN"), nil}, "INVALIDITEMRELATEDDOC"},
		{"unknown ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s("UNKNOWN")}, "INVALIDITEMRELATEDDOC"},
		{"empty string ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s("")}, "INVALIDITEMRELATEDDOC"},
		{"lowercase ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s(strings.ToLower(num))}, "INVALIDITEMRELATEDDOC"},
		{"ref with trailing space rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s(num + " ")}, "INVALIDITEMRELATEDDOC"},
		{"ref with leading space rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s(" " + num)}, "INVALIDITEMRELATEDDOC"},
		{"one valid one unknown ref rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{s(num), s("UNKNOWN")}, "INVALIDITEMRELATEDDOC"},
		{"docs without items rejected", []adversarialRelDoc{{nr, elec, num}}, []*string{}, "UNREFERENCEDRELATEDDOC"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := adversarialInvoice(tc.relDocs, tc.refs)
			err := (&strategy.InvoiceRelatedDocStrategy{Document: doc}).Validate()
			if tc.errorCode == "" {
				assert.Nil(t, err)
				return
			}
			if assert.NotNil(t, err) {
				assert.Equal(t, tc.errorCode, err.GetCode())
				assert.NotContains(t, err.Message, "No se encontr")
				assert.NotContains(t, strings.ToLower(err.Message), "unknownerror")
				assert.NotContains(t, err.Message, "validation_errors.")
				assert.NotContains(t, err.Message, "%!")
			}
		})
	}
}

// TestInvoiceRelatedDocLimitBoundary attacks the 50 related documents limit.
func TestInvoiceRelatedDocLimitBoundary(t *testing.T) {
	test.TestMain(t)

	build := func(count int) *invoice_models.ElectronicInvoice {
		docs := make([]adversarialRelDoc, count)
		for i := range docs {
			docs[i] = adversarialRelDoc{constants.NotaRemisionElectronica, constants.ElectronicDocument, fmt.Sprintf("DOC-%d", i)}
		}
		refs := make([]*string, count)
		for i := range refs {
			refs[i] = adversarialStr(fmt.Sprintf("DOC-%d", i))
		}
		return adversarialInvoice(docs, refs)
	}

	assert.Nil(t, (&strategy.InvoiceRelatedDocStrategy{Document: build(50)}).Validate())
	err := (&strategy.InvoiceRelatedDocStrategy{Document: build(51)}).Validate()
	require.NotNil(t, err)
	assert.Equal(t, "EXCEEDEDRELATEDDOCSLIMIT", err.GetCode())
}

// TestInvoiceRulesValidatorRegistersRelatedDocStrategy checks that the rules validator reports the related doc
// error of a valid invoice that gets a forbidden related document type.
func TestInvoiceRulesValidatorRegistersRelatedDocStrategy(t *testing.T) {
	test.TestMain(t)

	invoice, err := fixtures.BuildValidInvoice()
	require.NoError(t, err)

	rd := &models.RelatedDocument{}
	require.NoError(t, rd.SetDocumentType(constants.CCFElectronico))
	require.NoError(t, rd.SetGenerationType(constants.ElectronicDocument))
	rd.DocumentNumber = "001BEDAD-93F3-4F49-85D9-1E3618425F6B"
	require.NoError(t, rd.SetEmissionDate(utils.TimeNow()))
	invoice.RelatedDocuments = []interfaces.RelatedDocument{rd}

	dteErr := validator.NewInvoiceRulesValidator(invoice).Validate()
	require.NotNil(t, dteErr)

	codes := []string{}
	for _, business := range dteErr.BusinessErrors {
		codes = append(codes, business.GetCode())
	}
	assert.Contains(t, codes, "INVALIDRELATEDDOCDTETYPE")
}
