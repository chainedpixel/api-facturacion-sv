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
| `sequentialRepo` | Se obtiene el siguiente número secuencial de la BD |
| `authRepo` | Se obtiene información de la sucursal (año en DTE) |
| `reservationRepo` | Se gestiona el ciclo de vida de las reservas |

---

## Formato del Número de Control

```
DTE-{TipoDTE}-{CódigoEstablecimiento}{CódigoPOS}-{Secuencia}
```

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
func parseControlNumber(controlNumber string) (dteType, sequence, year string, error)
```

Se descompone un número de control en sus partes constituyentes.

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
func GetNextControlNumber(ctx, dteType, branchID, posCode, establishmentCode) (string, error)
```

Se obtiene el siguiente número de control para uso inmediato (sin reserva).

### `ReserveNextNumber`

```go
func ReserveNextNumber(ctx, dteType, branchID, posCode, establishmentCode, documentData, isContingency) (string, error)
```

Se reserva el siguiente número disponible. El flujo es:

1. **Buscar números liberados** — Se intenta reutilizar un número previamente liberado (con expiración de 1 hora)
2. **Si no hay liberados** — Se obtiene un nuevo número secuencial
3. **Crear la reserva** — Se registra con estado `RESERVED`

**Parámetros especiales:**
- `documentData` — Datos del documento para trazabilidad
- `isContingency` — Si es `true`, la reserva no tiene fecha de expiración

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
- `ExpiresAt` → 1 hora después (ventana de reuso)

### `ConfirmReservationByDocumentID`

```go
func ConfirmReservationByDocumentID(ctx, documentID) error
```

Se confirma la reserva buscando por ID del documento (útil en contingencia).

### `ReleaseReservationByDocumentID`

```go
func ReleaseReservationByDocumentID(ctx, documentID, rejectionReason, haciendaCode) error
```

Se libera la reserva buscando por ID del documento.

### `MarkReservationAsContingency`

```go
func MarkReservationAsContingency(ctx, reservationID, documentID, contingencyDocID) error
```

Se marca una reserva como contingencia cuando el documento entra en modo contingencia:
- `IsContingency` → `true`
- `ContingencyDocumentID` → ID del documento de contingencia
- `ExpiresAt` → Se elimina (sin expiración)

### `MarkReservationAsContingencyByControlNumber`

```go
func MarkReservationAsContingencyByControlNumber(ctx, controlNumber, documentID, contingencyDocID, branchID) error
```

Misma funcionalidad pero buscando por número de control.

---

## Modelo de Reserva

```go
type ReservedSequence struct {
    ID                    uint
    BranchID              uint
    DTEType               string
    SequenceNumber        uint
    Year                  int
    Status                string      // RESERVED, CONFIRMED, RELEASED
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
1. ReserveNextNumber() → "DTE-01-00010001-2024000000042" [RESERVED]
2. Transmitir a Hacienda → Éxito
3. ConfirmReservation() → [CONFIRMED]
```

### Flujo con Rechazo

```
1. ReserveNextNumber() → "DTE-01-00010001-2024000000042" [RESERVED]
2. Transmitir a Hacienda → Rechazado
3. ReleaseReservation() → [RELEASED] (expira en 1 hora)
4. ReserveNextNumber() → "DTE-01-00010001-2024000000042" [RESERVED] (reutilizado)
```

### Flujo con Contingencia

```
1. ReserveNextNumber() → "DTE-01-00010001-2024000000042" [RESERVED]
2. Transmitir a Hacienda → Error de red
3. MarkReservationAsContingency() → [RESERVED, isContingency=true]
4. (Retransmisión posterior)
5. ConfirmReservationByDocumentID() → [CONFIRMED]
```

---

## Notas

1. **Reutilización**: Los números liberados se pueden reutilizar dentro de 1 hora. Esto evita "huecos" en la secuencia.
2. **Contingencia sin expiración**: Las reservas en contingencia no expiran — se mantienen hasta que se confirmen o liberen explícitamente.
3. **Año configurable**: La inclusión del año en el número de control depende de la configuración del usuario (`YearInDTE`).
4. **Concurrencia**: El repositorio debe implementar control de concurrencia para evitar que dos solicitudes reserven el mismo número.
5. **Auditoría**: Cada reserva registra quién la creó, cuándo se confirmó/liberó, y por qué motivo fue rechazada.
