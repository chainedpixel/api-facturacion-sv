# Sistema de Eventos de Dominio

> **Paquete:** `internal/domain/core/event`

## Descripción General

El sistema de eventos de dominio permite que los componentes del sistema se comuniquen de forma desacoplada cuando ocurren hechos significativos (fallos de emisión, activación de contingencia, fallos de retransmisión). El publicador no conoce a los suscriptores; el bus in-process los conecta.

---

## Interfaces Principales

### `Event`

> **Archivo:** `event.go`

Interface que todo evento de dominio debe implementar.

```go
type Event interface {
    Name() string
    OccurredAt() time.Time
    AggregateID() string
    Payload() map[string]any
}
```

| Método | Descripción |
|---|---|
| `Name()` | Nombre del evento (ej. `"emission.failure"`). Se usa para enrutar al handler correcto. |
| `OccurredAt()` | Timestamp del momento en que ocurrió el evento. |
| `AggregateID()` | ID del agregado afectado (ej. `"dte:DTE-1-01-M-0001-..."`, `"branch:42"`). |
| `Payload()` | Mapa con los datos del evento. Se serializa a JSON para persistencia. |

---

### `Handler`

> **Archivo:** `bus.go`

Interface que todo suscriptor debe implementar.

```go
type Handler interface {
    Name() string
    Handle(ctx context.Context, evt Event) error
}
```

El `Name()` se usa para identificar el handler en el registro del bus. `Handle` es invocado sincrónicamente por el bus al publicar un evento.

---

### `Bus`

> **Archivo:** `bus.go`

Interface del bus de eventos in-process.

```go
type Bus interface {
    Publish(ctx context.Context, evt Event)
    Subscribe(eventName string, handler Handler)
}
```

| Método | Comportamiento |
|---|---|
| `Subscribe(eventName, handler)` | Registra un handler para un nombre de evento específico |
| `Publish(ctx, evt)` | Invoca sincrónicamente todos los handlers registrados para `evt.Name()` |

La implementación concreta se encuentra en `internal/infrastructure/adapters/events/`. Ver [Eventos de Dominio y Notificaciones](../../infrastructure/domain-events.md).

---

### `PersistableEvent`

> **Archivo:** `persistable.go`

Extensión de `Event` para eventos que deben persistirse en la tabla `domain_events`.

```go
type PersistableEvent interface {
    Event
    UserID() uint
    BranchID() uint
}
```

Los eventos que implementan esta interface son detectados por el bus y persisten automáticamente su payload a través del `Repository`.

---

### `Repository`

> **Archivo:** `repository.go`

Puerto para persistencia de eventos en base de datos.

```go
type Repository interface {
    Save(ctx, eventType, userID, branchID uint, payloadJSON string, occurredAt time.Time) error
}
```

La implementación concreta es `EventRepository` en `internal/infrastructure/adapters/repositories/event_repository.go`.

---

## Modelo de Persistencia

### `DomainEvent`

> **Archivo:** `domain_event_model.go`

Struct que representa una fila en la tabla `domain_events`.

```go
type DomainEvent struct {
    ID         uint   `json:"id,omitempty"`
    UserID     uint   `json:"user_id"`
    BranchID   uint   `json:"branch_id"`
    EventType  string `json:"event_type"`
    Payload    string `json:"payload"`     // JSON serializado del Payload()
    OccurredAt string `json:"occurred_at"`
}
```

---

## Eventos Definidos

> **Archivo:** `events.go`

### Constantes de nombres

```go
const (
    EventContingencyActivated    = "contingency.activated"
    EventEmissionFailure         = "emission.failure"
    EventRetransmissionJobFailed = "retransmission_job.failed"
)
```

---

### `ContingencyActivatedEvent`

Se publica cuando el sistema activa el modo de contingencia para una sucursal.

```go
type ContingencyActivatedEvent struct {
    BranchID        uint
    BranchAddress   string
    NIT             string
    ClientName      string
    ContingencyType string
    Reason          string
    AffectedDocs    int
    OccurredAtTime  time.Time
}
```

**`AggregateID()`** → `"branch:{BranchID}"`

**Payload:**

| Clave | Valor |
|---|---|
| `branch_id` | ID de la sucursal afectada |
| `nit` | NIT del emisor |
| `client_name` | Nombre del cliente |
| `contingency_type` | Tipo de contingencia activado |
| `reason` | Motivo de la activación |
| `affected_docs` | Número de documentos afectados |

---

### `EmissionFailureEvent`

Se publica cuando un DTE no puede transmitirse a Hacienda después de todos los reintentos.

```go
type EmissionFailureEvent struct {
    BranchID             uint
    BranchAddress        string
    NIT                  string
    ClientName           string
    ControlNumber        string
    GenerationCode       string
    DTEType              string
    ErrorCode            string
    LastError            string
    HaciendaObservations []string
    Attempts             int
    OccurredAtTime       time.Time
}
```

**`AggregateID()`** → `"dte:{ControlNumber}"` (o `"dte:{GenerationCode}"` si no hay número de control).

**Payload:**

| Clave | Valor |
|---|---|
| `control_number` | Número de control del DTE fallido |
| `generation_code` | UUID del documento |
| `dte_type` | Tipo de DTE (01, 03, etc.) |
| `error_code` | Código de error de Hacienda |
| `last_error` | Descripción del último error |
| `hacienda_observations` | Observaciones devueltas por Hacienda |
| `attempts` | Número de intentos realizados |

---

### `RetransmissionJobFailedEvent`

Se publica cuando el `RetransmissionJob` no puede retransmitir documentos pendientes.

```go
type RetransmissionJobFailedEvent struct {
    JobName         string
    FailedDocuments []RejectedDocSummary
    LastError       string
    OccurredAtTime  time.Time
}
```

**`AggregateID()`** → `"job:{JobName}"`

### `RejectedDocSummary`

Resumen por documento rechazado dentro de un evento de fallo del job.

```go
type RejectedDocSummary struct {
    ControlNumber string
    ErrorCode     string
    Description   string
    Observations  []string
}
```

---

## Flujo Completo

```
Componente del sistema (ej. GenericDTEUseCase)
  │
  ├── Crea evento: EmissionFailureEvent{ ... }
  │
  └── bus.Publish(ctx, evt)
        │
        ├── Busca handlers registrados para "emission.failure"
        │
        ├── Para cada handler:
        │     handler.Handle(ctx, evt)
        │     └── AdminEmailHandler:
        │           ├── Verifica cooldown Redis
        │           ├── Renderiza plantilla HTML
        │           └── Envía correo SMTP al administrador
        │
        └── Si evt implementa PersistableEvent:
              repository.Save(ctx, eventType, userID, branchID, payload, occurredAt)
```

---

## Notas

1. **Publicación síncrona**: El bus invoca los handlers en la misma goroutine del publicador. Los handlers deben ser rápidos o manejar su propia concurrencia.
2. **Agregar un nuevo evento**: Definir el struct en `events.go`, implementar la interface `Event`, declarar la constante del nombre. Si debe persistirse, implementar también `PersistableEvent`.
3. **Agregar un handler**: Implementar `event.Handler` y registrarlo con `bus.Subscribe(eventName, handler)` durante la inicialización del servidor.
4. **Cooldown en notificaciones**: El `AdminEmailHandler` usa Redis para evitar notificaciones duplicadas del mismo agregado en un período configurable (default 15 min). Ver [AdminEmailHandler](../../application/handlers/admin-email-handler.md).
