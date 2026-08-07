# Sistema de Errores del Dominio

> **Paquete:** `internal/domain/dte/common/dte_errors`
> **Paquete core:** `internal/domain/core/error`

## Descripción General

El sistema de errores del dominio proporciona tipos de error especializados que permiten diferenciar entre errores de validación, errores de negocio, y errores centinela. Todos implementan la interfaz `error` de Go.

---

## Tipos de Error

### DTEError — Error Principal

```go
type DTEError struct {
    ValidationErrors []error
    BusinessErrors   []*DTEError
    ErrorType        string
    Message          string
    Code             string
}
```

El error principal del dominio. Se puede crear de dos formas:

#### Constructor Simple

```go
func NewDTEErrorSimple(errorType string, params ...interface{}) *DTEError
```

Se crea un error con un tipo y parámetros opcionales. El `Message` se resuelve desde un mapa de constantes de error.

```go
// Ejemplo
err := NewDTEErrorSimple("InvalidIVAItemCalculation", itemNumber, expected, actual)
```

#### Constructor Compuesto

```go
func NewDTEErrorComposite(businessErrors []*DTEError) *DTEError
```

Se agrupan múltiples errores de negocio en uno solo. Se usa cuando una validación genera varios errores que se deben reportar juntos.

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Implementa `error`, retorna `Message` |
| `GetValidationErrorsString()` | `[]string` | Extrae todos los mensajes de error |
| `GetCode()` | `string` | Retorna el código de error |
| `GetMessage()` | `string` | Retorna el mensaje traducido |

---

### ValidationError — Error de Validación

```go
type ValidationError struct {
    ErrorType string
    Message   string
}
```

Se usa para errores de validación individual (un campo, una regla).

#### Constructor

```go
func NewValidationError(errorType string, params ...interface{}) *ValidationError
```

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Retorna `Message` |
| `GetType()` | `string` | Retorna `ErrorType` |

---

### CompositeError — Error Compuesto

```go
type CompositeError struct {
    Errors []error
}
```

Se agrupan múltiples errores de cualquier tipo.

#### Constructor

```go
func NewCompositeError(errors ...error) *CompositeError
```

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Todos los mensajes unidos por `;` |

---

## Errores Centinela (Core)

**Paquete:** `internal/domain/core/error/sentinels_error.go`

Errores predefinidos para condiciones específicas del negocio:

| Variable | Mensaje | Contexto |
|---|---|---|
| `ErrBranchMatrixNotFound` | — | La sucursal matriz no existe |
| `ErrAtLeastOneBranch` | — | El usuario no tiene sucursales |
| `ErrDontHaveBranchMatrix` | — | No hay sucursal tipo matriz |
| `ErrMoreThanOneBranchMatrix` | — | Más de una sucursal matriz |
| `ErrBranchMatrixWithoutAddress` | — | La sucursal matriz no tiene dirección |

---

## Constantes de Tipo de Error

Los tipos de error se definen como constantes string que se mapean a mensajes descriptivos:

### Errores de Validación General

| Tipo | Descripción |
|---|---|
| `RequiredField` | Campo obligatorio faltante |
| `RequiredFieldMissing` | Campo requerido no presente |
| `InvalidLength` | Longitud fuera de rango |
| `InvalidNumberRange` | Número fuera de rango |
| `InvalidValue` | Valor no válido |
| `InvalidEnum` | Enumeración no válida |
| `InvalidPattern` | No cumple patrón regex |
| `InvalidField` | Campo no válido |
| `RequiredField` | Campo requerido |

### Errores de Formato

| Tipo | Descripción |
|---|---|
| `InvalidNITFormat` | NIT no cumple `^([0-9]{14}\|[0-9]{9})$` |
| `InvalidDUIFormat` | DUI no cumple `^[0-9]{8}-[0-9]{1}$` |

### Errores de Ítems

| Tipo | Descripción |
|---|---|
| `ExceededItemsLimit` | Más de 2000 ítems |
| `InvalidItemNumber` | Número de ítem fuera de 1-2000 |
| `InvalidItemType` | Tipo de ítem no permitido |
| `MissingItemUnitPrice` | Precio unitario 0 con venta gravada |
| `ExcessiveItemTotal` | Total de ventas excede precio |
| `MixedSalesTypesNotAllowed` | Tipos de venta mezclados |

### Errores de IVA y Cálculos

| Tipo | Descripción |
|---|---|
| `InvalidIVAItemWithoutTaxedSale` | IVA sin venta gravada |
| `InvalidIVAItemCalculation` | Cálculo de IVA incorrecto |
| `InvalidTaxCalculation` | Cálculo de impuesto incorrecto |
| `MissingIVAInTaxes` | Falta IVA en impuestos |
| `MissingIVAForTaxedAmount` | Gravado sin IVA |
| `UnsupportedTaxCode` | Código de impuesto no soportado |
| `InvalidTaxCodeOnly20` | Solo IVA permitido para este tipo |
| `InvalidTaxForProduct` | Impuesto no permitido en producto |

### Errores de Totales

| Tipo | Descripción |
|---|---|
| `InvalidSubTotalCalculation` | Subtotal no coincide con fórmula |
| `InvalidTotalToPayCalculation` | Total a pagar incorrecto |
| `InvalidMonetaryAmount` | Más de 2 decimales |
| `DiscountExceedsSubtotal` | Descuento excede monto |
| `NegativeDiscount` | Descuento negativo |
| `InvalidSubTotal` | Subtotal incorrecto |
| `InvalidTotalOperation` | Total operación incorrecto |

### Errores de Documentos Relacionados

| Tipo | Descripción |
|---|---|
| `ExceededRelatedDocsLimit` | Más de 50 docs relacionados |
| `MixedDocumentTypesNotAllowed` | Tipos de doc mezclados |
| `InvalidRelatedDocDate` | Fecha futura |
| `InvalidRelatedDocDTEType` | Tipo de DTE no permitido |
| `MissingItemRelatedDoc` | Ítem sin doc relacionado |
| `InvalidItemRelatedDoc` | Referencia no existe |

### Errores de Contingencia

| Tipo | Descripción |
|---|---|
| `MissingContingencyType` | Tipo de contingencia faltante |
| `MissingContingencyReason` | Razón de contingencia faltante |
| `InvalidContingencyType` | Tipo de contingencia no permitido |
| `InvalidModelType` | Modelo no consistente con transmisión |

### Errores de Temporalidad

| Tipo | Descripción |
|---|---|
| `InvalidDateTime` | Fecha futura |
| `InvalidEmissionTime` | Hora futura (mismo día) |

### Errores de Pago

| Tipo | Descripción |
|---|---|
| `InvalidPaymentTypeOP2` | Efectivo en operación a crédito |
| `InvalidPaymentTerms` | Plazo/período en operación contado |
| `InvalidPaymentTermsOF` | Sin plazo/período en crédito |
| `InvalidPaymentTotal` | Suma de pagos no coincide |

### Errores de Otros Documentos

| Tipo | Descripción |
|---|---|
| `InvalidOtherDocsCount` | Más de 10 documentos |
| `InvalidAssociatedDocumentCode` | Código asociado no válido |
| `InvalidMedicalDocFields` | Campos médicos incorrectos |
| `MutuallyExclusiveFields` | NIT e Identificación simultáneos |

### Errores de Invalidación

| Tipo | Descripción |
|---|---|
| `DocumentAlreadyInvalid` | Ya fue invalidado |
| `DocumentReject` | Documento rechazado |
| `DocumentPending` | Documento pendiente |
| `InvalidDTETypeForInvalidation` | Tipo no invalidable |
| `InvalidDateForFEFX` | Factura fuera de 90 días |
| `InvalidDateForAllDTE` | DTE fuera de 24 horas |

### Errores de FSE

| Tipo | Descripción |
|---|---|
| `FSEItemInvalidEmpty` | Sin ítems |
| `FSEItemInvalidPurchase` | Purchase <= 0 |
| `FSETaxInvalidIVARetention` | Retención IVA negativa |
| `FSETaxInvalidTaxedSale` | TaxedSale != 0 |
| `FSEReceiverRequiredName` | Nombre faltante |
| `FSEReceiverInvalidNIT` | NIT inválido |

### Errores de Balance

| Tipo | Descripción |
|---|---|
| `NoRelatedDocs` | Sin docs relacionados |
| `DocumentNotReceived` | Doc original no recibido |
| `NotMatchingReceiverNIT` | NIT no coincide |
| `InvalidCreditNoteTransaction` | Excede saldo disponible |

### Errores de Retención

| Tipo | Descripción |
|---|---|
| `InvalidRetentionIVA` | Cálculo de IVA retenido incorrecto |
| `DateOutOfAllowedRange` | Fecha fuera de período |

---

## Flujo de Propagación de Errores

```
Value Object inválido
    │
    ▼
ValidateModel[T]() ──→ []error (ValidationErrors)
    │
    ▼
DTEService.Create()
    │
    ├── ValidateDTERules() ──→ *DTEError (estrategias de negocio)
    │
    ├── Se combinan ambos en DTEError compuesto
    │
    ▼
Application Layer ──→ Mapea a HTTP response
    │
    ├── DTEError → 400 Bad Request
    ├── ValidationError → 400 Bad Request
    └── error genérico → 500 Internal Server Error
```
