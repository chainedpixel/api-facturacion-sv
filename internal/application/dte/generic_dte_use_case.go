package dte

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	appPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	transmissionPorts "github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

const persistRetryDelay = 200 * time.Millisecond

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
	bus               event.Bus
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
	bus event.Bus,
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
		bus:               bus,
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

	confirmErr := retryOnce(ctx, "confirming reservation", func() error {
		return u.sequentialManager.ConfirmReservation(ctx, controlNumber, generationCode, claims.BranchID)
	})
	if confirmErr != nil {
		logs.Error("Reservation could not be confirmed after retry, continuing to persist the stamped document; manual reconciliation required", map[string]interface{}{
			"controlNumber":  controlNumber,
			"generationCode": generationCode,
			"error":          confirmErr.Error(),
		})
	}

	evidence := &dte.TransmissionEvidence{
		ReceptionStamp:   transmitResult.ReceptionStamp,
		SignedDocument:   transmitResult.SignedDocument,
		ProcessingDate:   transmitResult.ProcessingDate,
		HaciendaResponse: transmitResult.HaciendaResponse,
	}
	err = u.persistWithRetry(ctx, mhModel, evidence)
	if err != nil {
		return mhModel, options, u.reportStampedNotPersisted(ctx, claims, mhModel, evidence, generationCode, controlNumber, err)
	}

	if u.additionalOps != nil {
		err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
		if err != nil {
			logs.Error("Error executing additional operations", map[string]interface{}{"error": err.Error()})
			if u.shouldHandleAsContingency(err) {
				err = shared_error.NewFormattedGeneralServiceWithError("GenericDTEUseCase", "AdditionalOperations", err, "AdditionalOperationsFailed", generationCode)
			}
			return mhModel, options, err
		}
	}

	// Normativa v2.0 (Anexo I, regla 4.1): se responde el Archivo DTE listo para entregar al receptor
	// (estructura firmada + firmaElectronica + selloRecibido). Si no se pudiera armar, se responde la estructura.
	archive, archiveErr := dte_documents.BuildArchive(transmitResult.SignedDocument, transmitResult.ReceptionStamp)
	if archiveErr != nil {
		logs.Warn("Could not build DTE archive for response", map[string]interface{}{"error": archiveErr.Error()})
		return mhModel, options, nil
	}

	return archive, options, nil
}

// persistWithRetry saves the stamped document and retries once after a short delay when the first attempt fails.
func (u *GenericDTEUseCase) persistWithRetry(ctx context.Context, mhModel interface{}, evidence *dte.TransmissionEvidence) error {
	return retryOnce(ctx, "saving stamped document", func() error {
		return u.dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, evidence)
	})
}

// retryOnce runs the operation and, when it fails, runs it a second time after a fixed delay unless the context ends first.
func retryOnce(ctx context.Context, description string, operation func() error) error {
	err := operation()
	if err == nil {
		return nil
	}

	logs.Warn("Error "+description+", retrying once", map[string]interface{}{"error": err.Error()})

	select {
	case <-ctx.Done():
		return err
	case <-time.After(persistRetryDelay):
	}

	return operation()
}

// reportStampedNotPersisted records everything needed to recover a stamped document that could not be saved,
// notifies the administrator and builds the error returned to the client.
func (u *GenericDTEUseCase) reportStampedNotPersisted(
	ctx context.Context,
	claims *models.AuthClaims,
	mhModel interface{},
	evidence *dte.TransmissionEvidence,
	generationCode, controlNumber string,
	cause error,
) error {
	receptionStamp := ""
	if evidence.ReceptionStamp != nil {
		receptionStamp = *evidence.ReceptionStamp
	}
	dteType := extractDTEType(mhModel)

	logs.Error("CRITICAL: document stamped by Hacienda could not be saved, manual recovery required", map[string]interface{}{
		"generationCode": generationCode,
		"controlNumber":  controlNumber,
		"dteType":        dteType,
		"branchID":       claims.BranchID,
		"receptionStamp": receptionStamp,
		"processingDate": evidence.ProcessingDate,
		"signedDocument": evidence.SignedDocument,
		"error":          cause.Error(),
	})

	if u.bus != nil {
		u.bus.Publish(ctx, event.StampedDocumentNotPersistedEvent{
			BranchID:       claims.BranchID,
			NIT:            claims.NIT,
			ControlNumber:  controlNumber,
			GenerationCode: generationCode,
			DTEType:        dteType,
			ReceptionStamp: receptionStamp,
			ProcessingDate: evidence.ProcessingDate,
			LastError:      cause.Error(),
			OccurredAtTime: utils.TimeNow(),
		})
	}

	return shared_error.NewFormattedGeneralServiceWithError("GenericDTEUseCase", "PersistStampedDocument", cause, "StampedDocumentNotPersisted", generationCode, receptionStamp)
}

// extractDTEType extracts the DTE type code, returning an empty string when it is unavailable
func extractDTEType(mhModel interface{}) string {
	extractor, err := utils.ExtractAuxiliarIdentification(mhModel)
	if err != nil {
		return ""
	}

	return extractor.Identification.DTEType
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
