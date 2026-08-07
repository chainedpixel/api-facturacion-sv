package request_mapper

import (
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	identificationVO "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/invalidation"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/google/uuid"
)

type InvalidationMapper struct{}

func NewInvalidationMapper() *InvalidationMapper {
	return &InvalidationMapper{}
}

func (i *InvalidationMapper) MapToInvalidationData(req *structs.CreateInvalidationRequest, client *dte.IssuerDTE, baseDte *dte.DTEDetails, emissionDate time.Time) (*invalidation_models.InvalidationDocument, error) {
	if req == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Request")
	}

	reason, err := invalidation.MapInvalidationReasonRequest(req.Reason)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvalidationMapper", "MapToInvalidationData", err, "ErrorMapping", "Invalidation->Reason")
	}

	document, err := invalidation.MapInvalidatedDocument(baseDte, req, emissionDate)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvalidationMapper", "MapToInvalidationData", err, "ErrorMapping", "Invalidation->Document")
	}

	var documentType string
	if document.DocumentType != nil {
		documentType = document.DocumentType.GetValue()
	} else {
		documentType = baseDte.DTEType
	}

	identification, err := common.MapCommonRequestIdentification(1, 2, documentType)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvalidationMapper", "MapToInvalidationData", err, "ErrorMapping", "Invalidation->Identification")
	}

	newUUID := uuid.New().String()
	identification.GenerationCode = *identificationVO.NewValidatedGenerationCode(strings.ToUpper(newUUID))

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvalidationMapper", "MapToInvalidationData", err, "ErrorMapping", "Invalidation->Issuer")
	}

	return &invalidation_models.InvalidationDocument{
		Identification: identification,
		Reason:         reason,
		Issuer:         issuer,
		Document:       document,
	}, nil
}

func (i *InvalidationMapper) ValidateInvalidationReRequest(req *structs.CreateInvalidationRequest) error {
	if req == nil {
		return dte_errors.NewValidationError("RequiredField", "Request")
	}

	if req.Reason == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason")
	}

	if (req.Reason.Type == 1 || req.Reason.Type == 3) && req.ReplacementGenerationCode == nil {
		return shared_error.NewFormattedGeneralServiceError("InvalidationMapper", "validateInvoiceRequest", "InvalidReplacementCode1And3")
	}

	if req.Reason.Type == 2 && req.ReplacementGenerationCode != nil {
		return shared_error.NewFormattedGeneralServiceError("InvalidationMapper", "validateInvoiceRequest", "InvalidInvalidationType2")
	}

	if req.GenerationCode == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->GenerationCode")
	}

	if req.Reason.Type == 0 {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->Type")
	}

	if req.Reason.ResponsibleName == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->ResponsibleName")
	}

	if req.Reason.ResponsibleDocType == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->ResponsibleDocType")
	}

	if req.Reason.ResponsibleNumDoc == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->ResponsibleNumDoc")
	}

	if req.Reason.RequestorName == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->RequestorName")
	}

	if req.Reason.RequestorDocType == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->RequestorDocType")
	}

	if req.Reason.RequestorNumDoc == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Reason->RequestorNumDoc")
	}

	if req.Reason.Reason == nil && req.Reason.Type == 3 {
		return shared_error.NewFormattedGeneralServiceError("InvalidationMapper", "validateInvoiceRequest", "InvalidInvalidationType3")
	}

	return nil
}
