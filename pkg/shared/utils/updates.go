package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
)

const maxContingencyReasonLength = 500

// UpdateContingencyIdentification sets the deferred model, the contingency transmission type and the contingency
// type in the DTE identification. The contingency reason is required for type 5 (other) and defaults to null for
// types 1 to 4.
func UpdateContingencyIdentification(document interface{}, contiType *int8, reason *string) (map[string]interface{}, error) {
	var raw []byte
	switch v := document.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		jsonBytes, err := json.Marshal(document)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DTE JSON: %w", err)
		}
		raw = jsonBytes
	}

	dteDoc, err := decodePreservingNumbers(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DTE JSON: %w", err)
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
		if int(*contiType) == constants.OtroMotivo {
			identification["motivoContin"] = otherContingencyReason(reason)
		}
	}

	return dteDoc, nil
}

// decodePreservingNumbers decodes a JSON object keeping every number with its original text, so amounts and large
// integers are written back exactly as they were received.
func decodePreservingNumbers(raw []byte) (map[string]interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document map[string]interface{}
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	return document, nil
}

// otherContingencyReason returns the reason of a type 5 contingency: the given text, or the catalog description when
// it is empty, limited to the 500 characters accepted by the DTE.
func otherContingencyReason(reason *string) string {
	text := ""
	if reason != nil {
		text = strings.TrimSpace(*reason)
	}
	if text == "" {
		text = constants.GetContingencyReason(constants.OtroMotivo)
	}
	if runes := []rune(text); len(runes) > maxContingencyReasonLength {
		text = string(runes[:maxContingencyReasonLength])
	}
	return text
}
