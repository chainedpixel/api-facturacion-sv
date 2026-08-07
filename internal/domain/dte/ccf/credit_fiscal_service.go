package ccf

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	buisnessValidator "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type creditFiscalService struct {
	validator        *validator.CCFRulesValidator
	seqNumberManager dte_documents.SequentialNumberManager
}

// NewCCFService Creates a new Fiscal Credit Voucher service.
func NewCCFService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService {
	return &creditFiscalService{
		validator:        validator.NewCCFRulesValidator(nil),
		seqNumberManager: seqNumberManager,
	}
}

func (s *creditFiscalService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
	data := input.(*ccf_models.CCFData)
	baseDoc := createBaseDocument(data)

	creditFiscalDocument := &ccf_models.CreditFiscalDocument{
		DTEDocument:   baseDoc,
		CreditItems:   data.Items,
		CreditSummary: *data.CreditSummary,
	}

	if err := s.validate(creditFiscalDocument); err != nil {
		logs.Error("Failed to validate credit fiscal document basic validation", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := buisnessValidator.ValidateDTEDocument(creditFiscalDocument); err != nil {
		logs.Error("Failed to validate credit fiscal document generic validations", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := s.generateCodeAndIdentifiers(ctx, creditFiscalDocument, branchID); err != nil {
		return nil, err
	}

	return creditFiscalDocument, nil
}

func (s *creditFiscalService) validate(ccf *ccf_models.CreditFiscalDocument) error {
	s.validator = validator.NewCCFRulesValidator(ccf)
	err := s.validator.Validate()
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"CCFService",
			"Validate",
			err,
			"ValidationFailed",
		)
	}
	return nil
}

// generateControlNumber Generates a unique control number for the invoice.
func (s *creditFiscalService) generateControlNumber(ctx context.Context, ccf *ccf_models.CreditFiscalDocument, branchID uint) error {
	establishmentCode := ccf.Issuer.GetEstablishmentCode()
	posCode := ccf.Issuer.GetPOSCode()

	controlNumber, err := s.seqNumberManager.GetNextControlNumber(
		ctx,
		constants.CCFElectronico,
		branchID,
		posCode,
		establishmentCode,
	)
	if err != nil {
		return err
	}

	err = ccf.Identification.SetControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"CCFService",
			"GenerateControlNumber",
			err,
			"FailedToSetControlNumber",
		)
	}
	return nil
}

func (s *creditFiscalService) generateCodeAndIdentifiers(ctx context.Context, ccf *ccf_models.CreditFiscalDocument, branchID uint) error {
	if err := s.generateControlNumber(ctx, ccf, branchID); err != nil {
		return err
	}
	return ccf.Identification.GenerateCode()
}

func createBaseDocument(data *ccf_models.CCFData) *models.DTEDocument {
	var extInterface interfaces.Extension
	var thirdPartySale interfaces.ThirdPartySale
	var appendixes []interfaces.Appendix
	var relatedDocuments []interfaces.RelatedDocument
	var otherDocuments []interfaces.OtherDocuments
	baseItems := make([]interfaces.Item, len(data.Items))
	for i, item := range data.Items {
		baseItems[i] = &item
	}

	if data.Appendixes != nil {
		for _, appendix := range data.Appendixes {
			appendixes = append(appendixes, &appendix)
		}
	}

	if data.Extension != nil {
		extInterface = data.Extension
	}

	if data.RelatedDocs != nil {
		for _, relatedDoc := range data.RelatedDocs {
			relatedDocuments = append(relatedDocuments, &relatedDoc)
		}
	}

	if data.OtherDocs != nil {
		for _, otherDoc := range data.OtherDocs {
			otherDocuments = append(otherDocuments, &otherDoc)
		}
	}

	if data.ThirdPartySale != nil {
		thirdPartySale = data.ThirdPartySale
	}

	return &models.DTEDocument{
		Identification:   data.Identification,
		Issuer:           data.Issuer,
		Receiver:         data.Receiver,
		Items:            baseItems,
		RelatedDocuments: relatedDocuments,
		OtherDocuments:   otherDocuments,
		Summary:          data.CreditSummary.Summary,
		ThirdPartySale:   thirdPartySale,
		Extension:        extInterface,
		Appendix:         appendixes,
	}
}
