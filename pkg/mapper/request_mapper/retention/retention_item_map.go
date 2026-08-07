package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapRetentionItemList(req []structs.RetentionItem) ([]retention_models.RetentionItem, error) {
	if req == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Items")
	}

	var retentionItems []retention_models.RetentionItem
	for i, item := range req {
		retentionItem, err := mapRetentionItem(&item, i+1)
		if err != nil {
			return nil, err
		}
		retentionItems = append(retentionItems, *retentionItem)
	}

	return retentionItems, nil
}

func mapRetentionItem(req *structs.RetentionItem, i int) (*retention_models.RetentionItem, error) {
	if req == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "RetentionItem")
	}

	documentType, err := document.NewOperationType(req.DocumentType)
	if err != nil {
		return nil, err
	}

	err = validateRetentionFields(req)
	if err != nil {
		return nil, err
	}

	documentNumber, err := document.NewDocumentNumber(req.DocumentNumber, req.DocumentType)
	if err != nil {
		return nil, err
	}

	taxedAmount, err := financial.NewAmount(*req.TaxedAmount)
	if err != nil {
		return nil, err
	}

	ivaAmount, err := financial.NewAmount(*req.IvaAmount)
	if err != nil {
		return nil, err
	}

	emissionDate, err := temporal.NewEmissionDateFromString(*req.EmissionDate)
	if err != nil {
		return nil, err
	}

	dteType, err := document.NewDTEType(*req.DTEType)
	if err != nil {
		return nil, err
	}

	retentionCode, err := document.NewRetentionCode(req.RetentionCode)
	if err != nil {
		return nil, err
	}

	if req.Description == "" {
		return nil, dte_errors.NewValidationError("RequiredField", "Description")
	}

	return &retention_models.RetentionItem{
		Number:          *item.NewValidatedItemNumber(i),
		DocumentType:    *documentType,
		DocumentNumber:  documentNumber,
		Description:     req.Description,
		RetentionAmount: *taxedAmount,
		RetentionIVA:    *ivaAmount,
		EmissionDate:    *emissionDate,
		DTEType:         *dteType,
		ReceptionCodeMH: *retentionCode,
	}, nil
}

func validateRetentionFields(req *structs.RetentionItem) error {
	if req.TaxedAmount == nil {
		return dte_errors.NewValidationError("RequiredField", "TaxedAmount")
	}

	if req.IvaAmount == nil {
		return dte_errors.NewValidationError("RequiredField", "IVAAmount")
	}

	if req.EmissionDate == nil {
		return dte_errors.NewValidationError("RequiredField", "EmissionDate")
	}

	if req.DTEType == nil {
		return dte_errors.NewValidationError("RequiredField", "DTEType")
	}

	return nil
}
