package utils

import (
	"encoding/json"
	"math"
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
		NIT          string `json:"nit"`
		NumDocumento string `json:"numDocumento"`
	} `json:"receptor"`
}

// ReceiverID devuelve el NIT del receptor del documento relacionado. El CCF lo informa en "nit"; el CR
// identifica al receptor con tipo y numero de documento (36-NIT), por lo que se usa "numDocumento".
func (a AuxiliarReceiverExtractor) ReceiverID() string {
	if a.Receiver.NIT != "" {
		return a.Receiver.NIT
	}
	return a.Receiver.NumDocumento
}

type AuxiliarTotalAmountsExtractor struct {
	Summary struct {
		TotalTaxed      float64 `json:"totalGravada"`
		TotalExempt     float64 `json:"totalExenta"`
		TotalNotSubject float64 `json:"totalNoSuj"`

		DiscountNotSubject float64 `json:"descuNoSuj"`
		DiscountExempt     float64 `json:"descuExenta"`
		DiscountTaxed      float64 `json:"descuGravada"`
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

// NetTaxed returns the taxed total minus the global taxed discount, rounded to cents and never negative.
func (a AuxiliarTotalAmountsExtractor) NetTaxed() float64 {
	return netOfDiscount(a.Summary.TotalTaxed, a.Summary.DiscountTaxed)
}

// NetExempt returns the exempt total minus the global exempt discount, rounded to cents and never negative.
func (a AuxiliarTotalAmountsExtractor) NetExempt() float64 {
	return netOfDiscount(a.Summary.TotalExempt, a.Summary.DiscountExempt)
}

// NetNotSubject returns the not-subject total minus the global not-subject discount, rounded to cents and never negative.
func (a AuxiliarTotalAmountsExtractor) NetNotSubject() float64 {
	return netOfDiscount(a.Summary.TotalNotSubject, a.Summary.DiscountNotSubject)
}

func netOfDiscount(total, discount float64) float64 {
	return math.Max(0, math.Round((total-discount)*100)/100)
}
