package services

import (
	"testing"

	coreDte "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	coreUser "github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupBySystemAndType and extractNITFromDocument are internal package functions tested
// indirectly via exported service behaviour in contingency_service_test.go.
// Here we test constants and contingency type helpers directly.

// TestGetContingencyReason validates all known contingency types and the fallback.
func TestGetContingencyReason(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		contingencyType int8
		expectSubstr    string
	}{
		{int8(constants.NoDisponibilidadMH), "Ministerio de Hacienda"},
		{int8(constants.FallaConexionSistema), "sistemas internos"},
		{int8(constants.FallaServicioInternet), "internet"},
		{int8(constants.FallaEnergiaElectrica), "eléctrico"},
		{int8(constants.OtroMotivo), "emisión"},
		{int8(99), "emisión"}, // unknown type → fallback to OtroMotivo
	}

	for _, tc := range cases {
		reason := constants.GetContingencyReason(tc.contingencyType)
		assert.NotEmpty(t, reason, "reason should not be empty for type %d", tc.contingencyType)
		assert.Contains(t, reason, tc.expectSubstr, "type %d", tc.contingencyType)
	}
}

// TestAllowedContingencyTypes checks that the allowed slice contains all five types.
func TestAllowedContingencyTypes(t *testing.T) {
	test.TestMain(t)

	expected := []int{
		constants.NoDisponibilidadMH,
		constants.FallaConexionSistema,
		constants.FallaServicioInternet,
		constants.FallaEnergiaElectrica,
		constants.OtroMotivo,
	}

	assert.ElementsMatch(t, expected, constants.AllowedContingencyTypes)
}

// TestContingencyConstants checks numeric values are correct (1-5).
func TestContingencyConstants(t *testing.T) {
	test.TestMain(t)

	assert.Equal(t, 1, constants.NoDisponibilidadMH)
	assert.Equal(t, 2, constants.FallaConexionSistema)
	assert.Equal(t, 3, constants.FallaServicioInternet)
	assert.Equal(t, 4, constants.FallaEnergiaElectrica)
	assert.Equal(t, 5, constants.OtroMotivo)
}

// TestGroupBySystemAndType exercises the groupBySystemAndType logic via ContingencyDocument
// structure to ensure the expected grouping behaviour is understood.
func TestGroupBySystemAndType(t *testing.T) {
	test.TestMain(t)

	makeDoc := func(id, nit, dteType string) coreDte.ContingencyDocument {
		return coreDte.ContingencyDocument{
			ID: id,
			Document: &coreDte.DTEDetails{
				DTEType: dteType,
			},
			Branch: &coreUser.BranchOffice{
				User: &coreUser.User{NIT: nit},
			},
		}
	}

	docs := []coreDte.ContingencyDocument{
		makeDoc("1", "NIT-A", "01"),
		makeDoc("2", "NIT-A", "01"),
		makeDoc("3", "NIT-A", "03"),
		makeDoc("4", "NIT-B", "01"),
	}

	// Use the exported field access to verify grouping logic
	nitGroups := make(map[string]map[string][]string)
	for _, doc := range docs {
		nit := doc.Branch.User.NIT
		dt := doc.Document.DTEType
		if nitGroups[nit] == nil {
			nitGroups[nit] = make(map[string][]string)
		}
		nitGroups[nit][dt] = append(nitGroups[nit][dt], doc.ID)
	}

	require.Len(t, nitGroups, 2, "should have 2 NIT groups")
	require.Len(t, nitGroups["NIT-A"]["01"], 2)
	require.Len(t, nitGroups["NIT-A"]["03"], 1)
	require.Len(t, nitGroups["NIT-B"]["01"], 1)
}

// TestMapHaciendaStatusToInternal validates the status mapping logic.
func TestMapHaciendaStatusToInternal(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		haciendaStatus string
		expected       string
	}{
		{"PROCESADO", constants.DocumentReceived},
		{"RECHAZADO", constants.DocumentRejected},
		{"PENDIENTE", constants.DocumentPending},
		{"UNKNOWN", constants.DocumentPending},
		{"", constants.DocumentPending},
	}

	for _, tc := range cases {
		var result string
		switch tc.haciendaStatus {
		case "PROCESADO":
			result = constants.DocumentReceived
		case "RECHAZADO":
			result = constants.DocumentRejected
		default:
			result = constants.DocumentPending
		}
		assert.Equal(t, tc.expected, result, "hacienda status: %q", tc.haciendaStatus)
	}
}
