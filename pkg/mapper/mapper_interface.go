package mapper

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// DTEMapper is a generic interface for all DTE mappers.
type DTEMapper interface {
	MapToDomainModel(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error)
}

// ResponseMapperFunc defines the function type for response mapping.
type ResponseMapperFunc func(domain interface{}) interface{}

// MapperAdapter adapts any mapping function to the DTEMapper interface.
type MapperAdapter struct {
	MapFunc func(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error)
}

// MapToDomainModel implements DTEMapper, forwarding variadic params correctly.
func (a *MapperAdapter) MapToDomainModel(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error) {
	return a.MapFunc(req, issuer, params...)
}

// newTypedMapper creates a DTEMapper from a type-safe mapping function.
// It handles the type assertion internally and returns a descriptive error on mismatch.
func newTypedMapper[Req any, Res any](
	mapFn func(req *Req, issuer *dte.IssuerDTE) (Res, error),
) DTEMapper {
	var zero Req
	return &MapperAdapter{
		MapFunc: func(req interface{}, issuer *dte.IssuerDTE, _ ...interface{}) (interface{}, error) {
			typedReq, ok := req.(*Req)
			if !ok {
				return nil, fmt.Errorf("invalid request type: expected *%T, got %T", zero, req)
			}
			return mapFn(typedReq, issuer)
		},
	}
}

// newTypedMapperWithParams creates a DTEMapper from a type-safe mapping function that accepts extra params.
func newTypedMapperWithParams[Req any, Res any](
	mapFn func(req *Req, issuer *dte.IssuerDTE, params ...interface{}) (Res, error),
) DTEMapper {
	var zero Req
	return &MapperAdapter{
		MapFunc: func(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error) {
			typedReq, ok := req.(*Req)
			if !ok {
				return nil, fmt.Errorf("invalid request type: expected *%T, got %T", zero, req)
			}
			return mapFn(typedReq, issuer, params...)
		},
	}
}
