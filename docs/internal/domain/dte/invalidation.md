# Invalidación de DTE - Invalidation Service

> **Paquete:** `internal/domain/dte/invalidation`
> **Servicio:** `invalidationService`
> **Interfaz implementada:** `InvalidationManager`

## Descripción General

El servicio de invalidación gestiona la anulación de documentos tributarios electrónicos previamente emitidos y recibidos por Hacienda. La invalidación es un proceso regulado que requiere justificación y tiene restricciones temporales según el tipo de DTE.

Características principales:
- **No es un DTE en sí** — Es una operación sobre DTEs existentes
- **Restricciones temporales** — Ventana de tiempo limitada para invalidar
- **Tipos de invalidación** — Reemplazo, anulación, u otro motivo
- **Recuperación de balance** — Al invalidar notas de crédito/débito, se revierte el balance
- **Prevención de doble invalidación** — Un documento ya inválido no se puede invalidar nuevamente

---

## Estructura del Servicio

```go
type invalidationService struct {
    validator  *validator.InvalidationRulesValidator
    dteManager dte_documents.DTEManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.InvalidationRulesValidator` | Se validan reglas de negocio de invalidación |
| `dteManager` | `DTEManager` | Se accede al documento original y se actualiza su estado |

### Constructor

```go
func NewInvalidationService(dteManager dte_documents.DTEManager) InvalidationManager
```

### Interfaz `InvalidationManager`

```go
type InvalidationManager interface {
    Validate(ctx context.Context, branchID uint, document interface{}) error
    ValidateStatus(ctx context.Context, branchID uint, req interface{}) error
    InvalidateDocument(ctx context.Context, branchID uint, originalCode string) error
}
```

---

## Métodos

### `Validate`

```go
func (s *invalidationService) Validate(
    ctx context.Context,
    branchID uint,
    document interface{},
) error
```

Se valida la estructura del documento de invalidación contra las reglas de negocio. Ejecuta el `InvalidationRulesValidator` completo.

### `ValidateStatus`

```go
func (s *invalidationService) ValidateStatus(
    ctx context.Context,
    branchID uint,
    req interface{},
) error
```

Se valida el estado del documento original y, si es tipo reemplazo, el estado del documento de reemplazo:

1. **Documento original**: Debe estar en estado `RECEIVED`
2. **Documento de reemplazo** (tipo 1): Debe existir y estar en estado `RECEIVED`

#### Mapeo de Estados a Errores

| Estado del Original | Error |
|---|---|
| `RECEIVED` | Sin error (válido) |
| `INVALIDATED` | `DocumentAlreadyInvalid` |
| `REJECTED` | `DocumentReject` |
| `PENDING` | `DocumentPending` |

### `InvalidateDocument`

```go
func (s *invalidationService) InvalidateDocument(
    ctx context.Context,
    branchID uint,
    originalCode string,
) error
```

Se ejecuta la invalidación del documento:

1. Se busca el documento original por código de generación
2. Se actualiza el estado a `INVALIDATED`
3. **Si es nota de crédito o débito**: Se ejecuta `handleControlBalance()` para revertir el balance

---

## Recuperación de Balance

```go
func (s *invalidationService) handleControlBalance(
    ctx context.Context,
    branchID uint,
    originalCode string,
    doc *dte.DTEDocument,
) error
```

Cuando se invalida una nota de crédito o débito, se revierte el efecto en el balance del documento original:

### Nota de Crédito (DTE 05) Invalidada

Se extraen los montos (`TaxedSale`, `ExemptSale`, `NonSubjectSale`) de la nota de crédito y se suman de vuelta al balance del documento referenciado. Es decir, se "deshace" la reducción.

### Nota de Débito (DTE 06) Invalidada

Se extraen los montos de la nota de débito y se restan del balance del documento referenciado. Es decir, se "deshace" el incremento.

```go
dteManager.GenerateBalanceTransactionWithAmounts(
    ctx, branchID,
    transactionType,    // "CREDIT_REVERSAL" o "DEBIT_REVERSAL"
    originalDTE,
    adjustmentDTE,
    taxedSale, exemptSale, notSubjectSale,
)
```

---

## Modelo de Invalidación

### InvalidationDocument

```go
type InvalidationDocument struct {
    Identification *models.Identification
    Issuer         *models.Issuer
    Document       *InvalidatedDocument
    Reason         *InvalidationReason
}
```

### InvalidatedDocument

```go
type InvalidatedDocument struct {
    Type            string  // Tipo de DTE del documento a invalidar
    GenerationCode  string  // Código de generación UUID del documento
    ReceptionStamp  string  // Sello de recepción de Hacienda
    ControlNumber   string  // Número de control del documento
    EmissionDate    string  // Fecha de emisión original
}
```

### InvalidationReason

```go
type InvalidationReason struct {
    Type                int     // Tipo de invalidación (1, 2, 3)
    ResponsibleName     string  // Nombre del responsable
    ResponsibleDocType  string  // Tipo de documento del responsable
    ResponsibleDocNum   string  // Número de documento del responsable
    RequesterName       string  // Nombre del solicitante
    RequesterDocType    string  // Tipo de documento del solicitante
    RequesterDocNum     string  // Número de documento del solicitante
    Reason              string  // Motivo de la invalidación (requerido para tipo 3)
    ReplacementCode     string  // Código del documento de reemplazo (tipo 1 y 3)
}
```

---

## Tipos de Invalidación

| Tipo | Nombre | Descripción | ReplacementCode |
|---|---|---|---|
| `1` | Reemplazo | El documento se reemplaza por otro | **Requerido** |
| `2` | Anulación | El documento se anula completamente | **Debe ser null** |
| `3` | Otro motivo | Invalidación por otro motivo | **Requerido** |

### Reglas por Tipo

- **Tipo 1 (Reemplazo)**: Se requiere el código del documento de reemplazo. El documento de reemplazo debe existir y estar en estado `RECEIVED`.
- **Tipo 2 (Anulación)**: No debe tener código de reemplazo.
- **Tipo 3 (Otro)**: Se requiere el campo `Reason` explicando el motivo, y el código de reemplazo.

---

## Restricciones Temporales

### Para Facturas e Facturas de Exportación (DTE 01, 11)

```
Invalidación válida si: FechaInvalidación - FechaEmisión <= 90 días
```

### Para Todos los Demás DTEs

```
Invalidación válida si: FechaInvalidación - FechaEmisión <= 24 horas
```

---

## Validación del Sello de Recepción

El sello de recepción de Hacienda debe cumplir:

```
Patrón: ^[A-Z0-9]{40}$
```

Es decir, exactamente 40 caracteres alfanuméricos en mayúsculas.

---

## Estrategias de Validación

El `InvalidationRulesValidator` compone las siguientes estrategias:

| Estrategia | Propósito |
|---|---|
| `InvalidationBasicStrategy` | Campos requeridos de identificación y emisor |
| `InvalidationDocumentStrategy` | Validación del documento a invalidar |
| `InvalidationReasonStrategy` | Validación del motivo y tipo de invalidación |
| `InvalidationDateStrategy` | Restricciones temporales |

---

## Flujo Completo (incluyendo capa de aplicación)

```
Request de Invalidación
  │
  ▼
InvalidationUseCase (Application Layer)
  │
  ├── 1. Extraer contexto de autenticación
  ├── 2. invalidationService.Validate()         ← Validar estructura
  ├── 3. invalidationService.ValidateStatus()    ← Validar estados
  ├── 4. Cargar documento original
  ├── 5. Cargar info del emisor
  ├── 6. Mapear a modelo de dominio
  ├── 7. Validar documento completo
  ├── 8. Mapear a formato Hacienda
  ├── 9. Transmitir a Hacienda (sin contingencia)
  │
  └── 10. [Si éxito]:
        ├── invalidationService.InvalidateDocument()
        │     ├── Actualizar estado → INVALIDATED
        │     └── handleControlBalance() (si nota crédito/débito)
        └── Retornar resultado
```

> **Sin contingencia**: La invalidación NO soporta contingencia. Si Hacienda no está disponible, la operación falla.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| `DocumentAlreadyInvalid` | El documento ya fue invalidado |
| `DocumentReject` | El documento fue rechazado por Hacienda |
| `DocumentPending` | El documento está pendiente de procesamiento |
| `RequiredField` | Campos obligatorios faltantes |
| `InvalidPattern` | Sello de recepción con formato inválido |
| `InvalidDTETypeForInvalidation` | Tipo de DTE no válido para invalidación |
| `InvalidEnum` | Tipo de invalidación fuera de rango (1-3) |
| `InvalidDateForFEFX` | Factura fuera de ventana de 90 días |
| `InvalidDateForAllDTE` | DTE fuera de ventana de 24 horas |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/invalidation/invalidation_service.go` | Servicio principal |
| `internal/domain/dte/invalidation/models/` | Modelos de invalidación |
| `internal/domain/dte/invalidation/validator/` | Estrategias de validación |

---

## Notas

1. **Sin contingencia**: Las invalidaciones no entran en modo contingencia. Si falla la comunicación con Hacienda, se retorna error directo.
2. **Balance reversal**: La invalidación de notas de crédito/débito tiene efecto cascada en el balance del documento original.
3. **Idempotencia**: Se previene la doble invalidación verificando el estado actual.
4. **Ventanas temporales**: Considerar que las facturas tienen 90 días pero los demás DTEs solo 24 horas para invalidar.
5. **Responsable y solicitante**: Ambos campos son obligatorios y deben incluir nombre, tipo y número de documento.
