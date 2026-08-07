# Operaciones Adicionales — Post-Transmisión

> **Paquete:** `internal/application/dte`
> **Archivo:** `additional_operations.go`

## Descripción General

Las operaciones adicionales son funciones que se ejecutan **después de la transmisión exitosa** de un DTE a Hacienda. Se implementan como un hook configurable que el `GenericDTEUseCase` invoca como último paso.

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

**Efecto:** Se registra una **reducción** en el balance del documento original referenciado.

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

**Efecto:** Se registra un **incremento** en el balance del documento original referenciado.

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
  ├── ¿Es documento electrónico?
  │     │
  │     ├── Sí → GenerateBalanceTransaction()
  │     │        ├── Extrae montos del mhModel
  │     │        └── Registra transacción de balance
  │     │
  │     └── No (físico) → Saltar
  │
  ▼
Retornar nil o error
```

---

## Notas

1. **Solo documentos electrónicos**: Los documentos físicos (tipo 1) no generan transacciones de balance porque no están en el sistema.
2. **Error no bloqueante**: Si una transacción de balance falla, se loguea un warning pero se retorna error, lo que puede necesitar atención manual.
3. **Múltiples documentos**: Una nota de crédito/débito puede referenciar múltiples documentos, generando una transacción por cada uno.
