package retention_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"

// ExpectedTotalIVA calcula el IVA 13% del resumen de un CRE (Anexo IV, campo 157).
//   - Regla 157.4: con codigo de retencion C4 es "monto sujeto a retencion x 13%".
//   - Comprobado contra Hacienda (ambiente 00, 2026-11-04): con documentos fisicos Hacienda rechaza ese valor
//     ("CALCULO INCORRECTO") y acepta 0.00, tanto con C4 como con 22.
//
// Por eso la formula 157.4 se aplica solo cuando todos los documentos son electronicos (VERIFICAR: no se pudo
// probar porque requiere DTE recibidos de un proveedor); en los demas casos el valor es 0.00.
func ExpectedTotalIVA(totalSubjectRetention float64, retentionCode string, allElectronic bool) float64 {
	if retentionCode == constants.RetentionThirteenPercent && allElectronic {
		return totalSubjectRetention * 0.13
	}
	return 0
}
