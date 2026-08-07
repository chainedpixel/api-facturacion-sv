package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DTEHandler struct {
	GenericHandler      *GenericCreatorDTEHandler
	dteConsultUseCase   *dte.DTEConsultUseCase
	invalidationUseCase *dte.InvalidationUseCase
	respWriter          *response.ResponseWriter
}

func NewDTEHandler(
	dteConsultUseCase *dte.DTEConsultUseCase,
	invalidationUseCase *dte.InvalidationUseCase,
	genericHandler *GenericCreatorDTEHandler,
) *DTEHandler {
	return &DTEHandler{
		GenericHandler:      genericHandler,
		dteConsultUseCase:   dteConsultUseCase,
		invalidationUseCase: invalidationUseCase,
		respWriter:          response.NewResponseWriter(),
	}
}

// GetByGenerationCode handles the HTTP request to retrieve a DTE by its generation code
func (h *DTEHandler) GetByGenerationCode(w http.ResponseWriter, r *http.Request) {
	generationCode := helpers.GetRequestVar(r, "id")

	dte, err := h.dteConsultUseCase.GetByGenerationCode(r.Context(), generationCode)
	if err != nil {
		h.respWriter.HandleError(w, err)
		return
	}

	h.respWriter.Success(w, http.StatusOK, dte, nil)
}

// GetAll handles the HTTP request to retrieve all DTEs
func (h *DTEHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	dtes, err := h.dteConsultUseCase.GetAllDTEs(r.Context(), r)
	if err != nil {
		h.respWriter.HandleError(w, err)
		return
	}

	h.respWriter.Success(w, http.StatusOK, dtes, nil)
}

// InvalidateDocument handles the HTTP request to invalidate a DTE
func (h *DTEHandler) InvalidateDocument(w http.ResponseWriter, r *http.Request) {
	var req structs.CreateInvalidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logs.Error("Failed to decode request body", map[string]interface{}{"error": err.Error()})
		h.respWriter.Error(w, http.StatusBadRequest, "Invalid request format", nil)
		return
	}

	invalidation, err := h.invalidationUseCase.InvalidateDocument(r.Context(), req)
	if err != nil {
		h.respWriter.HandleError(w, err)
		return
	}

	h.respWriter.Success(w, http.StatusOK, invalidation, nil)
}
