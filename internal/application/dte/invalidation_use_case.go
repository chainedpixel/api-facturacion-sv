package dte

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	structs2 "github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	authManager "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	dteInterfaces "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
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

// InvalidateDocument transmite el evento de invalidacion y lo registra con su evidencia (estructura firmada,
// sello, fecha y respuesta de Hacienda). Normativa v2.0, regla 11: el evento se entrega al receptor como
// Archivo DTE, que se devuelve en la respuesta y queda disponible en GET /dte/{id}/eventos.
func (u *InvalidationUseCase) InvalidateDocument(ctx context.Context, request structs.CreateInvalidationRequest) (*dte.EventArchiveResponse, error) {
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

	if err := u.invalidationManager.ValidateDependencies(ctx, claims.BranchID, originalDTE); err != nil {
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
		u.registerRejectedInvalidation(ctx, mhInvalidation, request, claims.BranchID, err)
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

	event, err := buildInvalidationEvent(mhInvalidation, request, claims.BranchID, result)
	if err != nil {
		return nil, err
	}
	if err := u.dteManager.RegisterEvent(ctx, event); err != nil {
		logs.Error("Failed to register invalidation event", map[string]interface{}{
			"error": err.Error(),
			"code":  event.GenerationCode,
		})
		return nil, err
	}

	response := dteInterfaces.ToEventArchiveResponse(*event)
	return &response, nil
}

// registerRejectedInvalidation keeps an invalidation event that Hacienda rejected, with the transmitted JWS and the
// Hacienda response, so the attempt and its observations stay available. Other transmission failures are not stored,
// and a failure to store the attempt does not change the error returned to the consumer.
func (u *InvalidationUseCase) registerRejectedInvalidation(ctx context.Context, mhInvalidation *structs2.InvalidationResponse,
	request structs.CreateInvalidationRequest, branchID uint, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logs.Error("Recovered while registering rejected invalidation event", map[string]interface{}{"panic": fmt.Sprint(recovered)})
		}
	}()

	var haciendaErr *hacienda_error.HaciendaResponseError
	if !errors.As(err, &haciendaErr) || haciendaErr.Status != hacienda_error.StatusRejected {
		return
	}

	signedDocument := ""
	var transmissionErr *ports.TransmissionError
	if errors.As(err, &transmissionErr) {
		signedDocument = transmissionErr.SignedDocument
	}

	var response interface{} = haciendaErr
	if haciendaErr.Response != nil {
		response = haciendaErr.Response
	}
	rawResponse, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return
	}

	event, buildErr := buildInvalidationEvent(mhInvalidation, request, branchID, &transmitterModels.TransmitResult{
		SignedDocument:   signedDocument,
		ProcessingDate:   haciendaErr.ProcessedAt,
		HaciendaResponse: string(rawResponse),
	})
	if buildErr != nil {
		return
	}
	event.Status = constants.DocumentRejected

	if registerErr := u.dteManager.RegisterEvent(ctx, event); registerErr != nil {
		logs.Error("Failed to register rejected invalidation event", map[string]interface{}{
			"error": registerErr.Error(),
			"code":  event.GenerationCode,
		})
	}
}

// buildInvalidationEvent arma el registro del evento con la evidencia devuelta por Hacienda
func buildInvalidationEvent(mhInvalidation *structs2.InvalidationResponse, request structs.CreateInvalidationRequest,
	branchID uint, result *transmitterModels.TransmitResult) (*dte.DTEEvent, error) {
	jsonData, err := json.Marshal(mhInvalidation)
	if err != nil {
		return nil, err
	}

	relatedDTE := request.GenerationCode
	return &dte.DTEEvent{
		GenerationCode:   mhInvalidation.Identificacion.CodigoGeneracion,
		BranchID:         branchID,
		EventType:        dte.EventInvalidation,
		RelatedDTEID:     &relatedDTE,
		ReplacementDTEID: request.ReplacementGenerationCode,
		JSONData:         string(jsonData),
		SignedDocument:   result.SignedDocument,
		ReceptionStamp:   result.ReceptionStamp,
		ProcessingDate:   result.ProcessingDate,
		HaciendaResponse: result.HaciendaResponse,
		Status:           constants.DocumentReceived,
		CreatedAt:        utils.TimeNow(),
	}, nil
}
