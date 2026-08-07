# Health Checkers

> **Paquete:** `internal/infrastructure/adapters/health`

## Descripción General

El sistema de health checks verifica la conectividad y disponibilidad de cada componente externo del que depende el servicio. El resultado se expone a través del endpoint público `GET /health`.

---

## HealthService

> **Archivo:** `adapters/health/health_service.go`
> **Implementa:** `health.HealthManager`

### Estructura

```go
type healthService struct {
    checkers []health.ComponentChecker
}

type HealthServiceConfig struct {
    DB  *gorm.DB
    Bus event.Bus
}
```

### Flujo

```
CheckHealth()
  │
  ├── Por cada checker registrado:
  │     checker.Check()
  │     → { Status: "up" | "down", Details: string }
  │
  └── Resultado agregado:
        {
          Status:     "up" | "down",
          Components: map[name]Health,
          Timestamp:  "02-01-2006 15:04:05"
        }
```

### Estado General

- **up**: Todos los componentes están `up`
- **down**: Al menos un componente está `down`

### Checkers Registrados

| Checker | Nombre | Qué verifica |
|---|---|---|
| `DatabaseChecker` | `"database"` | Conectividad con la base de datos |
| `RedisChecker` | `"redis"` | Conectividad con Redis |
| `HaciendaChecker` | `"hacienda"` | Disponibilidad de la API de Hacienda |
| `FileSystemChecker` | `"filesystem"` | Acceso al sistema de archivos |
| `SignerChecker` | `"signer"` | Disponibilidad del servicio de firma |
| `DomainEventsChecker` | `"domain_events"` | Bus de eventos in-process |
| `SMTPChecker` | `"smtp"` | Servicio SMTP del notifier |

---

## DatabaseChecker

> **Archivo:** `adapters/health/checkers/database_checker.go`

### Estructura

```go
type DatabaseChecker struct {
    db *gorm.DB
}
```

### Implementación

Utiliza `dimiro1/health` para ejecutar un ping estándar a la base de datos. Reporta `down` si el ping falla o supera el timeout.

---

## RedisChecker

> **Archivo:** `adapters/health/checkers/cache_checker.go`

### Estructura

```go
type redisChecker struct{}
```

### Implementación

Realiza un dial TCP a `:6379` mediante `dimiro1/health/redis`. No requiere inyección de dependencias porque la dirección se resuelve desde la configuración del ambiente. Si la conexión falla, el detalle del error se incluye en el campo `Details` de la respuesta.

---

## HaciendaChecker

> **Archivo:** `adapters/health/checkers/hacienda_checker.go`

### Implementación

Envía una HTTP request al endpoint de Hacienda configurado en `config.MHPaths`. Reporta `down` si no hay respuesta o si el status HTTP indica error de servidor.

---

## FileSystemChecker

> **Archivo:** `adapters/health/checkers/filesystem_checker.go`

### Implementación

Ejecuta una operación de lectura/escritura de prueba en el directorio de trabajo del servicio. Verifica que el proceso tenga permisos de escritura en el filesystem local, necesarios para la generación de archivos temporales durante el firmado.

---

## SignerChecker

> **Archivo:** `adapters/health/checkers/signer_checker.go`

### Implementación

Envía una HTTP request de prueba al servicio de firma digital (Java/Spring Boot). Reporta `down` si el servicio no responde dentro del timeout configurado.

---

## DomainEventsChecker

> **Archivo:** `adapters/health/checkers/domain_events_checker.go`

### Estructura

```go
type DomainEventsChecker struct {
    bus event.Bus
}
```

### Implementación

Publica un evento `health.probe` en el bus in-process y espera la respuesta del subscriber interno con un timeout de 1 segundo. Verifica que el bus esté activo y procesando eventos correctamente.

---

## SMTPChecker

> **Archivo:** `adapters/health/checkers/smtp_checker.go`

### Implementación

Dos comportamientos según configuración:

| Condición | Resultado |
|---|---|
| `SMTP_HOST` vacío | `up` con detalle "no configurado" |
| `SMTP_HOST` presente | Abre una conexión real con `go-mail` (`Dial`, incluye `EHLO`, TLS y `AUTH PLAIN`) |

Si las credenciales son inválidas o el servidor no responde, retorna `down`.

---

## Notas

1. **Endpoint público**: `GET /health` no requiere autenticación para permitir que balanceadores de carga y sistemas de monitoreo lo consuman sin token.
2. **Interfaz `ComponentChecker`**: Agregar un nuevo checker solo requiere implementar `Check() models.Health` y `Name() string`, y registrarlo en `NewHealthService`.
3. **Fallo no es fatal**: Un checker en `down` no detiene el servicio — permite identificar degradación parcial sin interrumpir la operación.
