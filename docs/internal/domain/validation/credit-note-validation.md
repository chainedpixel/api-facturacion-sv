# Validación de Nota de Crédito (DTE 05)

> **Validador:** `CreditNoteRulesValidator`
> **Paquete:** `internal/domain/dte/credit_note/validator`

## Estrategias Compuestas

1. **CreditNoteItemStrategy** — Validación de ítems y reglas de impuestos
2. **CreditNoteTaxStrategy** — Validación de cálculos fiscales
3. **CreditNoteRelatedDocStrategy** — Validación de documentos relacionados

---

## 1. CreditNoteItemStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/credit_note_item_strategy.go`

### Reglas por Ítem

| Validación | Regla |
|---|---|
| Máximo ítems | 2000 |
| Tipos de venta | No mezclar (taxed, exempt, non-subject) |
| UnitPrice | No puede ser 0 cuando `TaxedSale > 0` |
| Docs relacionados | Requeridos para todos los ítems |
| Impuestos | Requeridos si `TaxedSale > 0` |

### Reglas de Impuestos por Tipo de Ítem

| Tipo | Código | Restricción |
|---|---|---|
| Producto | 1 | Solo IVA (código `20`) |
| Servicio | 2 | Cualquier impuesto |
| Producto y Servicio | 3 | Cualquier impuesto |
| Impuesto | 4 | Solo IVA, unidad de medida = `99` |

---

## 2. CreditNoteTaxStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/credit_note_tax_strategy.go`

Sigue las mismas reglas que el CCF con ajustes para notas de crédito:

### Totales Base

```
Sum(item.TaxedSale)      == Summary.TotalTaxed       (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)
```

### IVA

```
IVA = (TotalTaxed - TaxedDiscount) * 0.13    (±0.01)
```

### Percepción

```
IVAPerception = TotalTaxed * 0.01    (si aplica, ±0.01)
```

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount
```

### Total Operación

```
TotalOperation = SubTotal + Sum(Taxes)    (±0.01)
```

### Montos Monetarios

Máximo 2 decimales para todos los montos.

---

## 3. CreditNoteRelatedDocStrategy

**Archivo:** `internal/domain/dte/credit_note/validator/credit_note_related_doc_strategy.go`

| Validación | Regla |
|---|---|
| Máximo docs | 50 |
| Tipos válidos | DTEs de ajuste válidos (CCF, Retención) |
| Ítems | Todos deben referenciar un doc relacionado |
| Referencias | Deben existir en la lista del documento |

---

## Validación Adicional en el Servicio

Además de las estrategias del validador, el `creditNoteService` ejecuta:

### `validateRelatedDocs()` — Pre-validación

| Verificación | Descripción |
|---|---|
| Existencia | Al menos un documento relacionado |
| Estado | Documento original en estado `RECEIVED` |
| NIT | NIT del receptor coincide con el original |

### `ValidateForCreditNote()` — Validación de Balance (post-transmisión)

| Verificación | Descripción |
|---|---|
| Saldo disponible | Montos no exceden el saldo disponible del documento original |

---

## Errores Específicos

| Error | Causa |
|---|---|
| `NoRelatedDocs` | Sin documentos relacionados |
| `DocumentNotReceived` | Documento original no recibido |
| `NotMatchingReceiverNIT` | NIT no coincide |
| `InvalidCreditNoteTransaction` | Montos exceden saldo |
| `InvalidUnitPriceZero` | Precio unitario 0 con gravado |
| `MixedSalesTypesNotAllowed` | Tipos de venta mezclados |
| `MissingItemRelatedDoc` | Ítem sin doc relacionado |
| `MissingItemTaxes` | Gravado sin impuestos |
| `InvalidTaxType` | Tipo de impuesto no permitido |
| `InvalidUnitMeasure` | Tipo impuesto con unidad != 99 |
