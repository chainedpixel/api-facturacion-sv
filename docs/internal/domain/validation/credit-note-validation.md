# Validación de Nota de Crédito (DTE 05)

> **Validador:** `CreditNoteRulesValidator`
> **Paquete:** `internal/domain/dte/credit_note/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

Todas se ejecutan y los errores de estrategias distintas se agrupan:

1. **CreditNoteItemStrategy** — Validación de ítems y reglas de impuestos
2. **CreditNoteTaxStrategy** — Validación de cálculos fiscales
3. **CreditNoteRelatedDocStrategy** — Validación de documentos relacionados

Luego se aplican las [estrategias comunes](./README.md). La `PaymentTotalStrategy` no aplica a las notas de crédito.

---

## 1. CreditNoteItemStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/strategy/credit_note_item_strategy.go`

### Reglas por Ítem

| Validación | Regla | Error |
|---|---|---|
| Ítems | Al menos 1 | `RequiredField` |
| Máximo ítems | 2000 | `ExceededItemsLimit` |
| Tipos de venta | No mezclar (taxed, exempt, non-subject) | `MixedSalesTypesNotAllowed` |
| UnitPrice | No puede ser 0 cuando `TaxedSale > 0` | `InvalidUnitPriceZero` |
| Docs relacionados | Todo ítem debe tener `related_doc` (aunque el documento no traiga lista) | `MissingItemRelatedDoc` |
| Impuestos | Requeridos si `TaxedSale > 0` | `MissingItemTaxes` |
| Código de impuesto | Para ítems que no son tipo 4, cada código debe estar en el catálogo soportado | `InvalidTaxType` |

La venta del ítem **no** se contrasta contra `(UnitPrice * Quantity) - Discount` en NC (a diferencia de FE, CCF y FSE).

### Reglas del Tipo de Ítem 4 (Impuesto)

| Regla | Error |
|---|---|
| Unidad de medida = `99` | `InvalidUnitMeasure` |
| `taxes` debe ser exactamente `["20"]` | `InvalidTaxRulesCCF` |

---

## 2. CreditNoteTaxStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/strategy/credit_note_tax_strategy.go`

Orden interno: descuentos globales, totales base, IVA, percepción, montos monetarios y totales. Se devuelve el primer error.

### Descuentos Globales (no admitidos)

Las notas de crédito y débito no tienen descuentos globales en el esquema (Anexo VI de la Normativa v2.0). `NonSubjectDiscount`, `ExemptDiscount`, `TaxedDiscount` y `DiscountPercentage` deben ser `0`; el descuento se aplica en cada ítem. Error: `GlobalDiscountNotAllowedInAdjustment` (parámetro: nombre del campo). Esta es la primera verificación de la estrategia.

### Totales Base

```
Sum(item.TaxedSale)      == Summary.TotalTaxed        (±0.01)   InvalidTotalTaxed
Sum(item.ExemptSale)     == Summary.TotalExempt       (exacto)  InvalidTotalExempt
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (exacto)  InvalidTotalNonSubject
SubTotalSales = TotalTaxed + TotalExempt + TotalNonSubject      (±0.01)   InvalidSubTotalSales
```

### Tributos del Resumen

Si `TotalTaxed = 0` y existen tributos → `InvalidTaxes`. Si `TotalTaxed > 0` y no hay tributos → `MissingTaxes`.

```
IVA (20) = TotalTaxed * 0.13    (±0.01)    InvalidTaxCalculation
```

Con `TotalTaxed > 0` debe existir el tributo `20` con valor > 0 (`MissingIVAForTaxedAmount`). Las tasas de los otros tributos están en [Cálculos, sección 3.6](../../../api/calculations.md#36-tributos-del-resumen-campos-136-a-138).

### Percepción

```
Si IVAPerception != 0:
    IVAPerception = TotalTaxed * 0.01    (±0.01)    InvalidPerceptionAmount
```

La retención de IVA (`IVARetention`) no se recalcula; solo interviene en el total a pagar.

### Montos Monetarios

`IVAPerception`, `TotalOperation` y cada monto de pago deben ser múltiplos de 0.01 (`InvalidMonetaryAmount`).

### SubTotal

```
SubTotal = SubTotalSales - TaxedDiscount - ExemptDiscount - NonSubjectDiscount    (±0.01)    InvalidSubTotalCalculation
```

Como los descuentos son 0, `SubTotal = SubTotalSales`.

### Totales de NC y ND (`ValidateAdjustmentTotals`, regla compartida)

| Regla | Fórmula | Error |
|---|---|---|
| Retención de renta | `IncomeRetention` debe ser `0` (eliminada en NCE y NDE) | `IncomeRetentionNotAllowed` |
| Monto total de la operación (campo 150) | `TotalOperation = SubTotalSales + Sum(Taxes)` (±0.01) | `InvalidTotalOperation` |
| Total a pagar (campo 153) | `TotalToPay = TotalOperation + IVAPerception - IVARetention + TotalNonTaxed` (±0.01) | `InvalidTotalToPayCalculation` |

`TotalDiscount` no se contrasta contra la suma de descuentos por ítem en NC. `TotalNonTaxed` no se contrasta contra ningún ítem (los ítems de NC no tienen `NonTaxed`).

---

## 3. CreditNoteRelatedDocStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/strategy/credit_note_related_doc_strategy.go`

| Validación | Regla | Error |
|---|---|---|
| Documentos relacionados | Al menos uno | `RequiredField` |
| Máximo docs | 50 | `ExceededRelatedDocsLimit` |
| Tipos válidos | CCF (`03`) o Comprobante de Retención (`07`) | `InvalidRelatedDocTypeForCreditNote` |
| Ítems | Todos deben referenciar un doc relacionado | `MissingItemRelatedDoc` |
| Referencias | Deben existir en la lista del documento | `InvalidItemRelatedDoc` |

La estrategia común `RelatedDocsStrategy` aplica además el plazo de **3 meses** desde la generación del CCF o CR, calculado con `calendar.AddMonthsClamped` (tope en el último día del mes: 30 de noviembre + 3 meses = 28 de febrero) (`RelatedDocOutOfPeriod`); ver [README, RelatedDocsStrategy](./README.md#relateddocsstrategy).

---

## Validación Adicional en el Servicio

Además de las estrategias del validador, el `creditNoteService` ejecuta:

### `validateRelatedDocs()` — Pre-validación (antes del validador)

| Verificación | Descripción | Error |
|---|---|---|
| Existencia | Al menos un documento relacionado | `NoRelatedDocs` |
| Existencia en la base | Cada documento relacionado debe estar registrado | `FailedToGetDTE` |
| Estado | Documento original en estado `RECEIVED` | `RelatedDocumentNotReceived` |
| NIT | El NIT del receptor coincide con el del documento original | `NotMatchingReceiverNIT` |
| Fecha | Se toma la fecha de generación del documento original para el plazo de 3 meses | — |

### `ValidateForCreditNote()` — Validación de saldo (después de las validaciones, solo para relacionados electrónicos)

| Verificación | Descripción | Error |
|---|---|---|
| Saldo disponible | `TotalTaxed`, `TotalExempt` y `TotalNonSubject` de la NC no exceden el saldo restante del documento original | `InvalidCreditNoteTransaction` |

Ver [Cálculos, sección 3.12](../../../api/calculations.md#312-saldo-del-documento-relacionado-en-nc-y-nd).

---

## Errores Específicos

| Error | Causa |
|---|---|
| `NoRelatedDocs` | Sin documentos relacionados |
| `RelatedDocumentNotReceived` | Documento original no recibido por Hacienda |
| `NotMatchingReceiverNIT` | NIT no coincide |
| `InvalidCreditNoteTransaction` | Montos exceden saldo |
| `InvalidRelatedDocTypeForCreditNote` | Tipo de documento relacionado distinto de `03` o `07` |
| `GlobalDiscountNotAllowedInAdjustment` | Descuento global o porcentaje distinto de 0 |
| `InvalidUnitPriceZero` | Precio unitario 0 con gravado |
| `MixedSalesTypesNotAllowed` | Tipos de venta mezclados |
| `MissingItemRelatedDoc` | Ítem sin doc relacionado |
| `MissingItemTaxes` | Gravado sin impuestos |
| `InvalidTaxType` | Tipo de impuesto no permitido |
| `InvalidUnitMeasure`, `InvalidTaxRulesCCF` | Ítem tipo 4 con unidad distinta de 99 o impuestos distintos de `20` |
| `InvalidTotalTaxed`, `InvalidTotalExempt`, `InvalidTotalNonSubject`, `InvalidSubTotalSales` | Totales del resumen no coinciden con los ítems |
| `InvalidTaxes`, `MissingTaxes`, `MissingIVAForTaxedAmount`, `InvalidTaxCalculation` | Tributos del resumen |
| `InvalidPerceptionAmount` | Percepción distinta de 1% de `TotalTaxed` |
| `InvalidSubTotalCalculation`, `InvalidTotalOperation`, `InvalidTotalToPayCalculation` | Subtotal, monto total o total a pagar incorrectos |
| `IncomeRetentionNotAllowed` | `IncomeRetention` distinto de 0 |
