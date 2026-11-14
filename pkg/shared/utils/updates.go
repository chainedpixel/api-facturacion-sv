package utils

import (
	"encoding/json"
	"fmt"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
)

// UpdateContingencyIdentification sets the deferred model, the contingency transmission type and the contingency
// type in the DTE identification; the contingency reason is only kept for contingency type 5 (other) and is null
// for types 1 to 4.
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
		identification["motivoContin"] = nil
		if int(*contiType) == constants.OtroMotivo && reason != nil {
			identification["motivoContin"] = *reason
		}
	}

	return dteDoc, nil
}
