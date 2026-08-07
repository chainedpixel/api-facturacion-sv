# Nota de Crédito Electrónica (DTE 05) - Credit Note Service

> **Código DTE:** `05`
> **Paquete:** `internal/domain/dte/credit_note`
> **Servicio:** `creditNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Crédito es un documento de ajuste que **disminuye** el valor de un documento tributario previamente emitido (CCF). Se utiliza para devoluciones, descuentos posteriores, o correcciones a favor del receptor.

Características principales:
- **Siempre referencia un CCF** — Debe tener documentos relacionados válidos
- **Reduce el saldo** — Genera una transacción de balance que disminuye el monto original
- **Validación de estado** — El documento original debe estar en estado `RECEIVED`
- **Validación de NIT** — El NIT del receptor debe coincidir con el del documento original
- **Control de balance** — No puede exceder el saldo disponible del documento original

---

## Estructura del Servicio

```go
type creditNoteService struct {
    validator        *validator.CreditNoteRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
    dteManager       dte_documents.DTEManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.CreditNoteRulesValidator` | Se validan reglas de negocio de nota de crédito |
| `seqNumberManager` | `SequentialNumberManager` | Se gestiona la reserva de números de control |
| `dteManager` | `DTEManager` | Se accede al documento original para validaciones de estado y balance |

> **Dependencia adicional**: A diferencia de Factura/CCF/FSE, la Nota de Crédito requiere `DTEManager` para validar documentos relacionados.

### Constructor

```go
func NewCreditNoteService(
    seqNumberManager dte_documents.SequentialNumberManager,
    dteManager dte_documents.DTEManager,
) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *creditNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → CreditNoteInput
2. Validar documentos relacionados (estado + NIT)
3. Crear documento base (DTEDocument)
4. Ensamblar CreditNoteModel con ítems y resumen
5. Validar reglas de negocio
6. Generar código de generación (UUID) y número de control
7. Retornar CreditNoteModel validado
```

### Paso 2: Validación de Documentos Relacionados

```go
func (s *creditNoteService) validateRelatedDocs(
    ctx context.Context,
    data *credit_note_models.CreditNoteInput,
    branchID uint,
) error
```

Esta validación es **crítica** y se ejecuta antes de cualquier otra operación:

1. **Verificar existencia**: Se debe tener al menos un documento relacionado
2. **Por cada documento relacionado**:
   - Se busca el documento original usando `dteManager.VerifyStatus()`
   - Se verifica que el estado sea `RECEIVED` (no `REJECTED`, `INVALIDATED`, ni `PENDING`)
   - Se valida que el NIT del receptor coincida con el del documento original

#### Errores de Validación de Documentos Relacionados

| Error | Causa |
|---|---|
| `NoRelatedDocs` | No se proporcionaron documentos relacionados |
| `DocumentNotReceived` | El documento original no está en estado `RECEIVED` |
| `NotMatchingReceiverNIT` | El NIT del receptor no coincide con el del documento original |

---

## Modelos Específicos de Nota de Crédito

### CreditNoteModel

```go
type CreditNoteModel struct {
    *models.DTEDocument
    CreditItems   []CreditNoteItem
    CreditSummary CreditNoteSummary
}
```

### CreditNoteInput

```go
type CreditNoteInput struct {
    *models.InputDataCommon
    Items         []CreditNoteItem
    CreditSummary *CreditNoteSummary
}
```

### CreditNoteItem

```go
type CreditNoteItem struct {
    *models.Item
    NonSubjectSale financial.Amount  // Venta no sujeta
    ExemptSale     financial.Amount  // Venta exenta
    TaxedSale      financial.Amount  // Venta gravada
}
```

> **Sin IVAItem ni SuggestedPrice**: A diferencia de la factura, la nota de crédito no tiene IVA por ítem ni precio sugerido.

### CreditNoteSummary

```go
type CreditNoteSummary struct {
    *models.Summary
    TaxedDiscount   financial.Amount  // Descuento sobre gravado
    IVAPerception   financial.Amount  // Percepción de IVA
    IVARetention    financial.Amount  // Retención de IVA
    IncomeRetention financial.Amount  // Retención de renta
}
```

---

## Fórmulas de Cálculo

### IVA (igual que CCF)

```
IVA = (TotalTaxed - TaxedDiscount) * 0.13    (tolerancia ±0.01)
```

### Percepción de IVA

```
IVAPerception = TotalTaxed * 0.01    (cuando aplica, tolerancia ±0.01)
```

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount
```

### Total Operación

```
TotalOperation = SubTotal + Sum(Taxes)
```

---

## Control de Balance

La nota de crédito interactúa con el sistema de control de balance a través del `DTEManager`:

### Validación Pre-Emisión

Antes de confirmar la nota de crédito, se ejecuta `ValidateForCreditNote()`:

```go
dteManager.ValidateForCreditNote(ctx, branchID, originalDTE, document)
```

Se verifica que los montos de la nota de crédito no excedan el **saldo disponible** del documento original. El saldo disponible se calcula como:

```
SaldoDisponible = MontoOriginal - Sum(NotasDeCréditoPrevias)
```

### Transacción de Balance Post-Emisión

Después de la transmisión exitosa, se genera la transacción de balance en la capa de aplicación:

```go
dteManager.GenerateBalanceTransaction(ctx, branchID, "CREDIT", originalDTE, creditNoteID, document)
```

Se extraen los montos `TaxedSale`, `ExemptSale` y `NonSubjectSale` del documento y se registran como reducción del balance original.

---

## Reglas de Ítems

### Tipos de Venta

- Cada ítem puede tener solo un tipo de venta activo (misma regla que CCF)
- Si `TaxedSale > 0`: `UnitPrice` no puede ser 0
- Si `TaxedSale > 0`: debe tener impuestos asignados

### Documentos Relacionados en Ítems

- Si existen documentos relacionados, **todos los ítems deben referenciar uno**
- La referencia del ítem debe existir en la lista de documentos relacionados del DTE

### Reglas de Impuestos por Tipo

- **Producto (tipo 1):** Solo IVA (código `20`)
- **Impuesto (tipo 4):** Solo IVA, unidad de medida debe ser `99`

---

## Flujo Completo (incluyendo capa de aplicación)

```
Request
  │
  ▼
CreditNoteService.Create()
  │
  ├── validateRelatedDocs()     ← Verifica estado y NIT del doc original
  │     └── dteManager.VerifyStatus()
  │
  ├── createBaseDocument()      ← Convierte input a DTEDocument
  │
  ├── validate()                ← Reglas de negocio de nota de crédito
  │
  └── generateCodeAndIdentifiers()
        │
        ▼
Application Layer (GenericDTEUseCase)
  │
  ├── Map to Hacienda JSON
  ├── Transmit to Hacienda
  │
  ├── [Si éxito]:
  │     ├── dteManager.ValidateForCreditNote()  ← Valida balance disponible
  │     ├── ConfirmReservation()
  │     ├── Create() en BD
  │     └── GenerateBalanceTransaction()  ← Registra reducción de balance
  │
  └── [Si error]: Release o Contingencia
```

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido | El `interface{}` no es `*CreditNoteInput` |
| `NoRelatedDocs` | Sin documentos relacionados |
| `DocumentNotReceived` | Documento original no recibido |
| `NotMatchingReceiverNIT` | NIT del receptor no coincide |
| `InvalidCreditNoteTransaction` | Montos exceden saldo disponible |
| `InvalidTaxCodeOnly20` | Producto con impuesto diferente a IVA |
| `MissingItemRelatedDoc` | Ítem sin referencia a documento relacionado |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/credit_note/credit_note_service.go` | Servicio principal |
| `internal/domain/dte/credit_note/models/` | Modelos de nota de crédito |
| `internal/domain/dte/credit_note/validator/` | Estrategias de validación |
| `internal/domain/dte/dte_documents/dte_service.go` | DTEManager (balance) |

---

## Notas

1. **Dependencia del DTEManager**: Es el único DTE (junto con Nota de Débito) que requiere acceso a documentos existentes para validar.
2. **Balance**: La validación de balance ocurre en **dos momentos**: al validar (pre-emisión) y al registrar (post-transmisión).
3. **Invalidación reversa**: Si una nota de crédito es invalidada, se **recupera** el balance del documento original.
4. **No confundir con pagos**: La nota de crédito no aplica al `PaymentTotalStrategy` (no requiere formas de pago).
