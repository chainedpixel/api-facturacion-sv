package utils

import (
	"encoding/json"
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
)

// UpdateContingencyIdentification updates the contingency identification in the DTE JSON.
func UpdateContingencyIdentification(document interface{}, contiType *int8, reason *string) (map[string]interface{}, error) {
	var dteDoc map[string]interface{}
	switch v := document.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &dteDoc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DTE JSON: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(v, &dteDoc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DTE JSON: %w", err)
		}
	default:
		jsonBytes, err := json.Marshal(document)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DTE JSON: %w", err)
		}

		if err = json.Unmarshal(jsonBytes, &dteDoc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DTE JSON: %w", err)
		}
	}

	if identifi, exist := dteDoc["identificacion"]; exist {
		identification, ok := identifi.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to assert identification as map[string]interface{}")
		}
		identification["tipoModelo"] = constants.ModeloFacturacionDiferido
		identification["tipoOperacion"] = constants.TransmisionContingencia
		identification["tipoContingencia"] = *contiType
		identification["motivoContin"] = *reason
	}

	return dteDoc, nil
}

// SetReceptionStampIntoAppendix adds the reception stamp to the document appendix.
func SetReceptionStampIntoAppendix(document string, receptionStamp *string) (string, error) {
	if receptionStamp == nil || *receptionStamp == "" {
		return document, nil
	}

	dteInfo, err := ExtractAuxiliarIdentificationFromStringJSON(document)
	if err != nil || dteInfo.Identification.DTEType == "" {
		return "", fmt.Errorf("failed to determine DTE type: %w", err)
	}

	var dteDoc map[string]interface{}
	if err := json.Unmarshal([]byte(document), &dteDoc); err != nil {
		return "", fmt.Errorf("failed to unmarshal DTE JSON: %w", err)
	}

	stampEntry := map[string]interface{}{
		"Campo":    "Datos del documento",
		"Etiqueta": "Sello de recepción",
		"Valor":    *receptionStamp,
	}

	if appendix, exist := dteDoc["apendice"]; exist {
		if appendix == nil {
			dteDoc["apendice"] = []map[string]interface{}{stampEntry}
		} else {
			appendices, ok := appendix.([]interface{})
			if !ok {
				return "", fmt.Errorf("unexpected appendix type: expected []interface{}")
			}
			dteDoc["apendice"] = append(appendices, stampEntry)
		}
	}

	jsonData, err := json.Marshal(dteDoc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DTE JSON: %w", err)
	}

	return string(jsonData), nil
}
