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
1. Cast del input → CCFData
2. Crear documento base (DTEDocument) desde CCFData
3. Ensamblar CreditFiscalDocument con ítems y resumen específicos
4. Validar reglas de negocio del CCF
5. Generar código de generación (UUID) y número de control
6. Retornar CreditFiscalDocument validado
```

El flujo es idéntico al de factura en estructura, pero con reglas de validación y cálculos diferentes (ver sección de cálculos).

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
| **IVA** | Incluido en precio (`TaxedSale / 1.13 * 0.13`) | Separado del precio (`(TotalTaxed - TaxedDiscount) * 0.13`) |
| **Receptor** | Requerido, NRC opcional | Requerido, **NRC obligatorio** |
| **Percepción** | No aplica | `IVAPerception = TotalTaxed * 0.01` (cuando aplica) |
| **IVA por ítem** | Sí (`IVAItem`) | No (se calcula en resumen) |
| **Activity Code receptor** | Opcional | **Obligatorio** |
| **Activity Description receptor** | Opcional | **Obligatorio** |

---

## Fórmulas de Cálculo

### IVA del CCF (a nivel de resumen)

```
IVA = (TotalTaxed - TaxedDiscount) * 0.13
```

El descuento se aplica **antes** de calcular el IVA. Tolerancia: `±0.01`

### Percepción de IVA

```
IVAPerception = TotalTaxed * 0.01    (cuando aplica, tolerancia ±0.01)
```

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount
```

### Total a Pagar

```
TotalToPay = SubTotal + IVAPerception - IVARetention - IncomeRetention + TotalNonTaxed
```

### Cálculos de Impuestos por Código

| Código | Nombre | Fórmula |
|---|---|---|
| `20` (IVA) | Impuesto al Valor Agregado | `(TotalTaxed - TaxedDiscount) * 0.13` |
| `C3` (IVA Export) | IVA Exportación | `TotalTaxed * 0.00` |
| `59` (Tourism) | Turismo | `TotalTaxed * 0.01` |
| `71` (Tourism Airport) | Turismo Aeropuerto | Monto fijo |
| `D1` (FOVIAL) | FOVIAL | `TotalTaxed * 0.005` |
| `C8` (COTRANS) | COTRANS | Monto fijo |

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

- `TaxedSale > 0` → No puede tener `ExemptSale` ni `NonSubjectSale`
- `ExemptSale > 0` → No puede tener `TaxedSale` ni `NonSubjectSale`
- `NonSubjectSale > 0` → No puede tener `TaxedSale` ni `ExemptSale`

### Reglas de No Gravado

- Si `NonTaxed > 0`: no se puede mezclar con otros tipos de venta
- Si `NonTaxed > 0` y `UnitPrice == 0`: `UnitPrice` debe ser igual a `TaxedSale`

### Reglas de Impuestos por Tipo de Ítem

- **Producto (tipo 1):** Solo se permite IVA (código `20`)
- **Impuesto (tipo 4):** Solo se permite IVA, unidad de medida debe ser `99`
- Si `TaxedSale > 0`: debe tener impuestos asignados

---

## Documentos Relacionados

Los documentos relacionados válidos para CCF son:

- `04` — Nota de Remisión Electrónica
- `08` — Comprobante de Liquidación Electrónico
- `09` — Documento Contable de Liquidación Electrónico

Si existen documentos relacionados, cada ítem debe referenciar uno y la referencia debe existir en la lista de documentos del CCF.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido | El `interface{}` no es `*CCFData` |
| `MissingNRC` | Receptor sin NRC |
| `RequiredField` | ActivityCode o ActivityDescription del receptor faltante |
| `InvalidTaxCodeOnly20` | Producto con impuesto diferente a IVA |
| `InvalidMixedSalesWithNonTaxed` | Ítem con NonTaxed mezclado con otros tipos de venta |
| `InvalidTotalToPayNonTaxedCCF` | Total a pagar incorrecto con no gravado |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/ccf/credit_fiscal_service.go` | Servicio principal |
| `internal/domain/dte/ccf/models/` | Modelos del CCF |
| `internal/domain/dte/ccf/validator/` | Estrategias de validación |

---

## Notas

1. **IVA separado**: Es la diferencia fundamental con factura. El IVA se calcula sobre `(TotalTaxed - TaxedDiscount)`, no sobre cada ítem individual.
2. **Percepción**: Es un cargo adicional del 1% que se aplica en ciertos casos. Se suma al total a pagar.
3. **Receptor estricto**: Siempre se verifica NRC, código y descripción de actividad económica.
4. **Documentos relacionados**: Se restringen a tipos específicos (`04`, `08`, `09`) a diferencia de factura.
