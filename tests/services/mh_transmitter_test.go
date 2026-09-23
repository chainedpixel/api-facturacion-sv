package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	models2 "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleTestDocument returns a valid test document map for transmitter processing.
func sampleTestDocument() map[string]interface{} {
	return map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version":          float64(2),
			"tipoDte":          "01",
			"numeroControl":    "DTE-01-00000000-000000000000001",
			"codigoGeneracion": "12345678-1234-1234-1234-123456789012",
		},
	}
}

// TestNewMHTransmitter_TimeoutConfiguration verifies the HTTP client timeout is 8 seconds.
func TestNewMHTransmitter_TimeoutConfiguration(t *testing.T) {
	test.TestMain(t)

	trans := transmitter.NewMHTransmitter(&stubHaciendaAuth{}, nil)
	require.NotNil(t, trans)

	mhTrans, ok := trans.(*transmitter.MHTransmitter)
	require.True(t, ok)
	assert.Equal(t, 8*time.Second, mhTrans.GetTimeout())
}

// TestMHTransmitter_CheckDocumentStatus_HTTP200OK verifies that HTTP 200 OK is accepted.
func TestMHTransmitter_CheckDocumentStatus_HTTP200OK(t *testing.T) {
	test.TestMain(t)

	haciendaResponse := models2.HaciendaResponse{
		Version:            1,
		Ambient:            "00",
		VersionApp:         1,
		Status:             "PROCESADO",
		GenerationCode:     "12345678-1234-1234-1234-123456789012",
		ReceptionStamp:     "2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
		ProcessingDate:     "2026-03-09 10:00:00",
		ClassifyMessage:    "1",
		MessageCode:        "001",
		DescriptionMessage: "RECIBIDO",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(haciendaResponse)
	}))
	defer server.Close()

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&stubHaciendaAuth{}, nil)
	doc := sampleTestDocument()

	result, err := trans.CheckDocumentStatus(context.Background(), doc, "06140101010000")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "PROCESADO", result.Status)
	assert.NotNil(t, result.ReceptionStamp)
	assert.Equal(t, "2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM", *result.ReceptionStamp)
	assert.Equal(t, "RECIBIDO", result.MessageDesc)
}

// TestMHTransmitter_CheckDocumentStatus_HTTP202Accepted verifies that HTTP 202 Accepted is accepted.
func TestMHTransmitter_CheckDocumentStatus_HTTP202Accepted(t *testing.T) {
	test.TestMain(t)

	haciendaResponse := models2.HaciendaResponse{
		Version:            1,
		Ambient:            "00",
		VersionApp:         1,
		Status:             "RECIBIDO",
		GenerationCode:     "12345678-1234-1234-1234-123456789012",
		ReceptionStamp:     "",
		ProcessingDate:     "2026-03-09 10:00:00",
		MessageCode:        "002",
		DescriptionMessage: "EN PROCESO",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(haciendaResponse)
	}))
	defer server.Close()

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&stubHaciendaAuth{}, nil)
	doc := sampleTestDocument()

	result, err := trans.CheckDocumentStatus(context.Background(), doc, "06140101010000")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "RECIBIDO", result.Status)
}

// TestMHTransmitter_CheckDocumentStatus_HTTP400Error verifies that HTTP 400 returns an error.
func TestMHTransmitter_CheckDocumentStatus_HTTP400Error(t *testing.T) {
	test.TestMain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&stubHaciendaAuth{}, nil)
	doc := sampleTestDocument()

	result, err := trans.CheckDocumentStatus(context.Background(), doc, "06140101010000")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestMHTransmitter_CheckDocumentStatus_HTTP500Error verifies that HTTP 500 returns an error.
func TestMHTransmitter_CheckDocumentStatus_HTTP500Error(t *testing.T) {
	test.TestMain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&stubHaciendaAuth{}, nil)
	doc := sampleTestDocument()

	result, err := trans.CheckDocumentStatus(context.Background(), doc, "06140101010000")
	assert.Error(t, err)
	assert.Nil(t, result)
}
