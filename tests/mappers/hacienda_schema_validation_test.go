package mappers_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	contingency_models "github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
)

// Se valida el JSON completo que se envia a Hacienda contra los JSON Schema oficiales
// (copiados sin modificar en tests/testdata/svfe-json-schemas). A diferencia de las pruebas
// por campo, esta prueba detecta campos sobrantes (additionalProperties: false), campos
// requeridos ausentes, tipos, patrones, formatos y longitudes.

// ecmaRegexp adapta regexp2 (dialecto ECMAScript, el que exige JSON Schema) a la interfaz
// de la libreria. Se requiere porque los schemas usan lookahead, no soportado por regexp de Go.
type ecmaRegexp struct{ re *regexp2.Regexp }

func (r ecmaRegexp) MatchString(s string) bool {
	matched, err := r.re.MatchString(s)
	return err == nil && matched
}

func (r ecmaRegexp) String() string { return r.re.String() }

func ecmaRegexpEngine(pattern string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return ecmaRegexp{re: re}, nil
}

// assertMatchesHaciendaSchema serializa el documento y lo valida contra el schema indicado
// (ruta relativa a tests/testdata/svfe-json-schemas).
func assertMatchesHaciendaSchema(t *testing.T, schemaRelPath string, document interface{}) {
	t.Helper()

	schemaPath := filepath.Join(utils.FindProjectRoot(), "tests", "testdata", "svfe-json-schemas", schemaRelPath)

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft7)
	compiler.AssertFormat()
	compiler.UseRegexpEngine(ecmaRegexpEngine)

	schema, err := compiler.Compile(schemaPath)
	require.NoError(t, err, "no se pudo compilar el schema %s", schemaRelPath)

	payload, err := json.Marshal(document)
	require.NoError(t, err)

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	require.NoError(t, err)

	if err := schema.Validate(instance); err != nil {
		t.Errorf("el JSON no cumple %s:\n%v\n\nJSON enviado:\n%s", schemaRelPath, err, payload)
	}
}

func setVersion(t *testing.T, identification interface{}, version int) {
	t.Helper()
	id, ok := identification.(*models.Identification)
	require.True(t, ok)
	require.NoError(t, id.SetVersion(version))
}

func TestHaciendaSchema_Invoice(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidInvoice()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 2)

	assertMatchesHaciendaSchema(t, "v2/fe-f-v2.json", response_mapper.ToMHInvoice(doc))
}

func TestHaciendaSchema_CCF(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidCCF()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 4)

	assertMatchesHaciendaSchema(t, "v4/fe-ccf-v4.json", response_mapper.ToMHCreditFiscalInvoice(doc))
}

// TestHaciendaSchema_CreditSales validates FE and CCF credit sales, whose payments carry a null code and reference.
func TestHaciendaSchema_CreditSales(t *testing.T) {
	test.TestMain(t)

	creditPayments := []interfaces.PaymentType{fixtures.NewCreditPayment("01", 30)}

	invoice, err := fixtures.BuildValidInvoice()
	require.NoError(t, err)
	setVersion(t, invoice.GetIdentification(), 2)
	require.NoError(t, invoice.InvoiceSummary.SetOperationCondition(constants.Credit))
	require.NoError(t, invoice.InvoiceSummary.SetPaymentTypes(creditPayments))
	invoicePayload := response_mapper.ToMHInvoice(invoice)
	require.Len(t, invoicePayload.Resumen.Pagos, 1)
	require.Nil(t, invoicePayload.Resumen.Pagos[0].Codigo)
	assertMatchesHaciendaSchema(t, "v2/fe-f-v2.json", invoicePayload)

	ccf, err := fixtures.BuildValidCCF()
	require.NoError(t, err)
	setVersion(t, ccf.GetIdentification(), 4)
	require.NoError(t, ccf.CreditSummary.SetOperationCondition(constants.Credit))
	require.NoError(t, ccf.CreditSummary.SetPaymentTypes(creditPayments))
	ccfPayload := response_mapper.ToMHCreditFiscalInvoice(ccf)
	require.Len(t, ccfPayload.Resumen.Pagos, 1)
	require.Nil(t, ccfPayload.Resumen.Pagos[0].Codigo)
	assertMatchesHaciendaSchema(t, "v4/fe-ccf-v4.json", ccfPayload)
}

// Se cubren las secciones opcionales que los fixtures por defecto dejan en null:
// venta por cuenta de terceros (codDomiciliado, CAT-032) y apendice con elementos.
func TestHaciendaSchema_CCFWithThirdPartySaleAndAppendix(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidCCF()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 4)

	thirdPartySale := &models.ThirdPartySale{}
	require.NoError(t, thirdPartySale.SetNIT("06141804941035"))
	require.NoError(t, thirdPartySale.SetName("Empresa Tercero S.A. de C.V."))
	require.NoError(t, thirdPartySale.SetFiscalDomicile(constants.Domiciled))
	require.NoError(t, doc.SetThirdPartySale(thirdPartySale))

	appendix := &models.Appendix{}
	require.NoError(t, appendix.SetField("Datos del documento"))
	require.NoError(t, appendix.SetLabel("Sello de recepcion"))
	require.NoError(t, appendix.SetValue("2026A1B2C3D4E5F6A7B8C9D0E1F2A3B4C5D6"))
	require.NoError(t, doc.SetAppendix([]interfaces.Appendix{appendix}))

	assertMatchesHaciendaSchema(t, "v4/fe-ccf-v4.json", response_mapper.ToMHCreditFiscalInvoice(doc))
}

func TestHaciendaSchema_CreditNote(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidCreditNote()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 4)

	assertMatchesHaciendaSchema(t, "v4/fe-nc-v4.json", response_mapper.ToMHCreditNote(doc))
}

func TestHaciendaSchema_DebitNote(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidDebitNote()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 4)

	assertMatchesHaciendaSchema(t, "v4/fe-nd-v4.json", response_mapper.ToMHDebitNote(doc))
}

func TestHaciendaSchema_RemissionNote(t *testing.T) {
	test.TestMain(t)

	doc, receiver, err := fixtures.BuildValidRemissionNote()
	require.NoError(t, err)
	// En produccion el receptor se asigna al construir el modelo; el builder lo devuelve por separado
	doc.Receiver = receiver
	setVersion(t, doc.GetIdentification(), 4)

	assertMatchesHaciendaSchema(t, "v4/fe-nr-v4.json", response_mapper.ToMHRemissionNote(doc))
}

func TestHaciendaSchema_Retention(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidRetention()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 2)

	assertMatchesHaciendaSchema(t, "v2/fe-cr-v2.json", response_mapper.ToMHRetention(doc))
}

func TestHaciendaSchema_FSE(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildValidFSE()
	require.NoError(t, err)
	setVersion(t, doc.GetIdentification(), 2)

	assertMatchesHaciendaSchema(t, "v2/fe-fse-v2.json", response_mapper.ToMHFSE(doc))
}

func TestHaciendaSchema_Invalidation(t *testing.T) {
	test.TestMain(t)

	doc, err := fixtures.BuildInvalidationWithReplacement()
	require.NoError(t, err)

	assertMatchesHaciendaSchema(t, "v3/invalidacion-schema-v3.json", response_mapper.ToMHInvalidation(doc))
}

func TestHaciendaSchema_ContingencyEvent(t *testing.T) {
	test.TestMain(t)

	establishmentCodeMH := "M001"
	posCodeMH := "P001"
	event := contingency_models.ContingencyEvent{
		Identification: contingency_models.ContingencyIdentification{
			Version:          4,
			Ambient:          "00",
			GenerationCode:   "8E3E3C3B-2D5A-4C4F-9E1B-1A2B3C4D5E6F",
			TransmissionDate: "2026-11-03",
			TransmissionTime: "10:15:00",
		},
		Issuer: contingency_models.ContingencyIssuer{
			NIT:                  "06142803901121",
			Name:                 "Empresa de Prueba S.A. de C.V.",
			ResponsibleName:      "Responsable de Prueba",
			ResponsibleDocType:   "36",
			ResponsibleDocNumber: "06142803901121",
			EstablishmentType:    "01",
			Phone:                "22223333",
			Email:                "contacto@ejemplo.com",
			EstablishmentCodeMH:  &establishmentCodeMH,
			POSCode:              &posCodeMH,
		},
		DTEDetails: []contingency_models.DTEDetail{
			{ItemNumber: 1, GenerationCode: "1A2B3C4D-5E6F-4A1B-8C2D-3E4F5A6B7C8D", DocumentType: "01"},
		},
		Reason: contingency_models.ContingencyReason{
			StartDate:       "2026-11-03",
			EndDate:         "2026-11-03",
			StartTime:       "08:00:00",
			EndTime:         "09:00:00",
			ContingencyType: 1,
		},
	}

	assertMatchesHaciendaSchema(t, "v4/contingencia-schema-v4.json", event)
}
