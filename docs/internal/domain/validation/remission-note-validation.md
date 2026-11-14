# Validación de Nota de Remisión (DTE 04)

> **Validador:** `RemissionNoteRulesValidator`
> **Paquete:** `internal/domain/dte/remission_note/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

Todas se ejecutan y los errores se agrupan:

1. **RemissionNoteItemStrategy** — Validación de ítems de remisión
2. **RemissionNoteSummaryStrategy** — Validación de totales y monto total
3. **RemissionNoteReceiverStrategy** — Validación del receptor y BienTitulo

Luego se aplican las [estrategias comunes](./README.md). La `PaymentTotalStrategy` no aplica a la nota de remisión, y los documentos relacionados se validan con la `RelatedDocsStrategy` común (la NR no tiene una estrategia propia de documentos relacionados).

---

## 1. RemissionNoteItemStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/strategy/remission_note_item_strategy.go`

| Validación | Regla | Error |
|---|---|---|
| Mínimo ítems | Al menos 1 | `RequiredField` |
| NonSubjectSale, ExemptSale, TaxedSale | Cada uno `>= 0` | `InvalidAmount` |
| Al menos una venta | La suma de las tres ventas del ítem debe ser distinta de 0 | `AtLeastOneSaleTypeRequired` |
| Numeración | Secuencial comenzando desde 1 (1, 2, 3...); el número viene en el campo `number` de cada ítem de la solicitud | `InvalidSequence` |

La venta del ítem no se contrasta contra `(UnitPrice * Quantity) - Discount` en NR, ni se valida exclusividad de tipos de venta.

---

## 2. RemissionNoteSummaryStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/strategy/remission_note_summary_strategy.go`

### Consistencia de Totales

Cada total se compara solo si fue informado en el resumen (holgura ±0.01):

```
Sum(item.NonSubjectSale) == Summary.NonSubjectTotal   (±0.01)   InvalidTotalNonSubject
Sum(item.ExemptSale)     == Summary.ExemptTotal       (±0.01)   InvalidTotalExempt
Sum(item.TaxedSale)      == Summary.TaxedTotal        (±0.01)   InvalidTotalTaxed
```

### Montos No Negativos

`NonSubjectTotal`, `ExemptTotal`, `TaxedTotal` y `TotalAmount` no pueden ser negativos (`InvalidAmount`).

### Monto Total de la Operación (campo 150)

```
TotalAmount = Subtotal + Sum(Tributes)    (±0.01)    InvalidTotalOperation
```

Se evalúa si `Subtotal` y `TotalAmount` están presentes (ambos son obligatorios en la solicitud). No se recalculan `Subtotal` (a partir de los totales y descuentos), `SubtotalSales`, `TotalDiscount`, ni el valor de cada tributo contra su tasa.

---

## 3. RemissionNoteReceiverStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/strategy/remission_note_receiver_strategy.go`

### BienTitulo (Título del Bien)

| Validación | Regla | Error |
|---|---|---|
| Requerido | Sí | `RequiredBienTitulo` |
| Largo máximo | 2 caracteres | `InvalidLength` |
| Valores válidos | `01`, `02`, `03`, `04`, `05`, `99` | `InvalidBienTitulo` |

| Valor | Significado |
|---|---|
| `01` | Venta |
| `02` | Consignación |
| `03` | Exhibición |
| `04` | Traslado interno |
| `05` | Préstamo |
| `99` | Otros |

### Campos del Receptor

| Campo | Requerido | Validación |
|---|---|---|
| Name | Sí | No vacío (`RequiredField`) |
| DocumentType | Sí | No vacío (`RequiredField`) |
| DocumentNumber | Sí | No vacío; el formato de NIT y DUI se valida en `BasicRulesStrategy` |
| Address | Sí | Debe existir |
| Email | Sí | No vacío |
| NIT | Condicional | Requerido si DocumentType = `"36"` |

---

## 4. Documentos Relacionados

La NR no trae una estrategia propia: aplica `RelatedDocsStrategy` (ver [README](./README.md#relateddocsstrategy)). Los documentos relacionados que puede referenciar una NR son FE (`01`), CCF (`03`) y FEXE (`11`) (`InvalidRelatedDocDTEType`).

---

## Errores Específicos

| Error | Causa |
|---|---|
| `RequiredField` | Campo obligatorio faltante |
| `InvalidAmount` | Venta o total negativo |
| `AtLeastOneSaleTypeRequired` | Ítem con las tres ventas en 0 |
| `InvalidSequence` | Ítems no numerados secuencialmente |
| `InvalidTotalNonSubject`, `InvalidTotalExempt`, `InvalidTotalTaxed` | Totales no coinciden con la suma de ítems |
| `InvalidTotalOperation` | `TotalAmount` distinto de `Subtotal + Sum(Tributes)` |
| `RequiredBienTitulo`, `InvalidBienTitulo`, `InvalidLength` | BienTitulo faltante, inválido o con más de 2 caracteres |
| `InvalidRemissionNoteReceiver` | El receptor no es del tipo de receptor de remisión |
