# Repositorios — Adaptadores de Base de Datos

> **Paquete:** `internal/infrastructure/adapters/repositories`

## Descripción General

Los repositorios implementan las interfaces (ports) definidas en la capa de dominio, proporcionando la persistencia real a través de GORM (ORM para Go). Cada repositorio encapsula las operaciones SQL necesarias sin exponer detalles de la base de datos a las capas superiores.

---

## AuthRepository

> **Archivo:** `auth_repository.go`
> **Implementa:** `auth.AuthRepositoryPort`

### Estructura

```go
type AuthRepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `GetByNIT(ctx, nit)` | Se busca un usuario por su NIT |
| `GetByBranchApiKey(ctx, apiKey)` | Se busca un usuario a través del API key de su sucursal |
| `GetBranchByBranchApiKey(ctx, apiKey)` | Se obtiene una sucursal por su API key |
| `GetBranchByBranchID(ctx, branchID)` | Se obtiene una sucursal por su ID |
| `GetByBranchID(ctx, branchID)` | Se busca un usuario por el ID de su sucursal |
| `GetMatrixBranch(ctx, userID)` | Se obtiene la casa matriz del usuario |
| `Create(ctx, user)` | Se crea un usuario con todas sus sucursales (transaccional) |
| `Update(ctx, user)` | Se actualiza la información del usuario |
| `UpdateBranchOffices(ctx, userID, branches)` | Se actualizan sucursales (transaccional) |
| `DeleteBranchOffice(ctx, userID, branchID)` | Se elimina una sucursal |
| `GetAuthTypeByApiKey(ctx, apiKey)` | Se obtiene el tipo de autenticación por API key |
| `GetAuthTypeByNIT(ctx, nit)` | Se obtiene el tipo de autenticación por NIT |
| `GetIssuerInfoByBranchID(ctx, branchID)` | Se retorna información del emisor DTE (`IssuerDTE`) |

### Notas

- Las operaciones de escritura que involucran múltiples entidades (`Create`, `UpdateBranchOffices`) se ejecutan dentro de una transacción GORM.
- `GetIssuerInfoByBranchID` se usa para inyectar los datos del emisor en los mappers de DTE. Aplica estas reglas: correo y teléfono de la sucursal, y si son nulos se usan los del usuario; si la sucursal no tiene dirección se toma la de la casa matriz (`GetMatrixBranch`); `POSCode` y `EstablishmentCode` internos se truncan a 4 caracteres. Se devuelven también `EstablishmentCodeMH` y `POSCodeMH` (códigos asignados por Hacienda).
- `GetMatrixBranch` busca la sucursal del usuario con `establishment_type` de casa matriz y precarga su dirección.

---

## DTERepository

> **Archivo:** `dte_repository.go`
> **Implementa:** `dte_documents.DTERepositoryPort`

### Estructura

```go
type DTERepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `Create(ctx, document, transmission, status, evidence)` | Se almacena un DTE con su metadata y, si se recibe, la evidencia de transmisión |
| `Update(ctx, branchID, document)` | Se actualiza el estado, sello y evidencia del DTE (`dte_details`) y `updated_at` de `dte_documents` |
| `GetByGenerationCode(ctx, branchID, generationCode)` | Se obtiene un DTE por su UUID, incluidos `signed_document`, `processing_date` y `hacienda_response` |
| `GetActiveAdjustmentNotes(ctx, branchID, originalDTE)` | Se obtienen los códigos de generación de las notas de crédito y débito vigentes del documento |
| `VerifyStatus(ctx, branchID, id)` | Se consulta el estado actual de un DTE |
| `GetDTEBalanceControl(ctx, branchID, id)` | Se obtiene la información de control de balance |
| `GenerateBalanceTransaction(ctx, branchID, originalDTE, transaction)` | Se registra una transacción de balance |
| `GetTotalCount(ctx, filters)` | Se cuenta el total de documentos según filtros |
| `GetSummaryStats(ctx, filters)` | Se generan estadísticas resumidas |
| `GetPagedDocuments(ctx, filters)` | Se obtienen documentos con paginación |

### Evidencia de Transmisión

`Create` valida que el documento serializado tenga código de generación, número de control y tipo; si falta alguno devuelve `MissingDTEIdentification`. La evidencia (`dte.TransmissionEvidence`) se copia a `dte_details` con `applyEvidence`:

| Campo de evidencia | Columna de `dte_details` |
|---|---|
| `ReceptionStamp` | `reception_stamp` |
| `SignedDocument` | `signed_document` (JWS transmitido) |
| `ProcessingDate` | `processing_date` |
| `HaciendaResponse` | `hacienda_response` |

Las cadenas vacías se guardan como `NULL` (`optionalString`). `Update` hace lo mismo con `signed_document`, `processing_date` y `hacienda_response`. Ver [database.md](database.md).

### GetActiveAdjustmentNotes

Devuelve, sin repetidos y ordenados por código, los `adjustment_document_id` de `dte_balance_transactions` asociados al `dte_balance_control` del documento original (por sucursal y `original_dte_id`), cuya fila en `dte_details` sea de tipo nota de crédito (`05`) o nota de débito (`06`) y tenga estado `RECEIVED` o `PENDING`. Las notas rechazadas, invalidadas o vencidas no se consideran activas.

### Filtros Soportados

```
DTEFilters
  ├── BranchID        → Filtra por sucursal
  ├── DTEType/DTETypes → Filtra por tipo(s) de DTE
  ├── Status           → Filtra por estado (RECEIVED, INVALIDATED, REJECTED, PENDING, EXPIRED)
  ├── Transmission     → Filtra por tipo de transmisión (NORMAL, CONTINGENCY)
  ├── StartDate/EndDate → Rango de fechas
  ├── Page/PageSize    → Paginación
  └── IncludeAll       → Incluir todas las sucursales (admin)
```

`GetSummaryStats` cuenta por estado `RECEIVED`, `INVALIDATED`, `REJECTED` y `PENDING`, y por transmisión `NORMAL` y `CONTINGENCY`; los documentos `EXPIRED` suman al `Total` pero no tienen contador propio. `GetPagedDocuments` ordena por `dte_documents.created_at` descendente.

### Control de Balance

El repositorio gestiona dos tablas relacionadas:

1. **DTEBalanceControl** (`dte_balance_control`) — Registro principal del balance de un DTE (saldo disponible para notas de crédito/débito). Se crea con el hook `AfterCreate` de `DTEDocument`.
2. **DTEBalanceTransaction** (`dte_balance_transactions`) — Historial de transacciones que afectan el balance (cada nota de crédito/débito genera una entrada). `GenerateBalanceTransaction` bloquea la fila del control con `SELECT ... FOR UPDATE` dentro de una transacción.

---

## DTEEventRepository

> **Archivo:** `dte_event_repository.go`
> **Implementa:** `dte_documents.DTEEventRepositoryPort`

Persiste en la tabla `dte_events` los eventos transmitidos a Hacienda (invalidación y contingencia) con su evidencia.

| Método | Descripción |
|---|---|
| `Create(ctx, event)` | Se inserta el evento (`generation_code`, sucursal, tipo, documento relacionado y de reemplazo, `json_data`, `signed_document`, sello, fecha de procesamiento, respuesta de Hacienda, estado). `created_at` se fija con la hora actual |
| `ListByRelatedDTE(ctx, branchID, relatedDTEID)` | Se listan los eventos de una sucursal cuyo `related_dte_id` coincide, ordenados por `created_at` ascendente |

Los eventos de contingencia se guardan con `related_dte_id` nulo, por lo que `ListByRelatedDTE` solo devuelve eventos de invalidación. Lo consume `GET /api/v1/dte/{id}/eventos`.

---

## ReservedSequenceRepository

> **Archivo:** `reserved_sequence_repository.go`
> **Implementa:** `dte_documents.ReservedSequenceRepositoryPort`

### Estructura

```go
type ReservedSequenceRepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `Create(ctx, reservation)` | Se crea un nuevo registro de reservación |
| `GetOldestReleasedNumber(ctx, branchID, dteType, year)` | Se obtiene el número liberado más antiguo |
| `MarkAsReserved(ctx, reservationID, expiresAt)` | Se marca un número liberado como reservado |
| `UpdateStatus(ctx, branchID, dteType, seqNum, year, status, timestamp)` | Se actualiza el estado de una reservación |
| `UpdateStatusWithReason(ctx, ..., reason, haciendaCode)` | Se actualiza con motivo de rechazo |
| `UpdateStatusWithDocumentID(ctx, ..., documentID)` | Se asocia un documento a la reservación |
| `UpdateAsContingency(ctx, reservationID, documentID, contingencyDocID)` | Se marca como contingencia (`is_contingency = true`) |
| `UpdateAsContingencyByControlNumber(...)` | Se actualiza contingencia por número de control |
| `GetByDocumentID(ctx, documentID)` | Se obtiene la reservación por documento |
| `GetExpiredNonContingencyReservations(ctx)` | Se obtienen reservaciones expiradas no contingentes |

### Ciclo de Vida de Estados

Estados (`status`): `RESERVED`, `CONFIRMED` y `RELEASED`.

```
RESERVED ──► CONFIRMED  (documento aceptado; se asocia document_id y confirmed_at)
RESERVED ──► RELEASED   (rechazo, fallo o expiración; released_at, motivo y código de Hacienda opcionales)
RELEASED ──► RESERVED   (se reutiliza el correlativo liberado más antiguo, MarkAsReserved)
```

La contingencia no es un estado: se marca con el flag `is_contingency` y `contingency_document_id`.

- `GetOldestReleasedNumber` devuelve el correlativo `RELEASED` más antiguo (por `released_at`) de la sucursal, tipo y año, que no tenga código de Hacienda asociado.
- `GetExpiredNonContingencyReservations` devuelve las reservas en `RESERVED`, no contingentes, con `expires_at` vencido. Las consume el `ReservationCleanerJob` (ver [jobs.md](jobs.md)).

---

## ControlNumberRepository

> **Archivo:** `sequential_number_repository.go`
> **Implementa:** `ports.SequentialNumberRepositoryPort`

### Estructura

```go
type ControlNumberRepository struct {
    db *gorm.DB
}
```

### Método Principal

```go
func (r *ControlNumberRepository) GetNext(ctx, dteType, branchID) (int, error)
```

Se genera el siguiente número de control secuencial para un tipo de DTE, sucursal y año en curso (`control_number_sequences`, índice único por sucursal, tipo y año). Si no existe la fila del año, se crea con `last_number = 1`.

### Mecanismo de Bloqueo

Dentro de una transacción se lee la fila con `tx.Set("gorm:query_option", "FOR UPDATE")` para prevenir condiciones de carrera cuando múltiples solicitudes concurrentes intentan obtener el siguiente número:

```
BEGIN TRANSACTION
  → SELECT ... FOR UPDATE (bloquea la fila)
  → INCREMENT sequence
  → UPDATE sequence
COMMIT
```

---

## FailedSequenceNumberRepository

> **Archivo:** `failed_sequence_number_repository.go`
> **Implementa:** `ports.FailedSequenceNumberRepositoryPort`

### Estructura

```go
type FailedSequenceNumberRepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `RegisterFailedSequence(ctx, branchID, dteType, seqNum, year, reason, code, requestData, mhResponse)` | Se registra un intento fallido con detalles completos |
| `GetFailedSequences(ctx, branchID, dteType, limit)` | Se obtienen los fallos más recientes |
| `GetFailedSequencesByYear(ctx, branchID, dteType, year, limit)` | Se filtran fallos por año |

### Datos Almacenados

- `originalRequestData` — JSON del request original (serializado)
- `mhResponse` — JSON de la respuesta de Hacienda (serializado)
- `failureReason` — Motivo legible del fallo
- `responseCode` — Código de respuesta de Hacienda

---

## ContingencyRepository

> **Archivo:** `contingency_repository.go`
> **Implementa:** `contingency.ContingencyRepositoryPort`

### Estructura

```go
type ContingencyRepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `Create(ctx, doc)` | Se almacena un documento de contingencia con UUID generado automáticamente |
| `GetPending(ctx, limit)` | Se obtienen documentos con `dte_details.status = PENDING` y datos relacionados (preload de documento, sucursal, usuario y dirección), ordenados por vencimiento |
| `UpdateBatch(ctx, ids, observations, results, batchID, mhBatchID, status)` | Se actualiza un lote transaccionalmente |
| `MarkEventStamped(ctx, ids, stampedAt)` | Se guarda `event_stamped_at` en los documentos indicados, solo si estaba nulo |
| `GetFirstContingencyTimestamp(ctx, branchID)` | Se obtiene el `created_at` más antiguo de los documentos pendientes de la sucursal (o nulo si no hay) |

### Orden de GetPending

Las consultas de contingencia se ordenan por cercanía al vencimiento de la ventana de 72 horas:

1. Primero los documentos con `event_stamped_at` informado (`ORDER BY event_stamped_at IS NULL`), es decir, los que ya tienen el reloj de 72 horas en curso.
2. Luego por `event_stamped_at` ascendente (el más próximo a vencer primero).
3. Por último por `created_at` ascendente.

Los documentos sin sello de evento (aún no se envió el evento) quedan al final.

### Actualización por Lote

`UpdateBatch` es una operación transaccional que, por cada ID de `contingency_documents`:

1. Actualiza `batch_id`, `mh_batch_id` y, si hay, `observations` (la observación del mismo índice).
2. Lee el registro para obtener su `document_id`.
3. Actualiza `dte_details.status` con el estado recibido (por ejemplo `RECEIVED`, `REJECTED` o `EXPIRED`).
4. Si `results[id]` trae sello de recepción, guarda `reception_stamp` y, si viene, `processing_date`. La estructura firmada no se altera: el sello se guarda aparte y se agrega como `selloRecibido` al armar el Archivo DTE.

Si cualquier actualización falla (o hay un pánico), se revierte toda la transacción. El mismo método se usa para marcar documentos vencidos (`status = EXPIRED`, lote `expired`) y verificados (lote `verified`).

### MarkEventStamped

```sql
UPDATE contingency_documents
SET event_stamped_at = ?, updated_at = ?
WHERE id IN (...) AND event_stamped_at IS NULL
```

La condición `event_stamped_at IS NULL` garantiza que un reenvío del evento no reinicie la ventana de transmisión.

---

## EventRepository

> **Archivo:** `event_repository.go`
> **Implementa:** `event.Repository`

### Estructura

```go
type EventRepository struct {
    db *gorm.DB
}
```

### Métodos

| Método | Descripción |
|---|---|
| `Save(ctx, eventType, userID, branchID, payloadJSON, occurredAt)` | Se persiste un evento de dominio en la tabla `domain_events` |

### Notas

- Cada llamada a `Save` crea un nuevo registro en `DomainEvent` con el payload JSON del evento serializado.
- Se utiliza desde el handler de notificaciones de eventos de dominio para registrar en base de datos los eventos antes de intentar el envío por correo (ver [domain-events.md](domain-events.md)).
- Ver modelo `DomainEvent` en [Base de Datos](database.md) para la estructura completa de la tabla.

---

## Notas

1. **GORM como ORM**: Todos los repositorios dependen de `*gorm.DB`. Las operaciones complejas usan transacciones explícitas con `db.Transaction()`.
2. **Preload**: Se usa `Preload()` para cargar relaciones asociadas (ej. sucursales de usuario, detalles de contingencia).
3. **Row-level locking**: El `ControlNumberRepository` (opción `gorm:query_option`) y `GenerateBalanceTransaction` (`clause.Locking`) usan `FOR UPDATE` para garantizar consistencia.
4. **Serialización JSON**: Los repositorios de fallos serializan datos complejos (requests, responses) como JSON para auditoría.
5. **Filtros dinámicos**: El `DTERepository` construye queries GORM dinámicamente según los filtros proporcionados.
