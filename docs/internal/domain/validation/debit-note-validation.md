# Validación de Nota de Débito (DTE 06)

> **Validador:** `DebitNoteRulesValidator`
> **Paquete:** `internal/domain/dte/debit_note/validator`

## Estrategias Compuestas

1. **DebitNoteItemStrategy** — Validación de ítems y reglas de impuestos
2. **DebitNoteTaxStrategy** — Validación de cálculos fiscales
3. **DebitNoteRelatedDocStrategy** — Validación de documentos relacionados

---

## 1. DebitNoteItemStrategy

**Idéntico al `CreditNoteItemStrategy`** — Las mismas reglas aplican:

| Validación | Regla |
|---|---|
| Máximo ítems | 2000 |
| Tipos de venta | No mezclar (taxed, exempt, non-subject) |
| UnitPrice | No puede ser 0 cuando `TaxedSale > 0` |
| Docs relacionados | Requeridos para todos los ítems |
| Impuestos | Requeridos si `TaxedSale > 0` |
| Producto (tipo 1) | Solo IVA (código `20`) |
| Impuesto (tipo 4) | Solo IVA, unidad de medida = `99` |

---

## 2. DebitNoteTaxStrategy

Sigue las mismas reglas que el `CreditNoteTaxStrategy`.

---

## 3. DebitNoteRelatedDocStrategy

Sigue las mismas reglas que el `CreditNoteRelatedDocStrategy`.

---

## Diferencia Clave con Nota de Crédito

La diferencia principal no está en la validación sino en el **servicio**:

| Aspecto | Nota de Crédito | Nota de Débito |
|---|---|---|
| Validación de ítems | Idéntica | Idéntica |
| Validación fiscal | Idéntica | Idéntica |
| Cálculo TotalToPay | Delegado al validador | Calculado en `calculateTotalToPay()` antes de validar |
| Balance | `ValidateForCreditNote` (verifica saldo disponible) | `ValidateForDebitNote` (verifica existencia) |
| Transacción | `CREDIT` (resta) | `DEBIT` (suma) |

### Fórmula de TotalToPay en el Servicio

```go
func (s *debitNoteService) calculateTotalToPay(debitNote) error {
    SubTotal = (TotalTaxed - TaxedDiscount)
             + (TotalExempt - ExemptDiscount)
             + (TotalNonSubject - NonSubjectDiscount)

    TotalOperation = SubTotal
                   + IVAPerception
                   - IVARetention
                   - IncomeRetention
                   + Sum(OtherTaxes)
                   + NonTaxedAmount
}
```

Este cálculo se realiza **antes** de ejecutar las estrategias de validación, lo que significa que cuando el `DebitNoteTaxStrategy` valida el `TotalToPay`, ya fue calculado por el servicio.

---

## Errores

Los mismos errores que la nota de crédito aplican, más:

| Error | Causa |
|---|---|
| Error de cálculo | Falla en `calculateTotalToPay()` |
