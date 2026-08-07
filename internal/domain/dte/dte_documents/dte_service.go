package dte_documents

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type DTEService struct {
	repo DTERepositoryPort
}

func NewDTEService(repo DTERepositoryPort) DTEManager {
	return &DTEService{
		repo: repo,
	}
}

func (m *DTEService) Create(ctx context.Context, document interface{}, transmission, status string, receptionStamp *string) error {
	if transmission != constants.TransmissionContingency {
		if receptionStamp == nil || *receptionStamp == "" {
			return shared_error.NewFormattedGeneralServiceError("DTEService", "CreateDTE", "MissingReceptionStamp")
		}
		if err := m.setReceptionStampIntoAppendix(document, receptionStamp); err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("DTEService", "CreateDTE", err, "FailedToSetReceptionStamp")
		}
	}

	if err := m.repo.Create(ctx, document, transmission, status, receptionStamp); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "CreateDTE", err, "FailedToCreateDTE")
	}

	return nil
}

func (m *DTEService) GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, document interface{}) error {
	extractor, err := utils.ExtractSummaryTotalAmounts(document)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceError("DTEService", "GenerateBalanceTransaction", "FailedToExtractSummaryTotals")
	}

	transaction := dte.BalanceTransaction{
		AdjustmentDocumentID: adjustmentDTE,
		TransactionType:      transactionType,
		TaxedAmount:          extractor.Summary.TotalTaxed,
		ExemptAmount:         extractor.Summary.TotalExempt,
		NotSubjectAmount:     extractor.Summary.TotalNotSubject,
	}

	err = m.repo.GenerateBalanceTransaction(ctx, branchID, originalDTE, &transaction)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "GenerateBalanceTransaction", err, "FailedToGenerateBalanceTransaction")
	}

	return nil
}

func (m *DTEService) GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error {
	transaction := dte.BalanceTransaction{
		AdjustmentDocumentID: adjustmentDTE,
		TransactionType:      transactionType,
		TaxedAmount:          taxedSale,
		ExemptAmount:         exemptSale,
		NotSubjectAmount:     notSubjectSale,
	}

	err := m.repo.GenerateBalanceTransaction(ctx, branchID, originalDTE, &transaction)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "GenerateBalanceTransactionWithAmounts", err, "FailedToGenerateBalanceTransaction")
	}

	return nil
}

func (m *DTEService) ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error {

	doc := document.(*credit_note_models.CreditNoteModel)
	if doc == nil {
		return shared_error.NewGeneralServiceError("DTEService", "ValidateForCreditNote", "failed to cast document to CreditNoteModel", nil)
	}

	balanceControl, err := m.repo.GetDTEBalanceControl(ctx, branchID, originalDTE)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "IsValidForCreditNote", err, "FailedToGetBalanceControl")
	}

	if (balanceControl.RemainingTaxedAmount - doc.Summary.GetTotalTaxed()) < 0 {
		return dte_errors.NewValidationError("InvalidCreditNoteTransaction", "Taxed", originalDTE, doc.Summary.GetTotalTaxed(), balanceControl.RemainingTaxedAmount)
	}
	if (balanceControl.RemainingExemptAmount - doc.Summary.GetTotalExempt()) < 0 {
		return dte_errors.NewValidationError("InvalidCreditNoteTransaction", "Exempt", originalDTE, doc.Summary.GetTotalExempt(), balanceControl.RemainingExemptAmount)
	}
	if (balanceControl.RemainingNotSubjectAmount - doc.Summary.GetTotalNonSubject()) < 0 {
		return dte_errors.NewValidationError("InvalidCreditNoteTransaction", "Not Subject", originalDTE, doc.Summary.GetTotalNonSubject(), balanceControl.RemainingNotSubjectAmount)
	}

	return nil
}

func (m *DTEService) ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error {
	doc := document.(*debit_note_models.DebitNoteModel)
	if doc == nil {
		return shared_error.NewGeneralServiceError("DTEService", "ValidateForDebitNote", "failed to cast document to DebitNoteModel", nil)
	}

	_, err := m.repo.GetDTEBalanceControl(ctx, branchID, originalDTE)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "ValidateForDebitNote", err, "FailedToGetBalanceControl")
	}

	return nil
}

func (m *DTEService) UpdateDTE(ctx context.Context, branchID uint, document dte.DTEDetails) error {
	if err := m.repo.Update(ctx, branchID, document); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "UpdateDTE", err, "FailedToUpdateDTE")
	}

	return nil
}

func (m *DTEService) VerifyStatus(ctx context.Context, branchID uint, id string) (string, error) {
	status, err := m.repo.VerifyStatus(ctx, branchID, id)
	if err != nil {
		return "", shared_error.NewFormattedGeneralServiceWithError("DTEService", "VerifyStatus", err, "FailedToVerifyDTE")
	}

	return status, nil
}

func (m *DTEService) GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error) {
	dteDocument, err := m.repo.GetByGenerationCode(ctx, branchID, generationCode)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DTEService", "GetByGenerationCode", err, "FailedToGetDTE", generationCode)
	}

	return dteDocument, nil
}

func (m *DTEService) GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*dte.DTEResponse, error) {
	dteDocument, err := m.repo.GetByGenerationCode(ctx, branchID, generationCode)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DTEService", "GetByGenerationCode", err, "FailedToGetDTE", generationCode)
	}

	var jsonData map[string]interface{}
	if err = json.Unmarshal([]byte(dteDocument.Details.JSONData), &jsonData); err != nil {
		return nil, err
	}

	return &dte.DTEResponse{
		GenerationCode: dteDocument.Details.ID,
		ControlNumber:  dteDocument.Details.ControlNumber,
		Status:         dteDocument.Details.Status,
		Transmission:   dteDocument.Details.Transmission,
		ReceptionStamp: dteDocument.Details.ReceptionStamp,
		JSONData:       jsonData,
		CreatedAt:      dteDocument.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      dteDocument.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (m *DTEService) GetAllDTEs(ctx context.Context, filters *dte.DTEFilters) (*dte.DTEListResponse, error) {
	summaryStats, err := m.repo.GetSummaryStats(ctx, filters)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceError("DTEService", "GetAll", "FailedToGetSummaryStats")
	}

	response := &dte.DTEListResponse{
		Summary:   *summaryStats,
		Documents: []dte.DTEModelResponse{},
		Pagination: dte.DTEPaginationResponse{
			Page:       filters.Page,
			PageSize:   filters.PageSize,
			TotalPages: calculateTotalPages(summaryStats.Total, filters.PageSize),
		},
	}

	if summaryStats.Total == 0 {
		return response, nil
	}

	documents, err := m.repo.GetPagedDocuments(ctx, filters)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceError("DTEService", "GetAll", "FailedToGetPagedDoc")
	}
	response.Documents = documents

	return response, nil
}

func (m *DTEService) setReceptionStampIntoAppendix(document interface{}, receptionStamp *string) error {
	dteType, err := m.determineDTEType(document)
	if err != nil || dteType == "" {
		return shared_error.NewGeneralServiceError("DTEService", "setReceptionStampIntoAppendix", "failed to determine DTE type", nil)
	}

	appendix := &structs.DTEApendice{
		Campo:    "Datos del documento",
		Etiqueta: "Sello de recepción",
		Valor:    *receptionStamp,
	}

	switch dteType {
	case constants.FacturaElectronica:
		document.(*structs.InvoiceDTEResponse).Apendice =
			append(document.(*structs.InvoiceDTEResponse).Apendice, *appendix)
	case constants.CCFElectronico:
		document.(*structs.CCFDTEResponse).Apendice =
			append(document.(*structs.CCFDTEResponse).Apendice, *appendix)
	case constants.NotaCreditoElectronica:
		document.(*structs.CreditNoteDTEResponse).Apendice =
			append(document.(*structs.CreditNoteDTEResponse).Apendice, *appendix)
	case constants.NotaDebitoElectronica:
		document.(*structs.DebitNoteDTEResponse).Apendice =
			append(document.(*structs.DebitNoteDTEResponse).Apendice, *appendix)
	case constants.ComprobanteRetencionElectronico:
		document.(*structs.RetentionDTEResponse).Apendice =
			append(document.(*structs.RetentionDTEResponse).Apendice, *appendix)
	case constants.NotaRemisionElectronica:
		doc := document.(*structs.MHRemissionNote)
		doc.Appendix = append(doc.Appendix, *appendix)
	case constants.FacturaSujetoExcluidoElectronica:
		doc := document.(*structs.FSEDTEResponse)
		if doc.Apendice == nil {
			items := []structs.DTEApendice{*appendix}
			doc.Apendice = &items
		} else {
			*doc.Apendice = append(*doc.Apendice, *appendix)
		}
	default:
		return shared_error.NewFormattedGeneralServiceError("DTEService", "setReceptionStampIntoAppendix", "UnsupportedDTETypeForAppendix", dteType)
	}

	return nil
}

func (m *DTEService) determineDTEType(document interface{}) (string, error) {
	dteExtracted, err := utils.ExtractAuxiliarIdentification(document)

	if err != nil {
		return "", shared_error.NewGeneralServiceError("DTEManager", "determineDTEType", "failed to extract DTE identification", err)
	}

	return dteExtracted.Identification.DTEType, nil
}

func calculateTotalPages(totalItems int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}

	pages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		pages++
	}

	return pages
}
