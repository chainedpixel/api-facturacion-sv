# Nota de Crédito Electrónica (DTE 05) - Credit Note Service

> **Código DTE:** `05`
> **Paquete:** `internal/domain/dte/credit_note`
> **Servicio:** `creditNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Crédito es un documento de ajuste que **disminuye** el valor de un documento tributario previamente emitido (CCF o Comprobante de Retención). Se utiliza para devoluciones, descuentos posteriores, o correcciones a favor del receptor.

Características principales:
- **Siempre referencia un CCF o un Comprobante de Retención** — Debe tener documentos relacionados de tipo `03` o `07` (`ValidAdjustmentDTETypes`)
- **Reduce el saldo** — Genera una transacción de balance que disminuye el monto original
- **Validación de estado** — Cada documento relacionado debe estar en estado `RECEIVED`
- **Validación de NIT** — El NIT del receptor debe coincidir con el del documento relacionado
- **Control de balance** — No puede exceder el saldo disponible del documento original (solo para documentos relacionados electrónicos)

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
1. Cast del input → *CreditNoteInput
2. Validar documentos relacionados (existencia, estado y NIT)
3. Crear documento base (DTEDocument)
4. Ensamblar CreditNoteModel con ítems y resumen
5. Validar reglas específicas de la nota de crédito (CreditNoteRulesValidator)
6. Validar reglas genéricas de DTE (ValidateDTEDocument)
7. Por cada documento relacionado electrónico: ValidateForCreditNote (saldo disponible)
8. Generar código de generación (UUID) y número de control
9. Retornar CreditNoteModel validado
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
   - Se busca el documento original con `dteManager.GetByGenerationCode()` (si no existe, se propaga el error)
   - Se consulta su estado con `dteManager.VerifyStatus()` y debe ser `RECEIVED` (no `REJECTED`, `INVALIDATED`, `PENDING` ni `EXPIRED`)
   - Se asigna a la referencia la fecha de emisión (generación) del documento original, tomada de su JSON o, en su defecto, de su fecha de registro
   - Se valida que el NIT del receptor coincida con el del receptor del documento original

#### Errores de Validación de Documentos Relacionados

| Error | Causa |
|---|---|
| `NoRelatedDocs` | No se proporcionaron documentos relacionados |
| `RelatedDocumentNotReceived` | El documento relacionado no está en estado `RECEIVED` (parámetros: código del documento y estado actual) |
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

## Cálculos

Las fórmulas (IVA, percepción, subtotales, total de la operación y total a pagar) se documentan en [calculations.md](../../../api/calculations.md). Reglas específicas del resumen de la nota de crédito:

- No se admiten descuentos globales ni porcentaje de descuento (`GlobalDiscountNotAllowedInAdjustment`); los descuentos por ítem siguen permitidos.
- `IncomeRetention` debe ser cero (`IncomeRetentionNotAllowed`).
- Si `IVAPerception` es distinto de cero, debe corresponder al 1% del gravado.
- El total de la operación es el subtotal de ventas más los tributos del resumen; el total a pagar suma además `IVAPerception` y `TotalNonTaxed` y resta `IVARetention` (regla compartida `ValidateAdjustmentTotals`).

---

## Control de Balance

La nota de crédito interactúa con el sistema de control de balance a través del `DTEManager`:

### Validación Pre-Emisión

Antes de confirmar la nota de crédito, se ejecuta `ValidateForCreditNote()`:

```go
dteManager.ValidateForCreditNote(ctx, branchID, originalDTE, document)
```

Se ejecuta dentro de `Create` por cada documento relacionado de tipo electrónico (`ElectronicDocument`). Se verifica que cada monto de la nota (gravado, exento, no sujeto) no exceda el **saldo restante** del control de balance del documento original (`InvalidCreditNoteTransaction`). El saldo restante se reduce con cada nota de crédito registrada y se restituye cuando una nota es invalidada.

### Transacción de Balance Post-Emisión

Después de la transmisión exitosa, se genera la transacción de balance en la capa de aplicación:

```go
dteManager.GenerateBalanceTransaction(ctx, branchID, constants.NotaCreditoElectronica, originalDTE, creditNoteID, document)
```

Por cada documento relacionado electrónico se extraen los totales gravado, exento y no sujeto del resumen y se registran como reducción del balance original (el tipo de transacción `05` resta del saldo).

---

## Reglas de Ítems

### Tipos de Venta

- Máximo 2000 ítems (`ExceededItemsLimit`)
- Cada ítem puede tener solo un tipo de venta activo entre gravada, exenta y no sujeta (`MixedSalesTypesNotAllowed`)
- Si `TaxedSale > 0`: `UnitPrice` no puede ser 0 y debe tener impuestos asignados (`MissingItemTaxes`)
- Los impuestos de un ítem que no es de tipo Impuesto deben ser códigos permitidos (`InvalidTaxType`)

### Documentos Relacionados en Ítems

- **Todos los ítems deben referenciar un documento relacionado** (`MissingItemRelatedDoc`)
- La referencia del ítem debe existir en la lista de documentos relacionados del DTE (`InvalidItemRelatedDoc`)
- Los documentos relacionados son obligatorios (máximo 50) y su tipo debe ser `03` o `07` (`InvalidRelatedDocTypeForCreditNote`)

### Reglas de Impuestos por Tipo

- **Impuesto (tipo 4):** Exactamente un impuesto, IVA (código `20`), y unidad de medida `99`

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
  ├── ValidateDTEDocument()     ← Reglas genéricas
  │
  ├── dteManager.ValidateForCreditNote() ← Saldo disponible (documentos relacionados electrónicos)
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
| Pánico por cast (cast sin comprobación) | El `interface{}` no es `*CreditNoteInput` |
| `NoRelatedDocs` | Sin documentos relacionados |
| `RelatedDocumentNotReceived` | Documento relacionado no recibido |
| `NotMatchingReceiverNIT` | NIT del receptor no coincide |
| `InvalidCreditNoteTransaction` | Montos exceden saldo disponible |
| `InvalidRelatedDocTypeForCreditNote` | Documento relacionado distinto de CCF o Comprobante de Retención |
| `GlobalDiscountNotAllowedInAdjustment` | Descuento global o porcentaje de descuento distinto de cero |
| `IncomeRetentionNotAllowed` | Retención de renta distinta de cero |
| `MissingItemRelatedDoc` | Ítem sin referencia a documento relacionado |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/credit_note/credit_note_service.go` | Servicio principal |
| `internal/domain/dte/credit_note/credit_note_models/` | Modelos de nota de crédito |
| `internal/domain/dte/credit_note/validator/` | Estrategias de validación |
| `internal/domain/dte/dte_documents/dte_service.go` | DTEManager (balance) |

---

## Notas

1. **Dependencia del DTEManager**: Es, junto con la Nota de Débito, el DTE que requiere acceso a documentos existentes para validar.
2. **Balance**: La validación de balance ocurre en **dos momentos**: antes de la emisión (dentro de `Create`) y al registrar la transacción (post-transmisión).
3. **Invalidación reversa**: Si una nota de crédito es invalidada, se **recupera** el balance del documento original (ver [invalidation.md](invalidation.md)).
4. **Invalidación del documento original**: Mientras la nota esté vigente (`RECEIVED` o `PENDING`), el CCF o Comprobante de Retención al que ajusta no puede invalidarse.
4. **No confundir con pagos**: La nota de crédito no aplica al `PaymentTotalStrategy` (no requiere formas de pago).
