# Nota de Remisión Electrónica (DTE 04) - Remission Note Service

> **Código DTE:** `04`
> **Paquete:** `internal/domain/dte/remission_note`
> **Servicio:** `remissionNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Remisión es un documento que ampara el **traslado de mercadería** entre establecimientos del mismo contribuyente o hacia un tercero. No es un documento de venta directa — registra el movimiento físico de bienes.

Características principales:
- **Documento de traslado** — Ampara el movimiento físico de mercadería
- **Sin retenciones ni cargos financieros** — El resumen no define campos de retención; el monto total se valida como subtotal más los tributos del resumen
- **Receptor requerido** — `RemissionNoteReceiverStrategy` rechaza el documento si no hay receptor (`RequiredField` con `receiver`)
- **BienTitulo** — Campo especial del receptor que indica el título del bien trasladado
- **Documentos relacionados** — Los tipos permitidos (`ValidRemissionNoteDTETypesRelateDoc`) son `01`, `03` y `11`; ver [constants.md](../constants.md)

---

## Estructura del Servicio

```go
type remissionNoteService struct {
    validator        *validator.RemissionNoteRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
    dteManager       dte_documents.DTEManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.RemissionNoteRulesValidator` | Se validan reglas de negocio de nota de remisión |
| `seqNumberManager` | `SequentialNumberManager` | Se gestiona la reserva de números de control |
| `dteManager` | `DTEManager` | Se accede a servicios de gestión de DTE |

### Constructor

```go
func NewRemissionNoteService(
    seqNumberManager dte_documents.SequentialNumberManager,
    dteManager dte_documents.DTEManager,
) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *remissionNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → *RemissionNoteInput
2. Verificar que el resumen existe (SummaryIsRequired)
3. Crear documento base (DTEDocument)
4. Ensamblar RemissionNoteModel
5. Validar reglas específicas (RemissionNoteRulesValidator)
6. Validar reglas genéricas de DTE (ValidateDTEDocument)
7. Generar código de generación (UUID) y número de control
8. Retornar RemissionNoteModel validado
```

El `RemissionNoteRulesValidator` compone tres estrategias: `RemissionNoteItemStrategy`, `RemissionNoteSummaryStrategy` y `RemissionNoteReceiverStrategy`.

### Verificación de Resumen

Antes de continuar, se verifica que `RemissionSummary` no sea `nil`. Si lo es, se retorna un error `SummaryIsRequired`.

---

## Modelos Específicos de Nota de Remisión

### RemissionNoteModel

```go
type RemissionNoteModel struct {
    *models.DTEDocument
    RemissionItems []RemissionNoteItem
    Summary        *RemissionNoteSummary
}
```

### RemissionNoteInput

```go
type RemissionNoteInput struct {
    *models.InputDataCommon
    Receiver         *RemissionNoteReceiver
    Items            []RemissionNoteItem
    RemissionSummary *RemissionNoteSummary
    Extension        interfaces.Extension // remanente sin uso: no se recibe en el request ni se envía a Hacienda
    Appendixes       []models.Appendix
    ThirdPartySale   interfaces.ThirdPartySale
}
```

### RemissionNoteItem

```go
type RemissionNoteItem struct {
    *models.Item
    NonSubjectSale financial.Amount  // Venta no sujeta
    ExemptSale     financial.Amount  // Venta exenta
    TaxedSale      financial.Amount  // Venta gravada
}
```

### RemissionNoteSummary

```go
type RemissionNoteSummary struct {
    *models.Summary
    NonSubjectTotal    *financial.Amount
    ExemptTotal        *financial.Amount
    TaxedTotal         *financial.Amount
    SubtotalSales      *financial.Amount
    NonSubjectDiscount *financial.Amount
    ExemptDiscount     *financial.Amount
    TaxedDiscount      *financial.Amount
    DiscountPercent    *float64
    TotalDiscount      *financial.Amount
    Subtotal           *financial.Amount
    TotalAmount        *financial.Amount
    PaymentTypes       []interfaces.PaymentType
}
```

Los campos de totales son opcionales (punteros): las validaciones de coherencia solo se aplican a los que se envían. El `Summary` base embebido es el que se asigna al `DTEDocument` (`GetTotalTaxes()` proviene de él).

```go
// Receptor
type RemissionNoteReceiver struct {
    *models.Receiver
    BienTitulo *string
}
```

### RemissionNoteReceiver

```go
type RemissionNoteReceiver struct {
    *models.Receiver
    BienTitulo *string  // Título del bien trasladado
}
```

---

## Campo BienTitulo

El campo `BienTitulo` indica el tipo de título bajo el cual se traslada la mercadería:

| Valor | Significado |
|---|---|
| `01` | Propiedad |
| `02` | Consignación |
| `03` | Depósito |
| `04` | Exhibición |
| `05` | Reparación |
| `99` | Otro |

### Validaciones de BienTitulo

- **Obligatorio** (`RequiredBienTitulo`)
- **Máximo 2 caracteres** (`InvalidLength`)
- Debe ser uno de los valores válidos (`01`-`05`, `99`; `InvalidBienTitulo`)

---

## Reglas de Cálculo

Las fórmulas generales se documentan en [calculations.md](../../../api/calculations.md).

### Monto total de la operación

Cuando se envían `Subtotal` y `TotalAmount`, el monto total debe ser el subtotal más la suma de los tributos del resumen (`InvalidTotalOperation`, regla compartida `ValidateTotalOperation`).

### Validación de Totales

Cada total del resumen que se envía debe coincidir con la suma de los ítems (tolerancia de 0.01):

```
Sum(item.NonSubjectSale) == Summary.NonSubjectTotal    (InvalidTotalNonSubject)
Sum(item.ExemptSale)     == Summary.ExemptTotal        (InvalidTotalExempt)
Sum(item.TaxedSale)      == Summary.TaxedTotal         (InvalidTotalTaxed)
```

### Montos No Negativos

Los totales (no sujeto, exento, gravado y monto total) deben ser `>= 0` (`InvalidAmount`).

---

## Reglas de Ítems

### Validación por Ítem

- Al menos un ítem requerido (`RequiredField` con `items`)
- `NonSubjectSale >= 0`, `ExemptSale >= 0` y `TaxedSale >= 0` (`InvalidAmount`)
- La suma de los tres tipos de venta debe ser distinta de cero (`AtLeastOneSaleTypeRequired`)
- Los ítems deben estar numerados secuencialmente comenzando desde `1` (`InvalidSequence`)

---

## Reglas del Receptor

### Campos Requeridos

| Campo | Requerido | Validación |
|---|---|---|
| `Name` | Sí | No vacío |
| `DocumentType` | Sí | No vacío |
| `DocumentNumber` | Sí | No vacío |
| `Address` | Sí | Presente |
| `Email` | Sí | No vacío |
| `BienTitulo` | Sí | Valor válido (01-05, 99) |

Además, el receptor debe ser de tipo `*RemissionNoteReceiver` (`InvalidRemissionNoteReceiver`).

### Validación de NIT

Si `DocumentType` es `"36"` (NIT), se requiere el campo `NIT` del receptor.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Pánico por cast (cast sin comprobación) | El `interface{}` no es `*RemissionNoteInput` |
| `SummaryIsRequired` | `RemissionSummary` es `nil` |
| `ValidationFailed` | Reglas de negocio no cumplidas (envuelve el error compuesto) |
| `RequiredField` | Campos obligatorios faltantes (ítems, receptor, nombre, documento, dirección, correo, NIT) |
| `InvalidAmount` | Monto negativo |
| `AtLeastOneSaleTypeRequired` | Ítem sin ningún tipo de venta |
| `InvalidSequence` | Ítems no numerados secuencialmente |
| `InvalidTotalNonSubject`, `InvalidTotalExempt`, `InvalidTotalTaxed` | Totales no coinciden con la suma de ítems |
| `InvalidTotalOperation` | Monto total distinto de subtotal más tributos |
| `RequiredBienTitulo`, `InvalidBienTitulo`, `InvalidLength` | `BienTitulo` ausente o inválido |
| `InvalidRemissionNoteReceiver` | El receptor no es de tipo nota de remisión |

---

## Comparación con Otros DTEs

| Aspecto | Factura/CCF | Nota de Remisión |
|---|---|---|
| **Propósito** | Venta | Traslado de mercadería |
| **IVA** | Sí | No hay validación de IVA en el resumen |
| **Formas de pago** | Requeridas | `PaymentTotalStrategy` omite NC, ND y NRE |
| **Receptor** | Requerido | Requerido (con `BienTitulo`) |
| **BienTitulo** | No existe | Campo del receptor |
| **Monto total** | Según el tipo de documento | Subtotal más tributos del resumen |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/remission_note/remission_note_service.go` | Servicio principal |
| `internal/domain/dte/remission_note/remission_note_models/` | Modelos de nota de remisión |
| `internal/domain/dte/remission_note/validator/` | Estrategias de validación |

---

## Notas

1. **No es venta**: La nota de remisión ampara traslado de mercadería, no una transacción comercial.
2. **Sin validación de formas de pago**: `PaymentTotalStrategy` (validación genérica) omite la nota de remisión, y sus estrategias propias tampoco validan formas de pago.
3. **BienTitulo**: Es un campo exclusivo de la nota de remisión que no existe en otros DTEs.
4. **Receptor obligatorio**: Sin receptor, la validación falla con `RequiredField`; con receptor, todos sus campos listados son obligatorios.
5. **Numeración secuencial**: Los ítems deben estar numerados `1, 2, 3...` sin saltos.
