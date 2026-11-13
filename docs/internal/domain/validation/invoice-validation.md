# Validación de Factura Electrónica (DTE 01)

> **Validador:** `InvoiceRulesValidator`
> **Paquete:** `internal/domain/dte/invoice/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

El `InvoiceRulesValidator` ejecuta las siguientes estrategias (todas se ejecutan; los errores de estrategias distintas se agrupan):

1. **InvoiceItemsStrategy** — Validación de ítems y sus montos
2. **InvoiceTaxStrategy** — Validación de impuestos y cálculos fiscales
3. **InvoiceTotalsStrategy** — Validación de subtotales y totales

Luego se aplican las [estrategias comunes](./README.md) del sistema. La estrategia `InvoiceRelatedDocStrategy` existe en el paquete pero **no está registrada** en el validador, por lo que los tipos de documento relacionado de la FE solo se validan con la estrategia común.

---

## 1. InvoiceItemsStrategy

**Archivo:** `internal/domain/dte/invoice/validator/strategy/invoice_item_strategy.go`

### Validaciones por Ítem

Cada ítem pasa por la exclusividad de tipos de venta y luego por las reglas de la tabla.

| Validación | Regla | Tolerancia | Error |
|---|---|---|---|
| Máximo de ítems | <= 2000 | — | `ExceededItemsLimit` |
| Exclusividad de ventas | Un solo tipo activo entre `NonSubjectSale`, `ExemptSale`, `TaxedSale` y `NonTaxed` | — | `MixedSalesTypesNotAllowed`, `InvalidMixedSalesWithNonTaxed` |
| Precio unitario con venta gravada | Si `TaxedSale > 0` → `UnitPrice > 0` | — | `MissingItemUnitPrice` |
| IVA sin venta gravada | Si `IVAItem > 0` → `TaxedSale > 0` | — | `InvalidIVAItemWithoutTaxedSale` |
| Cálculo del IVA del ítem | `IVAItem ≈ (TaxedSale / 1.13) * 0.13` (campo 102) | ±0.01 | `InvalidIVAItemCalculation` |
| Total de ventas vs máximo | `NonSubjectSale + ExemptSale + TaxedSale <= UnitPrice * Quantity` | — | `ExcessiveItemTotal` |
| No gravado sin otras ventas | Si `NonTaxed != 0` → sin otros tipos de venta y `UnitPrice = 0` | — | `InvalidMixedSalesWithNonTaxed`, `InvalidUnitPriceForNonTaxed` |
| Venta del ítem | `NonSubjectSale + ExemptSale + TaxedSale ≈ (UnitPrice * Quantity) - Discount` (se omite si `UnitPrice = 0`) | ±0.01 | `InvalidItemSaleAmount` |
| Tributos del ítem | `taxes` no puede contener `20` ni `C3` (el IVA va incluido en el precio y se informa en `iva_item`) | — | `TaxNotAllowedInInvoice` |

### Validación de Totales vs Ítems

```
Sum(item.TaxedSale)      == Summary.TotalTaxed        (±0.01)   InvalidTotalTaxed
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)   InvalidTotalExempt
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)   InvalidTotalNonSubject
```

---

## 2. InvoiceTaxStrategy

**Archivo:** `internal/domain/dte/invoice/validator/strategy/invoice_tax_strategy.go`

Orden interno: base de totales, IVA y tributos, montos monetarios, totales; luego los ítems con documentos relacionados.

### Descuentos Base

`TaxedDiscount`, `ExemptDiscount` y `NonSubjectDiscount` no pueden ser mayores que `SubTotal` (`DiscountExceedsSubtotal`).

### Tributos del Resumen

Si `TotalTaxed = 0` y existen tributos → `InvalidTaxes`. Si `TotalTaxed > 0`, cada tributo del resumen se valida con la fórmula del código (ver [Cálculos, sección 3.6](../../../api/calculations.md#36-tributos-del-resumen-campos-136-a-138); `InvalidTaxCalculation`), con base `TotalTaxed` (sin restar el descuento).

En FE los tributos del resumen **no** incluyen `20` ni `C3` (`TaxNotAllowedInInvoiceSummary`, en la estrategia de totales) porque el IVA va incluido en el precio.

### IVA del Resumen (campo 157)

```
TotalIva = Sum(item.IVAItem) - (TaxedDiscount / 1.13) * 0.13     (±0.01)   InvalidTotalIVA
```

Si `TotalTaxed > 0` y `TotalIva <= 0` → `MissingIVAForTaxedAmount`.

### Montos Monetarios

`TotalOperation`, `IVARetention`, `IncomeRetention`, `TotalToPay` y cada `PaymentAmount` deben ser múltiplos exactos de 0.01 (`InvalidMonetaryAmount`). Además, todos los montos del resumen se leen con 2 decimales exactos (`InvalidDecimals`).

### Totales

| Fórmula | Tolerancia | Error |
|---|---|---|
| `SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount` | ±0.01 | `InvalidSubTotalCalculation` |
| `IncomeRetention` debe ser `0` (eliminada por la Normativa v2.0) | — | `IncomeRetentionNotAllowed` |
| Si `IVARetention > 0` → `TotalTaxed > 0` (el monto de la retención no se recalcula) | — | `InvalidIVARetentionWithoutTaxedAmount` |
| `Sum(item.NonTaxed)` y `TotalNonTaxed`: ambos cero o ambos distintos de cero | — | `InvalidNonTaxedAmount` |
| `Sum(item.NonTaxed) ≈ TotalNonTaxed` | ±0.01 | `InvalidNonTaxedAmountCalculation` |
| `TotalToPay = TotalOperation - IVARetention + TotalNonTaxed` (si el resultado es negativo se espera `0`) | ±0.01 | `InvalidTotalToPayCalculation` |

### Documentos Relacionados en Ítems

Si existen documentos relacionados:
- Cada ítem **debe** tener referencia a un documento relacionado (`MissingItemRelatedDoc`)
- La referencia del ítem **debe** existir en la lista de documentos relacionados (`InvalidItemRelatedDoc`)

---

## 3. InvoiceTotalsStrategy

**Archivo:** `internal/domain/dte/invoice/validator/strategy/invoice_total_strategy.go`

Se ejecutan en orden: tributos del resumen, suma de operaciones, subtotal, descuentos y total de la operación.

| Regla | Fórmula | Tolerancia | Error |
|---|---|---|---|
| Tributos del resumen sin `20` ni `C3` | — | — | `TaxNotAllowedInInvoiceSummary` |
| Suma de operaciones (campo 129) | `SubTotalSales = TotalNonSubject + TotalExempt + TotalTaxed` | ±0.01 | `InvalidSubTotalSales` |
| Subtotal (campo 139) | `SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount` | ±0.01 | `InvalidSubTotal` |
| Descuentos no negativos | `TotalDiscount`, `TaxedDiscount`, `ExemptDiscount`, `NonSubjectDiscount` >= 0 | — | `NegativeDiscount` |
| Descuentos acotados | Cada descuento global <= su total (`TotalTaxed`, `TotalExempt`, `TotalNonSubject`) | — | `DiscountExceedsBase` |
| Monto total de la operación (campo 150) | `TotalOperation = SubTotal + Sum(Taxes)` | ±0.01 | `InvalidTotalOperation` |

`TotalDiscount` no se contrasta contra la suma de descuentos (solo se valida que no sea negativo).

---

## Errores Específicos de Factura

Los mensajes están en `assets/i18n/es.yaml`.

| Error | Causa |
|---|---|
| `MissingItemUnitPrice` | `TaxedSale > 0` pero `UnitPrice == 0` |
| `InvalidIVAItemWithoutTaxedSale` | `IVAItem > 0` pero `TaxedSale == 0` |
| `InvalidIVAItemCalculation` | IVA del ítem no coincide con `(TaxedSale/1.13)*0.13` |
| `ExcessiveItemTotal` | `NonSubject + Exempt + Taxed > UnitPrice * Qty` |
| `InvalidItemSaleAmount` | La venta del ítem no coincide con `(UnitPrice * Qty) - Discount` |
| `InvalidMixedSalesWithNonTaxed` | `NonTaxed` mezclado con otros tipos de venta |
| `InvalidUnitPriceForNonTaxed` | `NonTaxed` con precio unitario distinto de 0 |
| `MixedSalesTypesNotAllowed` | Más de un tipo de venta activo en el ítem |
| `TaxNotAllowedInInvoice` | Tributo `20` o `C3` en un ítem |
| `TaxNotAllowedInInvoiceSummary` | Tributo `20` o `C3` en el resumen |
| `InvalidTotalTaxed`, `InvalidTotalExempt`, `InvalidTotalNonSubject` | Totales del resumen no coinciden con la suma de los ítems |
| `InvalidSubTotalSales` | Suma de operaciones incorrecta |
| `InvalidSubTotalCalculation`, `InvalidSubTotal` | Subtotal incorrecto (dos estrategias) |
| `InvalidTaxes`, `InvalidTaxCalculation` | Tributos presentes sin venta gravada; valor del tributo incorrecto |
| `InvalidTotalIVA`, `MissingIVAForTaxedAmount` | `TotalIva` incorrecto o ausente |
| `NegativeDiscount`, `DiscountExceedsBase`, `DiscountExceedsSubtotal` | Descuento negativo o mayor que su base |
| `InvalidTotalOperation` | `TotalOperation` incorrecto |
| `InvalidTotalToPayCalculation` | `TotalToPay` incorrecto |
| `IncomeRetentionNotAllowed` | `IncomeRetention` distinto de 0 |
| `InvalidIVARetentionWithoutTaxedAmount` | `IVARetention > 0` sin venta gravada |
| `InvalidNonTaxedAmount`, `InvalidNonTaxedAmountCalculation` | `TotalNonTaxed` no coincide con los ítems |
| `InvalidMonetaryAmount` | Monto con más de 2 decimales |
| `MissingItemRelatedDoc`, `InvalidItemRelatedDoc` | Referencia a documento relacionado ausente o inexistente |

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
  │         │     ├── Exclusividad de tipo de venta (incluye NonTaxed)
  │         │     ├── UnitPrice vs TaxedSale
  │         │     ├── IVAItem = (TaxedSale/1.13)*0.13
  │         │     ├── Venta = (UnitPrice*Qty) - Discount
  │         │     └── Sin tributos 20 ni C3
  │         │
  │         └── Totales:
  │               ├── Sum(TaxedSale) == TotalTaxed
  │               ├── Sum(ExemptSale) == TotalExempt
  │               └── Sum(NonSubjectSale) == TotalNonSubject
  │
  ├── [2] InvoiceTaxStrategy
  │         ├── Descuentos no exceden SubTotal
  │         ├── Tributos del resumen
  │         ├── TotalIva = Sum(IVAItem) - IVA del descuento
  │         ├── Montos con 2 decimales
  │         ├── SubTotal, retenciones, no gravado
  │         ├── TotalToPay = formula
  │         └── Documentos relacionados en ítems
  │
  └── [3] InvoiceTotalsStrategy
            ├── Sin tributos 20 ni C3 en el resumen
            ├── SubTotalSales y SubTotal
            ├── Descuentos >= 0 y <= base
            └── TotalOperation = SubTotal + Taxes
```
