package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/application/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
)

type AuthHandler struct {
	authUseCase *auth.AuthUseCase
	respWriter  *response.ResponseWriter
}

func NewAuthHandler(authUseCase *auth.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
		respWriter:  response.NewResponseWriter(),
	}
}

// Login authenticates a user with API credentials and returns a session token.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.AuthCredentials
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respWriter.Error(w, http.StatusBadRequest, "Invalid request format", nil)
		return
	}

	response, err := h.authUseCase.Login(r.Context(), &req)
	if err != nil {
		h.respWriter.HandleError(w, err)
		return
	}

	h.respWriter.Success(w, http.StatusOK, response, nil)
}

// Register creates a new user account along with its branches.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req user.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respWriter.Error(w, http.StatusBadRequest, "Invalid request format", nil)
		return
	}

	response, err := h.authUseCase.Register(r.Context(), &req)
	if err != nil {
		h.respWriter.HandleError(w, err)
		return
	}

	h.respWriter.Success(w, http.StatusCreated, response, nil)
}
