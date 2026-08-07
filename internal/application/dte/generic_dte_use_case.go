package dte

import (
	"context"
	"errors"
	"strings"

	"github.com/chainedpixel/ordo-factus/config"
	appPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	transmissionPorts "github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// GenericDTEUseCase implements a generic use case for any DTE type
type GenericDTEUseCase struct {
	authService       auth.AuthManager
	dteService        transmissionPorts.DTEManager
	transmitter       appPorts.BaseTransmitter
	service           ports.DTEService
	sequentialManager transmissionPorts.SequentialNumberManager
	mapper            mapper.DTEMapper
	responseMapper    mapper.ResponseMapperFunc
	additionalOps     AdditionalOperationsFunc
}

// NewGenericDTEUseCase creates a new instance of GenericDTEUseCase
func NewGenericDTEUseCase(
	authService auth.AuthManager,
	dteService transmissionPorts.DTEManager,
	transmitter appPorts.BaseTransmitter,
	service ports.DTEService,
	sequentialManager transmissionPorts.SequentialNumberManager,
	mapper mapper.DTEMapper,
	responseMapper mapper.ResponseMapperFunc,
	additionalOps AdditionalOperationsFunc,
) *GenericDTEUseCase {
	return &GenericDTEUseCase{
		authService:       authService,
		dteService:        dteService,
		transmitter:       transmitter,
		service:           service,
		sequentialManager: sequentialManager,
		mapper:            mapper,
		responseMapper:    responseMapper,
		additionalOps:     additionalOps,
	}
}

// Create processes any DTE type using a generic flow
func (u *GenericDTEUseCase) Create(ctx context.Context, req interface{}) (interface{}, *response.SuccessOptions, error) {
	claims := ctx.Value("claims").(*models.AuthClaims)
	token := ctx.Value("token").(string)

	issuer, err := u.authService.GetIssuer(ctx, claims.BranchID)
	if err != nil {
		logs.Error("Error getting issuer information", map[string]interface{}{"error": err.Error()})
		return nil, nil, err
	}

	domainModel, err := u.mapper.MapToDomainModel(req, issuer)
	if err != nil {
		logs.Error("Error mapping to domain model", map[string]interface{}{"error": err.Error()})
		return nil, nil, err
	}

	result, err := u.service.Create(ctx, domainModel, claims.BranchID)
	if err != nil {
		logs.Error("Error creating DTE at service level", map[string]interface{}{"error": err.Error()})
		return nil, nil, err
	}

	mhModel := u.responseMapper(result)

	generationCode, err := extractGenerationCode(mhModel)
	if err != nil {
		logs.Error("Error extracting generation code", map[string]interface{}{"error": err.Error()})
		return nil, nil, err
	}

	controlNumber, err := extractControlNumber(mhModel)
	if err != nil {
		logs.Error("Error extracting control number", map[string]interface{}{"error": err.Error()})
		return nil, nil, err
	}

	options := &response.SuccessOptions{
		Ambient:        config.Server.AmbientCode,
		GenerationCode: generationCode,
		EmissionDate:   utils.TimeNow(),
	}

	transmitResult, err := u.transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
	if err != nil {
		shouldHandleAsContingency := u.shouldHandleAsContingency(err)

		if shouldHandleAsContingency {
			logs.Info("Transmission failed - will handle as contingency, keeping reservation", map[string]interface{}{
				"controlNumber": controlNumber,
				"error":         err.Error(),
			})
		} else {
			rejectionReason := err.Error()
			haciendaCode := ""

			var haciendaErr *hacienda_error.HaciendaResponseError
			if errors.As(err, &haciendaErr) {
				haciendaCode = haciendaErr.Code
				rejectionReason = haciendaErr.Description
				if len(haciendaErr.Observations) > 0 {
					rejectionReason += " | Observaciones: " + strings.Join(haciendaErr.Observations, "; ")
				}
				logs.Info("Hacienda error detected for reservation release", map[string]interface{}{
					"code":        haciendaCode,
					"description": rejectionReason,
				})
			}

			releaseErr := u.sequentialManager.ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode, claims.BranchID)
			if releaseErr != nil {
				logs.Error("Error releasing reservation after transmission failure", map[string]interface{}{
					"controlNumber": controlNumber,
					"error":         releaseErr.Error(),
				})
			} else {
				logs.Info("Reservation released successfully after transmission failure", map[string]interface{}{
					"controlNumber":   controlNumber,
					"rejectionReason": rejectionReason,
					"haciendaCode":    haciendaCode,
				})
			}
		}

		logs.Error("Error transmitting document", map[string]interface{}{"error": err.Error()})
		return mhModel, options, err
	}
	options.ReceptionStamp = transmitResult.ReceptionStamp

	confirmErr := u.sequentialManager.ConfirmReservation(ctx, controlNumber, generationCode, claims.BranchID)
	if confirmErr != nil {
		logs.Error("Error confirming reservation after successful transmission", map[string]interface{}{
			"controlNumber": controlNumber,
			"error":         confirmErr.Error(),
		})
		return mhModel, options, confirmErr
	}

	err = u.dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, transmitResult.ReceptionStamp)
	if err != nil {
		logs.Error("Error saving document in database", map[string]interface{}{"error": err.Error()})
		return mhModel, options, err
	}

	if u.additionalOps != nil {
		err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
		if err != nil {
			logs.Error("Error executing additional operations", map[string]interface{}{"error": err.Error()})
			return mhModel, options, err
		}
	}

	return mhModel, options, nil
}

// extractGenerationCode extracts the generation code using reflection
func extractGenerationCode(mhModel interface{}) (string, error) {
	extractor, err := utils.ExtractAuxiliarIdentification(mhModel)
	if err != nil {
		return "", err
	}

	code := extractor.Identification.GenerationCode
	if code == "" {
		return "", dte_errors.NewValidationError("RequiredField", "GenerationCode")
	}

	return code, nil
}

// extractControlNumber extracts the control number using reflection
func extractControlNumber(mhModel interface{}) (string, error) {
	extractor, err := utils.ExtractAuxiliarIdentification(mhModel)
	if err != nil {
		return "", err
	}

	number := extractor.Identification.ControlNumber
	if number == "" {
		return "", dte_errors.NewValidationError("RequiredField", "ControlNumber")
	}

	return number, nil
}

func (u *GenericDTEUseCase) shouldHandleAsContingency(err error) bool {
	var validationErr *dte_errors.ValidationError
	if errors.As(err, &validationErr) {
		return false
	}

	var haciendaErr *hacienda_error.HaciendaResponseError
	if errors.As(err, &haciendaErr) {
		if haciendaErr.Status == "RECHAZADO" {
			return false
		}
	}

	var generalErr *shared_error.ServiceError
	if errors.As(err, &generalErr) {
		return false
	}

	var businessErr *dte_errors.DTEError
	if errors.As(err, &businessErr) {
		return false
	}

	return true
}
