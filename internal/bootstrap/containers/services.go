package containers

import (
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/chainedpixel/ordo-factus/config"
	notificationHandlers "github.com/chainedpixel/ordo-factus/internal/application/handlers/notification"
	appPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/service/strategies"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/notification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/metrics"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/test_endpoint"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/cache"
	adapterContingecy "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/contingency"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/crypt"
	adapterEvents "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/events"
	adapterHealth "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/health"
	adapterMetric "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/metrics"
	adapterEmail "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/signing"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/signing/signer"
	adapterTest "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/test_endpoint"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/tokens"
	adapterTransmitter "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter"
	batch "github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/batch"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type ServicesContainer struct {
	repos *RepositoryContainer

	cacheManager            ports.CacheManager
	tokenManager            ports.TokenManager
	authManager             auth.AuthManager
	cryptManager            ports.CryptManager
	transmitterManager      appPorts.DTETransmitter
	haciendaAuthManager     appPorts.HaciendaAuthManager
	signerManager           appPorts.SignerManager
	dteManager              dte_documents.DTEManager
	sequentialManager       dte_documents.SequentialNumberManager
	invalidationManager     invalidation.InvalidationManager
	transmitterBatchManager transmitter.BatchTransmitterPort
	contingencyEventManager contingency.ContingencyEventSender
	contingencyManager      contingency.ContingencyManager
	healthManager           health.HealthManager
	testManager             test_endpoint.TestManager
	metricsManager          metrics.MetricsManager
	invoiceManager          ports.DTEService
	ccfManager              ports.DTEService
	retentionManager        ports.DTEService
	creditNoteManager       ports.DTEService
	remissionNoteManager    ports.DTEService
	fseManager              ports.DTEService
	debitNoteManager        ports.DTEService
	eventBus                event.Bus
	mailer                  notification.Mailer
	mailRenderer            *adapterEmail.TemplateRenderer
}

func NewServicesContainer(repos *RepositoryContainer) *ServicesContainer {
	return &ServicesContainer{
		repos: repos,
	}
}

func (c *ServicesContainer) Initialize() error {
	var err error

	c.cryptManager = crypt.NewCryptService()
	c.cacheManager, err = cache.NewRedisTokenCache(config.NewRedisConfig(), c.cryptManager)
	if err != nil {
		return err
	}

	c.tokenManager = tokens.NewJWTService(config.Server.JWTSecret, c.cacheManager)
	c.authManager = strategies.NewAuthService(c.tokenManager, c.repos.AuthRepo(), c.cacheManager)
	c.signerManager = signer.NewDTESigner(c.repos.AuthRepo())
	c.haciendaAuthManager = signing.NewHaciendaAuthService(c.cacheManager, c.authManager)
	c.transmitterManager = adapterTransmitter.NewMHTransmitter(c.haciendaAuthManager, c.repos.FailedSequentialNumberRepo())
	c.dteManager = dte_documents.NewDTEService(c.repos.DTERepo())
	c.sequentialManager = dte_documents.NewSequentialNumberService(c.repos.SequentialNumberRepo(), c.repos.AuthRepo(), c.repos.ReservedSequenceRepo())
	c.invoiceManager = invoice.NewInvoiceService(c.sequentialManager)
	c.ccfManager = ccf.NewCCFService(c.sequentialManager)
	c.invalidationManager = invalidation.NewInvalidationService(c.dteManager)
	c.retentionManager = retention.NewRetentionService(c.sequentialManager, c.dteManager)
	c.creditNoteManager = credit_note.NewCreditNoteService(c.sequentialManager, c.dteManager)
	c.remissionNoteManager = remission_note.NewRemissionNoteService(c.sequentialManager, c.dteManager)
	c.fseManager = fse.NewFSEService(c.sequentialManager)
	c.debitNoteManager = debit_note.NewDebitNoteService(c.sequentialManager, c.dteManager)
	c.testManager = adapterTest.NewTestService(c.repos.db, c.repos.AuthRepo())
	c.metricsManager = adapterMetric.NewMetricService(c.cacheManager)

	c.eventBus = adapterEvents.NewInMemoryBus(repositories.NewEventRepository(c.repos.connection.Db))

	c.healthManager = adapterHealth.NewHealthService(&adapterHealth.HealthServiceConfig{
		DB:  c.repos.db,
		Bus: c.eventBus,
	})

	transmissionConf := models.NewTransmissionConfig(5*time.Second, 2*time.Minute, 2.0)
	c.transmitterBatchManager = batch.NewBatchTransmitterService(
		c.haciendaAuthManager,
		c.signerManager,
		c.repos.ContingencyRepo(),
		c.sequentialManager,
		transmissionConf,
		&transmitter.RealTimeProvider{},
		c.repos.connection,
	)

	c.contingencyEventManager = adapterContingecy.NewContingencyEventService(
		c.authManager,
		c.haciendaAuthManager,
		c.cacheManager,
		c.tokenManager,
		c.signerManager,
		c.repos.ContingencyRepo(),
		&transmitter.RealTimeProvider{},
		c.repos.connection,
	)

	c.contingencyManager = contingency.NewContingencyManager(
		c.authManager,
		c.dteManager,
		c.repos.ContingencyRepo(),
		c.haciendaAuthManager,
		c.cacheManager,
		c.tokenManager,
		c.signerManager,
		c.transmitterBatchManager,
		c.contingencyEventManager,
		c.sequentialManager,
		&transmitter.RealTimeProvider{},
		transmissionConf,
	)

	c.initNotificationStack()
	c.attachEventBusToPublishers()

	return nil
}

func (c *ServicesContainer) attachEventBusToPublishers() {
	if c.eventBus == nil {
		return
	}
	if cs, ok := c.contingencyManager.(*contingency.ContingencyService); ok {
		cs.SetEventBus(c.eventBus)
	}
	if bs, ok := c.transmitterBatchManager.(*batch.BatchTransmitterService); ok {
		bs.SetEventBus(c.eventBus)
	}
}

func (c *ServicesContainer) initNotificationStack() {
	c.mailer = adapterEmail.NewSMTPMailer()

	renderer, err := adapterEmail.NewTemplateRenderer()
	if err != nil {
		logs.Error("admin notifications disabled: failed to load email templates", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	c.mailRenderer = renderer

	cooldownClient := buildCooldownRedisClient()
	var cooldown adapterEmail.Cooldown
	if cooldownClient != nil {
		cooldown = adapterEmail.NewRedisCooldown(
			cooldownClient,
			time.Duration(config.Server.NotifyCooldownMinutes)*time.Minute,
		)
	}

	handler := notificationHandlers.NewAdminEmailHandler(c.mailer, cooldown, renderer)
	c.eventBus.Subscribe(event.EventContingencyActivated, handler)
	c.eventBus.Subscribe(event.EventEmissionFailure, handler)
	c.eventBus.Subscribe(event.EventRetransmissionJobFailed, handler)
}

func buildCooldownRedisClient() *redis.Client {
	if config.Redis == nil {
		return nil
	}
	opt, err := redis.ParseURL(config.NewRedisConfig().GetURL())
	if err != nil {
		logs.Warn("notification cooldown disabled: invalid redis URL", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	return redis.NewClient(opt)
}

func (c *ServicesContainer) CreditNoteManager() ports.DTEService {
	return c.creditNoteManager
}

func (c *ServicesContainer) RetentionManager() ports.DTEService {
	return c.retentionManager
}

func (c *ServicesContainer) MetricsManager() metrics.MetricsManager {
	return c.metricsManager
}

func (c *ServicesContainer) HealthManager() health.HealthManager {
	return c.healthManager
}

func (c *ServicesContainer) TestManager() test_endpoint.TestManager {
	return c.testManager
}

func (c *ServicesContainer) InvalidationManager() invalidation.InvalidationManager {
	return c.invalidationManager
}

func (c *ServicesContainer) ContingencyManager() contingency.ContingencyManager {
	return c.contingencyManager
}

func (c *ServicesContainer) DTEManager() dte_documents.DTEManager {
	return c.dteManager
}

func (c *ServicesContainer) CCFService() ports.DTEService {
	return c.ccfManager
}

func (c *ServicesContainer) InvoiceService() ports.DTEService {
	return c.invoiceManager
}

func (c *ServicesContainer) TransmitterManager() appPorts.DTETransmitter {
	return c.transmitterManager
}

func (c *ServicesContainer) SignerManager() appPorts.SignerManager {
	return c.signerManager
}

func (c *ServicesContainer) HaciendaAuthManager() appPorts.HaciendaAuthManager {
	return c.haciendaAuthManager
}

func (c *ServicesContainer) CacheManager() ports.CacheManager {
	return c.cacheManager
}

func (c *ServicesContainer) TokenManager() ports.TokenManager {
	return c.tokenManager
}

func (c *ServicesContainer) AuthManager() auth.AuthManager {
	return c.authManager
}

func (c *ServicesContainer) RemissionNoteManager() ports.DTEService {
	return c.remissionNoteManager
}

func (c *ServicesContainer) FSEManager() ports.DTEService {
	return c.fseManager
}

func (c *ServicesContainer) DebitNoteManager() ports.DTEService {
	return c.debitNoteManager
}

func (c *ServicesContainer) CryptManager() ports.CryptManager {
	return c.cryptManager
}

func (c *ServicesContainer) SequentialManager() dte_documents.SequentialNumberManager {
	return c.sequentialManager
}

func (c *ServicesContainer) EventBus() event.Bus {
	return c.eventBus
}

func (c *ServicesContainer) Mailer() notification.Mailer {
	return c.mailer
}

func (c *ServicesContainer) MailRenderer() *adapterEmail.TemplateRenderer {
	return c.mailRenderer
}
