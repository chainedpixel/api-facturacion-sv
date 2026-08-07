# API Facturación El Salvador

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![Go Version](https://img.shields.io/badge/DTE-2026-orange.svg)](https://factura.gob.sv)

API REST para la gestión, emisión y transmisión de Documentos Tributarios Electrónicos (DTE) para El Salvador, desarrollada en Go y diseñada para cumplir con los requisitos establecidos por el Ministerio de Hacienda.

---

## Características

- Emisión de Facturas Electrónicas (FE — código 01)
- Emisión de Comprobantes de Crédito Fiscal (CCF — código 03)
- Emisión de Notas de Remisión Electrónicas (código 04)
- Emisión de Notas de Crédito Electrónicas (código 05)
- Emisión de Notas de Débito Electrónicas (código 06)
- Emisión de Comprobantes de Retención Electrónicos (CRE — código 07)
- Emisión de Facturas de Sujeto Excluido (FSE — código 14)
- Invalidación (anulación) de documentos
- Gestión automática de números de control con ciclo reserva/confirmación/liberación
- Manejo de contingencias con retransmisión automática en segundo plano
- Consulta y listado de DTEs con filtros avanzados y paginación
- Autenticación JWT por sucursal con gestión de tokens de Hacienda
- Firmado electrónico de documentos via servicio externo
- Sistema de eventos de dominio in-process con notificación por correo al `ADMIN_EMAIL` ante incidentes (contingencia, fallas de emisión, fallos del job de retransmisión), con throttling vía Redis

---

## Arquitectura

Se implementa siguiendo la **Arquitectura Hexagonal** (Puertos y Adaptadores), combinada con principios de **Domain Driven Design (DDD)**:

```
Cliente HTTP
    |
Infraestructura  (Router Gorilla Mux, Handlers, Middleware, GORM, Redis, JWT, Transmisión)
    |
Aplicación       (Casos de uso, orquestación, GenericDTEUseCase, DTEUseCaseFactory)
    |
Dominio          (Value Objects, Servicios, Validadores, Puertos/Interfaces)
```

La flecha de dependencia siempre apunta hacia adentro. El dominio no tiene conocimiento de HTTP, bases de datos ni servicios externos.

### Capas

| Capa | Ubicación | Responsabilidad |
|------|-----------|-----------------|
| Dominio | `internal/domain/` | Lógica de negocio pura. Value objects inmutables y auto-validantes, validadores por cadena de estrategias, modelos y puertos (interfaces). Sin dependencias de infraestructura. |
| Aplicación | `internal/application/` | Casos de uso y orquestación. Coordina servicios de dominio con infraestructura sin contener reglas de negocio ni emitir I/O directo. |
| Infraestructura | `internal/infrastructure/` | Todo el I/O. Adaptadores para GORM/MySQL, Redis, JWT, transmisión HTTP a Hacienda, servicio de firma, métricas y health checks. |
| Bootstrap | `internal/bootstrap/` | Inyección manual de dependencias y arranque del servidor. Sin framework de DI. El grafo de objetos es explícito y completamente visible. |

Para mayor detalle, ver [ARCHITECTURE.md](./ARCHITECTURE.md) y la carpeta [docs/](./docs/).

---

## Tecnologías

| Tecnología | Propósito |
|-----------|-----------|
| Go 1.23 | Lenguaje principal |
| MySQL | Base de datos (via GORM) |
| Redis | Caché de tokens de Hacienda |
| Gorilla Mux | Router HTTP |
| GORM | ORM para acceso a base de datos |
| golang-jwt/jwt | Autenticación basada en tokens |
| shopspring/decimal | Aritmética decimal exacta para montos monetarios |
| Docker / Docker Compose | Contenerización y orquestación |

---

## Requisitos previos

- Docker y Docker Compose
- Go 1.23+ (solo para desarrollo local)
- Certificados de firma digital (para ambiente de producción o pruebas con Hacienda)

---

## Instalación

### Con Docker (Recomendado)

1. Clonar el repositorio:

```bash
git clone https://github.com/chainedpixel/ordo-factus.git
cd ordo-factus
```

2. Colocar los certificados de firma digital en la carpeta `scripts/temp` (crearla si no existe).

3. Iniciar los servicios:

```bash
docker-compose up -d
```

### Configuración

Las variables de entorno están predefinidas en `docker-compose.yml`. Las configuraciones principales son:

| Variable | Descripción |
|----------|-------------|
| DSN de base de datos | String de conexión MySQL |
| Dirección Redis | Host y puerto del servidor Redis |
| Secreto JWT | Clave para validación de API keys |
| URL del servicio de firma | URL base del servicio externo de firma (Spring Boot) |
| URL de la API de Hacienda | URL base del endpoint de transmisión del MH |
| Puerto | Puerto HTTP del servidor (por defecto `8080`) |

---

## Uso

### Autenticación

Todos los endpoints de DTE requieren un token JWT. Para obtenerlo:

```
POST /api/v1/auth/login    — Autenticación con API key, secret y credenciales de Hacienda
POST /api/v1/auth/register — Registro de un nuevo sistema (genera API keys por sucursal)
```

El token JWT se incluye en todas las peticiones protegidas como:

```
Authorization: Bearer <token>
```

### Endpoints de Documentos Tributarios

| Método | Path | Tipo DTE | Soporta Contingencia |
|--------|------|----------|----------------------|
| `POST` | `/api/v1/dte/invoices` | Factura Electrónica (01) | Sí |
| `POST` | `/api/v1/dte/ccf` | Comprobante de Crédito Fiscal (03) | Sí |
| `POST` | `/api/v1/dte/remissionnote` | Nota de Remisión Electrónica (04) | Sí |
| `POST` | `/api/v1/dte/creditnote` | Nota de Crédito Electrónica (05) | No |
| `POST` | `/api/v1/dte/debitnote` | Nota de Débito Electrónica (06) | Sí |
| `POST` | `/api/v1/dte/retention` | Comprobante de Retención (07) | No |
| `POST` | `/api/v1/dte/fse` | Factura de Sujeto Excluido (14) | Sí |
| `POST` | `/api/v1/dte/invalidation` | Anulación de DTE | No |

### Consulta de DTEs

```
GET /api/v1/dte          — Listar DTEs con filtros y paginación
GET /api/v1/dte/{id}     — Obtener DTE por código de generación (UUID)
```

Filtros disponibles en el listado: `status`, `type`, `transmission`, `startDate`, `endDate`, `page`, `page_size`.

### Monitoreo y Estado

```
GET /api/v1/health    — Estado del servicio (BD, Redis, Hacienda, signer, eventos de dominio, SMTP)
GET /api/v1/metrics   — Métricas de endpoints en formato Prometheus
GET /api/v1/test      — Prueba de componentes del sistema
```

El endpoint de health incluye dos checkers asociados al sistema de notificaciones:

- `domain_events`: publica un evento `health.probe` en el bus y verifica que un suscriptor interno responda en menos de 1 s.
- `smtp`: si `SMTP_HOST` está vacío reporta `UP` con el detalle "no configurado". En caso contrario hace un handshake completo (`EHLO` + TLS + `AUTH PLAIN`) y reporta `DOWN` con un detalle limpio si las credenciales son inválidas o el servidor no responde.

### Endpoint de Pruebas (solo `DEBUG=true`)

```
POST /api/v1/debug/notify-test[?event=contingency|emission|job]
```

Cuando `DEBUG=true` se registra un endpoint que envía un correo real al `ADMIN_EMAIL` usando la plantilla del evento indicado. Bypasea el cooldown de Redis para permitir verificaciones consecutivas. La colección Postman incluye los tres ejemplos en la carpeta `Debug`.

---

## Notificaciones por Correo al Admin

El microservicio publica eventos de dominio cuando ocurren incidentes operativos relevantes y un handler los traduce a correos enviados al administrador configurado en `ADMIN_EMAIL`:

| Evento | Cuándo se publica |
|---|---|
| `contingency.activated` | Al almacenar un DTE en modo contingencia |
| `emission.failure` | Cuando Hacienda rechaza un DTE durante la transmisión por lotes |
| `retransmission_job.failed` | Cuando el job programado de retransmisión finaliza con error |

Las plantillas HTML profesionales viven embebidas en `assets/mails/` (con SVG inline, paleta azul corporativa y soporte de dark mode). El cooldown se aplica por `<event_name>:<aggregate_id>` con TTL `NOTIFY_COOLDOWN_MINUTES` (15 min por defecto).

Variables de entorno relacionadas:

| Variable | Default | Descripción |
|---|---|---|
| `ADMIN_EMAIL` | — | Destinatario. Vacío deshabilita el envío. |
| `API_VERSION` | `3.0.0` | Versión mostrada en el footer del correo. |
| `SMTP_HOST` / `SMTP_PORT` | — / `587` | Host y puerto SMTP. Puerto `465` activa SSL implícito. |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | — / — | Credenciales (PLAIN auth). |
| `SMTP_FROM` | — | Remitente, requerido si `SMTP_HOST` está seteado. |
| `SMTP_TLS` | `true` | Activa STARTTLS (o SSL implícito en 465). |
| `NOTIFY_COOLDOWN_MINUTES` | `15` | TTL del cooldown del correo por evento. |

Detalle completo en [docs/internal/infrastructure/domain-events.md](./docs/internal/infrastructure/domain-events.md).

---

## Gestión de Números de Control

El Ministerio de Hacienda exige numeración única, consecutiva y sin saltos por tipo de documento. La API gestiona esto internamente; **nunca se debe enviar el número de control en el request**.

El ciclo de vida de un correlativo es el siguiente:

1. **Reserva**: Al recibir la petición, se reserva el siguiente número disponible.
2. **Uso**: Se genera y transmite el documento a Hacienda con el número reservado.
3. **Confirmación**: Si la transmisión es exitosa, el número queda confirmado permanentemente.
4. **Liberación (Rollback)**: Si hay un error de validación o Hacienda rechaza el documento (4xx), el número se libera y vuelve a estar disponible para la siguiente petición.

El sistema garantiza la integridad de la numeración incluso ante fallos de transmisión. Si se recibe un error 500 (error interno del servidor, no de Hacienda), es seguro reintentar la petición.

---

## Gestión de Contingencias

Cuando la API de Hacienda no está disponible, el sistema almacena automáticamente los documentos y los retransmite en segundo plano cuando se restaura la conectividad.

| Condición | Tipo de contingencia |
|-----------|----------------------|
| Error de red (conexión rechazada) | Falla de conexión al sistema |
| Red no disponible | Falla de servicio de Internet |
| HTTP 502/503/504 de Hacienda | No disponibilidad del MH |
| HTTP 408/429 (timeout o rate limiting) | No disponibilidad del MH |

Durante contingencia la respuesta es `HTTP 201` con el DTE completo pero sin sello de recepción (`reception_stamp: null`). El documento queda en estado `PENDING` hasta ser retransmitido exitosamente.

Los documentos que **no** activan contingencia son aquellos con errores de validación o rechazados definitivamente por Hacienda.

---

## Seguridad

- Autenticación basada en JWT por sucursal
- Validación estricta de entradas mediante Value Objects inmutables y auto-validantes
- Firmado digital de documentos via servicio externo
- API keys con hash seguro (el secreto en texto plano solo se retorna en el momento del registro)
- Circuit breaker para proteger contra fallos en cascada de la API de Hacienda

---

## Integración Continua (CI)

El proyecto utiliza un pipeline de integración continua con dos ramas específicas para la generación de builds:

- **release-amd64**: Compilación y despliegue para arquitectura `amd64`
- **release-arm64**: Compilación y despliegue para arquitectura `arm64`

Cada rama genera imágenes optimizadas para su respectiva arquitectura, asegurando compatibilidad en distintos entornos de ejecución.

---

## Contribución

Para contribuir a este proyecto:

1. Analizar la documentación existente antes de proponer implementaciones
2. Respetar la arquitectura establecida (Puertos y Adaptadores - Hexagonal)
3. Mantener consistencia con las implementaciones existentes
4. Validar contra el JSON Schema oficial de Hacienda
5. No asumir comportamientos no documentados

Documentación disponible:

- [Arquitectura detallada](./ARCHITECTURE.md)
- [Referencia de API para consumidores](./docs/api/README.md)
- [Eventos de dominio y notificaciones por correo](./docs/internal/infrastructure/domain-events.md)
- [Colección Postman](scripts/postman/Collection.postman_collection.json) y [environment](scripts/postman/Environment.postman_environment.json)

**Documentación interna por capa:**

| Capa | Documentación | Contenido |
|------|---------------|-----------|
| Dominio | [docs/internal/domain/](./docs/internal/domain/README.md) | 9 servicios DTE, 8 estrategias de validación, modelos, puertos, errores, constantes, números secuenciales |
| Aplicación | [docs/internal/application/](./docs/internal/application/README.md) | GenericDTEUseCase, Factory, Invalidación, Consulta, Transmitter, Mappers, Operaciones Adicionales, Auth |
| Infraestructura | [docs/internal/infrastructure/](./docs/internal/infrastructure/README.md) | Repositorios, JWT/Cache/Crypt, Transmisores, Contingencia, API Server, Middlewares, Circuit Breaker, Health/Métricas, BD |
| Bootstrap | [docs/internal/bootstrap/](./docs/internal/bootstrap/README.md) | Inyección de dependencias y arranque del servidor |

**Documentación complementaria:**

| Área | Documentación | Contenido |
|------|---------------|-----------|
| Mappers (pkg) | [docs/pkg/](./docs/pkg/README.md) | Sistema de mapeo, request mappers (HTTP→Dominio), response mappers (Dominio→Hacienda) |
| Entry Point | [docs/cmd/](./docs/cmd/README.md) | Punto de entrada, bootstrap, jobs asíncronos (retransmisión, limpieza) |
| Configuración | [docs/config/](./docs/config/README.md) | Variables de entorno, validación, drivers BD, Redis, traducciones i18n |
| Tests | [docs/test/](./docs/test/README.md) | Mocks (GoMock), fixtures (Builder Pattern), tests de servicios/mappers/integración |

- [JSON Schema y catálogos oficiales de Hacienda](https://factura.gob.sv/informacion-tecnica-y-funcional/)

---

## Agradecimientos

Este proyecto contó con las valiosas aportaciones de **[Denilson Chavez](https://www.linkedin.com/in/denilson-alexander-chavez-recinos-771837199/)**, cuyas recomendaciones y revisiones mejoraron significativamente la calidad del código y la arquitectura del sistema.