package dte

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/event"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation"
	domainPort "github.com/chainedpixel/api-facturacion-sv/internal/domain/ports"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper"
)

// DTEUseCaseFactory facilitates the creation of use cases for different DTE types
type DTEUseCaseFactory struct {
	authService       auth.AuthManager
	dteService        dte_documents.DTEManager
	transmitter       ports.BaseTransmitter
	sequentialManager dte_documents.SequentialNumberManager
	bus               event.Bus
	mapperFactory     *mapper.MapperFactory
	operationsFactory *DTEOperations
}

// NewDTEUseCaseFactory creates a new DTEUseCaseFactory instance
func NewDTEUseCaseFactory(
	authService auth.AuthManager,
	dteService dte_documents.DTEManager,
	transmitter ports.BaseTransmitter,
	sequentialManager dte_documents.SequentialNumberManager,
	bus event.Bus,
) *DTEUseCaseFactory {
	return &DTEUseCaseFactory{
		authService:       authService,
		dteService:        dteService,
		transmitter:       transmitter,
		sequentialManager: sequentialManager,
		bus:               bus,
		mapperFactory:     mapper.NewMapperFactory(),
		operationsFactory: NewDTEOperations(),
	}
}

// CreateInvoiceUseCase creates a use case for invoices
func (f *DTEUseCaseFactory) CreateInvoiceUseCase(invoiceService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		invoiceService,
		f.sequentialManager,
		f.mapperFactory.CreateInvoiceMapperAdapter(),
		f.mapperFactory.GetInvoiceResponseMapper(),
		f.operationsFactory.GetNoOperation(),
		f.bus,
	)
}

// CreateCCFUseCase creates a use case for CCF
func (f *DTEUseCaseFactory) CreateCCFUseCase(ccfService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		ccfService,
		f.sequentialManager,
		f.mapperFactory.CreateCCFMapperAdapter(),
		f.mapperFactory.GetCCFResponseMapper(),
		f.operationsFactory.GetNoOperation(),
		f.bus,
	)
}

// CreateCreditNoteUseCase creates a use case for credit notes
func (f *DTEUseCaseFactory) CreateCreditNoteUseCase(creditNoteService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		creditNoteService,
		f.sequentialManager,
		f.mapperFactory.CreateCreditNoteMapperAdapter(),
		f.mapperFactory.GetCreditNoteResponseMapper(),
		f.operationsFactory.GetCreditNoteOperations(f.dteService),
		f.bus,
	)
}

// CreateRetentionUseCase creates a use case for retentions
func (f *DTEUseCaseFactory) CreateRetentionUseCase(retentionService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		retentionService,
		f.sequentialManager,
		f.mapperFactory.CreateRetentionMapperAdapter(),
		f.mapperFactory.GetRetentionResponseMapper(),
		f.operationsFactory.GetNoOperation(),
		f.bus,
	)
}

// CreateDebitNoteUseCase creates a use case for debit notes
func (f *DTEUseCaseFactory) CreateDebitNoteUseCase(debitNoteService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		debitNoteService,
		f.sequentialManager,
		f.mapperFactory.CreateDebitNoteMapperAdapter(),
		f.mapperFactory.GetDebitNoteResponseMapper(),
		f.operationsFactory.GetDebitNoteOperations(f.dteService),
		f.bus,
	)
}

// CreateRemissionNoteUseCase creates a use case for remission notes
func (f *DTEUseCaseFactory) CreateRemissionNoteUseCase(remissionNoteService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		remissionNoteService,
		f.sequentialManager,
		f.mapperFactory.CreateRemissionNoteMapperAdapter(),
		f.mapperFactory.GetRemissionNoteResponseMapper(),
		f.operationsFactory.GetNoOperation(),
		f.bus,
	)
}

func (f *DTEUseCaseFactory) CreateInvalidationUseCase(
	invalidationManager invalidation.InvalidationManager,
) *InvalidationUseCase {
	return NewInvalidationUseCase(
		f.dteService,
		invalidationManager,
		f.authService,
		f.transmitter,
	)
}

// CreateFSEUseCase creates a use case for excluded subject invoices
func (f *DTEUseCaseFactory) CreateFSEUseCase(fseService domainPort.DTEService) *GenericDTEUseCase {
	return NewGenericDTEUseCase(
		f.authService,
		f.dteService,
		f.transmitter,
		fseService,
		f.sequentialManager,
		f.mapperFactory.CreateFSEMapperAdapter(),
		f.mapperFactory.GetFSEResponseMapper(),
		f.operationsFactory.GetNoOperation(),
		f.bus,
	)
}
