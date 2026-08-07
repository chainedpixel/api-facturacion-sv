package mapper

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
)

// MapperFactory creates request and response mapper adapters for each DTE type.
type MapperFactory struct{}

// NewMapperFactory creates a new MapperFactory instance.
func NewMapperFactory() *MapperFactory {
	return &MapperFactory{}
}

// CreateInvoiceMapperAdapter creates a DTEMapper for invoice requests.
func (f *MapperFactory) CreateInvoiceMapperAdapter() DTEMapper {
	m := request_mapper.NewInvoiceMapper()
	return newTypedMapper[structs.CreateInvoiceRequest](m.MapToInvoiceData)
}

// CreateCCFMapperAdapter creates a DTEMapper for Comprobante de Crédito Fiscal requests.
func (f *MapperFactory) CreateCCFMapperAdapter() DTEMapper {
	m := request_mapper.NewCCFMapper()
	return newTypedMapper[structs.CreateCreditFiscalRequest](m.MapToCCFData)
}

// CreateCreditNoteMapperAdapter creates a DTEMapper for credit note requests.
func (f *MapperFactory) CreateCreditNoteMapperAdapter() DTEMapper {
	m := request_mapper.NewCreditNoteMapper()
	return newTypedMapper[structs.CreateCreditNoteRequest](m.MapToCreditNoteData)
}

// CreateDebitNoteMapperAdapter creates a DTEMapper for debit note requests.
func (f *MapperFactory) CreateDebitNoteMapperAdapter() DTEMapper {
	m := request_mapper.NewDebitNoteMapper()
	return newTypedMapper[structs.CreateDebitNoteRequest](m.MapToDebitNoteData)
}

// CreateRetentionMapperAdapter creates a DTEMapper for retention requests.
func (f *MapperFactory) CreateRetentionMapperAdapter() DTEMapper {
	m := request_mapper.NewRetentionMapper()
	return newTypedMapper[structs.CreateRetentionRequest](m.MapToRetentionData)
}

// CreateRemissionNoteMapperAdapter creates a DTEMapper for remission note requests.
func (f *MapperFactory) CreateRemissionNoteMapperAdapter() DTEMapper {
	m := request_mapper.NewRemissionNoteMapper()
	return newTypedMapper[structs.CreateRemissionNoteRequest](m.MapToRemissionNoteData)
}

// CreateFSEMapperAdapter creates a DTEMapper for Factura de Sujeto Excluido requests.
func (f *MapperFactory) CreateFSEMapperAdapter() DTEMapper {
	m := request_mapper.NewFSEMapper()
	return newTypedMapper[structs.CreateFSERequest](m.MapToFSEData)
}

// CreateInvalidationMapperAdapter creates a DTEMapper for invalidation requests.
// It expects two extra params: *dte.DTEDetails and time.Time (emission date).
func (f *MapperFactory) CreateInvalidationMapperAdapter() DTEMapper {
	m := request_mapper.NewInvalidationMapper()
	return newTypedMapperWithParams[structs.CreateInvalidationRequest](
		func(req *structs.CreateInvalidationRequest, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error) {
			baseDte := params[0].(*dte.DTEDetails)
			emissionDate := params[1].(time.Time)
			return m.MapToInvalidationData(req, issuer, baseDte, emissionDate)
		},
	)
}

// GetInvoiceResponseMapper returns the response mapper function for invoices.
func (f *MapperFactory) GetInvoiceResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHInvoice(domain)
	}
}

// GetCCFResponseMapper returns the response mapper function for CCF documents.
func (f *MapperFactory) GetCCFResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHCreditFiscalInvoice(domain)
	}
}

// GetCreditNoteResponseMapper returns the response mapper function for credit notes.
func (f *MapperFactory) GetCreditNoteResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHCreditNote(domain)
	}
}

// GetDebitNoteResponseMapper returns the response mapper function for debit notes.
func (f *MapperFactory) GetDebitNoteResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHDebitNote(domain)
	}
}

// GetRetentionResponseMapper returns the response mapper function for retentions.
func (f *MapperFactory) GetRetentionResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHRetention(domain)
	}
}

// GetRemissionNoteResponseMapper returns the response mapper function for remission notes.
func (f *MapperFactory) GetRemissionNoteResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHRemissionNote(domain)
	}
}

// GetFSEResponseMapper returns the response mapper function for FSE documents.
func (f *MapperFactory) GetFSEResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHFSE(domain)
	}
}

// GetInvalidationResponseMapper returns the response mapper function for invalidations.
func (f *MapperFactory) GetInvalidationResponseMapper() ResponseMapperFunc {
	return func(domain interface{}) interface{} {
		return response_mapper.ToMHInvalidation(domain)
	}
}
