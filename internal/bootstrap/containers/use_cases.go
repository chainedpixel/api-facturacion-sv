package containers

import (
	"github.com/chainedpixel/ordo-factus/internal/application/auth"
	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/application/ports"
)

type UseCaseContainer struct {
	services *ServicesContainer

	dteConsult          *dte.DTEConsultUseCase
	invalidationUseCase *dte.InvalidationUseCase
	authUseCase         *auth.AuthUseCase
	baseTransmitter     ports.BaseTransmitter
	dteUseCaseFactory   *dte.DTEUseCaseFactory

	invoiceUseCase       *dte.GenericDTEUseCase
	ccfUseCase           *dte.GenericDTEUseCase
	retentionUseCase     *dte.GenericDTEUseCase
	creditNoteUseCase    *dte.GenericDTEUseCase
	remissionNoteUseCase *dte.GenericDTEUseCase
	fseUseCase           *dte.GenericDTEUseCase
	debitNoteUseCase     *dte.GenericDTEUseCase
}

func NewUseCaseContainer(services *ServicesContainer) *UseCaseContainer {
	return &UseCaseContainer{
		services: services,
	}
}

func (c *UseCaseContainer) Initialize() {
	c.authUseCase = auth.NewAuthUseCase(c.services.AuthManager(), c.services.CryptManager())
	c.baseTransmitter = dte.NewBaseTransmitter(c.services.TransmitterManager(), c.services.SignerManager())
	c.dteConsult = dte.NewDTEConsultUseCase(c.services.DTEManager())

	c.dteUseCaseFactory = dte.NewDTEUseCaseFactory(
		c.services.AuthManager(),
		c.services.DTEManager(),
		c.baseTransmitter,
		c.services.SequentialManager())

	c.invoiceUseCase = c.dteUseCaseFactory.CreateInvoiceUseCase(c.services.InvoiceService())
	c.ccfUseCase = c.dteUseCaseFactory.CreateCCFUseCase(c.services.CCFService())
	c.retentionUseCase = c.dteUseCaseFactory.CreateRetentionUseCase(c.services.RetentionManager())
	c.creditNoteUseCase = c.dteUseCaseFactory.CreateCreditNoteUseCase(c.services.CreditNoteManager())
	c.remissionNoteUseCase = c.dteUseCaseFactory.CreateRemissionNoteUseCase(c.services.RemissionNoteManager())
	c.fseUseCase = c.dteUseCaseFactory.CreateFSEUseCase(c.services.FSEManager())
	c.debitNoteUseCase = c.dteUseCaseFactory.CreateDebitNoteUseCase(c.services.DebitNoteManager())

	c.invalidationUseCase = c.dteUseCaseFactory.CreateInvalidationUseCase(c.services.InvalidationManager())

}

func (c *UseCaseContainer) DTEConsultUseCase() *dte.DTEConsultUseCase {
	return c.dteConsult
}

func (c *UseCaseContainer) InvoiceUseCase() *dte.GenericDTEUseCase {
	return c.invoiceUseCase
}

func (c *UseCaseContainer) CCFUseCase() *dte.GenericDTEUseCase {
	return c.ccfUseCase
}

func (c *UseCaseContainer) RetentionUseCase() *dte.GenericDTEUseCase {
	return c.retentionUseCase
}

func (c *UseCaseContainer) CreditNoteUseCase() *dte.GenericDTEUseCase {
	return c.creditNoteUseCase
}

func (c *UseCaseContainer) InvalidationUseCase() *dte.InvalidationUseCase {
	return c.invalidationUseCase
}

func (c *UseCaseContainer) RemissionNoteUseCase() *dte.GenericDTEUseCase {
	return c.remissionNoteUseCase
}

func (c *UseCaseContainer) FSEUseCase() *dte.GenericDTEUseCase {
	return c.fseUseCase
}

func (c *UseCaseContainer) DebitNoteUseCase() *dte.GenericDTEUseCase {
	return c.debitNoteUseCase
}

func (c *UseCaseContainer) AuthUseCase() *auth.AuthUseCase {
	return c.authUseCase
}
