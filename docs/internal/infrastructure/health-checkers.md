# Health Checkers

> **Paquete:** `internal/infrastructure/adapters/health`

## Descripción General

El sistema de health checks verifica la conectividad y disponibilidad de cada componente externo del que depende el servicio. El resultado se expone a través del endpoint público `GET /health` (registrado en la raíz del router, **sin** el prefijo `/api/v1`, ver [api-server.md](api-server.md)).

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
  ├── Por cada checker registrado (secuencialmente):
  │     checker.Check()
  │     → { Status: "UP" | "DOWN", Details: string }
  │
  └── Resultado agregado:
        {
          Status:     "UP" | "DOWN",
          Components: map[name]Health,
          Timestamp:  "02-01-2006 15:04:05"
        }
```

Los valores de estado son `UP` y `DOWN` (mayúsculas, `health/constants`). `Details` se omite del JSON cuando está vacío.

### Respuesta HTTP

`HealthHandler.CheckHealth` responde siempre `200` con el envoltorio estándar `{ "success": true, "data": { "status", "components", "timestamp" } }`, incluso cuando el estado general es `DOWN`: el estado se informa en el cuerpo, no en el código HTTP. Solo se responde `500` si `CheckHealth` devuelve error, lo cual el servicio actual nunca hace.

### Estado General

- **UP**: Todos los componentes están `UP`
- **DOWN**: Al menos un componente está `DOWN`

### Checkers Registrados

El orden de registro y las claves en `components` son:

| Checker | Nombre (clave) | Qué verifica |
|---|---|---|
| `DatabaseChecker` | `database` | Conectividad con la base de datos |
| `RedisChecker` | `redis` | Conectividad con Redis |
| `HaciendaChecker` | `hacienda` | Disponibilidad de los endpoints de Hacienda |
| `FileSystemChecker` | `filesystem` | Acceso de escritura al archivo de logs |
| `SignerChecker` | `dte_signer` | Disponibilidad del servicio de firma |
| `DomainEventsChecker` | `domain_events` | Bus de eventos in-process |
| `SMTPChecker` | `smtp` | Servicio SMTP del notifier |

Los textos de `Details` provienen de `assets/i18n/{es,en}.yaml` bajo `health.up.*`, `health.down.*`, `health.error.*` y `health.notconfigured.*`.

---

## DatabaseChecker

> **Archivo:** `adapters/health/checkers/database_checker.go`

### Estructura

```go
type databaseChecker struct {
    db *gorm.DB
}
```

### Implementación

Obtiene el `*sql.DB` de GORM (si falla, `DOWN` con el mensaje `FailedToGetDBConnection`) y ejecuta `dimiro1/health/db.NewMySQLChecker` sobre él. Reporta `DOWN` si la verificación falla, anexando el detalle del error. Nótese que se usa el checker de MySQL independientemente del driver configurado.

---

## RedisChecker

> **Archivo:** `adapters/health/checkers/cache_checker.go`

### Estructura

```go
type redisChecker struct{}
```

### Implementación

Realiza un dial TCP mediante `dimiro1/health/redis` a la dirección fija `tcp` `:6379` (puerto 6379 en el host local). La dirección **no** se resuelve desde la configuración de Redis del ambiente: si Redis corre en otro host o puerto, este checker reportará `DOWN` aunque el cache funcione. Si la conexión falla, el detalle del error se incluye en `Details`.

---

## HaciendaChecker

> **Archivo:** `adapters/health/checkers/hacienda_checker.go`

### Implementación

El cliente HTTP tiene timeout de 2 segundos. Se ejecutan dos fases:

1. **Disponibilidad de endpoints**: se envía `HEAD` (contexto de 3 s) a `config.MHPaths.AuthURL`, `ReceptionURL` y `ContingencyURL`. Un error de red o un status `>= 500` marca el componente `DOWN` (si el error contiene `dial tcp` se informa como falta de conexión a internet). El recorrido sigue el orden de un mapa de Go, por lo que no es determinista.
2. **Procesamiento de autenticación**: se envía un `POST` a `AuthURL` con credenciales de prueba (`test_user`/`test_password`; contexto de 5 s, aunque el cliente limita a 2 s). Solo se aceptan los status `200`, `400` y `401`; cualquier otro se reporta como respuesta inesperada y marca `DOWN`.

---

## FileSystemChecker

> **Archivo:** `adapters/health/checkers/filesystem_checker.go`

### Implementación

Intenta crear el directorio de logs y abrir (crear en modo append) el archivo `dte_microservice.log` bajo `config.Log.Path` desde la raíz del proyecto. Verifica así el permiso de escritura en el filesystem local.

> Observación del código: en `checkHealth` se invoca `result.Down()` cuando `checkFileSystem` falla, pero a continuación se invoca `result.Up()` de forma incondicional, por lo que el checker devuelve `UP` incluso si la prueba falló.

---

## SignerChecker

> **Archivo:** `adapters/health/checkers/signer_checker.go`

### Implementación

Realiza `GET` a `config.Signer.Health` (variable `SIGNER_HEALTH`) con timeout de 2 segundos. Reporta `DOWN` si hay error de red, si el status HTTP no es `200` o si el cuerpo JSON no indica estado `UP`. Se aceptan dos formatos de respuesta: `{"status":"UP"}` y `{"body":{"status":"UP"}}` (comparación sin distinguir mayúsculas).

---

## DomainEventsChecker

> **Archivo:** `adapters/health/checkers/domain_events_checker.go`

### Estructura

```go
type domainEventsChecker struct {
    bus    event.Bus
    signal chan string
}
```

### Implementación

Al construirse, se suscribe al bus con un handler interno para el evento `health.probe`. En cada verificación publica un `health.probe` con un UUID y espera durante 1 segundo a que el handler lo devuelva por el canal. Reporta `DOWN` si el bus es nulo (`domain_events_not_initialized`) o si el evento no se entrega dentro del timeout.

---

## SMTPChecker

> **Archivo:** `adapters/health/checkers/smtp_checker.go`

### Implementación

Dos comportamientos según configuración:

| Condición | Resultado |
|---|---|
| `SMTP_HOST` vacío | `UP` con detalle "no configurado" (`health.notconfigured.smtp`) |
| `SMTP_HOST` presente | Abre una conexión real con `go-mail` (`DialWithContext`, 5 s), con las mismas reglas de TLS que el mailer (SSL en el puerto 465, `STARTTLS` obligatorio en otros puertos si `SMTP_TLS=true`, sin TLS si es `false`) y `AUTH PLAIN` si hay credenciales |

Si la conexión falla, el detalle se clasifica según el texto del error: `smtp_invalid_credentials` (auth, credentials, 535, 5.7.8), `smtp_unreachable` (host inexistente, conexión rechazada, timeout, EOF, etc.) o `smtp` (otros).

---

## Notas

1. **Endpoint público**: `GET /health` no requiere autenticación para permitir que balanceadores de carga y sistemas de monitoreo lo consuman sin token.
2. **Interfaz `ComponentChecker`**: Agregar un nuevo checker solo requiere implementar `Check() models.Health` y `Name() string`, y registrarlo en `NewHealthService`.
3. **Fallo no es fatal**: Un checker en `DOWN` no detiene el servicio ni cambia el código HTTP de `/health`; permite identificar degradación parcial sin interrumpir la operación.
4. **Ejecución secuencial**: Los checkers se ejecutan uno tras otro en cada petición; el tiempo de respuesta es la suma de sus tiempos.
