package processors

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

func GetDocumentRequestData(document interface{}) (int, string, string, int, error) {
	var docMap map[string]interface{}

	jsonData, err := json.Marshal(document)
	if err != nil {
		return 0, "", "", 0, err
	}

	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return 0, "", "", 0, err
	}

	identification, ok := docMap["identificacion"].(map[string]interface{})
	if !ok {
		logs.Error("Missing or invalid identificacion object")
		return 0, "", "", 0, fmt.Errorf("missing or invalid identificacion object")
	}

	version, ok := identification["version"].(float64)
	if !ok {
		logs.Error("Missing or invalid version field")
		return 0, "", "", 0, fmt.Errorf("missing or invalid version field")
	}

	dteType, ok := identification["tipoDte"].(string)
	if !ok {
		documento, docOk := docMap["documento"].(map[string]interface{})
		if !docOk || documento == nil {
			logs.Error("Missing or invalid tipoDte field")
			return 0, "", "", 0, fmt.Errorf("missing or invalid tipoDte field")
		}
		dteType, ok = documento["tipoDte"].(string)
		if !ok {
			logs.Error("Missing or invalid tipoDte field")
			return 0, "", "", 0, fmt.Errorf("missing or invalid tipoDte field")
		}
	}

	var controlNumber string
	if cn, okCn := identification["numeroControl"].(string); okCn {
		controlNumber = cn
	} else if documento, docOk := docMap["documento"].(map[string]interface{}); docOk && documento != nil {
		if cn, okCn := documento["numeroControl"].(string); okCn {
			controlNumber = cn
		}
	}

	generationCode, ok := identification["codigoGeneracion"].(string)
	if !ok {
		logs.Error("Missing or invalid codigoGeneracion field")
		return 0, "", "", 0, fmt.Errorf("missing or invalid codigoGeneracion field")
	}

	sequenceNumber := 1
	if controlNumber != "" {
		if seq, err := extractCorrelativo(controlNumber); err == nil {
			sequenceNumber = seq
		}
	}

	return int(version), dteType, generationCode, sequenceNumber, nil
}

func extractCorrelativo(numeroControl string) (int, error) {
	parts := strings.Split(numeroControl, "-")
	if len(parts) != 4 {
		return 0, fmt.Errorf("invalid numeroControl format")
	}
	lastPart := parts[3]

	correlativo, err := strconv.Atoi(lastPart)
	if err != nil {
		return 0, fmt.Errorf("invalid correlativo number: %w", err)
	}

	return correlativo, nil
}
