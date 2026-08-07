package utils

import (
	"encoding/json"
)

type AuxiliarIdentificationExtractor struct {
	Identification struct {
		DTEType        string `json:"tipoDte"`
		ControlNumber  string `json:"numeroControl"`
		GenerationCode string `json:"codigoGeneracion"`
	} `json:"identificacion"`
	Issuer struct {
		NIT string `json:"nit"`
	} `json:"emisor"`
}

type AuxiliarReceiverExtractor struct {
	Receiver struct {
		NIT string `json:"nit"`
	} `json:"receptor"`
}

type AuxiliarTotalAmountsExtractor struct {
	Summary struct {
		TotalTaxed      float64 `json:"totalGravada"`
		TotalExempt     float64 `json:"totalExenta"`
		TotalNotSubject float64 `json:"totalNoSuj"`
	} `json:"resumen"`
}

type AuxiliarDTETotalAmountsExtractor struct {
	Summary struct {
		TotalTaxed      float64 `json:"total_taxed"`
		TotalExempt     float64 `json:"total_exempt"`
		TotalNotSubject float64 `json:"total_non_subject"`
	} `json:"summary"`
}

type AuxiliarRelatedDocAndItemsExtractor struct {
	RelatedDocs []struct {
		GenerationType int    `json:"tipoGeneracion"`
		DocumentNumber string `json:"numeroDocumento"`
	} `json:"documentoRelacionado"`
	Items []struct {
		TaxedAmount      float64 `json:"ventaGravada"`
		ExemptAmount     float64 `json:"ventaExenta"`
		NotSubjectAmount float64 `json:"ventaNoSuj"`
		RelatedDoc       string  `json:"numeroDocumento"`
	} `json:"cuerpoDocumento"`
}

func ExtractAuxiliarIdentification(document interface{}) (AuxiliarIdentificationExtractor, error) {
	var identification AuxiliarIdentificationExtractor

	jsonData, err := json.Marshal(document)
	if err != nil {
		return identification, err
	}

	if err := json.Unmarshal(jsonData, &identification); err != nil {
		return identification, err
	}

	return identification, nil
}

func ExtractAuxiliarIdentificationFromStringJSON(document interface{}) (AuxiliarIdentificationExtractor, error) {
	var identification AuxiliarIdentificationExtractor

	jsonData := []byte(document.(string))

	if err := json.Unmarshal(jsonData, &identification); err != nil {
		return identification, err
	}

	return identification, nil
}

func ExtractSummaryTotalAmounts(document interface{}) (AuxiliarTotalAmountsExtractor, error) {
	var summary AuxiliarTotalAmountsExtractor

	jsonData, err := json.Marshal(document)
	if err != nil {
		return summary, err
	}

	if err = json.Unmarshal(jsonData, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func ExtractSummaryTotalAmountsFromStringJSON(document interface{}) (AuxiliarTotalAmountsExtractor, error) {
	var summary AuxiliarTotalAmountsExtractor

	jsonData := []byte(document.(string))

	if err := json.Unmarshal(jsonData, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func ExtractRelatedDocAndItemsFromStringJSON(document interface{}) AuxiliarRelatedDocAndItemsExtractor {
	var relatedDocAndItems AuxiliarRelatedDocAndItemsExtractor

	jsonData := []byte(document.(string))

	_ = json.Unmarshal(jsonData, &relatedDocAndItems)
	return relatedDocAndItems
}

func ExtractDTEReceiverFromString(document interface{}) (AuxiliarReceiverExtractor, error) {
	var receiver AuxiliarReceiverExtractor

	jsonData := []byte(document.(string))

	if err := json.Unmarshal(jsonData, &receiver); err != nil {
		return receiver, err
	}

	return receiver, nil
}
