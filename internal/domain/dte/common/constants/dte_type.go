package constants

const (
	FacturaElectronica                = "01"
	CCFElectronico                    = "03"
	NotaRemisionElectronica           = "04"
	NotaCreditoElectronica            = "05"
	NotaDebitoElectronica             = "06"
	ComprobanteRetencionElectronico   = "07"
	ComprobanteLiquidacionElectronico = "08"
	DocContableLiquidacionElectronico = "09"
	FacturaExportacionElectronica     = "11"
	FacturaSujetoExcluidoElectronica  = "14"
	ComprobanteDonacionElectronico    = "15"

	NIT             = "36"
	DUI             = "13"
	CarnetResidente = "02"
	Pasaporte       = "03"
	OtroDocumento   = "37"
)

var (
	ValidDTETypes = map[string]bool{
		FacturaElectronica:                true,
		CCFElectronico:                    true,
		NotaRemisionElectronica:           true,
		NotaCreditoElectronica:            true,
		NotaDebitoElectronica:             true,
		ComprobanteRetencionElectronico:   true,
		ComprobanteLiquidacionElectronico: true,
		DocContableLiquidacionElectronico: true,
		FacturaExportacionElectronica:     true,
		FacturaSujetoExcluidoElectronica:  true,
		ComprobanteDonacionElectronico:    true,
	}

	ValidReceiverDTETypes = []string{
		NIT,
		DUI,
		CarnetResidente,
		Pasaporte,
		OtroDocumento,
	}

	ValidRetentionDTETypes = map[string]bool{
		FacturaElectronica:               true,
		CCFElectronico:                   true,
		FacturaSujetoExcluidoElectronica: true,
	}

	ValidAdjustmentDTETypes = map[string]bool{
		CCFElectronico:                  true,
		ComprobanteRetencionElectronico: true,
	}

	ValidCCFDTETypesRelateDoc = map[string]bool{
		NotaRemisionElectronica:           true,
		ComprobanteLiquidacionElectronico: true,
		DocContableLiquidacionElectronico: true,
	}

	ValidInvoiceDTETypesRelateDoc = map[string]bool{
		NotaRemisionElectronica:           true,
		DocContableLiquidacionElectronico: true,
	}

	ValidDTETypesForContingency = map[string]bool{
		FacturaElectronica:               true,
		CCFElectronico:                   true,
		NotaRemisionElectronica:          true,
		NotaCreditoElectronica:           true,
		NotaDebitoElectronica:            true,
		FacturaExportacionElectronica:    true,
		FacturaSujetoExcluidoElectronica: true,
	}
)

func ShowValidRelatedDocTypes(valids map[string]bool) string {
	var result string
	for k := range valids {
		result += k + ", "
	}
	if len(result) > 2 {
		result = result[:len(result)-2]
	}
	return result
}
