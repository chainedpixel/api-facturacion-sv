package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/response"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
)

// GenericCreatorDTEHandler handles requests to create any type of DTE document
type GenericCreatorDTEHandler struct {
	documentConfigs    map[string]helpers.DocumentConfig
	respWriter         *response.ResponseWriter
	contingencyHandler *helpers.ContingencyHandler
}

// NewGenericDTEHandler creates a new instance of GenericCreatorDTEHandler
func NewGenericDTEHandler(contingencyHandler *helpers.ContingencyHandler) *GenericCreatorDTEHandler {
	return &GenericCreatorDTEHandler{
		documentConfigs:    make(map[string]helpers.DocumentConfig),
		respWriter:         response.NewResponseWriter(),
		contingencyHandler: contingencyHandler,
	}
}

// RegisterDocument registers a new document type to be handled
func (h *GenericCreatorDTEHandler) RegisterDocument(path string, config helpers.DocumentConfig) {
	h.documentConfigs[path] = config
}

// HandleCreate handles the creation of any document type
func (h *GenericCreatorDTEHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	config, err := h.getDocumentTypeFromPath(path)
	if err != nil {
		h.respWriter.Error(w, http.StatusNotFound, "Document type not supported", nil)
		return
	}

	requestType := reflect.TypeOf(config.RequestType)
	request := reflect.New(requestType.Elem()).Interface()

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		logs.Error("Failed to decode request body", map[string]interface{}{"error": err.Error()})
		h.respWriter.Error(w, http.StatusBadRequest, "Invalid request format", nil)
		return
	}

	resp, options, err := config.UseCase.Create(r.Context(), request)
	if err != nil {
		logs.Warn("Error processing document because", map[string]interface{}{"error": err.Error()})

		if config.UsesContingency {
			err = h.handleErrorForContingency(r.Context(), resp, config.DocumentType, options, err, w)
			if err != nil {
				h.respWriter.HandleError(w, err)
				return
			}
			return
		} else {
			h.respWriter.HandleError(w, err)
			return
		}
	}

	h.respWriter.Success(w, http.StatusCreated, resp, options)
}

// handleErrorForContingency stores the document in contingency when the error calls for it and responds with the
// signed deferred document, the same one that is later transmitted to Hacienda
func (h *GenericCreatorDTEHandler) handleErrorForContingency(ctx context.Context, dte interface{}, dteType string, options *response.SuccessOptions, err error, w http.ResponseWriter) error {
	logs.Warn("Error transmitting DTE because", map[string]interface{}{
		"error": err.Error(),
	})

	deferred := h.contingencyHandler.HandleContingency(ctx, dte, dteType, err)
	if deferred == nil {
		logs.Error("Error creating DTE contingency", map[string]interface{}{"error": err.Error()})
		return err
	}

	h.respWriter.Success(w, http.StatusCreated, deferred.Response(), options)
	return nil
}

// GetDocumentConfigs returns the registered document configuration
func (h *GenericCreatorDTEHandler) GetDocumentConfigs() map[string]helpers.DocumentConfig {
	return h.documentConfigs
}

// getDocumentTypeFromPath retrieves the document type based on the path
func (h *GenericCreatorDTEHandler) getDocumentTypeFromPath(path string) (helpers.DocumentConfig, error) {
	for key := range h.documentConfigs {
		if strings.Contains(path, key) {
			return h.documentConfigs[key], nil
		}
	}

	return helpers.DocumentConfig{}, errors.New("error was found in the path")
}
