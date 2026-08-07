# Validación de Factura Electrónica (DTE 01)

> **Validador:** `InvoiceRulesValidator`
> **Paquete:** `internal/domain/dte/invoice/validator`

## Estrategias Compuestas

El `InvoiceRulesValidator` ejecuta las siguientes estrategias en orden:

1. **InvoiceItemsStrategy** — Validación de ítems y sus montos
2. **InvoiceTaxStrategy** — Validación de impuestos y cálculos fiscales
3. **InvoiceTotalsStrategy** — Validación de subtotales y totales

Además, se aplican las [estrategias comunes](./README.md) del sistema.

---

## 1. InvoiceItemsStrategy

**Archivo:** `internal/domain/dte/invoice/validator/invoice_item_strategy.go`

### Validaciones por Ítem

| Validación | Regla | Tolerancia |
|---|---|---|
| Máximo de ítems | <= 2000 | — |
| Precio unitario con venta gravada | Si `TaxedSale > 0` → `UnitPrice > 0` | — |
| IVA sin venta gravada | Si `IVAItem > 0` → `TaxedSale > 0` | — |
| Cálculo del IVA | `IVAItem ≈ (TaxedSale / 1.13) * 0.13` | ±0.01 |
| Total de ventas vs máximo | `NonSubjectSale + ExemptSale + TaxedSale <= UnitPrice * Quantity` | — |
| No gravado exclusivo | Si `NonTaxed > 0` → no puede tener otros tipos de venta | — |
| No gravado sin precio | Si `NonTaxed > 0` → `UnitPrice = 0` | — |
| Venta gravada consistente | Si `TaxedSale > 0` → `TaxedSale ≈ (UnitPrice * Quantity) - Discount` | ±0.01 |

### Reglas de Tipo de Ítem

| Tipo de Ítem | Restricción |
|---|---|
| Producto (1) | No puede tener impuesto IVA a nivel de ítem |
| Si hay IVA en resumen | No puede haber productos con impuesto |

### Exclusividad de Tipos de Venta

Cada ítem solo puede tener **un tipo de venta activo**:
- No gravado con otros → Error
- Solo un tipo de los tres (NonSubject, Exempt, Taxed)

### Validación de Totales vs Ítems

```
Sum(item.TaxedSale)      == Summary.TotalTaxed       (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)
```

---

## 2. InvoiceTaxStrategy

**Archivo:** `internal/domain/dte/invoice/validator/invoice_tax_strategy.go`

### Validación de Descuentos Base

```
Descuento no puede exceder el monto base
```

### Cálculo de IVA

Si `TotalTaxed > 0`, se valida cada impuesto en el resumen:

| Código | Fórmula | Tolerancia |
|---|---|---|
| `20` (IVA) | `baseTaxed * 0.13` | ±0.01 |
| `C3` (IVA Export) | `baseTaxed * 0.00` | ±0.01 |
| `59` (Tourism) | `baseTaxed * 0.01` | ±0.01 |
| `71` (Tourism Airport) | Monto fijo | — |
| `D1` (FOVIAL) | `baseTaxed * 0.005` | ±0.01 |
| `C8` (COTRANS) | Monto fijo | — |
| `D5` (Special) | Sin validación | — |

Donde `baseTaxed = TotalTaxed` para factura (el descuento no se aplica antes del IVA a diferencia del CCF).

### Validación de Montos Monetarios

Todos los montos monetarios deben tener **máximo 2 decimales**:
- `TotalOperation`
- `IVARetention`
- `IncomeRetention`
- `TotalToPay`
- `PaymentAmounts`

### Validación de Totales

| Fórmula | Tolerancia |
|---|---|
| `SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount` | ±0.01 |
| Si `TotalTaxed > 0` → debe existir IVA en impuestos | — |
| `IVARetention` solo si `TotalTaxed > 0` | — |
| `IncomeRetention` solo si `TotalTaxed > 0` | — |
| Si hay ítems con `NonTaxed` → resumen debe tener `TotalNonTaxed` | — |
| `TotalToPay = SubTotal + Taxes - IVARetention - IncomeRetention + NonTaxed` | ±0.01 |

### Validación de Documentos Relacionados en Ítems

Si existen documentos relacionados:
- Cada ítem **debe** tener referencia a un documento relacionado
- La referencia del ítem **debe** existir en la lista de documentos relacionados

---

## 3. InvoiceTotalsStrategy

**Archivo:** `internal/domain/dte/invoice/validator/invoice_total_strategy.go`

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount
         + TotalExempt - ExemptDiscount
         + TotalNonSubject - NonSubjectDiscount
```
Tolerancia: ±0.0001

### Validación de Descuentos

| Regla | Descripción |
|---|---|
| No negativos | Ningún descuento puede ser negativo |
| TaxedDiscount <= TotalTaxed | El descuento no excede el gravado |
| ExemptDiscount <= TotalExempt | El descuento no excede el exento |
| NonSubjectDiscount <= TotalNonSubject | El descuento no excede el no sujeto |

### Total Operación

```
TotalOperation = SubTotal + Sum(Taxes)
```
Tolerancia: ±0.0001

---

## Errores Específicos de Factura

| Error | Mensaje | Causa |
|---|---|---|
| `MissingItemUnitPrice` | Precio unitario requerido | `TaxedSale > 0` pero `UnitPrice == 0` |
| `InvalidIVAItemWithoutTaxedSale` | IVA sin venta gravada | `IVAItem > 0` pero `TaxedSale == 0` |
| `InvalidIVAItemCalculation` | Cálculo de IVA incorrecto | IVA no coincide con `(TaxedSale/1.13)*0.13` |
| `ExcessiveItemTotal` | Total excede precio | `NonSubject + Exempt + Taxed > UnitPrice * Qty` |
| `InvalidMixedSalesWithNonTaxed` | No gravado mezclado | NonTaxed activo con otros tipos |
| `InvalidTaxForProduct` | Impuesto en producto | Producto con impuesto IVA |
| `MixedSalesTypesNotAllowed` | Tipos mezclados | Más de un tipo de venta activo |
| `InvalidTaxedAmount` | Monto gravado incorrecto | TaxedSale no coincide con cálculo |
| `InvalidSubTotal` | Subtotal incorrecto | No coincide con fórmula |
| `NegativeDiscount` | Descuento negativo | Descuento < 0 |
| `DiscountExceedsBase` | Descuento excede base | Descuento > monto |
| `InvalidTotalOperation` | Total operación incorrecto | No coincide con SubTotal + Taxes |
| `InvalidTotalToPayCalculation` | Total a pagar incorrecto | No coincide con fórmula |
| `MissingIVAForTaxedAmount` | IVA faltante | Gravado > 0 pero sin IVA |

---

## Diagrama de Validación

```
ElectronicInvoice
  │
  ▼
InvoiceRulesValidator.Validate()
  │
  ├── [1] InvoiceItemsStrategy
  │         ├── Por cada ítem:
  │         │     ├── UnitPrice vs TaxedSale
  │         │     ├── IVA = (TaxedSale/1.13)*0.13
  │         │     ├── Exclusividad de tipo de venta
  │         │     ├── NonTaxed exclusivo
  │         │     └── TaxedSale = (UnitPrice*Qty) - Discount
  │         │
  │         └── Totales:
  │               ├── Sum(TaxedSale) == TotalTaxed
  │               ├── Sum(ExemptSale) == TotalExempt
  │               └── Sum(NonSubjectSale) == TotalNonSubject
  │
  ├── [2] InvoiceTaxStrategy
  │         ├── Descuentos no exceden base
  │         ├── IVA: baseTaxed * 0.13
  │         ├── Montos con max 2 decimales
  │         ├── SubTotal = formula
  │         ├── Retenciones solo con gravado
  │         └── TotalToPay = formula
  │
  └── [3] InvoiceTotalsStrategy
            ├── SubTotal = formula (±0.0001)
            ├── Descuentos >= 0
            ├── Descuentos <= base
            └── TotalOperation = SubTotal + Taxes
```
