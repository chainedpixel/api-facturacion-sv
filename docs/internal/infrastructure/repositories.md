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
| `GetIssuerInfoByBranchID(ctx, branchID)` | Se retorna información del emisor DTE |

### Notas

- Las operaciones de escritura que involucran múltiples entidades (`Create`, `UpdateBranchOffices`) se ejecutan dentro de una transacción GORM.
- `GetIssuerInfoByBranchID` se usa para inyectar los datos del emisor en los mappers de DTE.

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
| `Create(ctx, document, transmission, status)` | Se almacena un DTE con su metadata |
| `Update(ctx, branchID, document)` | Se actualiza el estado y detalles del DTE |
| `GetByGenerationCode(ctx, branchID, generationCode)` | Se obtiene un DTE por su UUID |
| `VerifyStatus(ctx, branchID, id)` | Se consulta el estado actual de un DTE |
| `GetDTEBalanceControl(ctx, branchID, id)` | Se obtiene la información de control de balance |
| `GenerateBalanceTransaction(ctx, branchID, originalDTE, transaction)` | Se registra una transacción de balance |
| `GetTotalCount(ctx, filters)` | Se cuenta el total de documentos según filtros |
| `GetSummaryStats(ctx, filters)` | Se generan estadísticas resumidas |
| `GetPagedDocuments(ctx, filters)` | Se obtienen documentos con paginación |

### Filtros Soportados

```
DTEFilters
  ├── BranchID        → Filtra por sucursal
  ├── DTEType/DTETypes → Filtra por tipo(s) de DTE
  ├── Status           → Filtra por estado (RECEIVED, INVALIDATED, REJECTED, PENDING)
  ├── Transmission     → Filtra por tipo de transmisión (NORMAL, CONTINGENCY)
  ├── StartDate/EndDate → Rango de fechas
  ├── Page/PageSize    → Paginación
  └── IncludeAll       → Incluir todas las sucursales (admin)
```

### Control de Balance

El repositorio gestiona dos tablas relacionadas:

1. **DTEBalanceControl** — Registro principal del balance de un DTE (saldo disponible para notas de crédito/débito)
2. **DTEBalanceTransaction** — Historial de transacciones que afectan el balance (cada nota de crédito/débito genera una entrada)

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
| `UpdateAsContingency(ctx, reservationID, documentID, contingencyDocID)` | Se marca como contingencia |
| `UpdateAsContingencyByControlNumber(...)` | Se actualiza contingencia por número de control |
| `GetByDocumentID(ctx, documentID)` | Se obtiene la reservación por documento |
| `GetExpiredNonContingencyReservations(ctx)` | Se obtienen reservaciones expiradas no contingentes |

### Ciclo de Vida de Estados

```
Released → Reserved → Confirmed
                   → Rejected (con motivo y código Hacienda)
                   → Contingency (con ID de documento contingente)
```

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

Se genera el siguiente número de control secuencial para un tipo de DTE y sucursal.

### Mecanismo de Bloqueo

Se utiliza `FOR UPDATE` (row-level locking) dentro de una transacción para prevenir condiciones de carrera cuando múltiples solicitudes concurrentes intentan obtener el siguiente número:

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
| `GetPending(ctx, limit)` | Se obtienen documentos pendientes con datos relacionados (preload) |
| `UpdateBatch(ctx, ids, observations, stamps, batchID, mhBatchID, status)` | Se actualiza un lote transaccionalmente |
| `GetFirstContingencyTimestamp(ctx, branchID)` | Se obtiene el timestamp del primer documento pendiente |

### Actualización por Lote

`UpdateBatch` es una operación transaccional que:

1. Itera sobre cada ID del lote
2. Asigna el `batchID` y `mhBatchID` a cada documento
3. Actualiza las observaciones y sellos de recepción (manipulación JSON directa)
4. Cambia el estado a procesado

Si cualquier actualización falla, se revierte toda la transacción.

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
- Se utiliza por el `AdminEmailHandler` para registrar en base de datos todos los eventos de notificación antes de intentar el envío por correo.
- Ver modelo `DomainEvent` en [Base de Datos](database.md) para la estructura completa de la tabla.

---

## Notas

1. **GORM como ORM**: Todos los repositorios dependen de `*gorm.DB`. Las operaciones complejas usan transacciones explícitas con `db.Transaction()`.
2. **Preload**: Se usa `Preload()` para cargar relaciones asociadas (ej. sucursales de usuario, detalles de contingencia).
3. **Row-level locking**: El `ControlNumberRepository` usa `FOR UPDATE` para garantizar consistencia en generación de números secuenciales.
4. **Serialización JSON**: Los repositorios de fallos serializan datos complejos (requests, responses) como JSON para auditoría.
5. **Filtros dinámicos**: El `DTERepository` construye queries GORM dinámicamente según los filtros proporcionados.
