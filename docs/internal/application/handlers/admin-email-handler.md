# AdminEmailHandler

> **Paquete:** `internal/application/handlers/notification`
> **Archivo:** `admin_email_handler.go`

## Descripción General

`AdminEmailHandler` es un handler de eventos de dominio que envía notificaciones por correo electrónico al administrador del sistema cuando se produce un evento relevante. Implementa la interfaz `event.Handler` del bus de eventos in-process.

Su comportamiento es silencioso por diseño: si el cooldown está activo, el envío se omite sin error. Solo falla cuando el renderizado o el envío del correo reportan un error explícito.

---

## Estructura

```go
type AdminEmailHandler struct {
    mailer   notification.Mailer
    cooldown email.Cooldown
    render   renderer
}
```

| Campo | Tipo | Propósito |
|---|---|---|
| `mailer` | `notification.Mailer` | Envío del correo a través del servicio SMTP |
| `cooldown` | `email.Cooldown` | Control de frecuencia para evitar spam de notificaciones |
| `render` | `renderer` | Genera el asunto y cuerpo del correo a partir del evento |

### Interfaz `renderer` (interna)

```go
type renderer interface {
    Render(evt event.Event) (subject, html, plain string, err error)
}
```

La implementación concreta es `TemplateRenderer` en `internal/infrastructure/adapters/notifier/email/template_renderer.go`, que renderiza plantillas HTML/plain-text desde `assets/mails/`.

---

## Métodos

### `NewAdminEmailHandler`

```go
func NewAdminEmailHandler(m notification.Mailer, c email.Cooldown, r renderer) *AdminEmailHandler
```

Constructor. El parámetro `cooldown` puede ser `nil`; en ese caso se omite la verificación y todos los eventos generan un envío.

---

### `Name`

```go
func (h *AdminEmailHandler) Name() string // → "admin_email_handler"
```

Se usa por el bus de eventos para identificar y registrar el handler.

---

### `Handle`

```go
func (h *AdminEmailHandler) Handle(ctx context.Context, evt event.Event) error
```

#### Flujo

```
Handle(ctx, evt)
  │
  ├── [1] Verificar cooldown (si está configurado)
  │     cooldown.ShouldNotify(ctx, evt.Name(), evt.AggregateID())
  │     ├── Error en Redis → log warning, continuar igual (fail-open)
  │     └── Cooldown activo (ok=false) → log debug, return nil (silencioso)
  │
  ├── [2] Renderizar correo
  │     render.Render(evt) → subject, html, plain
  │     └── Error → log error, return err
  │
  └── [3] Enviar correo
        mailer.SendAdminAlert(ctx, subject, html, plain)
        └── Error → log error, return err
```

#### Comportamiento ante fallos del cooldown

Si `ShouldNotify` retorna error (ej. Redis no disponible), el handler **continúa con el envío** (fail-open). Esto garantiza que los eventos críticos no se pierdan por una falla de Redis, a costa de posibles envíos duplicados.

---

## Dependencias e Interfaces

### `notification.Mailer`

```go
// internal/domain/core/notification/mailer_port.go
type Mailer interface {
    SendAdminAlert(ctx context.Context, subject, htmlBody, plainBody string) error
}
```

Implementación concreta: `SMTPMailer` en `internal/infrastructure/adapters/notifier/email/smtp_mailer.go`.

---

### `email.Cooldown`

```go
// internal/infrastructure/adapters/notifier/email/cooldown.go
type Cooldown interface {
    ShouldNotify(ctx context.Context, eventName, aggregateID string) (bool, error)
}
```

Implementación concreta: `RedisCooldown`.

#### Mecanismo

```
ShouldNotify(ctx, eventName, aggregateID)
  │
  ├── Clave Redis: "notify:cooldown:{eventName}:{aggregateID}"
  ├── SETNX con TTL configurado (default: 15 minutos)
  ├── ok=true  → primera notificación del periodo → enviar
  └── ok=false → ya existe la clave → cooldown activo → omitir
```

El TTL evita que un mismo evento (identificado por su nombre y el ID del agregado) genere múltiples correos en el mismo período. Configurado al inicializar `RedisCooldown` con `NewRedisCooldown(client, ttl)`.

---

## Registro en el Bus de Eventos

`AdminEmailHandler` se registra como subscriber en el bus de eventos de dominio durante la inicialización del servidor. Cuando el bus publica un evento, todos los handlers registrados reciben la llamada `Handle` de forma síncrona.

---

## Notas

1. **Fail-open en cooldown**: Un error de Redis no bloquea la notificación. Se acepta el riesgo de correos duplicados para garantizar entrega en eventos críticos.
2. **Handler silencioso**: Un cooldown activo retorna `nil` (sin error). El bus de eventos no debe interpretar la omisión como fallo.
3. **Separación de renderizado**: El `renderer` encapsula toda la lógica de plantillas. `AdminEmailHandler` no conoce el formato del correo ni las rutas de los templates.
4. **Plantillas**: Los archivos HTML/plain-text se encuentran en `assets/mails/`. Ver documentación del notifier en [Eventos de Dominio y Notificaciones](../../infrastructure/domain-events.md).
