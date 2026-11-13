# Capa de Infraestructura

**Ubicación:** `internal/infrastructure/`

## Qué es la Capa de Infraestructura

La capa de infraestructura implementa todos los detalles técnicos que las capas superiores definen como interfaces (ports). Aquí se encuentra el código que interactúa con sistemas externos: base de datos, Redis, API de Hacienda, servicio de firma digital, y el servidor HTTP.

Esta capa **nunca es importada** por dominio ni aplicación. Las capas superiores solo conocen las interfaces; la infraestructura proporciona las implementaciones concretas que se inyectan en tiempo de inicialización.

## Organización de Paquetes

| Paquete | Propósito |
|---|---|
| `adapters/repositories/` | Implementaciones de repositorios con GORM (persistencia), incluidos los eventos transmitidos (`dte_events`) |
| `adapters/tokens/` | Servicio JWT (generación y validación de tokens) |
| `adapters/cache/` | Cache Redis (tokens, credenciales, métricas) |
| `adapters/crypt/` | Operaciones criptográficas (API keys, cifrado AES) |
| `adapters/transmitter/` | Transmisión individual de DTEs y eventos de invalidación a Hacienda |
| `adapters/transmitter/batch/` | Transmisión por lotes (contingencia) |
| `adapters/signing/` | Autenticación con Hacienda (token en cache) |
| `adapters/signing/signer/` | Firma digital de documentos |
| `adapters/contingency/` | Preparación, firma y envío del evento de contingencia (con registro de evidencia y sello) |
| `adapters/circuit/` | Circuit breaker para protección de servicios externos |
| `adapters/health/` | Health checks de componentes |
| `adapters/metrics/` | Recolección de métricas de rendimiento |
| `adapters/test_endpoint/` | Servicio de prueba de componentes |
| `adapters/events/` | Bus de eventos de dominio in-process |
| `adapters/notifier/email/` | Mailer SMTP, cooldown en Redis y renderer de plantillas |
| `api/server/` | Servidor HTTP y configuración de rutas |
| `api/routes/` | Registro de rutas por grupo (DTE, auth, health, metrics, test, debug) |
| `api/handlers/` | Handlers HTTP y creación genérica de documentos |
| `api/helpers/` | Configuración de documentos y clasificación de errores para contingencia |
| `api/middleware/` | Middlewares (auth, error, metrics, CORS, timeout) |
| `api/response/` | Escritor de respuestas estandarizado |
| `database/db_models/` | Modelos de base de datos (GORM structs) |
| `database/` | Migraciones (solo con `RUN_MIGRATION=true`) |
| `jobs/` | Trabajos programados (retransmisión de contingencia, limpieza de métricas, liberación de reservas); se programan en `cmd/setup/jobs.go` |

---

## Documentación Detallada

### Persistencia

| Documento | Descripción |
|---|---|
| [Repositorios](repositories.md) | AuthRepository, DTERepository, DTEEventRepository, ReservedSequenceRepository, ControlNumberRepository, FailedSequenceNumberRepository, ContingencyRepository, EventRepository |
| [Base de Datos](database.md) | 15 modelos de BD (User, BranchOffice, DTEDocument, DTEDetails, DTEEvent, etc.), columnas de evidencia de transmisión y migraciones (`RUN_MIGRATION`) |

### Autenticación y Seguridad

| Documento | Descripción |
|---|---|
| [Servicios de Auth](auth-services.md) | JWTService (tokens validados contra cache y firma), RedisTokenCache (cache cifrado), CryptService (API keys y AES) |

### Comunicación con Hacienda

| Documento | Descripción |
|---|---|
| [Transmisores](transmitter.md) | MHTransmitter (individual e invalidación, con JWS y respuesta completa), BatchTransmitterService (lotes), HaciendaAuthService (token), DTESigner (firma) |
| [Contingencia](contingency.md) | ContingencyEventService (orquestador), ContingencyHTTPService (envío), detección de duplicados, ciclo de retransmisión y ventana de 72 horas |

### API HTTP

| Documento | Descripción |
|---|---|
| [Servidor HTTP](api-server.md) | Server (Gorilla Mux), rutas completas (incluidas `/dte/{id}/archivo` y `/dte/{id}/eventos`), ResponseWriter, formato de errores, generación de QR |
| [Middlewares](middleware.md) | CORS, Error, Timeout, DBConnection, Auth, TokenExtraction, Metrics — orden de ejecución |

### Resiliencia y Observabilidad

| Documento | Descripción |
|---|---|
| [Circuit Breaker](circuit-breaker.md) | Patrón Closed/Open/Half-Open, threshold=3, reset=5min, thread safety |
| [Health Checkers](health-checkers.md) | 7 health checkers (`GET /health`, sin prefijo `/api/v1`): database, redis, hacienda, filesystem, dte_signer, domain_events, smtp |
| [Métricas y Test de Sistema](endpoint-metrics.md) | MetricManager (métricas por endpoint en Redis, expuestas en JSON por `GET /api/v1/metrics`), TestService (pruebas activas de componentes) |
| [Eventos de Dominio y Notificaciones](domain-events.md) | Bus in-process, mailer SMTP (go-mail), cooldown en Redis, plantillas en `assets/mails/`, alertas de contingencia vencida |
| [Jobs Programados](jobs.md) | RetransmissionJob (30 min en producción, 5 min en pruebas), ReservationCleanerJob (10 min), MetricsCleanupJob (6 h) — protección anti-concurrencia con atomic.Bool |

---

## Dirección de Dependencias

```
Capa de Aplicación (interfaces/ports)
  │
  │  implementa ▼
  │
Capa de Infraestructura
  ├── Repositorios      → GORM → MySQL/PostgreSQL
  ├── Cache             → go-redis → Redis
  ├── JWT               → golang-jwt
  ├── Crypt             → crypto/rand + cryptopasta
  ├── Transmitter       → net/http → API Hacienda
  ├── Signer            → net/http → Servicio de firma (Java/Spring)
  ├── Server            → Gorilla Mux → HTTP
  └── Circuit Breaker   → sync.Mutex (in-memory)
```

La infraestructura **importa** las interfaces del dominio y la aplicación. Nunca al revés. Las dependencias externas (GORM, Redis, HTTP clients) quedan encapsuladas dentro de cada adaptador.

---

## Servicios Externos

| Servicio | Protocolo | Timeout | Uso |
|---|---|---|---|
| Base de datos | GORM/SQL | Configurado en driver | Persistencia de usuarios, DTEs, eventos, secuencias |
| Redis | go-redis | Por operación | Cache de tokens, credenciales, métricas |
| API Hacienda (recepción e invalidación) | HTTP POST | 8s | Transmisión de DTEs y eventos de invalidación |
| API Hacienda (auth) | HTTP POST | 8s | Autenticación |
| API Hacienda (consulta) | HTTP POST | 8s | Consulta de estado |
| API Hacienda (contingencia) | HTTP POST | 8s | Envío del evento de contingencia |
| API Hacienda (lotes) | HTTP POST/GET | 30s | Envío y consulta de lotes de contingencia |
| Servicio de firma | HTTP POST | 2s | Firma digital de documentos |

Timeout de petición del servidor: 14 s (TimeoutMiddleware); `ReadTimeout`/`WriteTimeout` del servidor: 15 s.
