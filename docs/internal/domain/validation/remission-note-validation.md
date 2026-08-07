# Validación de Nota de Remisión (DTE 04)

> **Validador:** `RemissionNoteRulesValidator`
> **Paquete:** `internal/domain/dte/remission_note/validator`

## Estrategias Compuestas

1. **RemissionNoteItemStrategy** — Validación de ítems de remisión
2. **RemissionNoteSummaryStrategy** — Validación de totales y subtotales
3. **RemissionNoteReceiverStrategy** — Validación del receptor y BienTitulo

---

## 1. RemissionNoteItemStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/remission_note_item_strategy.go`

| Validación | Regla |
|---|---|
| Mínimo ítems | Al menos 1 |
| NonSubjectSale | >= 0 |
| ExemptSale | >= 0 |
| TaxedSale | >= 0 |
| Al menos una venta | Algún tipo de venta debe ser > 0 |
| Numeración | Secuencial comenzando desde 1 (1, 2, 3...) |

---

## 2. RemissionNoteSummaryStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/remission_note_summary_strategy.go`

### Consistencia de Totales

```
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)
Sum(item.TaxedSale)      == Summary.TotalTaxed        (±0.01)
```

### Montos No Negativos

```
TotalNonSubject >= 0
TotalExempt     >= 0
TotalTaxed      >= 0
```

### SubTotal = Total

```
SubTotal == TotalAmount    (regla específica de remisión)
```

---

## 3. RemissionNoteReceiverStrategy

**Archivo:** `internal/domain/dte/remission_note/validator/remission_note_receiver_strategy.go`

### BienTitulo (Título del Bien)

| Validación | Regla |
|---|---|
| Requerido | Sí |
| Largo máximo | 2 caracteres |
| Valores válidos | `01`, `02`, `03`, `04`, `05`, `99` |

| Valor | Significado |
|---|---|
| `01` | Propiedad |
| `02` | Consignación |
| `03` | Depósito |
| `04` | Exhibición |
| `05` | Reparación |
| `99` | Otro |

### Campos del Receptor

| Campo | Requerido | Validación |
|---|---|---|
| Name | Sí | No vacío |
| DocumentType | Sí | Tipo válido |
| DocumentNumber | Sí | Formato según tipo |
| Address | Sí | No vacío |
| Email | Sí | Formato válido |
| NIT | Condicional | Requerido si DocumentType = `"36"` |

---

## Errores Específicos

| Error | Causa |
|---|---|
| `ValidationFailed` | Error general de validación |
| `RequiredField` | Campo obligatorio faltante |
| `InvalidValue` | Valor fuera de rango |
| `InvalidSequence` | Ítems no numerados secuencialmente |
| `TotalMismatch` | Totales no coinciden con suma |
| `InvalidLength` | BienTitulo excede 2 caracteres |
| `InvalidCalculation` | SubTotal != TotalAmount |
