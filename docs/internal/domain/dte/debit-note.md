# Nota de Débito Electrónica (DTE 06) - Debit Note Service

> **Código DTE:** `06`
> **Paquete:** `internal/domain/dte/debit_note`
> **Servicio:** `debitNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Débito es un documento de ajuste que **incrementa** el valor de un documento tributario previamente emitido (CCF). Se utiliza para cargos adicionales, intereses por mora, o correcciones a favor del emisor.

Características principales:
- **Siempre referencia un CCF** — Debe tener documentos relacionados válidos
- **Incrementa el saldo** — Genera una transacción de balance que aumenta el monto adeudado
- **Cálculo especial de TotalToPay** — Incluye lógica de cálculo propia con impuestos, percepciones y retenciones
- **Validación de estado** — El documento original debe estar en estado `RECEIVED`

---

## Estructura del Servicio

```go
type debitNoteService struct {
    validator        *validator.DebitNoteRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
    dteManager       dte_documents.DTEManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.DebitNoteRulesValidator` | Se validan reglas de negocio de nota de débito |
| `seqNumberManager` | `SequentialNumberManager` | Se gestiona la reserva de números de control |
| `dteManager` | `DTEManager` | Se accede al documento original para validaciones |

### Constructor

```go
func NewDebitNoteService(
    seqNumberManager dte_documents.SequentialNumberManager,
    dteManager dte_documents.DTEManager,
) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *debitNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → DebitNoteInput
2. Validar documentos relacionados (estado)
3. Crear documento base (DTEDocument)
4. Ensamblar DebitNoteModel con ítems y resumen
5. Calcular TotalToPay
6. Validar reglas de negocio
7. Generar código de generación (UUID) y número de control
8. Retornar DebitNoteModel validado
```

> **Diferencia con Nota de Crédito**: La Nota de Débito incluye un paso adicional (paso 5) de cálculo del `TotalToPay` antes de la validación.

### Paso 2: Validación de Documentos Relacionados

```go
func (s *debitNoteService) validateRelatedDocs(
    ctx context.Context,
    data *debit_note_models.DebitNoteInput,
    branchID uint,
) error
```

Similar a la nota de crédito:
1. Se verifica que exista al menos un documento relacionado
2. Por cada documento, se verifica el estado `RECEIVED`

### Paso 5: Cálculo del Total a Pagar

```go
func (s *debitNoteService) calculateTotalToPay(debitNote *debit_note_models.DebitNoteModel) error
```

El cálculo del `TotalToPay` en la nota de débito incluye una lógica específica:

```
SubTotal = (TotalTaxed - TaxedDiscount) + (TotalExempt - ExemptDiscount) + (TotalNonSubject - NonSubjectDiscount)

TotalOperation = SubTotal + IVAPerception - IVARetention - IncomeRetention + OtherTaxes + NonTaxedAmount
```

---

## Modelos Específicos de Nota de Débito

### DebitNoteModel

```go
type DebitNoteModel struct {
    *models.DTEDocument
    DebitItems   []DebitNoteItem
    DebitSummary DebitNoteSummary
}
```

### DebitNoteInput

```go
type DebitNoteInput struct {
    *models.InputDataCommon
    Items        []DebitNoteItem
    DebitSummary *DebitNoteSummary
}
```

### DebitNoteItem

```go
type DebitNoteItem struct {
    *models.Item
    NonSubjectSale financial.Amount  // Venta no sujeta
    ExemptSale     financial.Amount  // Venta exenta
    TaxedSale      financial.Amount  // Venta gravada
}
```

> Idéntico al `CreditNoteItem` en estructura.

### DebitNoteSummary

```go
type DebitNoteSummary struct {
    *models.Summary
    TaxedDiscount   financial.Amount  // Descuento sobre gravado
    IVAPerception   financial.Amount  // Percepción de IVA
    IVARetention    financial.Amount  // Retención de IVA
    IncomeRetention financial.Amount  // Retención de renta
}
```

---

## Fórmulas de Cálculo

### SubTotal

```
SubTotal = (TotalTaxed - TaxedDiscount)
         + (TotalExempt - ExemptDiscount)
         + (TotalNonSubject - NonSubjectDiscount)
```

### Total Operación (con impuestos)

```
TotalOperation = SubTotal
               + IVAPerception
               - IVARetention
               - IncomeRetention
               + Sum(OtherTaxes)
               + NonTaxedAmount
```

### IVA

```
IVA = (TotalTaxed - TaxedDiscount) * 0.13
```

---

## Comparación: Nota de Crédito vs Nota de Débito

| Aspecto | Nota de Crédito (05) | Nota de Débito (06) |
|---|---|---|
| **Efecto** | Disminuye saldo | Incrementa saldo |
| **Transacción** | `CREDIT` | `DEBIT` |
| **Cálculo TotalToPay** | Delegado al validador | Calculado en servicio (`calculateTotalToPay`) |
| **Validación de balance** | Verifica saldo disponible | Verifica documento existe |
| **Paso adicional** | No | Sí (cálculo de TotalToPay) |

---

## Control de Balance

### Validación Pre-Emisión

```go
dteManager.ValidateForDebitNote(ctx, branchID, originalDTE, document)
```

Se verifica que el documento original existe y está en estado válido.

### Transacción de Balance Post-Emisión

```go
dteManager.GenerateBalanceTransaction(ctx, branchID, "DEBIT", originalDTE, debitNoteID, document)
```

Se registra un incremento en el balance del documento original. A diferencia de la nota de crédito que reduce, la nota de débito **incrementa** el monto adeudado.

---

## Reglas de Ítems

Las reglas de ítems son **idénticas a las de la nota de crédito**:

- Máximo 2000 ítems
- Exclusividad de tipos de venta (no mezclar taxed, exempt, non-subject)
- `UnitPrice` no puede ser 0 cuando `TaxedSale > 0`
- Todos los ítems deben referenciar un documento relacionado
- Impuestos requeridos si `TaxedSale > 0`
- Producto (tipo 1): solo IVA (código `20`)
- Impuesto (tipo 4): solo IVA, unidad de medida `99`

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido | El `interface{}` no es `*DebitNoteInput` |
| `NoRelatedDocs` | Sin documentos relacionados |
| `DocumentNotReceived` | Documento original no en estado `RECEIVED` |
| Error de cálculo | Falla al calcular `TotalToPay` |
| Errores de validación de ítems | Mismos que nota de crédito |

---

## Flujo Completo

```
Request
  │
  ▼
DebitNoteService.Create()
  │
  ├── validateRelatedDocs()       ← Verifica estado doc original
  │
  ├── createBaseDocument()        ← Convierte input a DTEDocument
  │
  ├── calculateTotalToPay()       ← Cálculo específico de nota de débito
  │
  ├── validate()                  ← Reglas de negocio
  │
  └── generateCodeAndIdentifiers()
        │
        ▼
Application Layer
  │
  ├── Transmit to Hacienda
  │
  └── [Si éxito]:
        ├── ValidateForDebitNote()
        ├── ConfirmReservation()
        ├── Create() en BD
        └── GenerateBalanceTransaction("DEBIT")  ← Incrementa balance
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/debit_note/debit_note_service.go` | Servicio principal |
| `internal/domain/dte/debit_note/models/` | Modelos de nota de débito |
| `internal/domain/dte/debit_note/validator/` | Estrategias de validación |

---

## Notas

1. **Paso adicional**: El `calculateTotalToPay()` se ejecuta **antes** de la validación, diferenciándolo de la nota de crédito.
2. **Balance invertido**: Mientras la nota de crédito genera transacción `CREDIT` (resta), la nota de débito genera `DEBIT` (suma).
3. **Invalidación reversa**: Si una nota de débito es invalidada, se **revierte** el incremento de balance.
4. **Validadores compartidos**: El `DebitNoteItemStrategy` es idéntico al `CreditNoteItemStrategy`.
