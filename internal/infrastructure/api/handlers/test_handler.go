package handlers

import (
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/domain/test_endpoint"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type TestHandler struct {
	testManager    test_endpoint.TestManager
	responseWriter *response.ResponseWriter
}

func NewTestHandler(testManager test_endpoint.TestManager) *TestHandler {
	return &TestHandler{
		testManager:    testManager,
		responseWriter: response.NewResponseWriter(),
	}
}

// RunSystemTest executes the end-to-end system smoke test and returns its result.
func (h *TestHandler) RunSystemTest(w http.ResponseWriter, r *http.Request) {
	result, err := h.testManager.RunSystemTest(r.Context())
	if err != nil {
		logs.Error("System test failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.responseWriter.Error(w, http.StatusInternalServerError, "System test failed", nil)
		return
	}

	h.responseWriter.Success(w, http.StatusOK, result, nil)
}
