package dte

import (
	"context"

	structs2 "github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"

	"github.com/chainedpixel/ordo-factus/internal/application/ports"
	authManager "github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	dteInterfaces "github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type InvalidationUseCase struct {
	dteManager          dteInterfaces.DTEManager
	authManager         authManager.AuthManager
	invalidationManager invalidation.InvalidationManager
	mapper              *request_mapper.InvalidationMapper
	transmitter         ports.BaseTransmitter
}

func NewInvalidationUseCase(dteManager dteInterfaces.DTEManager, invalidationManager invalidation.InvalidationManager, authManager authManager.AuthManager, transmitter ports.BaseTransmitter) *InvalidationUseCase {
	return &InvalidationUseCase{
		dteManager:          dteManager,
		invalidationManager: invalidationManager,
		authManager:         authManager,
		transmitter:         transmitter,
		mapper:              request_mapper.NewInvalidationMapper(),
	}
}

func (u *InvalidationUseCase) InvalidateDocument(ctx context.Context, request structs.CreateInvalidationRequest) (*structs2.InvalidationResponse, error) {
	claims := ctx.Value("claims").(*models.AuthClaims)
	token := ctx.Value("token").(string)

	if err := u.mapper.ValidateInvalidationReRequest(&request); err != nil {
		return nil, err
	}

	if err := u.invalidationManager.ValidateStatus(ctx, claims.BranchID, request); err != nil {
		return nil, err
	}

	originalDTE, err := u.dteManager.GetByGenerationCode(ctx, claims.BranchID, request.GenerationCode)
	if err != nil {
		return nil, err
	}

	issuer, err := u.authManager.GetIssuer(ctx, claims.BranchID)
	if err != nil {
		return nil, err
	}

	invalidationDocument, err := u.mapper.MapToInvalidationData(&request, issuer, originalDTE.Details, originalDTE.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err = u.invalidationManager.Validate(ctx, claims.BranchID, invalidationDocument); err != nil {
		return nil, err
	}

	mhInvalidation := response_mapper.ToMHInvalidation(invalidationDocument)
	if mhInvalidation == nil {
		logs.Error("Error mapping invoice to hacienda model", map[string]interface{}{"error": "nil model"})
		return nil, shared_error.NewFormattedGeneralServiceError("InvalidationUseCase", "InvalidateDocument", "ErrorMapping", "MH model")
	}

	result, err := u.transmitter.RetryTransmission(ctx, mhInvalidation, token, claims.NIT)
	if err != nil {
		return nil, err
	}
	if result.Status != ReceivedStatus {
		logs.Warn("Error transmitting invalidation", map[string]interface{}{"error": "TransmissionFailed"})
		return nil, dte_errors.NewDTEErrorSimple("TransmissionFailed")
	}

	if err := u.invalidationManager.InvalidateDocument(ctx, claims.BranchID, request.GenerationCode); err != nil {
		logs.Error("Failed to update original DTE status", map[string]interface{}{
			"error": err.Error(),
			"code":  request.GenerationCode,
		})
		return nil, err
	}

	return mhInvalidation, nil
}
