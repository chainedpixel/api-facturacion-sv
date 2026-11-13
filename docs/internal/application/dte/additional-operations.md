# Operaciones Adicionales — Post-Transmisión

> **Paquete:** `internal/application/dte`
> **Archivo:** `additional_operations.go`

## Descripción General

Las operaciones adicionales son funciones que se ejecutan **después de la transmisión exitosa y de guardar el documento** (con su evidencia de transmisión). Se implementan como un hook configurable que el `GenericDTEUseCase` invoca antes de armar la respuesta del Archivo DTE.

---

## Tipo de Función

```go
type AdditionalOperationsFunc func(
    ctx context.Context,
    result interface{},      // Modelo de dominio (resultado del service.Create)
    branchID uint,
    mhModel interface{},     // Modelo Hacienda (formato MH)
) error
```

---

## DTEOperations

```go
type DTEOperations struct{}
func NewDTEOperations() *DTEOperations
```

### `GetNoOperation` — Sin Operaciones

```go
func (o *DTEOperations) GetNoOperation() AdditionalOperationsFunc
```

Retorna una función no-op (`return nil`). Se usa para DTEs que no requieren operaciones post-transmisión:
- Factura (01)
- CCF (03)
- Retención (07)
- Nota de Remisión (04)
- FSE (14)

---

### `GetCreditNoteOperations` — Transacciones de Balance (Crédito)

```go
func (o *DTEOperations) GetCreditNoteOperations(
    dteService dte_documents.DTEManager,
) AdditionalOperationsFunc
```

Se genera una transacción de balance por cada documento relacionado electrónico:

```
Por cada relatedDoc donde GetGenerationType() == ElectronicDocument:
  dteService.GenerateBalanceTransaction(
      ctx,
      branchID,
      constants.NotaCreditoElectronica,        // "05"
      relatedDoc.GetDocumentNumber(),           // Número del doc original
      creditNote.GetIdentification().GetGenerationCode(), // UUID de la nota
      mhModel,                                  // Formato Hacienda
  )
```

**Efecto:** Se inserta una transacción de balance (`dte_balance_transactions`) sobre el control de saldo del documento original (`dte_balance_control`). El hook `AfterCreate` de `DTEBalanceTransaction` resta los montos gravado, exento y no sujeto de los saldos restantes cuando el tipo de transacción es `05`.

Los montos se extraen del resumen del modelo Hacienda (`utils.ExtractSummaryTotalAmounts`).

---

### `GetDebitNoteOperations` — Transacciones de Balance (Débito)

```go
func (o *DTEOperations) GetDebitNoteOperations(
    dteService dte_documents.DTEManager,
) AdditionalOperationsFunc
```

Idéntico a la operación de crédito pero:
- Se type-asserts a `*DebitNoteModel`
- Se usa `constants.NotaDebitoElectronica` (`"06"`) como tipo de transacción

**Efecto:** Se inserta una transacción de balance con tipo `06`. El hook `AfterCreate` de `DTEBalanceTransaction` suma los montos a los saldos restantes para cualquier tipo distinto de `05`.

---

## Flujo de Balance

```
Nota de Crédito/Débito transmitida exitosamente
  │
  ▼
additionalOps(ctx, result, branchID, mhModel)
  │
  ▼
Por cada documento relacionado:
  │
  ├── ¿Es documento electrónico? (GetGenerationType() == ElectronicDocument)
  │     │
  │     ├── Sí → DTEManager.GenerateBalanceTransaction()
  │     │        ├── Extrae montos del resumen del mhModel
  │     │        └── Inserta la transacción (bajo bloqueo de la fila de control de saldo)
  │     │
  │     └── No (físico) → Saltar
  │
  ▼
Retornar nil o error
```

---

## Notas

1. **Solo documentos electrónicos**: Los documentos físicos (tipo 1) no generan transacciones de balance porque no están en el sistema.
2. **Error devuelto al llamador**: Si una transacción de balance falla, se registra un warning y se devuelve el error. En ese punto el documento ya fue transmitido y guardado, por lo que el consumidor recibe un error aunque Hacienda ya otorgó el sello y puede requerir atención manual. Si hay varios documentos relacionados, las transacciones ya insertadas no se revierten.
3. **Múltiples documentos**: Una nota de crédito/débito puede referenciar múltiples documentos, generando una transacción por cada uno.
4. **Tipo del resultado**: Si `result` no es `*CreditNoteModel` (o `*DebitNoteModel`) se devuelve el error `invalid result type`.
5. **Reversión por invalidación**: Al invalidar una nota, `InvalidationManager.InvalidateDocument` genera una transacción con tipo `INVALIDATED` (`GenerateBalanceTransactionWithAmounts`); el hook de `AfterCreate` la suma al saldo (ver [invalidation-use-case.md](./invalidation-use-case.md)).
