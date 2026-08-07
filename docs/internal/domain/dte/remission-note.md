# Nota de Remisión Electrónica (DTE 04) - Remission Note Service

> **Código DTE:** `04`
> **Paquete:** `internal/domain/dte/remission_note`
> **Servicio:** `remissionNoteService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Nota de Remisión es un documento que ampara el **traslado de mercadería** entre establecimientos del mismo contribuyente o hacia un tercero. No es un documento de venta directa — registra el movimiento físico de bienes.

Características principales:
- **Documento de traslado** — Ampara el movimiento físico de mercadería
- **SubTotal = TotalAmount** — No hay operaciones financieras adicionales (no hay IVA ni retenciones)
- **Receptor opcional** — Puede o no tener receptor dependiendo del tipo de traslado
- **BienTitulo** — Campo especial del receptor que indica el título del bien trasladado
- **Sin formas de pago** — No aplica el `PaymentTotalStrategy`

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
1. Cast del input → RemissionNoteInput
2. Verificar que el resumen existe (SummaryIsRequired)
3. Crear documento base (DTEDocument)
4. Ensamblar RemissionNoteModel
5. Validar reglas de negocio
6. Generar código de generación (UUID) y número de control
7. Retornar RemissionNoteModel validado
```

### Verificación de Resumen

Antes de continuar, se verifica que el resumen no sea `nil`. Si lo es, se retorna un error `SummaryIsRequired`.

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
    Items             []RemissionNoteItem
    RemissionSummary  *RemissionNoteSummary
    Receiver          *RemissionNoteReceiver    // Receptor opcional
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
    // Extiende Summary sin campos adicionales especiales
    // La diferencia clave es la regla: SubTotal = TotalAmount
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

- **Requerido** cuando existe receptor
- **Máximo 2 caracteres**
- Debe ser uno de los valores válidos (`01`-`05`, `99`)

---

## Reglas de Cálculo

### Regla Principal: SubTotal = TotalAmount

En la nota de remisión, el subtotal debe ser igual al monto total, ya que no hay transacciones financieras adicionales.

### Validación de Totales

```
Sum(item.NonSubjectSale) == Summary.TotalNonSubject    (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt        (±0.01)
Sum(item.TaxedSale)      == Summary.TotalTaxed         (±0.01)
```

### Montos No Negativos

Todos los totales deben ser `>= 0`.

---

## Reglas de Ítems

### Validación por Ítem

- Al menos un ítem requerido
- `NonSubjectSale >= 0`
- `ExemptSale >= 0`
- `TaxedSale >= 0`
- Al menos un tipo de venta debe tener valor (`> 0`)
- Los ítems deben estar numerados secuencialmente comenzando desde `1`

---

## Reglas del Receptor

### Campos Requeridos (cuando el receptor existe)

| Campo | Requerido | Validación |
|---|---|---|
| `Name` | Sí | No vacío |
| `DocumentType` | Sí | Tipo válido |
| `DocumentNumber` | Sí | Formato válido según tipo |
| `Address` | Sí | No vacío |
| `Email` | Sí | Formato válido |
| `BienTitulo` | Sí | Valor válido (01-05, 99) |

### Validación de NIT

Si `DocumentType` es `"36"` (NIT), se requiere el campo `NIT` del receptor.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido | El `interface{}` no es `*RemissionNoteInput` |
| `SummaryIsRequired` | Resumen es `nil` |
| `ValidationFailed` | Reglas de negocio no cumplidas |
| `RequiredField` | Campos obligatorios faltantes |
| `InvalidValue` | Valores fuera del rango permitido |
| `InvalidSequence` | Ítems no numerados secuencialmente |
| `TotalMismatch` | Totales no coinciden con suma de ítems |

---

## Comparación con Otros DTEs

| Aspecto | Factura/CCF | Nota de Remisión |
|---|---|---|
| **Propósito** | Venta | Traslado de mercadería |
| **IVA** | Sí | No |
| **Formas de pago** | Requeridas | No aplican |
| **Receptor** | Requerido | Opcional |
| **BienTitulo** | No existe | Campo del receptor |
| **SubTotal** | Calculado con descuentos | Igual al total |
| **Operación financiera** | Sí | No |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/remission_note/remission_note_service.go` | Servicio principal |
| `internal/domain/dte/remission_note/models/` | Modelos de nota de remisión |
| `internal/domain/dte/remission_note/validator/` | Estrategias de validación |

---

## Notas

1. **No es venta**: La nota de remisión ampara traslado de mercadería, no una transacción comercial.
2. **Sin PaymentTotalStrategy**: Al no ser una operación de venta, no se aplican validaciones de formas de pago.
3. **BienTitulo**: Es un campo exclusivo de la nota de remisión que no existe en otros DTEs.
4. **Receptor condicional**: La presencia del receptor depende del tipo de traslado, pero si existe, sus campos son obligatorios.
5. **Numeración secuencial**: Los ítems deben estar numerados `1, 2, 3...` sin saltos.
