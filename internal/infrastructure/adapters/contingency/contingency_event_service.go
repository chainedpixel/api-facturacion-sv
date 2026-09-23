package contingency

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/config/drivers"
	haciendaPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
	authPorts "github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// ContingencyEventService orchestrates preparation and transmission of contingency events.
type ContingencyEventService struct {
	authManager  auth.AuthManager
	repo         contingency.ContingencyRepositoryPort
	timeProvider authPorts.TimeProvider
	tokenSvc     *contingencyTokenService
	httpSvc      *contingencyHTTPService
	connection   *drivers.DbConnection
}

// NewContingencyEventService creates a new ContingencyEventService with its dependencies.
func NewContingencyEventService(
	authManager auth.AuthManager,
	haciendaAuth haciendaPorts.HaciendaAuthManager,
	cache authPorts.CacheManager,
	tokenService authPorts.TokenManager,
	signer haciendaPorts.SignerManager,
	repo contingency.ContingencyRepositoryPort,
	timeProvider authPorts.TimeProvider,
	connection *drivers.DbConnection,
) *ContingencyEventService {
	return &ContingencyEventService{
		authManager:  authManager,
		repo:         repo,
		timeProvider: timeProvider,
		connection:   connection,
		tokenSvc:     newContingencyTokenService(cache, tokenService, authManager),
		httpSvc:      newContingencyHTTPService(haciendaAuth, signer, cache),
	}
}

// PrepareAndSendContingencyEvent builds a contingency event from the given documents and sends it to Hacienda.
func (s *ContingencyEventService) PrepareAndSendContingencyEvent(ctx context.Context, docs []dte.ContingencyDocument) error {
	if len(docs) == 0 {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "PrepareAndSendContingencyEvent", "no documents provided for contingency event", nil)
	}

	sqlDb, err := s.connection.Db.DB()
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "PrepareAndSendContingencyEvent", "failed to get sql db", err)
	}
	sqlDb.Ping()

	client, err := s.authManager.GetIssuer(ctx, docs[0].BranchID)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "PrepareAndSendContingencyEvent", "failed to get issuer info", err)
	}

	reason, err := s.prepareContingencyReason(ctx, docs[0])
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "PrepareAndSendContingencyEvent", "failed to prepare contingency reason", err)
	}

	event := &models.ContingencyEvent{
		Identification: models.ContingencyIdentification{
			Version:          4,
			Ambient:          config.Server.AmbientCode,
			GenerationCode:   strings.ToUpper(uuid.New().String()),
			TransmissionDate: utils.TimeNow().Format("2006-01-02"),
			TransmissionTime: utils.TimeNow().Format("15:04:05"),
		},
		Issuer: models.ContingencyIssuer{
			NIT:                  client.NIT,
			Name:                 client.BusinessName,
			ResponsibleName:      client.BusinessName,
			ResponsibleDocType:   constants.NIT,
			ResponsibleDocNumber: client.NIT,
			EstablishmentType:    client.EstablishmentType,
			Phone:                *client.Phone,
			Email:                *client.Email,
			EstablishmentCodeMH:  client.EstablishmentCodeMH,
			POSCode:              client.POSCode,
		},
		DTEDetails: s.prepareDTEDetails(docs),
		Reason:     reason,
	}

	return s.sendContingencyEvent(ctx, event, docs[0].BranchID, docs)
}

func (s *ContingencyEventService) prepareDTEDetails(docs []dte.ContingencyDocument) []models.DTEDetail {
	details := make([]models.DTEDetail, len(docs))
	for i, doc := range docs {
		details[i] = models.DTEDetail{
			ItemNumber:     i + 1,
			GenerationCode: doc.Document.ID,
			DocumentType:   doc.Document.DTEType,
		}
	}
	return details
}

func (s *ContingencyEventService) prepareContingencyReason(ctx context.Context, doc dte.ContingencyDocument) (models.ContingencyReason, error) {
	now := s.timeProvider.Now()

	startTime, err := s.repo.GetFirstContingencyTimestamp(ctx, doc.BranchID)
	if err != nil {
		return models.ContingencyReason{}, shared_error.NewGeneralServiceError("ContingencyEventService", "prepareContingencyReason", "failed to get first contingency timestamp", err)
	}

	if startTime == nil {
		return models.ContingencyReason{}, shared_error.NewGeneralServiceError("ContingencyEventService", "prepareContingencyReason", "no pending contingency documents found for branch", nil)
	}

	var contingencyReason *string
	if doc.Reason != "" {
		contingencyReason = &doc.Reason
	}

	return models.ContingencyReason{
		StartDate:         startTime.Format("2006-01-02"),
		EndDate:           now.Format("2006-01-02"),
		StartTime:         startTime.Add(-60 * time.Second).Format("15:04:05"),
		EndTime:           now.Add(10 * time.Second).Format("15:04:05"),
		ContingencyType:   doc.ContingencyType,
		ContingencyReason: contingencyReason,
	}, nil
}

func (s *ContingencyEventService) sendContingencyEvent(ctx context.Context, event *models.ContingencyEvent, branchID uint, docs []dte.ContingencyDocument) error {
	client, err := s.authManager.GetByNIT(ctx, event.Issuer.NIT)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "sendContingencyEvent", "failed to get client", err)
	}

	token, err := s.tokenSvc.GenerateTokenForUser(client, branchID)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "sendContingencyEvent", "failed to generate matching token", err)
	}

	isDuplicate, err := s.httpSvc.SignAndSend(ctx, event, client.NIT, token)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyEventService", "sendContingencyEvent", "failed to sign and send contingency event", err)
	}

	if isDuplicate {
		return &contingency.ContingencyEventExistsError{
			Message:   "contingency event already exists in Hacienda",
			Documents: docs,
		}
	}

	return nil
}
