# Nota de Débito Electrónica (DTE 06) - Debit Note Service

> **Código DTE:** `06`
> **Paquete:** `internal/domain/dte/debit_note`
> **Servicio:** `debitNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Débito es un documento de ajuste que **incrementa** el valor de un documento tributario previamente emitido (CCF o Comprobante de Retención). Se utiliza para cargos adicionales, intereses por mora, o correcciones a favor del emisor.

Características principales:
- **Siempre referencia un CCF o un Comprobante de Retención** — Debe tener documentos relacionados de tipo `03` o `07` (`ValidAdjustmentDTETypes`)
- **Incrementa el saldo** — Genera una transacción de balance sobre el documento original
- **Cálculo propio de TotalToPay** — El servicio fija el total a pagar con `calculateTotalToPay` al final de `Create`
- **Validación de estado y NIT** — Cada documento relacionado debe estar en estado `RECEIVED` y el NIT del receptor debe coincidir con el del documento original

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
1. Cast del input → *DebitNoteInput
2. Validar documentos relacionados (existencia, estado y NIT)
3. Crear documento base (DTEDocument)
4. Ensamblar DebitNoteModel con ítems y resumen
5. Validar reglas específicas de la nota de débito (DebitNoteRulesValidator)
6. Validar reglas genéricas de DTE (ValidateDTEDocument)
7. Por cada documento relacionado electrónico: ValidateForDebitNote (existe control de balance)
8. Generar código de generación (UUID) y número de control
9. Calcular TotalToPay (calculateTotalToPay)
10. Retornar DebitNoteModel validado
```

> **Diferencia con Nota de Crédito**: La Nota de Débito incluye un paso final (paso 9) que fija el `TotalToPay` con `SetForceTotalToPay`, después de la validación y de generar los identificadores.

### Paso 2: Validación de Documentos Relacionados

```go
func (s *debitNoteService) validateRelatedDocs(
    ctx context.Context,
    data *debit_note_models.DebitNoteInput,
    branchID uint,
) error
```

Igual que en la nota de crédito:
1. Se verifica que exista al menos un documento relacionado (`NoRelatedDocs`)
2. Por cada documento relacionado se busca el original (`GetByGenerationCode`), se verifica el estado `RECEIVED` (`RelatedDocumentNotReceived`, con código y estado actual), se asigna a la referencia la fecha de generación del original y se valida que el NIT del receptor coincida con el del original (`NotMatchingReceiverNIT`)

### Paso 9: Cálculo del Total a Pagar

```go
func (s *debitNoteService) calculateTotalToPay(debitNote *debit_note_models.DebitNoteModel) error
```

Se calcula a partir del resumen y se fija con `SetForceTotalToPay`:

1. Subtotal: gravado, exento y no sujeto, cada uno menos su descuento.
2. Si el total gravado es mayor que cero, se suma `IVAPerception` y se restan `IVARetention` e `IncomeRetention`.
3. Se suman los tributos del resumen (`TotalTaxes`).
4. Si `TotalNonTaxed` es mayor que cero, se suma.

Las fórmulas generales y la tolerancia se documentan en [calculations.md](../../../api/calculations.md).

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

## Cálculos

Las fórmulas (IVA, percepción, subtotales, total de la operación y total a pagar) se documentan en [calculations.md](../../../api/calculations.md). Reglas específicas del resumen de la nota de débito (las mismas que la nota de crédito):

- No se admiten descuentos globales ni porcentaje de descuento (`GlobalDiscountNotAllowedInAdjustment`).
- `IncomeRetention` debe ser cero (`IncomeRetentionNotAllowed`).
- El total de la operación es el subtotal de ventas más los tributos del resumen y el total a pagar se valida con la regla compartida `ValidateAdjustmentTotals`.

---

## Comparación: Nota de Crédito vs Nota de Débito

| Aspecto | Nota de Crédito (05) | Nota de Débito (06) |
|---|---|---|
| **Efecto** | Disminuye saldo | Incrementa saldo |
| **Transacción** | Tipo `05` (resta del saldo restante) | Tipo `06` (suma al saldo restante) |
| **Cálculo TotalToPay** | Validado por el validador | Fijado en el servicio (`calculateTotalToPay`) |
| **Validación de balance** | Verifica saldo disponible | Verifica que exista el control de balance |
| **Paso adicional** | No | Sí (cálculo de TotalToPay) |

---

## Control de Balance

### Validación Pre-Emisión

```go
dteManager.ValidateForDebitNote(ctx, branchID, originalDTE, document)
```

Se ejecuta dentro de `Create` por cada documento relacionado electrónico. Solo verifica que exista el control de balance del documento original (`GetDTEBalanceControl`); no compara montos.

### Transacción de Balance Post-Emisión

```go
dteManager.GenerateBalanceTransaction(ctx, branchID, constants.NotaDebitoElectronica, originalDTE, debitNoteID, document)
```

Se registra la transacción por cada documento relacionado electrónico. En el modelo de persistencia (`DTEBalanceTransaction.AfterCreate`) solo el tipo `05` resta del saldo restante; cualquier otro tipo, incluido `06`, suma.

---

## Reglas de Ítems

Las reglas de ítems son **idénticas a las de la nota de crédito**:

- Máximo 2000 ítems
- Exclusividad de tipos de venta (no mezclar taxed, exempt, non-subject)
- `UnitPrice` no puede ser 0 cuando `TaxedSale > 0`
- Todos los ítems deben referenciar un documento relacionado
- Impuestos requeridos si `TaxedSale > 0`
- Impuesto (tipo 4): exactamente un impuesto, IVA, y unidad de medida `99`
- Los impuestos de un ítem que no es de tipo Impuesto deben ser códigos permitidos (`InvalidTaxType`)
- Los documentos relacionados son obligatorios (máximo 50) y su tipo debe ser `03` o `07`; el error usado es `InvalidRelatedDocTypeForCreditNote`

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Pánico por cast (cast sin comprobación) | El `interface{}` no es `*DebitNoteInput` |
| `NoRelatedDocs` | Sin documentos relacionados |
| `RelatedDocumentNotReceived` | Documento relacionado no en estado `RECEIVED` |
| `NotMatchingReceiverNIT` | NIT del receptor no coincide con el del documento original |
| `InvalidRelatedDocTypeForCreditNote` | Documento relacionado distinto de CCF o Comprobante de Retención |
| `GlobalDiscountNotAllowedInAdjustment` | Descuento global o porcentaje de descuento distinto de cero |
| `IncomeRetentionNotAllowed` | Retención de renta distinta de cero |
| Errores de validación de ítems | Mismos que nota de crédito |

---

## Flujo Completo

```
Request
  │
  ▼
DebitNoteService.Create()
  │
  ├── validateRelatedDocs()       ← Verifica estado y NIT del doc original
  │
  ├── createBaseDocument()        ← Convierte input a DTEDocument
  │
  ├── validate()                  ← Reglas de negocio
  │
  ├── ValidateDTEDocument()       ← Reglas genéricas
  │
  ├── ValidateForDebitNote()      ← Existe control de balance (docs electrónicos)
  │
  ├── generateCodeAndIdentifiers()
  │
  └── calculateTotalToPay()       ← Cálculo específico de nota de débito
        │
        ▼
Application Layer
  │
  ├── Transmit to Hacienda
  │
  └── [Si éxito]:
        ├── ConfirmReservation()
        ├── Create() en BD
        └── GenerateBalanceTransaction(tipo 06)  ← Registra la transacción de balance
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/debit_note/debit_note_service.go` | Servicio principal |
| `internal/domain/dte/debit_note/debit_note_models/` | Modelos de nota de débito |
| `internal/domain/dte/debit_note/validator/` | Estrategias de validación |

---

## Notas

1. **Paso adicional**: El `calculateTotalToPay()` se ejecuta **después** de la validación y de generar los identificadores, diferenciándolo de la nota de crédito.
2. **Balance**: La nota de crédito genera una transacción de tipo `05` (resta) y la nota de débito de tipo `06` (suma).
3. **Invalidación**: Al invalidar una nota de débito se registra una transacción de balance de tipo `DEBIT_NOTE_INVALID` (ver [invalidation.md](invalidation.md)), que el modelo de persistencia resta del saldo restante (la nota de débito los había sumado al emitirse).
4. **Validadores compartidos**: El `DebitNoteItemStrategy` es idéntico al `CreditNoteItemStrategy`.
