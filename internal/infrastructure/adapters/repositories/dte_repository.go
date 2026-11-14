package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

type DTERepository struct {
	db *gorm.DB
}

func NewDTERepository(db *gorm.DB) dte_documents.DTERepositoryPort {
	return &DTERepository{
		db: db,
	}
}

func (D *DTERepository) Create(ctx context.Context, document interface{}, transmission, status string, evidence *dte.TransmissionEvidence) error {
	claims := ctx.Value("claims").(*models.AuthClaims)
	var dteResponse utils.AuxiliarIdentificationExtractor

	jsonData, err := json.Marshal(document)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(jsonData, &dteResponse); err != nil {
		return err
	}

	if dteResponse.Identification.GenerationCode == "" ||
		dteResponse.Identification.ControlNumber == "" ||
		dteResponse.Identification.DTEType == "" {
		return shared_error.NewFormattedGeneralServiceError("DTERepository", "Create", "MissingDTEIdentification")
	}

	dteDocument := &db_models.DTEDocument{
		BranchID:  claims.BranchID,
		CreatedAt: utils.TimeNow(),
		UpdatedAt: utils.TimeNow(),
		Document: &db_models.DTEDetails{
			ID:            dteResponse.Identification.GenerationCode,
			Transmission:  transmission,
			Status:        status,
			DTEType:       dteResponse.Identification.DTEType,
			ControlNumber: dteResponse.Identification.ControlNumber,
			JSONData:      string(jsonData),
		},
	}
	applyEvidence(dteDocument.Document, evidence)

	result := D.db.WithContext(ctx).Create(dteDocument)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// applyEvidence copia la evidencia de transmision (sello, JWS, fecha y respuesta de Hacienda) al modelo
func applyEvidence(details *db_models.DTEDetails, evidence *dte.TransmissionEvidence) {
	if evidence == nil {
		return
	}
	details.ReceptionStamp = evidence.ReceptionStamp
	details.SignedDocument = optionalString(evidence.SignedDocument)
	details.ProcessingDate = optionalString(evidence.ProcessingDate)
	details.HaciendaResponse = optionalString(evidence.HaciendaResponse)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (D *DTERepository) GetDTEBalanceControl(ctx context.Context, branchID uint, id string) (*dte.BalanceControl, error) {
	var balanceControl db_models.DTEBalanceControl

	result := D.db.WithContext(ctx).
		Preload("Transactions").
		Where("branch_id = ? AND original_dte_id = ?", branchID, id).
		First(&balanceControl)
	if result.Error != nil {
		return nil, handleGormErr(result.Error, "GetDTEBalanceControl")
	}

	transactions := make([]dte.BalanceTransaction, 0, len(balanceControl.Transactions))
	for _, transaction := range balanceControl.Transactions {
		transactions = append(transactions, dte.BalanceTransaction{
			BalanceControlID:     transaction.BalanceControlID,
			AdjustmentDocumentID: transaction.AdjustmentDocumentID,
			TransactionType:      transaction.TransactionType,
			TaxedAmount:          transaction.TaxedAmount,
			ExemptAmount:         transaction.ExemptAmount,
			NotSubjectAmount:     transaction.NotSubjectAmount,
			CreatedAt:            transaction.CreatedAt,
		})
	}

	return &dte.BalanceControl{
		ID:                        balanceControl.ID,
		BranchID:                  balanceControl.BranchID,
		OriginalDTEID:             balanceControl.OriginalDTEID,
		OriginalTaxedAmount:       balanceControl.OriginalTaxedAmount,
		OriginalExemptAmount:      balanceControl.OriginalExemptAmount,
		OriginalNotSubjectAmount:  balanceControl.OriginalTotalNotSubjectAmount,
		RemainingTaxedAmount:      balanceControl.RemainingTaxedAmount,
		RemainingExemptAmount:     balanceControl.RemainingExemptAmount,
		RemainingNotSubjectAmount: balanceControl.RemainingNotSubjectAmount,
		Transactions:              transactions,
	}, nil
}

func (D *DTERepository) GenerateBalanceTransaction(ctx context.Context, branchID uint, originalDTE string, transaction *dte.BalanceTransaction) error {
	return D.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var balanceControl db_models.DTEBalanceControl

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("branch_id = ? AND original_dte_id = ?", branchID, originalDTE).
			First(&balanceControl).Error; err != nil {
			return handleGormErr(err, "GenerateBalanceTransaction")
		}

		dteTransaction := &db_models.DTEBalanceTransaction{
			BalanceControlID:     balanceControl.ID,
			BalanceControl:       &balanceControl,
			AdjustmentDocumentID: transaction.AdjustmentDocumentID,
			TransactionType:      transaction.TransactionType,
			TaxedAmount:          transaction.TaxedAmount,
			ExemptAmount:         transaction.ExemptAmount,
			NotSubjectAmount:     transaction.NotSubjectAmount,
		}

		if err := tx.Create(dteTransaction).Error; err != nil {
			return err
		}

		return nil
	})
}

func (D *DTERepository) Update(ctx context.Context, branchID uint, document dte.DTEDetails) error {
	dbModel := &db_models.DTEDetails{
		ID:             document.ID,
		DTEType:        document.DTEType,
		ControlNumber:  document.ControlNumber,
		Transmission:   document.Transmission,
		Status:         document.Status,
		ReceptionStamp: document.ReceptionStamp,
		JSONData:       document.JSONData,
	}
	dbModel.SignedDocument = optionalString(document.SignedDocument)
	dbModel.ProcessingDate = optionalString(document.ProcessingDate)
	dbModel.HaciendaResponse = optionalString(document.HaciendaResponse)

	if err := D.db.WithContext(ctx).
		Model(&db_models.DTEDocument{}).
		Where("document_id = ? AND branch_id = ?", document.ID, branchID).
		Updates(map[string]interface{}{
			"updated_at": utils.TimeNow(),
		}).Error; err != nil {
		return err
	}

	if err := D.db.WithContext(ctx).
		Model(&db_models.DTEDetails{}).
		Where("id = ?", document.ID).
		Updates(dbModel).Error; err != nil {
		return err
	}

	return nil
}

func (D *DTERepository) VerifyStatus(ctx context.Context, branchID uint, id string) (string, error) {
	var status string

	result := D.db.WithContext(ctx).
		Model(&db_models.DTEDocument{}).
		Joins("JOIN dte_details ON dte_documents.document_id = dte_details.id").
		Select("dte_details.status").
		Where("document_id = ? AND branch_id = ?", id, branchID).
		First(&status)
	if result.Error != nil {
		return "", handleGormErr(result.Error, "VerifyStatus")
	}

	return status, nil
}

// GetTotalCount retrieves the total count of documents matching the given filters
func (D *DTERepository) GetTotalCount(ctx context.Context, filters *dte.DTEFilters) (int64, error) {
	var totalCount int64

	query := D.db.WithContext(ctx).
		Table("dte_documents").
		Joins("JOIN dte_details ON dte_documents.document_id = dte_details.id")

	loadFilters(query, filters)

	if err := query.Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}

// GetSummaryStats retrieves the overall statistics for all documents based on the given filters
func (D *DTERepository) GetSummaryStats(ctx context.Context, filters *dte.DTEFilters) (*dte.ListSummary, error) {
	summary := &dte.ListSummary{}

	totalCount, err := D.GetTotalCount(ctx, filters)
	if err != nil {
		return nil, err
	}
	summary.Total = totalCount

	if totalCount == 0 {
		return summary, nil
	}

	query := D.db.WithContext(ctx).
		Table("dte_documents").
		Joins("JOIN dte_details ON dte_documents.document_id = dte_details.id")

	loadFilters(query, filters)

	type StatusTypeCount struct {
		Status       string `gorm:"column:status"`
		Transmission string `gorm:"column:transmission"`
		Count        int64  `gorm:"column:count"`
	}
	var statusCounts []StatusTypeCount

	if err := query.Select("dte_details.status, dte_details.transmission, COUNT(*) as count").
		Group("dte_details.status, dte_details.transmission").
		Find(&statusCounts).Error; err != nil {
		return nil, err
	}

	for _, sc := range statusCounts {
		switch sc.Status {
		case constants.DocumentReceived:
			summary.Received += sc.Count
		case constants.DocumentInvalid:
			summary.Invalid += sc.Count
		case constants.DocumentRejected:
			summary.Rejected += sc.Count
		case constants.DocumentPending:
			summary.Pending += sc.Count
		}

		switch sc.Transmission {
		case constants.TransmissionNormal:
			summary.ByNormal += sc.Count
		case constants.TransmissionContingency:
			summary.ByContingency += sc.Count
		}
	}

	return summary, nil
}

// GetPagedDocuments retrieves documents in a paginated manner
func (D *DTERepository) GetPagedDocuments(ctx context.Context, filters *dte.DTEFilters) ([]dte.DTEModelResponse, error) {
	query := D.db.WithContext(ctx).
		Table("dte_documents").
		Joins("JOIN dte_details ON dte_documents.document_id = dte_details.id")

	loadFilters(query, filters)

	type DocumentResult struct {
		ID           string    `gorm:"column:id"`
		JSONData     string    `gorm:"column:json_data"`
		Status       string    `gorm:"column:status"`
		Transmission string    `gorm:"column:transmission"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}

	var documents []DocumentResult

	query = query.Order("dte_documents.created_at DESC")

	if filters.Page > 0 && filters.PageSize > 0 {
		offset := (filters.Page - 1) * filters.PageSize
		query = query.Offset(offset).Limit(filters.PageSize)
	}

	if err := query.Select("dte_details.id, dte_details.json_data, dte_details.status, dte_details.transmission, dte_documents.created_at").
		Find(&documents).Error; err != nil {
		return nil, err
	}

	result := make([]dte.DTEModelResponse, 0, len(documents))
	for _, doc := range documents {
		result = append(result, dte.DTEModelResponse{
			Status:           doc.Status,
			TransmissionType: doc.Transmission,
			Document:         json.RawMessage(doc.JSONData),
		})
	}

	return result, nil
}

func (D *DTERepository) GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error) {
	var document db_models.DTEDocument

	result := D.db.WithContext(ctx).
		Preload("Document").
		Where("branch_id = ? AND document_id = ?", branchID, generationCode).
		First(&document)
	if result.Error != nil {
		return nil, handleGormErr(result.Error, "GetByGenerationCode")
	}

	return &dte.DTEDocument{
		DocumentID: document.Document.ID,
		BranchID:   document.BranchID,
		CreatedAt:  document.CreatedAt,
		UpdatedAt:  document.UpdatedAt,
		Details: &dte.DTEDetails{
			ID:               document.Document.ID,
			DTEType:          document.Document.DTEType,
			ControlNumber:    document.Document.ControlNumber,
			Transmission:     document.Document.Transmission,
			Status:           document.Document.Status,
			ReceptionStamp:   document.Document.ReceptionStamp,
			JSONData:         document.Document.JSONData,
			SignedDocument:   valueOrEmpty(document.Document.SignedDocument),
			ProcessingDate:   valueOrEmpty(document.Document.ProcessingDate),
			HaciendaResponse: valueOrEmpty(document.Document.HaciendaResponse),
		},
	}, nil
}

// GetActiveAdjustmentNotes returns the received or pending credit and debit notes registered in the balance
// control of the given document.
func (D *DTERepository) GetActiveAdjustmentNotes(ctx context.Context, branchID uint, originalDTE string) ([]string, error) {
	var notes []string

	result := D.db.WithContext(ctx).
		Table("dte_balance_transactions AS t").
		Joins("JOIN dte_balance_control AS c ON c.id = t.balance_control_id").
		Joins("JOIN dte_details AS d ON d.id = t.adjustment_document_id").
		Where("c.branch_id = ? AND c.original_dte_id = ?", branchID, originalDTE).
		Where("d.dte_type IN ?", []string{constants.NotaCreditoElectronica, constants.NotaDebitoElectronica}).
		Where("d.status IN ?", []string{constants.DocumentReceived, constants.DocumentPending}).
		Distinct().
		Order("t.adjustment_document_id").
		Pluck("t.adjustment_document_id", &notes)
	if result.Error != nil {
		return nil, handleGormErr(result.Error, "GetActiveAdjustmentNotes")
	}

	return notes, nil
}

func loadFilters(query *gorm.DB, filters *dte.DTEFilters) {
	if filters.IssuerNIT != "" {
		query = query.Where(
			"dte_documents.branch_id IN (?)",
			query.Session(&gorm.Session{NewDB: true}).
				Table("branch_offices").
				Select("branch_offices.id").
				Joins("JOIN users ON users.id = branch_offices.user_id").
				Where("users.nit = ?", filters.IssuerNIT),
		)
	} else if filters.BranchID != 0 {
		query = query.Where("dte_documents.branch_id = ?", filters.BranchID)
	}

	if len(filters.DTETypes) > 0 {
		query = query.Where("dte_details.dte_type IN ?", filters.DTETypes)
	} else if filters.DTEType != "" {
		query = query.Where("dte_details.dte_type = ?", filters.DTEType)
	}

	if filters.Status != "" {
		query = query.Where("dte_details.status = ?", filters.Status)
	}

	if filters.StartDate != nil {
		query = query.Where("dte_documents.created_at >= ?", filters.StartDate)
	}

	if filters.EndDate != nil {
		query = query.Where("dte_documents.created_at <= ?", filters.EndDate)
	}

	if filters.Transmission != "" {
		query = query.Where("dte_details.transmission = ?", filters.Transmission)
	}
}

func handleGormErr(err error, operation string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shared_error.NewFormattedGeneralServiceError("DTERepo", operation, "NotFound")
	}

	return err
}
