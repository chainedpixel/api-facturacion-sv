package containers

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/handlers"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

type HandlerContainer struct {
	useCases *UseCaseContainer
	services *ServicesContainer

	authHandler        *handlers.AuthHandler
	dteHandler         *handlers.DTEHandler
	healthHandler      *handlers.HealthHandler
	testHandler        *handlers.TestHandler
	metricsHandler     *handlers.MetricsHandler
	contingencyHandler *helpers.ContingencyHandler
	debugNotifyHandler *handlers.DebugNotifyHandler
}

func NewHandlerContainer(useCases *UseCaseContainer, services *ServicesContainer) *HandlerContainer {
	return &HandlerContainer{
		useCases: useCases,
		services: services,
	}
}

func (c *HandlerContainer) Initialize() {
	c.contingencyHandler = helpers.NewContingencyHandler(c.services.contingencyManager)
	c.healthHandler = handlers.NewHealthHandler(c.services.HealthManager())
	c.testHandler = handlers.NewTestHandler(c.services.TestManager())
	c.authHandler = handlers.NewAuthHandler(c.useCases.AuthUseCase())
	c.metricsHandler = handlers.NewMetricsHandler(c.services.MetricsManager())
	c.dteHandler = handlers.NewDTEHandler(c.useCases.DTEConsultUseCase(), c.useCases.InvalidationUseCase(),
		c.initializeGenericCreatorHandler(c.contingencyHandler),
	)
	c.debugNotifyHandler = handlers.NewDebugNotifyHandler(c.services.Mailer(), c.services.MailRenderer())
}

func (c *HandlerContainer) initializeGenericCreatorHandler(contingencyHandler *helpers.ContingencyHandler) *handlers.GenericCreatorDTEHandler {
	genericHandler := handlers.NewGenericDTEHandler(contingencyHandler)

	genericHandler.RegisterDocument("/dte/invoices", helpers.DocumentConfig{
		UseCase:         c.useCases.InvoiceUseCase(),
		RequestType:     &structs.CreateInvoiceRequest{},
		DocumentType:    constants.FacturaElectronica,
		UsesContingency: true,
	})

	genericHandler.RegisterDocument("/dte/ccf", helpers.DocumentConfig{
		UseCase:         c.useCases.CCFUseCase(),
		RequestType:     &structs.CreateCreditFiscalRequest{},
		DocumentType:    constants.CCFElectronico,
		UsesContingency: true,
	})

	genericHandler.RegisterDocument("/dte/creditnote", helpers.DocumentConfig{
		UseCase:         c.useCases.CreditNoteUseCase(),
		RequestType:     &structs.CreateCreditNoteRequest{},
		DocumentType:    constants.NotaCreditoElectronica,
		UsesContingency: false,
	})

	genericHandler.RegisterDocument("/dte/retention", helpers.DocumentConfig{
		UseCase:         c.useCases.RetentionUseCase(),
		RequestType:     &structs.CreateRetentionRequest{},
		DocumentType:    constants.ComprobanteRetencionElectronico,
		UsesContingency: false,
	})

	genericHandler.RegisterDocument("/dte/remissionnote", helpers.DocumentConfig{
		UseCase:         c.useCases.RemissionNoteUseCase(),
		RequestType:     &structs.CreateRemissionNoteRequest{},
		DocumentType:    constants.NotaRemisionElectronica,
		UsesContingency: true,
	})

	genericHandler.RegisterDocument("/dte/fse", helpers.DocumentConfig{
		UseCase:         c.useCases.FSEUseCase(),
		RequestType:     &structs.CreateFSERequest{},
		DocumentType:    constants.FacturaSujetoExcluidoElectronica,
		UsesContingency: true,
	})

	genericHandler.RegisterDocument("/dte/debitnote", helpers.DocumentConfig{
		UseCase:         c.useCases.DebitNoteUseCase(),
		RequestType:     &structs.CreateDebitNoteRequest{},
		DocumentType:    constants.NotaDebitoElectronica,
		UsesContingency: true,
	})

	return genericHandler
}

func (c *HandlerContainer) MetricsHandler() *handlers.MetricsHandler {
	return c.metricsHandler
}

func (c *HandlerContainer) HealthHandler() *handlers.HealthHandler {
	return c.healthHandler
}

func (c *HandlerContainer) TestHandler() *handlers.TestHandler {
	return c.testHandler
}

func (c *HandlerContainer) DTEHandler() *handlers.DTEHandler {
	return c.dteHandler
}

func (c *HandlerContainer) AuthHandler() *handlers.AuthHandler {
	return c.authHandler
}

func (c *HandlerContainer) DebugNotifyHandler() *handlers.DebugNotifyHandler {
	return c.debugNotifyHandler
}
