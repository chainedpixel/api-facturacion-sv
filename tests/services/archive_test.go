package services

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	test "github.com/chainedpixel/ordo-factus/tests"
)

// jws arma un JWS de prueba cuyo payload es exactamente la estructura indicada
func jws(payload string) string {
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".firma"
}

// Normativa v2.0, Anexo I regla 4.1: Archivo DTE = estructura firmada integra + firmaElectronica + selloRecibido
func TestBuildArchive_KeepsSignedStructureAndAddsSignatureAndStamp(t *testing.T) {
	test.TestMain(t)
	structure := `{"identificacion":{"version":2,"codigoGeneracion":"ABC"},"apendice":null}`
	signed := jws(structure)
	stamp := "2026917C89FAE3A64A0AA18ED01933D5DE4BWBHE"

	archive, err := dte_documents.BuildArchive(signed, &stamp)
	require.NoError(t, err)

	expected := `{"identificacion":{"version":2,"codigoGeneracion":"ABC"},"apendice":null,` +
		`"firmaElectronica":"` + signed + `","selloRecibido":"` + stamp + `"}`
	assert.Equal(t, expected, string(archive), "la estructura debe conservarse byte a byte, sin reordenar llaves")
	assert.True(t, json.Valid(archive))
}

func TestBuildArchive_WithoutStampSendsNull(t *testing.T) {
	test.TestMain(t)
	archive, err := dte_documents.BuildArchive(jws(`{"identificacion":{}}`), nil)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(archive, &parsed))
	assert.Nil(t, parsed["selloRecibido"])
	assert.NotEmpty(t, parsed["firmaElectronica"])
}

func TestBuildArchive_RejectsInvalidSignedDocument(t *testing.T) {
	test.TestMain(t)
	for _, signed := range []string{"", "no-es-un-jws", "a.b", "a.%%%.c", jws(`[1,2]`)} {
		_, err := dte_documents.BuildArchive(signed, nil)
		assert.Error(t, err, "JWS invalido: %q", signed)
	}
}
