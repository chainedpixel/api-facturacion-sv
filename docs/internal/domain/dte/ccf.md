# Comprobante de Crédito Fiscal Electrónico (DTE 03) - CCF Service

> **Código DTE:** `03`
> **Paquete:** `internal/domain/dte/ccf`
> **Servicio:** `creditFiscalService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

El Comprobante de Crédito Fiscal (CCF) es el documento tributario utilizado en transacciones entre contribuyentes registrados (B2B). A diferencia de la factura, el IVA se calcula **separado del precio** y el receptor **debe ser un contribuyente con NRC**.

---

## Estructura del Servicio

```go
type creditFiscalService struct {
    validator        *validator.CCFRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.CCFRulesValidator` | Se validan reglas de negocio específicas del CCF |
| `seqNumberManager` | `dte_documents.SequentialNumberManager` | Se gestiona la reserva de números de control |

### Constructor

```go
func NewCCFService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *creditFiscalService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → *CCFData
2. Crear documento base (DTEDocument) desde CCFData
3. Ensamblar CreditFiscalDocument con ítems y resumen específicos
4. Validar reglas específicas del CCF (CCFRulesValidator)
5. Validar reglas genéricas de DTE (ValidateDTEDocument)
6. Generar número de control (SequentialNumberManager) y código de generación (UUID)
7. Retornar CreditFiscalDocument validado
```

El flujo es idéntico al de factura en estructura, pero con reglas de validación y cálculos diferentes. Las fórmulas se documentan en [calculations.md](../../../api/calculations.md). El `CCFRulesValidator` compone cuatro estrategias: `CCFItemStrategy`, `CCFTaxStrategy`, `CCFReceiverStrategy` y `CCFRelatedDocStrategy`, y agrupa los errores en un error compuesto.

---

## Modelos Específicos del CCF

### CreditFiscalDocument

```go
type CreditFiscalDocument struct {
    *models.DTEDocument
    CreditItems   []CreditItem
    CreditSummary CreditSummary
}
```

### CCFData (Input)

```go
type CCFData struct {
    *models.InputDataCommon
    Items         []CreditItem
    CreditSummary *CreditSummary
}
```

### CreditItem

```go
type CreditItem struct {
    *models.Item
    NonSubjectSale financial.Amount  // Venta no sujeta
    ExemptSale     financial.Amount  // Venta exenta
    TaxedSale      financial.Amount  // Venta gravada
    SuggestedPrice financial.Amount  // Precio sugerido
    NonTaxed       financial.Amount  // Monto no gravado
}
```

> **Diferencia clave con Factura:** El CCF no tiene campo `IVAItem` por ítem. El IVA se calcula a nivel de resumen, no por ítem.

### CreditSummary

```go
type CreditSummary struct {
    *models.Summary
    TaxedDiscount           financial.Amount  // Descuento sobre gravado
    IVAPerception           financial.Amount  // Percepción de IVA
    IVARetention            financial.Amount  // Retención de IVA
    BalanceInFavor          financial.Amount  // Saldo a favor
    IncomeRetention         financial.Amount  // Retención de renta
    ElectronicPaymentNumber *string           // Número de pago electrónico
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `TaxedDiscount` | `Amount` | Descuento aplicado al monto gravado |
| `IVAPerception` | `Amount` | Percepción de IVA (1% del gravado, cuando aplica) |
| `IVARetention` | `Amount` | Retención de IVA |
| `BalanceInFavor` | `Amount` | Saldo a favor del contribuyente |
| `IncomeRetention` | `Amount` | Retención de renta |

---

## Diferencias Clave con Factura (DTE 01)

| Aspecto | Factura (01) | CCF (03) |
|---|---|---|
| **IVA** | Incluido en precio | Separado del precio, calculado en el resumen |
| **Receptor** | Requerido, NRC opcional | Requerido, **NRC obligatorio** |
| **Percepción** | No aplica | Campo `IVAPerception` (cuando aplica) |
| **Retención de renta** | No aplica | No permitida (`IncomeRetentionNotAllowed`) |
| **IVA por ítem** | Sí (`IVAItem`) | No (se calcula en resumen) |
| **Activity Code receptor** | Opcional | **Obligatorio** |
| **Activity Description receptor** | Opcional | **Obligatorio** |

---

## Cálculos

Las fórmulas del CCF (IVA sobre la base gravada menos el descuento, percepción de IVA, subtotales, total de la operación y total a pagar) y la tolerancia aplicada se documentan en [calculations.md](../../../api/calculations.md). Los porcentajes y montos fijos por código de impuesto están en [constants.md](../constants.md).

Reglas del resumen que valida `CCFTaxStrategy`:

- Los totales del resumen (gravado, exento, no sujeto) deben coincidir con la suma de los ítems.
- Cada descuento no puede exceder su base.
- Si el gravado es mayor que cero debe existir IVA (código `20`); si es cero no pueden existir impuestos en el resumen.
- Si `IVAPerception` es distinto de cero, debe corresponder al 1% del gravado.
- `IncomeRetention` debe ser cero: la retención de renta no se admite en el CCF.
- El total a pagar suma el monto total de la operación, `IVAPerception` y `TotalNonTaxed`, y resta `IVARetention`; si el resultado es negativo se espera `0.00`.
- `TotalNonTaxed` del resumen debe igualar la suma de `NonTaxed` de los ítems.

---

## Reglas de Validación del Receptor

El receptor en un CCF debe cumplir:

1. **NRC obligatorio** — El receptor debe tener Número de Registro de Contribuyente
2. **ActivityCode obligatorio** — Se requiere el código de actividad económica
3. **ActivityDescription obligatorio** — Se requiere la descripción de la actividad

Estas validaciones se ejecutan en `CCFReceiverStrategy`.

---

## Reglas de Ítems

### Exclusividad de Tipos de Venta

Cada ítem solo puede tener **un tipo de venta activo**:

- `ExemptSale > 0` → No puede tener `TaxedSale`, `NonSubjectSale` ni `NonTaxed`, ni impuestos
- `NonSubjectSale > 0` → No puede tener `TaxedSale`, `ExemptSale` ni `NonTaxed`, ni impuestos
- `NonTaxed > 0` → No puede tener otro tipo de venta ni impuestos

### Reglas de No Gravado

- Si `NonTaxed > 0`: no se puede mezclar con otros tipos de venta, no puede tener impuestos y el precio unitario debe ser `0` (`InvalidUnitPriceWithNonTaxed`)
- Con `TaxedSale > 0`, el precio unitario no puede ser `0` (`InvalidUnitPriceZero`)
- La venta del ítem (suma de no sujeta, exenta y gravada) debe corresponder a `(precio unitario x cantidad) - descuento`

### Reglas de Impuestos por Tipo de Ítem

- **Producto (tipo 1):** Solo se permite IVA (código `20`)
- **Impuesto (tipo 4):** Debe tener exactamente un impuesto, IVA (`InvalidTaxRulesCCF`), y unidad de medida `99`
- Si `TaxedSale > 0`: debe tener impuestos asignados (`MissingTaxesItem`)

---

## Documentos Relacionados

Los documentos relacionados válidos para CCF son:

- `04` — Nota de Remisión Electrónica
- `08` — Comprobante de Liquidación Electrónico
- `09` — Documento Contable de Liquidación Electrónico

Se admiten hasta 50 documentos relacionados (`ExceededRelatedDocsLimit`). Si existen, cada ítem debe referenciar uno (`MissingItemRelatedDoc`) que exista en la lista del CCF (`InvalidItemRelatedDoc`) y cada documento debe estar referenciado por al menos un ítem (`UnreferencedRelatedDoc`); sin documentos relacionados ningún ítem puede llevar referencia. Un tipo no permitido produce `InvalidRelatedDocDTEType`.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Pánico por cast (cast sin comprobación) | El `interface{}` no es `*CCFData` |
| `MissingNRC` | Receptor sin NRC |
| `RequiredField` | ActivityCode o ActivityDescription del receptor faltante |
| `InvalidTaxCodeOnly20` | Producto con impuesto diferente a IVA |
| `InvalidMixedSalesWithNonTaxed` | Ítem con NonTaxed mezclado con otros tipos de venta |
| `InvalidTotalToPayCalculation` | Total a pagar incorrecto |
| `IncomeRetentionNotAllowed` | Retención de renta distinta de cero |
| `InvalidPerceptionAmount` | Percepción de IVA distinta del 1% del gravado |
| `InvalidTaxCalculation` | Impuesto del resumen distinto del esperado |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/ccf/credit_fiscal_service.go` | Servicio principal |
| `internal/domain/dte/ccf/ccf_models/` | Modelos del CCF |
| `internal/domain/dte/ccf/validator/` | Estrategias de validación |

---

## Notas

1. **IVA separado**: Es la diferencia fundamental con factura. El IVA se calcula sobre `(TotalTaxed - TaxedDiscount)`, no sobre cada ítem individual.
2. **Percepción**: Es un cargo adicional del 1% del gravado que se aplica en ciertos casos. Se suma al total a pagar.
3. **Receptor estricto**: Siempre se verifica NRC, código y descripción de actividad económica.
4. **Documentos relacionados**: Se restringen a tipos específicos (`04`, `08`, `09`) a diferencia de factura.
