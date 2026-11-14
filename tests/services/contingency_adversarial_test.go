package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDte "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/user"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const (
	adversarialNITOne = "06140101010000"
	adversarialNITTwo = "06149999990000"
)

// structuredDTE is a typed DTE used to feed the identification update with a struct.
type structuredDTE struct {
	Identificacion map[string]interface{} `json:"identificacion"`
	Monto          float64                `json:"monto"`
}

// selectiveSigner builds a compact JWS like fakeContingencySigner but fails for payloads containing failMarker.
type selectiveSigner struct {
	mu         sync.Mutex
	failMarker string
	payloads   []string
}

func (s *selectiveSigner) SignDTE(_ context.Context, document json.RawMessage, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloads = append(s.payloads, string(document))
	if s.failMarker != "" && strings.Contains(string(document), s.failMarker) {
		return "", errors.New("signing refused")
	}
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString(document) + ".c2lnbmF0dXJl", nil
}

func (s *selectiveSigner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.payloads)
}

// batchCall is one TransmitBatch invocation.
type batchCall struct {
	nit       string
	dteType   string
	documents []string
}

// recordingBatchTransmitter records every batch in order and stops the flow after recording it.
type recordingBatchTransmitter struct {
	mu    sync.Mutex
	calls []batchCall
}

func (b *recordingBatchTransmitter) TransmitBatch(_ context.Context, nit string, dteType string, documents []string, _ string, _ authModels.HaciendaCredentials) (*transmitterModels.BatchResponse, string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, batchCall{nit: nit, dteType: dteType, documents: append([]string(nil), documents...)})
	return nil, "", errors.New("batch captured")
}

func (b *recordingBatchTransmitter) VerifyContingencyBatchStatus(context.Context, string, string, string, uint, map[string]coreDte.ContingencyDocument) error {
	return nil
}

func (b *recordingBatchTransmitter) GetDTEVersion(string) int { return 1 }

func (b *recordingBatchTransmitter) documentsOf(dteType string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var all []string
	for _, call := range b.calls {
		if call.dteType == dteType {
			all = append(all, call.documents...)
		}
	}
	return all
}

func (b *recordingBatchTransmitter) allDocuments() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var all []string
	for _, call := range b.calls {
		all = append(all, call.documents...)
	}
	return all
}

// storeHarness exposes the captured persistence data of a store flow.
type storeHarness struct {
	svc        contingency.ContingencyManager
	mu         sync.Mutex
	stored     []interface{}
	evidence   []string
	createHits int
	repoHits   int
}

func newStoreHarness(ctrl *gomock.Controller, signer interface {
	SignDTE(context.Context, json.RawMessage, string) (string, error)
}, createErr, repoErr error) *storeHarness {
	harness := &storeHarness{}
	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().
		Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, gomock.Any()).
		DoAndReturn(func(_ interface{}, document interface{}, _ string, _ string, given *coreDte.TransmissionEvidence) error {
			harness.mu.Lock()
			defer harness.mu.Unlock()
			harness.createHits++
			if createErr != nil {
				return createErr
			}
			harness.stored = append(harness.stored, document)
			if given != nil {
				harness.evidence = append(harness.evidence, given.SignedDocument)
			}
			return nil
		}).AnyTimes()
	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ interface{}, _ interface{}) error {
		harness.mu.Lock()
		defer harness.mu.Unlock()
		harness.repoHits++
		return repoErr
	}).AnyTimes()
	mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeq.EXPECT().MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	harness.svc = contingency.NewContingencyManager(nil, mockDTE, mockRepo, nil, nil, nil, signer, nil,
		mocks.NewMockContingencyEventSender(ctrl), mockSeq, nil, nil)
	return harness
}

func dteJSONWithCode(code string) string {
	return fmt.Sprintf(`{"identificacion":{"tipoDte":"01","numeroControl":"DTE-01-00000001-%s","codigoGeneracion":"%s"},"emisor":{"nit":"0614-010101-000-0"}}`, code, code)
}

// TestUpdateContingencyIdentificationInputKinds checks that every supported document representation produces the
// same deferred identification.
func TestUpdateContingencyIdentificationInputKinds(t *testing.T) {
	test.TestMain(t)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(minimalDTEJSON()), &asMap))
	var asStruct structuredDTE
	require.NoError(t, json.Unmarshal([]byte(minimalDTEJSON()), &asStruct))

	inputs := map[string]interface{}{
		"string":      minimalDTEJSON(),
		"bytes":       []byte(minimalDTEJSON()),
		"raw message": json.RawMessage(minimalDTEJSON()),
		"struct":      asStruct,
		"struct ptr":  &asStruct,
		"map":         asMap,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			contingencyType := int8(constants.OtroMotivo)
			reason := "reason"
			updated, err := utils.UpdateContingencyIdentification(input, &contingencyType, &reason)
			require.NoError(t, err)
			decoded := decodeDeferredIdentification(t, updated)
			assert.Equal(t, constants.ModeloFacturacionDiferido, decoded.Identificacion.TipoModelo)
			assert.Equal(t, constants.TransmisionContingencia, decoded.Identificacion.TipoOperacion)
			require.NotNil(t, decoded.Identificacion.MotivoContin)
			assert.Equal(t, "reason", *decoded.Identificacion.MotivoContin)
		})
	}
}

// TestUpdateContingencyIdentificationMalformedInput checks the failure paths and the lenient paths of the update.
func TestUpdateContingencyIdentificationMalformedInput(t *testing.T) {
	test.TestMain(t)

	contingencyType := int8(constants.NoDisponibilidadMH)
	reason := "reason"

	for name, input := range map[string]interface{}{
		"invalid json string":   "{not json",
		"invalid json bytes":    []byte("{"),
		"invalid json raw":      json.RawMessage("nope"),
		"empty string":          "",
		"json array":            "[1,2,3]",
		"identificacion string": `{"identificacion":"text"}`,
		"identificacion number": `{"identificacion":7}`,
		"identificacion array":  `{"identificacion":[]}`,
		"unmarshalable value":   make(chan int),
		"identificacion null":   []byte(`{"identificacion":null}`),
	} {
		t.Run(name, func(t *testing.T) {
			updated, err := utils.UpdateContingencyIdentification(input, &contingencyType, &reason)
			require.Error(t, err)
			assert.Nil(t, updated)
		})
	}

	t.Run("missing identificacion leaves the document untouched", func(t *testing.T) {
		updated, err := utils.UpdateContingencyIdentification(`{"emisor":{"nit":"1"}}`, &contingencyType, &reason)
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"emisor": map[string]interface{}{"nit": "1"}}, updated)
	})
}

// TestUpdateContingencyIdentificationReasonEdges checks the reason handling of type 5 and the null reason of the
// other types.
func TestUpdateContingencyIdentificationReasonEdges(t *testing.T) {
	test.TestMain(t)

	otherType := int8(constants.OtroMotivo)
	defaultReason := constants.GetContingencyReason(constants.OtroMotivo)
	require.NotEmpty(t, defaultReason)

	reasonOf := func(t *testing.T, reason *string) string {
		t.Helper()
		updated, err := utils.UpdateContingencyIdentification(minimalDTEJSON(), &otherType, reason)
		require.NoError(t, err)
		decoded := decodeDeferredIdentification(t, updated)
		require.NotNil(t, decoded.Identificacion.MotivoContin)
		return *decoded.Identificacion.MotivoContin
	}

	t.Run("nil reason uses the default text", func(t *testing.T) {
		assert.Equal(t, defaultReason, reasonOf(t, nil))
	})

	for _, blank := range []string{"", " ", "   ", "\t", "\n", " \t\r\n "} {
		blank := blank
		t.Run(fmt.Sprintf("blank %q uses the default text", blank), func(t *testing.T) {
			assert.Equal(t, defaultReason, reasonOf(t, &blank))
		})
	}

	t.Run("surrounding whitespace is trimmed", func(t *testing.T) {
		padded := "  \n reason with spaces \t "
		assert.Equal(t, "reason with spaces", reasonOf(t, &padded))
	})

	t.Run("exactly 500 characters is kept", func(t *testing.T) {
		exact := strings.Repeat("a", 500)
		assert.Equal(t, exact, reasonOf(t, &exact))
	})

	t.Run("multibyte reason is truncated by runes", func(t *testing.T) {
		long := strings.Repeat("ñáé", 400)
		require.Greater(t, len(long), 1000)
		got := reasonOf(t, &long)
		assert.True(t, utf8.ValidString(got), "truncation must not split a character")
		assert.Equal(t, 500, utf8.RuneCountInString(got))
		assert.Equal(t, string([]rune(long)[:500]), got)
	})

	t.Run("multibyte reason of 500 runes over 500 bytes is not truncated", func(t *testing.T) {
		exact := strings.Repeat("ñ", 500)
		got := reasonOf(t, &exact)
		assert.Equal(t, exact, got)
	})

	t.Run("other types ignore the reason", func(t *testing.T) {
		for _, contingencyType := range []int8{1, 2, 3, 4} {
			given := "must be ignored"
			updated, err := utils.UpdateContingencyIdentification(minimalDTEJSON(), &contingencyType, &given)
			require.NoError(t, err)
			identification := updated["identificacion"].(map[string]interface{})
			value, present := identification["motivoContin"]
			assert.True(t, present, "type %d keeps an explicit null reason", contingencyType)
			assert.Nil(t, value)
		}
	})
}

// TestUpdateContingencyIdentificationOverwritesExistingValues checks that stale deferred values never survive.
func TestUpdateContingencyIdentificationOverwritesExistingValues(t *testing.T) {
	test.TestMain(t)

	document := `{"identificacion":{"tipoModelo":1,"tipoOperacion":1,"tipoContingencia":5,"motivoContin":"stale","codigoGeneracion":"X"}}`

	contingencyType := int8(constants.NoDisponibilidadMH)
	updated, err := utils.UpdateContingencyIdentification(document, &contingencyType, nil)
	require.NoError(t, err)
	decoded := decodeDeferredIdentification(t, updated)
	assert.Equal(t, constants.ModeloFacturacionDiferido, decoded.Identificacion.TipoModelo)
	assert.Equal(t, constants.TransmisionContingencia, decoded.Identificacion.TipoOperacion)
	require.NotNil(t, decoded.Identificacion.TipoContingencia)
	assert.Equal(t, int(constants.NoDisponibilidadMH), *decoded.Identificacion.TipoContingencia)
	assert.Nil(t, decoded.Identificacion.MotivoContin, "a stale reason is erased for types 1 to 4")
	assert.Equal(t, "X", decoded.Identificacion.CodigoGeneracion)

	otherType := int8(constants.OtroMotivo)
	fresh := "fresh"
	updated, err = utils.UpdateContingencyIdentification(document, &otherType, &fresh)
	require.NoError(t, err)
	decoded = decodeDeferredIdentification(t, updated)
	require.NotNil(t, decoded.Identificacion.MotivoContin)
	assert.Equal(t, "fresh", *decoded.Identificacion.MotivoContin)
}

// TestUpdateContingencyIdentificationPreservesNumbers checks that decimals, integers and zero survive the round trip.
func TestUpdateContingencyIdentificationPreservesNumbers(t *testing.T) {
	test.TestMain(t)

	document := `{"identificacion":{"codigoGeneracion":"X"},` +
		`"resumen":{"totalGravada":0.12345678,"totalPagar":100,"totalDescu":0,"iva":13.00000001,"grande":123456789012,"negativo":-5.5},` +
		`"cuerpoDocumento":[{"cantidad":1,"precioUni":99.99999999,"montoDescu":0}]}`

	contingencyType := int8(constants.NoDisponibilidadMH)
	updated, err := utils.UpdateContingencyIdentification(json.RawMessage(document), &contingencyType, nil)
	require.NoError(t, err)

	raw, err := json.Marshal(updated)
	require.NoError(t, err)
	output := string(raw)
	for _, fragment := range []string{
		`"totalGravada":0.12345678`, `"totalPagar":100`, `"totalDescu":0`, `"iva":13.00000001`,
		`"grande":123456789012`, `"negativo":-5.5`,
		`"precioUni":99.99999999`, `"montoDescu":0`, `"cantidad":1`,
	} {
		assert.Contains(t, output, fragment)
	}
	assert.NotContains(t, output, "e-", "no exponent notation may leak into the amounts")
}

// TestUpdateContingencyIdentificationSmallAmountsKeepPlainNotation checks that amounts below 0.000001 keep their
// original plain notation in the deferred document instead of being rewritten in exponent notation.
func TestUpdateContingencyIdentificationSmallAmountsKeepPlainNotation(t *testing.T) {
	test.TestMain(t)

	contingencyType := int8(constants.NoDisponibilidadMH)
	updated, err := utils.UpdateContingencyIdentification(
		`{"identificacion":{"codigoGeneracion":"X"},"resumen":{"precio":0.00000001}}`, &contingencyType, nil)
	require.NoError(t, err)

	raw, err := json.Marshal(updated)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"precio":0.00000001`)
}

// TestStoreDocumentInContingencyAllTypes checks every contingency type end to end through the store flow.
func TestStoreDocumentInContingencyAllTypes(t *testing.T) {
	test.TestMain(t)

	for contingencyType := int8(1); contingencyType <= 5; contingencyType++ {
		contingencyType := contingencyType
		t.Run(fmt.Sprintf("type %d", contingencyType), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			signer := &fakeContingencySigner{}
			harness := newStoreHarness(ctrl, signer, nil, nil)

			deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
				constants.FacturaElectronica, contingencyType, "  detailed reason  ")
			require.NoError(t, err)
			require.NotNil(t, deferred)

			assert.Equal(t, 1, signer.callCount(), "the signer is called exactly once")
			require.Len(t, harness.stored, 1)
			require.Len(t, harness.evidence, 1)
			assert.Equal(t, deferred.SignedDocument, harness.evidence[0])
			assert.Equal(t, normalizedJSON(t, deferred.Document), normalizedJSON(t, harness.stored[0]))
			assert.Equal(t, normalizedJSON(t, deferred.Document), jwsPayload(t, deferred.SignedDocument))

			response := deferred.Response()
			require.Equal(t, deferred.SignedDocument, response["firmaElectronica"])
			delete(response, "firmaElectronica")
			assert.Equal(t, jwsPayload(t, deferred.SignedDocument), normalizedJSON(t, response))
			assert.Equal(t, normalizedJSON(t, harness.stored[0]), normalizedJSON(t, response))

			decoded := decodeDeferredIdentification(t, jwsPayload(t, deferred.SignedDocument))
			require.NotNil(t, decoded.Identificacion.TipoContingencia)
			assert.Equal(t, int(contingencyType), *decoded.Identificacion.TipoContingencia)
			if contingencyType == constants.OtroMotivo {
				require.NotNil(t, decoded.Identificacion.MotivoContin)
				assert.Equal(t, "detailed reason", *decoded.Identificacion.MotivoContin)
			} else {
				assert.Nil(t, decoded.Identificacion.MotivoContin)
			}
		})
	}
}

// TestStoreDocumentInContingencyFailureModes checks that every failure returns an error, a nil result, and never
// leaves a half stored document.
func TestStoreDocumentInContingencyFailureModes(t *testing.T) {
	test.TestMain(t)

	t.Run("signer error stores nothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		signer := &fakeContingencySigner{err: errors.New("signer down")}
		harness := newStoreHarness(ctrl, signer, nil, nil)

		deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
			constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
		require.Error(t, err)
		assert.Nil(t, deferred)
		assert.Equal(t, 1, signer.callCount())
		assert.Zero(t, harness.createHits)
		assert.Zero(t, harness.repoHits)
	})

	t.Run("dte manager error is returned and the contingency record is not created", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		harness := newStoreHarness(ctrl, &fakeContingencySigner{}, errors.New("db down"), nil)

		deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
			constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
		require.Error(t, err)
		assert.Nil(t, deferred)
		assert.Equal(t, 1, harness.createHits)
		assert.Zero(t, harness.repoHits)
	})

	t.Run("repository error after the dte was stored is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		harness := newStoreHarness(ctrl, &fakeContingencySigner{}, nil, errors.New("repo down"))

		deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
			constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
		require.Error(t, err)
		assert.Nil(t, deferred)
		assert.Equal(t, 1, harness.createHits)
		assert.Equal(t, 1, harness.repoHits)
	})

	t.Run("malformed json fails before signing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		signer := &fakeContingencySigner{}
		harness := newStoreHarness(ctrl, signer, nil, nil)

		deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), "{broken",
			constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
		require.Error(t, err)
		assert.Nil(t, deferred)
		assert.Zero(t, signer.callCount())
		assert.Zero(t, harness.createHits)
	})

	t.Run("document without identification codes is not persisted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		harness := newStoreHarness(ctrl, &fakeContingencySigner{}, nil, nil)

		deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1),
			`{"identificacion":{"tipoDte":"01"}}`, constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
		require.Error(t, err)
		assert.Nil(t, deferred)
		assert.Zero(t, harness.createHits)
		assert.Zero(t, harness.repoHits)
	})
}

// TestDeferredDocumentResponseIsIsolated checks that Response returns an independent top level copy.
func TestDeferredDocumentResponseIsIsolated(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	harness := newStoreHarness(ctrl, &fakeContingencySigner{}, nil, nil)

	deferred, err := harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
		constants.FacturaElectronica, constants.NoDisponibilidadMH, "")
	require.NoError(t, err)

	before := normalizedJSON(t, deferred.Document)
	require.NotContains(t, deferred.Document, "firmaElectronica")

	first := deferred.Response()
	first["firmaElectronica"] = "tampered"
	first["injected"] = true
	delete(first, "emisor")
	first["identificacion"] = "replaced"

	assert.Equal(t, before, normalizedJSON(t, deferred.Document), "mutating the response must not touch the document")
	assert.NotContains(t, deferred.Document, "firmaElectronica")
	assert.NotContains(t, deferred.Document, "injected")
	assert.Contains(t, deferred.Document, "emisor")

	second := deferred.Response()
	assert.Equal(t, deferred.SignedDocument, second["firmaElectronica"])
	assert.NotContains(t, second, "injected")
	assert.Contains(t, second, "emisor")
	assert.Equal(t, before["emisor"], normalizedJSON(t, second)["emisor"])
}

// TestStoreDocumentInContingencyConcurrent checks that parallel calls never share signatures or structures.
func TestStoreDocumentInContingencyConcurrent(t *testing.T) {
	test.TestMain(t)

	const workers = 20
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	signer := &fakeContingencySigner{}
	harness := newStoreHarness(ctrl, signer, nil, nil)

	results := make([]*contingency.DeferredDocument, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code := fmt.Sprintf("CODE-%03d", i)
			results[i], errs[i] = harness.svc.StoreDocumentInContingency(buildContingencyCtx(1), dteJSONWithCode(code),
				constants.FacturaElectronica, int8(i%5)+1, "concurrent reason")
		}(i)
	}
	wg.Wait()

	assert.Equal(t, workers, signer.callCount())
	seen := make(map[string]bool)
	for i := 0; i < workers; i++ {
		require.NoError(t, errs[i])
		require.NotNil(t, results[i])
		code := fmt.Sprintf("CODE-%03d", i)

		decoded := decodeDeferredIdentification(t, jwsPayload(t, results[i].SignedDocument))
		assert.Equal(t, code, decoded.Identificacion.CodigoGeneracion, "worker %d received a foreign signature", i)
		require.NotNil(t, decoded.Identificacion.TipoContingencia)
		assert.Equal(t, int(int8(i%5)+1), *decoded.Identificacion.TipoContingencia)
		assert.Equal(t, code, decodeDeferredIdentification(t, results[i].Document).Identificacion.CodigoGeneracion)
		assert.False(t, seen[results[i].SignedDocument], "duplicated signature")
		seen[results[i].SignedDocument] = true
	}
	assert.Len(t, harness.stored, workers)
	assert.ElementsMatch(t, func() []string {
		var signatures []string
		for _, result := range results {
			signatures = append(signatures, result.SignedDocument)
		}
		return signatures
	}(), harness.evidence)
}

type retransmitHarness struct {
	svc       contingency.ContingencyManager
	batch     *recordingBatchTransmitter
	events    [][]coreDte.ContingencyDocument
	updates   []coreDte.DTEDetails
	expiredBy []string
	mu        sync.Mutex
}

// retransmitHarnessOptions tunes the retransmission fixture.
type retransmitHarnessOptions struct {
	batchSize int
	sendEvent func(group []coreDte.ContingencyDocument) error
}

func newRetransmitHarness(ctrl *gomock.Controller, signer *selectiveSigner, docs []coreDte.ContingencyDocument, options retransmitHarnessOptions) *retransmitHarness {
	harness := &retransmitHarness{batch: &recordingBatchTransmitter{}}
	if options.batchSize == 0 {
		options.batchSize = 100
	}

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().GetPending(gomock.Any(), gomock.Any()).Return(docs, nil)
	mockRepo.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ids []string, _ []string, _ map[string]coreDte.ReceptionResult, _ string, _ string, _ string) error {
			harness.mu.Lock()
			defer harness.mu.Unlock()
			harness.expiredBy = append(harness.expiredBy, ids...)
			return nil
		}).AnyTimes()

	mockEvent := mocks.NewMockContingencyEventSender(ctrl)
	mockEvent.EXPECT().PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, group []coreDte.ContingencyDocument) error {
			harness.events = append(harness.events, group)
			if options.sendEvent != nil {
				return options.sendEvent(group)
			}
			return nil
		}).AnyTimes()

	mockAuth := mocks.NewMockAuthManager(ctrl)
	mockAuth.EXPECT().GetBranchByBranchID(gomock.Any(), gomock.Any()).
		Return(&user.BranchOffice{ID: 1, User: &user.User{ID: 7, NIT: adversarialNITOne}}, nil).AnyTimes()
	mockCache := mocks.NewMockCacheManager(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).Return(`{"IssuedAt":1,"ExpiresAt":4102444800}`, nil).AnyTimes()
	mockCache.EXPECT().GetCredentials(gomock.Any()).Return(&authModels.HaciendaCredentials{Username: "u", Password: "p"}, nil).AnyTimes()

	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().UpdateDTE(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uint, details coreDte.DTEDetails) error {
			harness.mu.Lock()
			defer harness.mu.Unlock()
			harness.updates = append(harness.updates, details)
			return nil
		}).AnyTimes()

	harness.svc = contingency.NewContingencyManager(mockAuth, mockDTE, mockRepo, nil, mockCache, fakeContingencyTokens{}, signer, harness.batch,
		mockEvent, mocks.NewMockSequentialNumberManager(ctrl), nil, &transmitterModels.TransmissionConfig{BatchSize: options.batchSize})
	return harness
}

func adversarialJWS(id string) string {
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString([]byte(`{"id":"`+id+`"}`)) + ".c2lnbmF0dXJl"
}

func pendingDocument(id, nit, dteType string, contingencyType int8, signed bool) coreDte.ContingencyDocument {
	details := &coreDte.DTEDetails{
		ID:       id,
		DTEType:  dteType,
		JSONData: fmt.Sprintf(`{"identificacion":{"codigoGeneracion":"%s"}}`, id),
	}
	if signed {
		details.SignedDocument = adversarialJWS(id)
	}
	return coreDte.ContingencyDocument{
		ID:              "contingency-" + id,
		DocumentID:      id,
		BranchID:        1,
		ContingencyType: contingencyType,
		CreatedAt:       time.Now(),
		Document:        details,
		Branch:          &user.BranchOffice{ID: 1, User: &user.User{ID: 7, NIT: nit}},
	}
}

func eventIDs(group []coreDte.ContingencyDocument) []string {
	ids := make([]string, 0, len(group))
	for _, doc := range group {
		ids = append(ids, doc.Document.ID)
	}
	return ids
}

// TestContingencyRetransmitGroupsByIssuerAndType checks one event per issuer and contingency type, in order of first
// appearance, with the stored signatures sent byte for byte and no signing.
func TestContingencyRetransmitGroupsByIssuerAndType(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docs := []coreDte.ContingencyDocument{
		pendingDocument("A", adversarialNITOne, constants.FacturaElectronica, 1, true),
		pendingDocument("B", adversarialNITTwo, constants.FacturaElectronica, 1, true),
		pendingDocument("C", adversarialNITOne, constants.CCFElectronico, 5, true),
		pendingDocument("D", adversarialNITOne, constants.CCFElectronico, 1, true),
		pendingDocument("E", adversarialNITTwo, constants.FacturaElectronica, 1, true),
		pendingDocument("F", adversarialNITTwo, constants.FacturaElectronica, 5, true),
	}
	signer := &selectiveSigner{}
	harness := newRetransmitHarness(ctrl, signer, docs, retransmitHarnessOptions{})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	require.Len(t, harness.events, 4)
	assert.Equal(t, []string{"A", "D"}, eventIDs(harness.events[0]))
	assert.Equal(t, []string{"B", "E"}, eventIDs(harness.events[1]))
	assert.Equal(t, []string{"C"}, eventIDs(harness.events[2]))
	assert.Equal(t, []string{"F"}, eventIDs(harness.events[3]))
	for _, group := range harness.events {
		for _, doc := range group {
			assert.Equal(t, group[0].ContingencyType, doc.ContingencyType)
			assert.Equal(t, group[0].Branch.User.NIT, doc.Branch.User.NIT)
		}
	}

	assert.Zero(t, signer.callCount(), "documents with a stored signature are never signed again")
	assert.Empty(t, harness.updates, "documents with a stored signature are never rewritten")

	expected := map[string][]string{
		constants.FacturaElectronica: {adversarialJWS("A"), adversarialJWS("B"), adversarialJWS("E"), adversarialJWS("F")},
		constants.CCFElectronico:     {adversarialJWS("C"), adversarialJWS("D")},
	}
	for dteType, signatures := range expected {
		assert.ElementsMatch(t, signatures, harness.batch.documentsOf(dteType), "dte type %s", dteType)
	}
	for _, call := range harness.batch.calls {
		for _, document := range call.documents {
			id := jwsPayload(t, document)["id"].(string)
			var owner coreDte.ContingencyDocument
			for _, doc := range docs {
				if doc.Document.ID == id {
					owner = doc
				}
			}
			assert.Equal(t, owner.Branch.User.NIT, call.nit, "document %s sent under a foreign issuer", id)
			assert.Equal(t, owner.Document.DTEType, call.dteType, "document %s sent under a foreign dte type", id)
		}
	}
}

// TestContingencyRetransmitLegacyDocuments checks that documents without a stored signature are signed exactly once,
// that a failing one is skipped, and that its siblings still go.
func TestContingencyRetransmitLegacyDocuments(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docs := []coreDte.ContingencyDocument{
		pendingDocument("SIGNED-1", adversarialNITOne, constants.FacturaElectronica, 1, true),
		pendingDocument("LEGACY-OK", adversarialNITOne, constants.FacturaElectronica, 1, false),
		pendingDocument("LEGACY-FAILME", adversarialNITOne, constants.FacturaElectronica, 1, false),
		pendingDocument("SIGNED-2", adversarialNITOne, constants.FacturaElectronica, 1, true),
	}
	signer := &selectiveSigner{failMarker: "LEGACY-FAILME"}
	harness := newRetransmitHarness(ctrl, signer, docs, retransmitHarnessOptions{})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	assert.Equal(t, 2, signer.callCount(), "only the two legacy documents are signed, once each")
	require.Len(t, harness.updates, 1, "only the successfully signed legacy document is persisted")
	assert.Equal(t, "LEGACY-OK", harness.updates[0].ID)
	assert.NotEmpty(t, harness.updates[0].SignedDocument)

	sent := harness.batch.documentsOf(constants.FacturaElectronica)
	require.Len(t, sent, 3)
	assert.Equal(t, adversarialJWS("SIGNED-1"), sent[0])
	assert.Equal(t, harness.updates[0].SignedDocument, sent[1])
	assert.Equal(t, adversarialJWS("SIGNED-2"), sent[2])
	for _, document := range sent {
		assert.NotContains(t, string(mustDecodePayload(t, document)), "LEGACY-FAILME")
	}
}

func mustDecodePayload(t *testing.T, jws string) []byte {
	t.Helper()
	parts := strings.Split(jws, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	return raw
}

// TestContingencyRetransmitAllLegacyFailing checks that a batch where nothing could be signed is never transmitted.
func TestContingencyRetransmitAllLegacyFailing(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docs := []coreDte.ContingencyDocument{
		pendingDocument("FAILME-1", adversarialNITOne, constants.FacturaElectronica, 1, false),
		pendingDocument("FAILME-2", adversarialNITOne, constants.FacturaElectronica, 1, false),
	}
	signer := &selectiveSigner{failMarker: "FAILME"}
	harness := newRetransmitHarness(ctrl, signer, docs, retransmitHarnessOptions{})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	assert.Equal(t, 2, signer.callCount())
	assert.Empty(t, harness.batch.calls)
	assert.Empty(t, harness.updates)
}

// TestContingencyRetransmitBatchSplitting checks that splitting keeps every signature intact, once, in order.
func TestContingencyRetransmitBatchSplitting(t *testing.T) {
	test.TestMain(t)

	for _, batchSize := range []int{1, 2, 3, 7, 50} {
		batchSize := batchSize
		t.Run(fmt.Sprintf("batch size %d", batchSize), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var docs []coreDte.ContingencyDocument
			var expected []string
			for i := 0; i < 7; i++ {
				id := fmt.Sprintf("DOC-%d", i)
				docs = append(docs, pendingDocument(id, adversarialNITOne, constants.FacturaElectronica, 1, true))
				expected = append(expected, adversarialJWS(id))
			}
			signer := &selectiveSigner{}
			harness := newRetransmitHarness(ctrl, signer, docs, retransmitHarnessOptions{batchSize: batchSize})

			require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

			assert.Len(t, harness.events, 1)
			assert.Equal(t, expected, harness.batch.allDocuments(), "signatures lost, duplicated or reordered")
			expectedBatches := (7 + batchSize - 1) / batchSize
			assert.Len(t, harness.batch.calls, expectedBatches)
			for _, call := range harness.batch.calls {
				assert.LessOrEqual(t, len(call.documents), batchSize)
				assert.NotEmpty(t, call.documents)
			}
			assert.Zero(t, signer.callCount())
		})
	}
}

// TestContingencyRetransmitEventFailureIsolated checks that a failed event only blocks its own group.
func TestContingencyRetransmitEventFailureIsolated(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docs := []coreDte.ContingencyDocument{
		pendingDocument("A", adversarialNITOne, constants.FacturaElectronica, 1, true),
		pendingDocument("B", adversarialNITTwo, constants.CCFElectronico, 1, true),
		pendingDocument("C", adversarialNITOne, constants.FacturaElectronica, 5, true),
	}
	harness := newRetransmitHarness(ctrl, &selectiveSigner{}, docs, retransmitHarnessOptions{
		sendEvent: func(group []coreDte.ContingencyDocument) error {
			if group[0].Branch.User.NIT == adversarialNITTwo {
				return errors.New("event rejected")
			}
			return nil
		},
	})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	require.Len(t, harness.events, 3)
	assert.ElementsMatch(t, []string{adversarialJWS("A"), adversarialJWS("C")}, harness.batch.documentsOf(constants.FacturaElectronica))
	assert.Empty(t, harness.batch.documentsOf(constants.CCFElectronico), "documents of a failed event are not transmitted")
}

// TestContingencyRetransmitEventAlreadyExists checks that an existing event does not stop the transmission of the
// stored signatures.
func TestContingencyRetransmitEventAlreadyExists(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docs := []coreDte.ContingencyDocument{
		pendingDocument("A", adversarialNITOne, constants.FacturaElectronica, 1, true),
		pendingDocument("B", adversarialNITOne, constants.FacturaElectronica, 1, true),
	}
	broken := pendingDocument("BROKEN", adversarialNITOne, constants.FacturaElectronica, 1, true)
	broken.Document.JSONData = "{not json"

	signer := &selectiveSigner{}
	harness := newRetransmitHarness(ctrl, signer, docs, retransmitHarnessOptions{
		sendEvent: func(group []coreDte.ContingencyDocument) error {
			return &contingency.ContingencyEventExistsError{Message: "already exists", Documents: []coreDte.ContingencyDocument{broken}}
		},
	})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	assert.Equal(t, []string{adversarialJWS("A"), adversarialJWS("B")}, harness.batch.documentsOf(constants.FacturaElectronica))
	assert.Zero(t, signer.callCount())
	assert.Empty(t, harness.updates, "an unreadable existing document is not rewritten")
}

// TestContingencyRetransmitExpiredDocuments checks that documents past the transmission window are not sent nor
// reported in an event, and that valid ones still are.
func TestContingencyRetransmitExpiredDocuments(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	old := time.Now().Add(-73 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	expired := pendingDocument("EXPIRED", adversarialNITOne, constants.FacturaElectronica, 1, true)
	expired.EventStampedAt = &old
	fresh := pendingDocument("FRESH", adversarialNITOne, constants.FacturaElectronica, 1, true)
	fresh.EventStampedAt = &recent
	unstamped := pendingDocument("UNSTAMPED", adversarialNITOne, constants.FacturaElectronica, 1, true)

	harness := newRetransmitHarness(ctrl, &selectiveSigner{}, []coreDte.ContingencyDocument{expired, fresh, unstamped}, retransmitHarnessOptions{})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	assert.Equal(t, []string{"contingency-EXPIRED"}, harness.expiredBy)
	require.Len(t, harness.events, 1)
	assert.Equal(t, []string{"FRESH", "UNSTAMPED"}, eventIDs(harness.events[0]))
	assert.Equal(t, []string{adversarialJWS("FRESH"), adversarialJWS("UNSTAMPED")}, harness.batch.allDocuments())
}

// TestContingencyRetransmitOnlyExpiredDocuments checks that nothing is sent when every document expired.
func TestContingencyRetransmitOnlyExpiredDocuments(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	old := time.Now().Add(-100 * time.Hour)
	expired := pendingDocument("EXPIRED", adversarialNITOne, constants.FacturaElectronica, 1, true)
	expired.EventStampedAt = &old

	harness := newRetransmitHarness(ctrl, &selectiveSigner{}, []coreDte.ContingencyDocument{expired}, retransmitHarnessOptions{})

	require.NoError(t, harness.svc.RetransmitPendingDocuments(context.Background()))

	assert.Empty(t, harness.events)
	assert.Empty(t, harness.batch.calls)
}
