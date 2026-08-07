package invalidation

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

func MapInvalidatedDocument(baseDTE *dte.DTEDetails, request *structs.CreateInvalidationRequest, emissionDate time.Time) (*invalidation_models.InvalidatedDocument, error) {
	if baseDTE == nil {
		return nil, shared_error.NewFormattedGeneralServiceError("InvalidationMapper", "MapToInvalidatedDocument", "InvalidBaseDTE")
	}

	var dteData map[string]interface{}
	if err := json.Unmarshal([]byte(baseDTE.JSONData), &dteData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DTE data: %w", err)
	}

	docType, numDoc, name, email, phone, err := extractReceptorData(dteData)
	if err != nil {
		return nil, err
	}

	montoIVA, err := extractIVAAmount(dteData)
	if err != nil {
		return nil, err
	}

	doc := &invalidation_models.InvalidatedDocument{
		Type:           *document.NewValidatedDTEType(baseDTE.DTEType),
		GenerationCode: *identification.NewValidatedGenerationCode(baseDTE.ID),
		ControlNumber:  *identification.NewValidatedControlNumber(baseDTE.ControlNumber),
		ReceptionStamp: *baseDTE.ReceptionStamp,
		EmissionDate:   *temporal.NewValidatedEmissionDate(emissionDate),
		IVAAmount:      financial.NewValidatedAmount(montoIVA),
	}

	if phone != nil {
		doc.Phone = base.NewValidatedPhone(*phone)
	}

	if name != nil {
		doc.Name = name
	}

	if docType != nil && numDoc != nil {
		doc.DocumentType = document.NewValidatedDTEType(*docType)
		doc.DocumentNumber = identification.NewValidatedDocumentNumber(*numDoc)
	}

	if email != nil {
		doc.Email = base.NewValidatedEmail(*email)
	}

	if request.ReplacementGenerationCode != nil {
		doc.ReplacementCode = identification.NewValidatedGenerationCode(*request.ReplacementGenerationCode)
	}

	return doc, nil
}

func extractReceptorData(dteData map[string]interface{}) (*string, *string, *string, *string, *string, error) {
	var docType, numDoc, name, phone, email *string

	receptor, ok := dteData["receptor"].(map[string]interface{})
	if !ok {
		sujetoExcluido, okSujeto := dteData["sujetoExcluido"].(map[string]interface{})
		if !okSujeto {
			return nil, nil, nil, nil, nil, nil
		}
		receptor = sujetoExcluido
	}

	if nombreValue, ok := receptor["nombre"].(string); ok {
		name = &nombreValue
	} else {
		name = new(string)
	}

	if telefonoValue, ok := receptor["telefono"].(string); ok {
		phone = &telefonoValue
	} else {
		phone = new(string)
	}

	if correoValue, ok := receptor["correo"].(string); ok {
		email = &correoValue
	} else {
		email = new(string)
	}

	if tipoDocValue, ok := receptor["tipoDocumento"].(string); ok {
		docType = &tipoDocValue
	} else {
		if nitValue, ok := receptor["nit"].(string); ok {
			nit := constants.NIT
			return &nit, &nitValue, name, email, phone, nil
		}
		docType = new(string)
	}

	if numDocValue, ok := receptor["numDocumento"].(string); ok {
		numDoc = &numDocValue
	} else {
		numDoc = new(string)
	}

	return docType, numDoc, name, email, phone, nil
}

func extractIVAAmount(dteData map[string]interface{}) (float64, error) {
	resumen, ok := dteData["resumen"].(map[string]interface{})
	if !ok {
		return 0, shared_error.NewGeneralServiceError("InvalidationMapper", "extractIVAAmount", "invalid resumen data in DTE", nil)
	}

	ivaAmount, ok := resumen["totalIva"].(float64)
	if ok {
		return ivaAmount, nil
	}

	tributos, ok := resumen["tributos"].([]interface{})
	if !ok {
		return 0, nil
	}

	for _, t := range tributos {
		tributo := t.(map[string]interface{})
		if tributo["codigoTributo"] == constants.TaxIVA {
			return tributo["valorTributo"].(float64), nil
		}
	}

	return 0, nil
}
