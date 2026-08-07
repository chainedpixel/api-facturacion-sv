# Sistema de Eventos de Dominio y Notificaciones por Correo

> **Paquetes:**
> - `internal/domain/core/event/` — contratos del bus
> - `internal/domain/core/notification/` — puerto del mailer
> - `internal/infrastructure/adapters/events/` — bus en memoria
> - `internal/infrastructure/adapters/notifier/email/` — mailer SMTP, cooldown y renderer
> - `internal/application/handlers/notification/` — handler del correo al admin
> - `assets/mails/` — plantillas HTML, SVG y texto plano

## Descripción General

El sistema permite que distintos puntos del microservicio publiquen **eventos de dominio** y que un **handler de correo** notifique al administrador (variable de entorno `ADMIN_EMAIL`) cuando ocurren incidentes operativos relevantes:

1. **Activación de contingencia** en una sucursal.
2. **Falla consistente de emisión** de un DTE (rechazado por Hacienda).
3. **Fallo del job de retransmisión** de contingencia.

El bus es **in-process y asíncrono** (basado en goroutines), no requiere broker externo. Cada notificación pasa por un **cooldown en Redis** (15 minutos por defecto) para evitar inundar la bandeja durante incidentes prolongados.

Si el SMTP no está configurado, el sistema **degrada gracefully**: solo deja de enviar correos, sin bloquear la emisión de DTE.

---

## Arquitectura por Capas

```
                  ┌──────────────────────────────────────────┐
                  │  Servicios que publican (publishers)     │
                  │   - ContingencyService                   │
                  │   - BatchTransmitterService              │
                  │   - RetransmissionJob                    │
                  └────────────────────┬─────────────────────┘
                                       │ Publish(ctx, evt)
                                       ▼
        ┌──────────────────────────────────────────────────────┐
        │   event.Bus (puerto)  ──►  events.InMemoryBus        │
        │   - dispatch async vía goroutines                    │
        │   - persistencia opcional en domain_events           │
        │   - recovery() ante panic en handlers                │
        └────────────────────┬─────────────────────────────────┘
                             │ Handle(ctx, evt)
                             ▼
        ┌──────────────────────────────────────────────────────┐
        │   notification.AdminEmailHandler                     │
        │   1. ShouldNotify? (Cooldown / Redis)                │
        │   2. Render(evt) → subject + html + plain            │
        │   3. SendAdminAlert(...) (Mailer)                    │
        └────────────────────┬─────────────────────────────────┘
                             ▼
        ┌──────────────────────────────────────────────────────┐
        │   email.SMTPMailer (go-mail)                         │
        │   STARTTLS + autenticación según .env                │
        └──────────────────────────────────────────────────────┘
```

---

## Contratos del Dominio

> **Paquete:** `internal/domain/core/event/`

```go
type Event interface {
    Name() string
    OccurredAt() time.Time
    AggregateID() string
    Payload() map[string]any
}

type Handler interface {
    Name() string
    Handle(ctx context.Context, evt Event) error
}

type Bus interface {
    Publish(ctx context.Context, evt Event)
    Subscribe(eventName string, handler Handler)
}

type PersistableEvent interface {
    Event
    UserID() uint
    BranchID() uint
}

type Repository interface {
    Save(ctx context.Context, eventType string, userID, branchID uint, payloadJSON string, occurredAt time.Time) error
}
```

### Tipos auxiliares

```go
type RejectedDocSummary struct {
    ControlNumber string
    ErrorCode     string
    Description   string
    Observations  []string
}
```

### Eventos concretos

| Constante (Name) | Struct | AggregateID | Cuándo se publica |
|---|---|---|---|
| `contingency.activated` | `ContingencyActivatedEvent` | `branch:<id>` | Cuando un DTE se almacena en modo contingencia |
| `emission.failure` | `EmissionFailureEvent` | `dte:<controlNumber>` (fallback: `dte:<generationCode>`) | Cuando MH rechaza un DTE en transmisión por lotes |
| `retransmission_job.failed` | `RetransmissionJobFailedEvent` | `job:<jobName>` | Cuando el batch de retransmisión tiene rechazos **o** cuando el job falla por error de infraestructura |

#### Campos clave por evento

**`ContingencyActivatedEvent`**

| Campo | Tipo | Descripción |
|---|---|---|
| `BranchID` | `uint` | ID interno de la sucursal |
| `BranchAddress` | `string` | Dirección legible (opcional, poblado cuando el publisher tiene acceso) |
| `NIT` | `string` | NIT del emisor (tomado de los claims del JWT) |
| `ClientName` | `string` | Nombre comercial (opcional) |
| `ContingencyType` | `string` | Código numérico ("1"–"5"); el renderer lo convierte a etiqueta descriptiva |
| `Reason` | `string` | Motivo libre reportado |
| `AffectedDocs` | `int` | Documentos almacenados en contingencia |

**`EmissionFailureEvent`**

| Campo | Tipo | Descripción |
|---|---|---|
| `ControlNumber` | `string` | Número de control DTE (ej. `DTE-01-0001-000000001`) — mostrado en el correo |
| `GenerationCode` | `string` | UUID de generación — usado como fallback en `AggregateID` y para persistencia |
| `DTEType` | `string` | Código de tipo DTE ("01", "03", etc.); el renderer lo convierte a etiqueta |
| `ErrorCode` | `string` | Código MH del rechazo |
| `LastError` | `string` | Descripción del rechazo recibida de MH |
| `HaciendaObservations` | `[]string` | Lista de observaciones del campo `observaciones` de la respuesta MH |
| `NIT`, `BranchID`, `BranchAddress`, `ClientName` | varios | Campos de identificación del emisor (opcionales según el publisher) |

**`RetransmissionJobFailedEvent`**

| Campo | Tipo | Descripción |
|---|---|---|
| `JobName` | `string` | Nombre del job (`contingency_retransmission`) |
| `FailedDocuments` | `[]RejectedDocSummary` | Lista de documentos rechazados con detalle de MH (vacía en fallos de infraestructura) |
| `LastError` | `string` | Error de ejecución del job (vacío cuando el fallo es por rechazos de MH) |

### Puerto del Mailer

> **Paquete:** `internal/domain/core/notification/`

```go
type Mailer interface {
    SendAdminAlert(ctx context.Context, subject, htmlBody, plainBody string) error
}
```

---

## Implementaciones de Infraestructura

### Bus en memoria

> **Archivo:** `internal/infrastructure/adapters/events/in_memory_bus.go`

| Característica | Detalle |
|---|---|
| Dispatch | Cada handler se ejecuta en su propia goroutine (no bloquea al publisher) |
| Resiliencia | `recover()` por handler — un panic no afecta a los demás |
| Persistencia | Si el evento implementa `PersistableEvent` y el bus tiene un `Repository`, se guarda el payload serializado en `domain_events` |
| Concurrencia | Mapa de subscriptores protegido con `sync.RWMutex` |

### Repositorio de eventos

> **Archivo:** `internal/infrastructure/adapters/repositories/event_repository.go`

Inserta cada evento persistible en la tabla `domain_events` (modelo GORM ya existente en `db_models/domain_events_model.go`).

### Mailer SMTP

> **Archivo:** `internal/infrastructure/adapters/notifier/email/smtp_mailer.go`

Usa `github.com/wneessen/go-mail`. Comportamiento:

- Si `SMTP_HOST`, `SMTP_FROM` o `ADMIN_EMAIL` están vacíos → log `WARN` y retorna `nil` (degradación elegante).
- Manejo de TLS según puerto cuando `SMTP_TLS=true`:
  - `465` → SSL/TLS implícito (SMTPS, ej. Gmail).
  - cualquier otro puerto → `STARTTLS` (ej. 587).
  - `SMTP_TLS=false` → sin TLS (solo dev).
- Autenticación PLAIN cuando `SMTP_USERNAME` o `SMTP_PASSWORD` están seteados.

### Cooldown en Redis

> **Archivo:** `internal/infrastructure/adapters/notifier/email/cooldown.go`

API:

```go
type Cooldown interface {
    ShouldNotify(ctx context.Context, eventName, aggregateID string) (bool, error)
}
```

- Llave: `notify:cooldown:<eventName>:<aggregateID>`
- Implementación: `SET ... NX EX <ttl>` — atómico.
- TTL configurable vía `NOTIFY_COOLDOWN_MINUTES` (default 15).

### Renderer de plantillas

> **Archivo:** `internal/infrastructure/adapters/notifier/email/template_renderer.go`

- Carga las plantillas desde `assets/mails/` con `embed.FS` (binario self-contained).
- Por cada evento produce `(subject, html, plain)`.
- Inyecta automáticamente `APIVersion` (de `config.Server.APIVersion`), `Environment` (`MH_AMBIENT_CODE` mapeado a `test`/`production`) y `Timestamp`.

---

## Handler del Correo al Admin

> **Archivo:** `internal/application/handlers/notification/admin_email_handler.go`

Flujo:

```
Handle(ctx, evt)
  ├── cooldown.ShouldNotify(evt.Name, evt.AggregateID)
  │       └── false  →  log debug y retorna nil
  ├── render.Render(evt)  →  subject, html, plain
  └── mailer.SendAdminAlert(subject, html, plain)
```

Errores del mailer se loguean como `ERROR` pero **no se propagan** al publisher: una caída de SMTP nunca debe revertir la lógica de negocio que originó el evento.

---

## Plantillas de Correo

> **Carpeta:** `assets/mails/`

```
assets/mails/
├── mails.go                          go:embed FS
├── layout.html                       layout base (header gradiente azul, footer con API v{{.APIVersion}})
├── contingency_activated.html        bloque {{define "content"}}
├── emission_failure.html             bloque {{define "content"}}
├── retransmission_job_failed.html    bloque {{define "content"}}
├── partials/
│   ├── icon_warning.svg              SVG ámbar inline (24×24)
│   ├── icon_error.svg                SVG rojo inline
│   └── icon_info.svg                 SVG azul inline
└── plain/
    ├── contingency_activated.txt
    ├── emission_failure.txt
    └── retransmission_job_failed.txt
```

### UI

- Layout `table`-based de 600 px, compatible con Outlook / Gmail / Apple Mail.
- Paleta corporativa azul (`#1E3A8A` indigo, `#2563EB` blue).
- SVG inline al lado del título, badge de tipo de alerta en el header.
- Tabla key/value con datos del evento, callout con borde azul para el mensaje de error principal.
- Footer con `API v{{.APIVersion}}`, ambiente, timestamp y `AggregateID`.
- Soporte de `prefers-color-scheme: dark` y `@media (max-width: 600px)`.

---

## Variables de Entorno

| Variable | Tipo | Default | Descripción                                                                                |
|---|---|---|--------------------------------------------------------------------------------------------|
| `ADMIN_EMAIL` | string | — | Destinatario de las alertas. Vacío deshabilita el envío.                                   |
| `API_VERSION` | string | `3.0.0` | Versión de la API mostrada en el footer del correo.                                        |
| `SMTP_HOST` | string | — | Host SMTP. Vacío deshabilita el envío.                                                     |
| `SMTP_PORT` | int | `587` | Puerto SMTP.                                                                               |
| `SMTP_USERNAME` | string | — | Usuario SMTP (opcional).                                                                   |
| `SMTP_PASSWORD` | string | — | Clave SMTP (opcional).                                                                     |
| `SMTP_FROM` | string | — | Remitente, ej. `Ordo Factus <noreply@example.com>`. Requerido si `SMTP_HOST` está seteado. |
| `SMTP_TLS` | bool | `true` | `true` aplica `STARTTLS`; `false` desactiva TLS (solo dev).                                |
| `NOTIFY_COOLDOWN_MINUTES` | int | `15` | TTL del cooldown por `<event_name>:<aggregate_id>` en Redis.                               |

`ADMIN_EMAIL`, `API_VERSION` y `NOTIFY_COOLDOWN_MINUTES` viven en el struct `config.Server`. El bloque SMTP completo está en `config.SMTP` (`config/env_structs.go`).

---

## Inyección de Dependencias

> **Archivo:** `internal/bootstrap/containers/services.go`

Métodos clave:

| Método | Función |
|---|---|
| `initNotificationStack()` | Construye `EventBus` (con repositorio), `Mailer`, `Cooldown` y `AdminEmailHandler`, y suscribe el handler a los tres eventos. |
| `attachEventBusToPublishers()` | Inyecta el bus en `ContingencyService` y `BatchTransmitterService` vía `SetEventBus(...)`. |
| `EventBus()` / `Mailer()` | Accesores expuestos por el contenedor. |

El bus también se entrega al job de retransmisión desde `cmd/setup/jobs.go` (`SetupJobs(..., bus event.Bus)`), que llama a `RetransmissionJob.SetEventBus(bus)`.

---

## Puntos de Publicación

| Origen | Evento publicado | Notas |
|---|---|---|
| `contingency_service.go` → `StoreDocumentInContingency` | `ContingencyActivatedEvent` | Popula `NIT` desde los JWT claims |
| `batch_transmitter_service.go` → bloque de rechazos | `EmissionFailureEvent` (uno por DTE rechazado) | Incluye `ControlNumber` de `doc.Document.ControlNumber` y `HaciendaObservations` de la respuesta MH |
| `batch_transmitter_service.go` → bloque de rechazos | `RetransmissionJobFailedEvent` | Se publica **además** del anterior cuando hay ≥1 rechazo; lleva `FailedDocuments` con el resumen de todos los rechazos del batch |
| `contingency_retransmission_job.go` → `handleExecutionError` | `RetransmissionJobFailedEvent` | Solo cuando el job falla por error de infraestructura; `FailedDocuments` vacío, `LastError` con el mensaje de error |

Todos los publishers comparten el patrón `if s.bus != nil { s.bus.Publish(...) }`, por lo que su comportamiento es idéntico cuando no hay bus inyectado (tests, modo deshabilitado).

---

## Persistencia (tabla `domain_events`)

| Columna | Tipo | Descripción |
|---|---|---|
| `id` | uint PK | Autoincrement |
| `user_id` | uint | Owner del branch |
| `branch_id` | uint | Sucursal afectada |
| `event_type` | varchar(50) | `contingency.activated`, `emission.failure`, etc. |
| `payload` | json | Map serializado del evento |
| `occurred_at` | timestamp | Momento del evento |

Solo los eventos que implementan `PersistableEvent` (con `UserID()` y `BranchID()`) se persisten. Los eventos system-wide (job failures) se publican pero no se almacenan en esta tabla.

---

## Tests

> **Ubicación:** `tests/events/`, `tests/notifier/`, `tests/handlers/`, `tests/repositories/`

| Archivo                                       | Cobertura                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
|-----------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `tests/events/in_memory_bus_test.go`          | Dispatch, aislamiento de panics, persistencia condicional                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `tests/notifier/cooldown_test.go`             | Cooldown con `miniredis`: primer envío permitido, segundo bloqueado, expiración por TTL                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `tests/notifier/template_renderer_test.go`    | Renderizado correcto de los tres eventos; resolución de etiquetas de `ContingencyType` (1-5) y `DTEType` (11 códigos); fallback a valor raw para tipos desconocidos; campos opcionales (NIT, ClientName, BranchAddress) presentes cuando se populan y ausentes cuando están vacíos; ausencia de basura en HTML (sin `&mdash;` huérfano, sin filas vacías); `ControlNumber` en subject y cuerpo del correo de emisión; lista de `FailedDocuments` con código, descripción y observaciones en el correo de retransmisión; subject dinámico según tipo de fallo (rechazos MH vs error de infraestructura); integridad estructural con eventos mínimos; round-trip de `Payload()` para los campos nuevos |
| `tests/notifier/smtp_mailer_test.go`          | Servidor SMTP fake recibe `subject`, `html` y `plain`; degradación cuando faltan envs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `tests/handlers/admin_email_handler_test.go`  | Cooldown, errores del cooldown, errores de render, sin cooldown, dedupe                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `tests/handlers/test_notify_handler_test.go`  | Endpoint `GET /notify-test`: variantes `contingency`/`emission`/`job` y respuesta `503` cuando el stack no está inicializado                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `tests/repositories/event_repository_test.go` | INSERT en `domain_events` validado con `sqlmock`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |

---

## Health Checks

> **Archivos:** `internal/infrastructure/adapters/health/checkers/domain_events_checker.go`, `smtp_checker.go`

Se exponen dos checkers en el endpoint de health:

| Componente | Estado | Significado |
|---|---|---|
| `domain_events` | `UP` | El bus respondió a un evento `health.probe` en menos de 1 s |
| `domain_events` | `DOWN` | Bus no inicializado o no entregó el evento dentro del timeout |
| `smtp` | `UP` (no configurado) | `SMTP_HOST` vacío — envío deshabilitado a propósito |
| `smtp` | `UP` | El cliente go-mail completa `Dial` (incluye `EHLO`, TLS y `AUTH PLAIN`) |
| `smtp` | `DOWN` | Falla en handshake: timeout, host inválido, TLS roto **o credenciales inválidas** |

Los textos están traducidos en `assets/i18n/{es,en}.yaml` bajo `health.up.domain_events`, `health.up.smtp`, `health.down.*` y `health.notconfigured.smtp` (esta última accesible vía `utils.TranslateHealthNotConfigured`).

---

## Endpoint de Prueba (solo `DEBUG=true`)

> **Archivos:** `internal/infrastructure/api/handlers/test_handler.go`, `internal/infrastructure/api/routes/test_routes.go`

Cuando `DEBUG=true` en `.env`, el servidor expone un endpoint que envía un correo real de prueba al `ADMIN_EMAIL` usando el `Mailer` y el `TemplateRenderer` configurados. **Bypassa el cooldown** de Redis para permitir reintentos consecutivos durante la verificación.

```
GET /api/v1/notify-test[?event=contingency|emission|job]
```

| Parámetro | Valor | Resultado |
|---|---|---|
| `event=contingency` (default) | `ContingencyActivatedEvent` | Correo "Contingencia activada" |
| `event=emission` | `EmissionFailureEvent` | Correo "Falla consistente de emisión" con DTE de prueba |
| `event=job` | `RetransmissionJobFailedEvent` | Correo "Job de retransmisión con fallos" |

### Respuestas

- `200 OK` — `{ "event": "contingency.activated", "subject": "...", "sent": true }`
- `503 Service Unavailable` — el stack de notificaciones no se inicializó (templates rotos o mailer no construido).
- `500 Internal Server Error` — fallo de render o de transporte SMTP. El cuerpo incluye el mensaje del error.

La ruta vive en el subrouter protegido (requiere JWT) y solo se registra cuando `config.Server.Debug == true`. En producción (`DEBUG=false`) el endpoint no existe y devuelve `404`.

---

## Verificación End-to-End

1. Levantar **MailHog** local:
   ```
   docker run -p 1025:1025 -p 8025:8025 mailhog/mailhog
   ```
2. Configurar `.env`:
   ```
   SMTP_HOST=localhost
   SMTP_PORT=1025
   SMTP_TLS=false
   SMTP_FROM=Ordo <noreply@local>
   ADMIN_EMAIL=admin@test.local
   API_VERSION=3.0.0
   NOTIFY_COOLDOWN_MINUTES=1
   ```
3. Forzar contingencia con `FORCE_CONTINGENCY=true` y emitir un DTE → debe llegar el correo *Contingencia activada* en `http://localhost:8025`.
4. Emitir varios DTE seguidos → llega un único correo durante el TTL (cooldown OK).
5. Esperar el TTL → próxima emisión vuelve a enviar correo.
6. Validar:
   - Header con gradiente azul.
   - SVG warning visible al lado del título.
   - Tabla con datos de sucursal, motivo, hora.
   - Footer con `API v3.0.0` y ambiente.

---

## Cómo Agregar un Nuevo Evento

1. Definir la struct y la constante de nombre en `internal/domain/core/event/events.go` implementando `Event` (y opcionalmente `PersistableEvent`).
2. Crear las plantillas en `assets/mails/` y agregarlas al `go:embed` de `mails.go`.
3. En `template_renderer.go` agregar el patrón de subject, badge, título, intro e ícono para el nuevo evento.
4. Suscribir el handler en `services.go` → `initNotificationStack()`:
   ```go
   bus.Subscribe(event.EventMiNuevo, handler)
   ```
5. Inyectar el bus en el servicio que lo emite (`SetEventBus`) y publicar:
   ```go
   if s.bus != nil { s.bus.Publish(ctx, event.MiNuevoEvent{...}) }
   ```
6. Agregar tests en `tests/notifier/template_renderer_test.go` y, si aplica, en `tests/events/`.
