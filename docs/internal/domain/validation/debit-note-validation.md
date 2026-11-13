# Validación de Nota de Débito (DTE 06)

> **Validador:** `DebitNoteRulesValidator`
> **Paquete:** `internal/domain/dte/debit_note/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

Todas se ejecutan y los errores de estrategias distintas se agrupan:

1. **DebitNoteItemStrategy** — Validación de ítems y reglas de impuestos
2. **DebitNoteTaxStrategy** — Validación de cálculos fiscales
3. **DebitNoteRelatedDocStrategy** — Validación de documentos relacionados

Luego se aplican las [estrategias comunes](./README.md). La `PaymentTotalStrategy` no aplica a las notas de débito.

---

## 1. DebitNoteItemStrategy

Mismas reglas que [`CreditNoteItemStrategy`](./credit-note-validation.md#1-creditnoteitemstrategy), con `DebitItems` en lugar de `CreditItems`:

| Validación | Regla | Error |
|---|---|---|
| Ítems | Al menos 1; máximo 2000 | `RequiredField`, `ExceededItemsLimit` |
| Tipos de venta | No mezclar (taxed, exempt, non-subject) | `MixedSalesTypesNotAllowed` |
| UnitPrice | No puede ser 0 cuando `TaxedSale > 0` | `InvalidUnitPriceZero` |
| Docs relacionados | Todo ítem debe tener `related_doc` | `MissingItemRelatedDoc` |
| Impuestos | Requeridos si `TaxedSale > 0`; códigos del catálogo soportado en ítems que no son tipo 4 | `MissingItemTaxes`, `InvalidTaxType` |
| Impuesto (tipo 4) | `taxes = ["20"]` y unidad de medida = `99` | `InvalidTaxRulesCCF`, `InvalidUnitMeasure` |

La venta del ítem no se contrasta contra `(UnitPrice * Quantity) - Discount`.

---

## 2. DebitNoteTaxStrategy

**Archivo:** `internal/domain/dte/debit_note/validator/strategy/debit_note_tax_strategy.go`

Orden interno: descuentos globales no admitidos, totales base, descuentos, IVA, percepción y totales. Se devuelve el primer error. Las reglas son las de [`CreditNoteTaxStrategy`](./credit-note-validation.md#2-creditnotetaxstrategy) con estas diferencias:

| Aspecto | Nota de Crédito | Nota de Débito |
|---|---|---|
| Descuentos globales y porcentaje | Rechazados (`GlobalDiscountNotAllowedInAdjustment`) | Rechazados (`GlobalDiscountNotAllowedInAdjustment`) |
| Descuentos negativos o mayores que su base | No evalúa (los globales son 0) | `NegativeDiscount`, `DiscountExceedsBase` |
| `TotalDiscount` | No se contrasta | `TotalDiscount = Sum(item.Discount)` (±0.01), error `InvalidTotalDiscountCalculation` |
| Subtotal | `SubTotalSales - descuentos` | `TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount` (equivalentes porque los descuentos son 0) |
| Múltiplos de 0.01 de percepción, total de operación y pagos | `InvalidMonetaryAmount` | No se ejecuta esta validación (los montos del resumen sí se leen con 2 decimales exactos, `InvalidDecimals`) |
| Valor de cada tributo del resumen | Se rechaza si difiere de la fórmula (`InvalidTaxCalculation`) | **No se rechaza**: la diferencia solo se registra en el log; sí se exige presencia de tributos y del tributo `20` |
| Documento nulo | Sin error propio | `NullDocument` |

Las reglas compartidas se mantienen: percepción `TotalTaxed * 0.01` (±0.01), `IncomeRetention = 0`, `TotalOperation = SubTotalSales + Sum(Taxes)` (±0.01) y `TotalToPay = TotalOperation + IVAPerception - IVARetention + TotalNonTaxed` (±0.01).

---

## 3. DebitNoteRelatedDocStrategy

Mismas reglas que [`CreditNoteRelatedDocStrategy`](./credit-note-validation.md#3-creditnoterelateddocstrategy): al menos un documento relacionado, máximo 50, tipos `03` (CCF) o `07` (CR) (`InvalidRelatedDocTypeForCreditNote`), todos los ítems referencian un documento relacionado existente en la lista, y plazo de 3 meses desde la generación del CCF o CR (estrategia común).

---

## Diferencias en el Servicio respecto a la Nota de Crédito

| Aspecto | Nota de Crédito | Nota de Débito |
|---|---|---|
| Pre-validación (`validateRelatedDocs`) | Existencia, estado `RECEIVED` y NIT del receptor | Igual |
| Validación de ítems y fiscal | Ver arriba | Ver arriba |
| Balance | `ValidateForCreditNote` (verifica que los totales no excedan el saldo) | `ValidateForDebitNote` (solo verifica que exista el control de saldo) |
| Total a pagar | Se conserva el valor validado | Se **recalcula** y reemplaza el valor enviado (ver abajo) |
| Transacción de saldo | Resta del saldo | Suma al saldo |

### Recalculo de `TotalToPay` en el servicio

`calculateTotalToPay()` se ejecuta **después** de las validaciones (el valor enviado por el cliente se valida primero con la fórmula de `ValidateAdjustmentTotals` y luego se reemplaza):

```
SubTotal = (TotalTaxed - TaxedDiscount) + (TotalExempt - ExemptDiscount) + (TotalNonSubject - NonSubjectDiscount)

TotalToPay = SubTotal
           + Sum(Taxes)
           + TotalNonTaxed                                        (si TotalNonTaxed > 0)
           + IVAPerception - IVARetention - IncomeRetention       (solo si TotalTaxed > 0)
```

Como `IncomeRetention` es 0, el resultado coincide con la fórmula validada, salvo que con `TotalTaxed = 0` la percepción y la retención de IVA no se incluyen en el valor guardado.

---

## Errores

Los mismos errores que la nota de crédito aplican, más:

| Error | Causa |
|---|---|
| `InvalidTotalDiscountCalculation` | `TotalDiscount` no coincide con la suma de descuentos de los ítems |
| `NegativeDiscount`, `DiscountExceedsBase` | Descuento global negativo o mayor que su base |
| `NullDocument` | Documento nulo en la estrategia de impuestos o de documentos relacionados |
