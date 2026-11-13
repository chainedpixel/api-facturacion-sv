# Sistema de Números Secuenciales

> **Paquete:** `internal/domain/dte/dte_documents`
> **Servicio:** `sequentialNumberService`
> **Interfaz:** `SequentialNumberManager`

## Descripción General

El sistema de números secuenciales gestiona el ciclo de vida de los números de control para los DTEs. Cada documento tributario debe tener un número de control único y secuencial, asignado antes de la transmisión. El sistema implementa un mecanismo de **reserva** para evitar conflictos en escenarios concurrentes.

---

## Estructura del Servicio

```go
type sequentialNumberService struct {
    sequentialRepo  ports.SequentialNumberRepositoryPort
    authRepo        auth.AuthRepositoryPort
    reservationRepo ReservedSequenceRepositoryPort
}
```

### Dependencias

| Dependencia | Propósito |
|---|---|
| `sequentialRepo` | `ports.SequentialNumberRepositoryPort`: se obtiene el siguiente número secuencial (`GetNext(ctx, dteType, branchID)`); el repositorio mantiene un contador por sucursal, tipo de DTE y año calendario actual |
| `authRepo` | `auth.AuthRepositoryPort`: se obtiene el usuario de la sucursal (`YearInDTE`) |
| `reservationRepo` | `ReservedSequenceRepositoryPort`: se gestiona el ciclo de vida de las reservas |

Constructor: `NewSequentialNumberService(sequentialRepo, authRepo, reservationRepo) SequentialNumberManager`.

---

## Formato del Número de Control

```
DTE-{TipoDTE}-{CódigoEstablecimiento}{CódigoPOS}-{Secuencia}
```

Si el código de establecimiento o de punto de venta es `nil`, se usa `"0000"`. El value object `ControlNumber` valida el patrón `^DTE-[0-9]{2}-[A-Z0-9]{8}-[0-9]{15}$`.

### Con Año (cuando `user.YearInDTE = true`)

```
DTE-XX-XXXX-YYYYNNNNNNNNNNN
```

**Ejemplo:** `DTE-03-00010001-2024000000001`

### Sin Año

```
DTE-XX-XXXX-NNNNNNNNNNNNNNN
```

**Ejemplo:** `DTE-03-00010001-000000000000001`

### Parsing del Número de Control

```go
func (m *sequentialNumberService) parseControlNumber(controlNumber string) (dteType string, seqNum uint, year int, err error)
```

Se descompone un número de control con el patrón `^DTE-(\d{2})-([A-Z0-9]+)-(\d{15})$`. Si los primeros cuatro dígitos de la parte numérica forman un año entre 2020 y 2099, se interpretan como año y el resto como secuencia; en caso contrario toda la parte numérica es la secuencia y el año es el actual.

---

## Ciclo de Vida de una Reserva

```
                         ┌─────────────┐
                         │   RELEASED  │
                         │ (disponible │
                         │  para reuso)│
                         └──────┬──────┘
                                │ (reutilizado)
                                ▼
┌──────────┐  reservar   ┌─────────────┐  confirmar   ┌─────────────┐
│ (nuevo)  │────────────→│  RESERVED   │─────────────→│  CONFIRMED  │
└──────────┘             │ (bloqueado) │              │  (final)    │
                         └──────┬──────┘              └─────────────┘
                                │
                                │ rechazar/liberar
                                ▼
                         ┌─────────────┐
                         │   RELEASED  │
                         │ (registra   │
                         │  motivo)    │
                         └─────────────┘
```

### Estados

| Estado | Descripción | Siguiente Estado |
|---|---|---|
| `RESERVED` | Número reservado, bloqueado para uso | `CONFIRMED` o `RELEASED` |
| `CONFIRMED` | Documento transmitido exitosamente | (estado final) |
| `RELEASED` | Número liberado para reuso | `RESERVED` (si se reutiliza) |

---

## Métodos

### `GetNextControlNumber`

```go
func GetNextControlNumber(ctx, dteType, branchID, posCode, establishmentCode *string) (string, error)
```

Se obtiene el siguiente número de control. Es el método que usan los servicios de cada DTE y **sí crea una reserva**: aplica los valores por defecto (`"0000"`) y delega en `ReserveNextNumber(..., "", false)`, descartando el identificador de la reserva.

### `ReserveNextNumber`

```go
func ReserveNextNumber(ctx, dteType, branchID, posCode, establishmentCode *string, documentData string, isContingency bool) (string, uint, error)
```

Se reserva el siguiente número disponible y se retorna el número de control y el identificador de la reserva. El flujo es:

1. **Obtener el usuario** de la sucursal (para `YearInDTE`)
2. **Buscar números liberados** — Se toma el número liberado más antiguo del año actual, del mismo tipo y sucursal, **sin código de Hacienda** (`hacienda_code` vacío); si se reutiliza se marca `RESERVED` con expiración de 1 hora (sin expiración si `isContingency`). Si otra solicitud lo tomó antes, se genera uno nuevo
3. **Si no hay liberados** — Se obtiene un nuevo número secuencial (`sequentialRepo.GetNext`)
4. **Crear la reserva** — Se registra con estado `RESERVED`, año actual y `ExpiresAt` de 1 hora (nulo si `isContingency`)

**Parámetros especiales:**
- `documentData` — Se ignora (reservado para trazabilidad)
- `isContingency` — Si es `true`, la reserva no tiene fecha de expiración

Los números liberados por un rechazo de Hacienda (con código) no se reutilizan.

### `ConfirmReservation`

```go
func ConfirmReservation(ctx, controlNumber, documentID, branchID) error
```

Se confirma la reserva cuando el documento fue transmitido exitosamente a Hacienda. Se actualiza:
- Estado → `CONFIRMED`
- `DocumentID` → ID del documento
- `ConfirmedAt` → Timestamp actual

### `ReleaseReservation`

```go
func ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode, branchID) error
```

Se libera la reserva cuando el documento fue rechazado por Hacienda. Se actualiza:
- Estado → `RELEASED`
- `RejectionReason` → Motivo del rechazo
- `HaciendaCode` → Código de respuesta de Hacienda
- `ReleasedAt` → Timestamp actual
- `IsContingency` → `false` y `ContingencyDocumentID` → vacío

La expiración de 1 hora se asigna al volver a reservar el número, no al liberarlo.

### `ConfirmReservationByDocumentID`

```go
func ConfirmReservationByDocumentID(ctx, documentID) error
```

Se confirma la reserva buscando por ID del documento (útil en contingencia). Solo actualiza estado y fecha de confirmación.

### `ReleaseReservationByDocumentID`

```go
func ReleaseReservationByDocumentID(ctx, documentID, rejectionReason, haciendaCode) error
```

Se libera la reserva buscando por ID del documento.

### `MarkReservationAsContingency`

```go
func MarkReservationAsContingency(ctx, reservationID, documentID, contingencyDocID) error
```

Se marca una reserva como contingencia cuando el documento entra en modo contingencia (`ReservedSequenceRepositoryPort.UpdateAsContingency`):
- `IsContingency` → `true`
- `ContingencyDocumentID` → ID del documento de contingencia
- Se elimina la expiración de la reserva (según el adaptador de persistencia)

### `MarkReservationAsContingencyByControlNumber`

```go
func MarkReservationAsContingencyByControlNumber(ctx, controlNumber, documentID, contingencyDocID, branchID) error
```

Misma funcionalidad pero buscando por número de control. `ContingencyService` la usa al almacenar un documento en contingencia.

---

## Modelo de Reserva

```go
type ReservedSequence struct {
    ID                    uint
    BranchID              uint
    DTEType               string
    SequenceNumber        uint
    Year                  int
    Status                string      // ReservationStatusReserved, ReservationStatusConfirmed, ReservationStatusReleased
    DocumentID            *string
    ReservedAt            time.Time
    ConfirmedAt           *time.Time
    ReleasedAt            *time.Time
    ExpiresAt             *time.Time
    RejectionReason       *string
    HaciendaCode          *string
    IsContingency         bool
    ContingencyDocumentID *string
}
```

---

## Escenarios

### Flujo Normal (Exitoso)

```
1. GetNextControlNumber() → "DTE-01-00010001-2024000000042" [RESERVED] (con YearInDTE)
2. Transmitir a Hacienda → Éxito
3. ConfirmReservation() → [CONFIRMED]
```

### Flujo con Rechazo

```
1. GetNextControlNumber() → "DTE-01-00010001-2024000000042" [RESERVED]
2. Transmitir a Hacienda → Rechazado
3. ReleaseReservation() → [RELEASED] con motivo y código de Hacienda
   (un número liberado con código de Hacienda no se reutiliza)
```

### Flujo con Contingencia

```
1. GetNextControlNumber() → "DTE-01-00010001-2024000000042" [RESERVED]
2. Transmitir a Hacienda → Error de red
3. MarkReservationAsContingencyByControlNumber() → [RESERVED, isContingency=true]
4. (Retransmisión posterior)
5. ConfirmReservationByDocumentID() → [CONFIRMED]
   o ReleaseReservationByDocumentID() → [RELEASED] si Hacienda rechaza
```

---

## Notas

1. **Reutilización**: Los números liberados sin código de Hacienda del año actual se reutilizan (del más antiguo al más reciente). Esto evita "huecos" en la secuencia.
2. **Contingencia sin expiración**: Las reservas en contingencia no expiran — se mantienen hasta que se confirmen o liberen explícitamente. Las reservas vencidas que no son de contingencia las procesa el job `reservation_cleaner_job` mediante `GetExpiredNonContingencyReservations`.
3. **Año configurable**: La inclusión del año en el número de control depende de la configuración del usuario (`YearInDTE`).
4. **Concurrencia**: `GetNext` incrementa el contador dentro de una transacción con bloqueo `FOR UPDATE` y `MarkAsReserved` falla si otra solicitud tomó el número liberado.
5. **Contador por año**: El contador se identifica por sucursal, tipo de DTE y año calendario (`year`), de modo que la secuencia reinicia cada año.
6. **Auditoría**: Cada reserva registra quién la creó, cuándo se confirmó/liberó, y por qué motivo fue rechazada.
