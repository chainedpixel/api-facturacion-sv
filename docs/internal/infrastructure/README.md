# Capa de Infraestructura

**Ubicación:** `internal/infrastructure/`

## Qué es la Capa de Infraestructura

La capa de infraestructura implementa todos los detalles técnicos que las capas superiores definen como interfaces (ports). Aquí se encuentra el código que interactúa con sistemas externos: base de datos, Redis, API de Hacienda, servicio de firma digital, y el servidor HTTP.

Esta capa **nunca es importada** por dominio ni aplicación. Las capas superiores solo conocen las interfaces; la infraestructura proporciona las implementaciones concretas que se inyectan en tiempo de inicialización.

## Organización de Paquetes

| Paquete | Propósito |
|---|---|
| `adapters/repositories/` | Implementaciones de repositorios con GORM (persistencia) |
| `adapters/tokens/` | Servicio JWT (generación y validación de tokens) |
| `adapters/cache/` | Cache Redis (tokens, credenciales, métricas) |
| `adapters/crypt/` | Operaciones criptográficas (API keys, cifrado AES) |
| `adapters/transmitter/` | Transmisión individual de DTEs a Hacienda |
| `adapters/transmitter/batch/` | Transmisión por lotes (contingencia) |
| `adapters/signing/` | Autenticación OAuth con Hacienda |
| `adapters/signing/signer/` | Firma digital de documentos |
| `adapters/contingency/` | Preparación y envío de eventos de contingencia |
| `adapters/circuit/` | Circuit breaker para protección de servicios externos |
| `adapters/health/` | Health checks de componentes |
| `adapters/metrics/` | Recolección de métricas de rendimiento |
| `adapters/test_endpoint/` | Servicio de prueba de componentes |
| `adapters/events/` | Bus de eventos de dominio in-process |
| `adapters/notifier/email/` | Mailer SMTP, cooldown en Redis y renderer de plantillas |
| `api/server/` | Servidor HTTP y configuración de rutas |
| `api/middleware/` | Middlewares (auth, error, metrics, CORS, timeout) |
| `api/response/` | Escritor de respuestas estandarizado |
| `database/db_models/` | Modelos de base de datos (GORM structs) |
| `database/` | Migraciones automáticas |
| `jobs/` | Trabajos programados (retransmisión, limpieza de métricas, liberación de reservas) |

---

## Documentación Detallada

### Persistencia

| Documento | Descripción |
|---|---|
| [Repositorios](repositories.md) | AuthRepository, DTERepository, ReservedSequenceRepository, ControlNumberRepository, FailedSequenceNumberRepository, ContingencyRepository |
| [Base de Datos](database.md) | 14 modelos de BD (User, BranchOffice, DTEDocument, etc.) y sistema de migraciones |

### Autenticación y Seguridad

| Documento | Descripción |
|---|---|
| [Servicios de Auth](auth-services.md) | JWTService (tokens), RedisTokenCache (cache cifrado), CryptService (API keys y AES) |

### Comunicación con Hacienda

| Documento | Descripción |
|---|---|
| [Transmisores](transmitter.md) | MHTransmitter (individual), BatchTransmitterService (lotes), HaciendaAuthService (OAuth), DTESigner (firma) |
| [Contingencia](contingency.md) | ContingencyEventService (orquestador), ContingencyHTTPService (envío), detección de duplicados |

### API HTTP

| Documento | Descripción |
|---|---|
| [Servidor HTTP](api-server.md) | Server (Gorilla Mux), rutas completas, ResponseWriter, generación de QR |
| [Middlewares](middleware.md) | Auth, Error, Metrics, Timeout, CORS, DBConnection, TokenExtraction — orden de ejecución |

### Resiliencia y Observabilidad

| Documento | Descripción |
|---|---|
| [Circuit Breaker](circuit-breaker.md) | Patrón Closed/Open/Half-Open, threshold=3, reset=5min, thread safety |
| [Health Checkers](health-checkers.md) | 7 health checkers: DatabaseChecker, RedisChecker, HaciendaChecker, FileSystemChecker, SignerChecker, DomainEventsChecker, SMTPChecker |
| [Métricas y Test de Sistema](endpoint-metrics.md) | MetricManager (métricas por endpoint en Redis), TestService (pruebas activas de componentes) |
| [Eventos de Dominio y Notificaciones](domain-events.md) | Bus in-process, mailer SMTP (go-mail), cooldown en Redis, plantillas en `assets/mails/` |
| [Jobs Programados](jobs.md) | RetransmissionJob (10min), MetricsCleanupJob, ReservationCleanerJob (5min) — protección anti-concurrencia con atomic.Bool |

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
| Base de datos | GORM/SQL | Configurado en driver | Persistencia de usuarios, DTEs, secuencias |
| Redis | go-redis | Por operación | Cache de tokens, credenciales, métricas |
| API Hacienda (recepción) | HTTP POST | 30s | Transmisión de DTEs |
| API Hacienda (auth) | HTTP POST | 30s | Autenticación OAuth |
| API Hacienda (consulta) | HTTP POST | 30s | Consulta de estado |
| Servicio de firma | HTTP POST | 2s | Firma digital de documentos |
