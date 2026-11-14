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
- **Ajuste de balance** — Al invalidar notas de crédito/débito, se revierte su efecto en el balance
- **Prevención de doble invalidación** — Un documento ya inválido no se puede invalidar nuevamente

---

## Estructura del Servicio

```go
type invalidationService struct {
    dteManager dte_documents.DTEManager
    deadlines  *policy.Deadlines
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `dteManager` | `DTEManager` | Se accede al documento original, se consultan notas vigentes y se actualiza su estado |
| `deadlines` | `*policy.Deadlines` | Se calculan los plazos de transmisión de la invalidación |

### Constructor

```go
func NewInvalidationService(dteManager dte_documents.DTEManager, deadlines *policy.Deadlines) InvalidationManager
```

Si `deadlines` es `nil` se usa un calendario con los asuetos nacionales únicamente. El validador de reglas (`InvalidationRulesValidator`) se crea en cada llamada a `Validate`, con `utils.TimeNow()` como instante de evaluación.

### Interfaz `InvalidationManager`

```go
type InvalidationManager interface {
    Validate(ctx context.Context, branchID uint, document *invalidation_models.InvalidationDocument) error
    ValidateStatus(ctx context.Context, branchID uint, req structs.CreateInvalidationRequest) error
    ValidateDependencies(ctx context.Context, branchID uint, doc *dte.DTEDocument) error
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
    document *invalidation_models.InvalidationDocument,
) error
```

Se valida la estructura del documento de invalidación contra las reglas de negocio. Ejecuta el `InvalidationRulesValidator` completo (cuatro estrategias, los errores se agrupan en un error compuesto).

### `ValidateStatus`

```go
func (s *invalidationService) ValidateStatus(
    ctx context.Context,
    branchID uint,
    req structs.CreateInvalidationRequest,
) error
```

Se consulta con `DTEManager.VerifyStatus` el estado del documento a invalidar y, si el tipo de invalidación no es `2` y se envió un código de reemplazo, el estado del documento de reemplazo. Solo los estados listados abajo producen error; cualquier otro estado (por ejemplo `RECEIVED`) no se rechaza en este paso.

#### Mapeo de Estados a Errores

| Estado | Error (parámetro: `document to invalidate` o `replacement document`) |
|---|---|
| `INVALIDATED` | `DocumentAlreadyInvalid` |
| `REJECTED` | `DocumentReject` |
| `PENDING` | `DocumentPending` |
| `EXPIRED` | `DocumentExpired` |
| Otros | Sin error |

### `ValidateDependencies`

```go
func (s *invalidationService) ValidateDependencies(
    ctx context.Context,
    branchID uint,
    doc *dte.DTEDocument,
) error
```

Solo aplica a documentos cuyo tipo está en `ValidAdjustmentDTETypes` (CCF `03` y Comprobante de Retención `07`); para el resto retorna `nil`. Se consultan las notas vigentes con `DTEManager.GetActiveAdjustmentNotes` y, si existe alguna, se retorna `PendingAdjustmentNotes` con el código del documento y la lista de notas. Ver sección Dependencias.

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

Cuando se invalida una nota de crédito o débito, se registra una transacción de balance sobre los documentos relacionados de tipo electrónico (`ElectronicDocument`):

1. Se extraen del JSON de la nota los documentos relacionados y los ítems (`utils.ExtractRelatedDocAndItemsFromStringJSON`).
2. Por cada ítem cuyo documento relacionado es electrónico se invoca:

```go
dteManager.GenerateBalanceTransactionWithAmounts(
    ctx, branchID,
    transactionType,           // "INVALIDATED" (nota de crédito) o "DEBIT_NOTE_INVALID" (nota de débito)
    relatedDocumentNumber,     // documento original afectado
    originalCode,              // nota invalidada
    taxedAmount, exemptAmount, notSubjectAmount,
)
```

El tipo de transacción es `constants.DocumentInvalid` (`INVALIDATED`) salvo que el documento invalidado sea una nota de débito (`NotaDebitoElectronica`), caso en el que se usa `constants.DebitNoteInvalid` (`DEBIT_NOTE_INVALID`, definida en `constants/dte_states.go`).

El efecto sobre el saldo se aplica en el modelo de persistencia (`DTEBalanceTransaction.AfterCreate`): si el tipo de transacción es `05` o `DEBIT_NOTE_INVALID` se restan los montos del saldo restante; con cualquier otro tipo (incluido `INVALIDATED`) se suman. Por tanto, la invalidación de una nota de crédito devuelve los montos al saldo del documento original, y la invalidación de una nota de débito resta los montos que esa nota había sumado.

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
    Type            document.DTEType               // json "type": tipo de DTE a invalidar
    GenerationCode  identification.GenerationCode  // json "generationCode"
    ReceptionStamp  string                         // json "receptionStamp": sello de recepción
    ControlNumber   identification.ControlNumber   // json "controlNumber"
    EmissionDate    temporal.EmissionDate          // json "emissionDate": fecha de emisión original
    DocumentType    *document.DTEType              // json "documentType": tipo de documento del receptor
    DocumentNumber  *identification.DocumentNumber // json "documentNumber"
    Name            *string                        // json "name": nombre del receptor
    Email           *base.Email                    // json "email"
    ReplacementCode *identification.GenerationCode // json "replacementCode,omitempty"
    IVAAmount       *financial.Amount              // json "IVAAmount,omitempty"
    Phone           *base.Phone                    // json "phone,omitempty"

    GenerationDateTime time.Time // json "-": fecha y hora de generación del documento
    StampDate          time.Time // json "-": fecha del sello de recepción
    IssuerActivity     string    // json "-": código de actividad del emisor
}
```

Los tres últimos campos no se serializan; los usa `InvalidationDateStrategy` para calcular plazos y validar la fecha del evento.

### InvalidationReason

```go
type InvalidationReason struct {
    Type               document.InvalidationType     // json "type": 1, 2 o 3
    ResponsibleName    string                        // json "responsibleName"
    ResponsibleDocType document.DTEType              // json "responsibleDocType"
    ResponsibleDocNum  identification.DocumentNumber // json "responsibleDocNum"
    RequesterName      string                        // json "requesterName"
    RequesterDocType   document.DTEType              // json "requesterDocType"
    RequesterDocNum    identification.DocumentNumber // json "requesterDocNum"
    Reason             *document.InvalidationReason  // json "reason": obligatorio para tipo 3
}
```

El código de reemplazo no forma parte de `InvalidationReason`: se ubica en `InvalidatedDocument.ReplacementCode`. El value object `InvalidationReason` acepta entre 5 y 250 caracteres; `InvalidationType` acepta de 1 a 3.

---

## Tipos de Invalidación

| Tipo | Nombre | Descripción | ReplacementCode |
|---|---|---|---|
| `1` | Reemplazo | El documento se reemplaza por otro | **Requerido** (nulo si el documento invalidado es `05` u `08`) |
| `2` | Anulación | El documento se anula completamente | **Debe ser null** |
| `3` | Otro motivo | Invalidación por otro motivo | **Requerido** (nulo si el documento invalidado es `05` u `08`) |

### Reglas por Tipo

- **Tipo 1 (Reemplazo)**: Se requiere el código del documento de reemplazo, salvo que el documento invalidado sea una Nota de Crédito (`05`) o un Comprobante de Liquidación (`08`) (Anexo V, campo `codigoGeneracionR`), para los cuales debe ser nulo. Si se envía, su estado se consulta en `ValidateStatus` y se rechaza si es `INVALIDATED`, `REJECTED`, `PENDING` o `EXPIRED`.
- **Tipo 2 (Anulación)**: No debe tener código de reemplazo.
- **Tipo 3 (Otro)**: Se requiere el campo `Reason` explicando el motivo, y el código de reemplazo (con la misma excepción para `05` y `08`).
- La regla del código de reemplazo para los tipos 1 y 3 la evalúa `InvalidationReasonStrategy` con `policy.RequiresReplacement(dteType)`: si se requiere y falta, `InvalidReplacementCode1And3`; si no se requiere (NC `05` o CLE `08`) y se envía, `ReplacementCodeNotAllowedForType`; el tipo 2 con código de reemplazo devuelve `InvalidInvalidationType2`.

---

## Restricciones Temporales

Calculadas por `internal/domain/dte/invalidation/policy` con el calendario de `internal/domain/dte/common/calendar`:

| Tipo de DTE | Plazo para transmitir la invalidación (contado desde el sello) | Fecha del evento (`fecEmi`/`horEmi`) |
|---|---|---|
| FE (`01`), FEXE (`11`), FSE (`14`) | Hasta las 23:59:59 del mismo día, 3 meses después (2 años para FE y FEXE de emisores con actividad 21001, 21008, 46482, 46484, 46491 o 47721) | Fecha y hora actual; no puede superar el mismo periodo contado desde la generación |
| Resto (`03`, `04`, `05`, `06`, `07`) | Hasta el décimo día hábil del mes siguiente al mes del sello | Misma fecha y hora de generación del documento (la hora actual si se generó hoy) |

- La fecha del sello es `processing_date` (`fhProcesamiento`); para documentos antiguos sin ese dato se usa la fecha de registro.
- Días hábiles: lunes a viernes sin asuetos nacionales fijos, Semana Santa (de Lunes Santo a Lunes de Pascua), ni los asuetos de `MH_EXTRA_HOLIDAYS`.

## Dependencias

`ValidateDependencies` consulta `DTEManager.GetActiveAdjustmentNotes`: si un CCF o CR tiene NC/ND en estado `RECEIVED` o `PENDING`, la invalidación se rechaza con `PendingAdjustmentNotes` y la lista de códigos. No se encadenan invalidaciones.

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
  ├── 1. Extraer contexto de autenticación (claims y token)
  ├── 2. Validar la solicitud (mapper)
  ├── 3. invalidationService.ValidateStatus()        ← Validar estados
  ├── 4. Cargar documento original
  ├── 5. invalidationService.ValidateDependencies()  ← NC/ND vigentes de un CCF o CR
  ├── 6. Cargar info del emisor
  ├── 7. Mapear a modelo de dominio (InvalidationDocument)
  ├── 8. invalidationService.Validate()              ← Validar reglas del evento
  ├── 9. Mapear a formato Hacienda
  ├── 10. Transmitir a Hacienda (sin contingencia); si el estado no es RECEIVED → TransmissionFailed
  │
  └── 11. [Si éxito]:
        ├── invalidationService.InvalidateDocument()
        │     ├── Actualizar estado → INVALIDATED
        │     └── handleControlBalance() (si nota crédito/débito)
        ├── dteManager.RegisterEvent()   ← Guarda el evento con su evidencia (JWS, sello, respuesta)
        └── Retornar el evento con su Archivo DTE
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
| `InvalidEnum` | Tipo de invalidación fuera de rango (1-3) o tipo de documento del responsable o solicitante no permitido |
| `InvalidationDeadlineExpired` | Plazo de transmisión vencido |
| `InvalidationEventDateOutOfRange` | Fecha del evento de FE/FEXE/FSE fuera del periodo desde la generación |
| `InvalidationEventDateMismatch` | Fecha del evento distinta a la generación (resto de DTE) |
| `MissingStampDate` | No se conoce la fecha del sello |
| `DocumentWithoutStamp` | El documento no tiene sello de recepción (se lanza en el mapeo de la solicitud, `pkg/mapper`, no en el dominio) |
| `PendingAdjustmentNotes` | CCF o CR con NC/ND vigentes |

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/invalidation/invalidation_service.go` | Servicio principal |
| `internal/domain/dte/invalidation/invalidation_models/` | Modelos de invalidación |
| `internal/domain/dte/invalidation/policy/` | Plazos y fecha del evento por tipo de DTE |
| `internal/domain/dte/common/calendar/` | Calendario de días hábiles |
| `internal/domain/dte/invalidation/validator/` | Estrategias de validación |

---

## Notas

1. **Sin contingencia**: Las invalidaciones no entran en modo contingencia. Si falla la comunicación con Hacienda, se retorna error directo.
2. **Balance reversal**: La invalidación de notas de crédito suma sus montos al balance del documento original y la de notas de débito los resta.
3. **Idempotencia**: Se previene la doble invalidación verificando el estado actual.
4. **Ventanas temporales**: FE, FEXE y FSE tienen 3 meses desde el sello; el resto, hasta el décimo día hábil del mes siguiente al sello.
5. **Responsable y solicitante**: Ambos campos son obligatorios y deben incluir nombre, tipo y número de documento.
