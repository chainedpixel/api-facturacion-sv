package dte_documents

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/shopspring/decimal"
)

type DTEService struct {
	repo      DTERepositoryPort
	eventRepo DTEEventRepositoryPort
}

func NewDTEService(repo DTERepositoryPort, eventRepo DTEEventRepositoryPort) DTEManager {
	return &DTEService{
		repo:      repo,
		eventRepo: eventRepo,
	}
}

// RegisterEvent guarda un evento transmitido a Hacienda con su evidencia (Normativa v2.0, regla 11)
func (m *DTEService) RegisterEvent(ctx context.Context, event *dte.DTEEvent) error {
	if err := m.eventRepo.Create(ctx, event); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "RegisterEvent", err, "FailedToRegisterEvent")
	}
	return nil
}

// GetActiveAdjustmentNotes returns the generation codes of the received or pending credit and debit notes
// that adjust the given document.
func (m *DTEService) GetActiveAdjustmentNotes(ctx context.Context, branchID uint, originalDTE string) ([]string, error) {
	notes, err := m.repo.GetActiveAdjustmentNotes(ctx, branchID, originalDTE)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DTEService", "GetActiveAdjustmentNotes", err, "FailedToGetDTE", originalDTE)
	}
	return notes, nil
}

// GetArchive devuelve el Archivo DTE de un documento: estructura firmada + firmaElectronica + selloRecibido.
// Los documentos emitidos antes de guardar el JWS no tienen Archivo DTE disponible.
func (m *DTEService) GetArchive(ctx context.Context, branchID uint, generationCode string) (json.RawMessage, error) {
	doc, err := m.repo.GetByGenerationCode(ctx, branchID, generationCode)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DTEService", "GetArchive", err, "FailedToGetDTE", generationCode)
	}
	if doc.Details.SignedDocument == "" {
		return nil, dte_errors.NewValidationError("ArchiveNotAvailable", generationCode)
	}
	return BuildArchive(doc.Details.SignedDocument, doc.Details.ReceptionStamp)
}

// GetEvents devuelve los eventos de un documento (por ejemplo su invalidacion) con la respuesta de Hacienda
// y el Archivo DTE de cada evento, listo para entregar al receptor
func (m *DTEService) GetEvents(ctx context.Context, branchID uint, generationCode string) ([]dte.EventArchiveResponse, error) {
	// Se valida que el documento exista en la sucursal: un codigo inexistente responde no encontrado, no lista vacia
	if _, err := m.GetByGenerationCode(ctx, branchID, generationCode); err != nil {
		return nil, err
	}

	events, err := m.eventRepo.ListByRelatedDTE(ctx, branchID, generationCode)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DTEService", "GetEvents", err, "FailedToGetEvents", generationCode)
	}

	result := make([]dte.EventArchiveResponse, 0, len(events))
	for _, event := range events {
		result = append(result, ToEventArchiveResponse(event))
	}
	return result, nil
}

// ToEventArchiveResponse arma la respuesta de un evento con su Archivo DTE (si se guardo el JWS)
func ToEventArchiveResponse(event dte.DTEEvent) dte.EventArchiveResponse {
	response := dte.EventArchiveResponse{DTEEvent: event}
	if event.HaciendaResponse != "" {
		response.HaciendaResponse = json.RawMessage(event.HaciendaResponse)
	}
	if event.SignedDocument != "" {
		if archive, err := BuildArchive(event.SignedDocument, event.ReceptionStamp); err == nil {
			response.Archive = archive
		}
	}
	return response
}

// Create guarda el DTE junto con la evidencia de su transmision. Normativa v2.0 (Anexo I, regla 4.1): la
// estructura firmada no se altera (el sello ya no se inserta en el apendice); el sello y la firma se
// guardan aparte y se agregan como selloRecibido y firmaElectronica al armar el Archivo DTE.
func (m *DTEService) Create(ctx context.Context, document interface{}, transmission, status string, evidence *dte.TransmissionEvidence) error {
	if transmission != constants.TransmissionContingency {
		if evidence == nil || evidence.ReceptionStamp == nil || *evidence.ReceptionStamp == "" {
			return shared_error.NewFormattedGeneralServiceError("DTEService", "CreateDTE", "MissingReceptionStamp")
		}
	}

	if err := m.repo.Create(ctx, document, transmission, status, evidence); err != nil {
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

	balanceControl, err := m.repo.GetDTEBalanceControl(ctx, branchID, originalDTE)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DTEService", "ValidateForDebitNote", err, "FailedToGetBalanceControl")
	}

	available := decimal.NewFromFloat(balanceControl.OriginalNotSubjectAmount).
		Sub(decimal.NewFromFloat(balanceControl.DebitNotesNotSubjectAmount()))
	requested := decimal.NewFromFloat(doc.Summary.GetTotalNonSubject()).Round(2)
	if requested.GreaterThan(available) {
		return dte_errors.NewValidationError("InvalidDebitNoteNotSubjectAmount", requested.InexactFloat64(), originalDTE, available.InexactFloat64())
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
		ProcessingDate: dteDocument.Details.ProcessingDate,
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
