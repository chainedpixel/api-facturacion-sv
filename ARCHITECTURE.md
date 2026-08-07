# Arquitectura

`github.com/chainedpixel/ordo-factus` es un servidor HTTP API para emitir documentos tributarios electrónicos (DTE) de El Salvador contra la API del Ministerio de Hacienda (MH). Sigue la **Arquitectura Hexagonal** (también conocida como Puertos y Adaptadores).

---

## Tabla de Contenidos

1. [Vista General de Alto Nivel](#1-vista-general-de-alto-nivel)
2. [Responsabilidades por Capa](#2-responsabilidades-por-capa)
   - [Dominio](#21-dominio)
   - [Aplicación](#22-aplicación)
   - [Infraestructura](#23-infraestructura)
   - [Bootstrap](#24-bootstrap)
3. [Árbol de Directorios](#3-árbol-de-directorios)
4. [Ciclo de Vida de una Petición](#4-ciclo-de-vida-de-una-petición)
5. [Pipeline de Documentos DTE](#5-pipeline-de-documentos-dte)
6. [Reglas de Dependencia](#6-reglas-de-dependencia)
7. [Patrones de Diseño Clave](#7-patrones-de-diseño-clave)
8. [Dependencias Externas](#8-dependencias-externas)
9. [Configuración y Entorno](#9-configuración-y-entorno)
10. [Lectura Adicional](#10-lectura-adicional)

---

## 1. Vista General de Alto Nivel

```
┌──────────────────────────────────────────────────────────┐
│                      Cliente HTTP                        │
└────────────────────────────┬─────────────────────────────┘
                             │ HTTP/JSON
┌────────────────────────────▼─────────────────────────────┐
│                   Capa de Infraestructura                │
│  Router Gorilla Mux · Handlers · Middleware              │
│  Repositorios GORM · Caché Redis · Tokens JWT            │
│  Transmisor HTTP a Hacienda · Adaptador de firma         │
└────────────────────────────┬─────────────────────────────┘
                             │ Interfaces de puertos
┌────────────────────────────▼──────────────────────────────┐
│                   Capa de Aplicación                      │
│  GenericDTEUseCase · InvalidationUseCase                  │
│  DTEConsultUseCase · AuthUseCase                          │
│  DTEUseCaseFactory                                        │
└────────────────────────────┬──────────────────────────────┘
                             │ Interfaces de dominio
┌────────────────────────────▼───────────────────────────────┐
│                    Capa de Dominio                         │
│  Value Objects · Agregados · Servicios de Dominio          │
│  Validadores (cadena de estrategias) · Interfaces de puerto│
└────────────────────────────────────────────────────────────┘
```

La flecha de dependencia siempre apunta **hacia adentro**: la infraestructura depende de la aplicación, la aplicación depende del dominio. El dominio no tiene conocimiento de HTTP, bases de datos ni servicios externos.

---

## 2. Responsabilidades por Capa

### 2.1 Dominio

**Ubicación:** `internal/domain/`

El núcleo de negocio puro. No se permiten imports de infraestructura aquí (sin `net/http`, sin ORM, sin cliente Redis). Los módulos externos permitidos se limitan a `github.com/shopspring/decimal` (aritmética monetaria) y `github.com/google/uuid` (códigos de generación).

Sub-paquetes principales:

| Paquete | Contenido |
|---|---|
| `domain/dte/common/value_objects/` | Primitivos inmutables y auto-validantes: `NIT`, `Amount`, `ControlNumber`, `EmissionDate`, `DTEType`, etc. |
| `domain/dte/common/models/` | Structs de modelo compartidos embebidos por cada tipo de DTE |
| `domain/dte/common/validator/` | `BaseValidator` (basado en reflexión) + `DTERulesValidator` (cadena de estrategias) |
| `domain/dte/common/dte_errors/` | `DTEError`, `ValidationError`, `CompositeError` — tipos de error del dominio con i18n |
| `domain/dte/common/interfaces/` | Interfaces de Go para documentos DTE, ítems, resúmenes y receptores |
| `domain/dte/{invoice,ccf,fse,credit_note,debit_note,retention,remission_note,invalidation}/` | Un paquete por tipo de DTE; cada uno contiene modelos, un servicio de dominio y estrategias de validación específicas del tipo |
| `domain/dte/dte_documents/` | `SequentialNumberManager` (ciclo de vida del número de control) + puerto `DTEManager` |
| `domain/dte/contingency/` | Modelos de documentos en contingencia |
| `domain/ports/` | Interfaces de puertos que el dominio expone para que la infraestructura las implemente: `DTEService`, `SequentialNumberRepositoryPort`, `CacheManager`, `CircuitManager`, `CryptManager`, `TimeProvider` |
| `domain/auth/` | Modelos de autenticación y puerto del repositorio |
| `domain/core/` | Modelos de persistencia transversales: `DTEDocument`, `User`, `BranchOffice`, `DomainEvent` |
| `domain/core/event/` | Contratos del bus de eventos: `Event`, `Bus`, `Handler`, `PersistableEvent`, `Repository` y los eventos concretos (`ContingencyActivatedEvent`, `EmissionFailureEvent`, `RetransmissionJobFailedEvent`) |
| `domain/core/notification/` | Puerto `Mailer` para el envío de alertas al administrador |

### 2.2 Aplicación

**Ubicación:** `internal/application/`

Nivel de orquestación. El código de aplicación coordina los servicios de dominio y los puertos de infraestructura; no contiene reglas de negocio y no emite I/O directo.

| Paquete | Contenido |
|---|---|
| `application/dte/` | `GenericDTEUseCase`, `InvalidationUseCase`, `DTEConsultUseCase`, `DTEUseCaseFactory` |
| `application/auth/` | `AuthUseCase` (login + registro) |
| `application/ports/` | Interfaces de puertos que posee la capa de aplicación: `BaseTransmitter`, `SignerManager`, `DTETransmitter`, `HaciendaAuthManager` |
| `application/handlers/notification/` | `AdminEmailHandler` — escucha eventos del bus, consulta el cooldown y envía la alerta vía `Mailer` |

**`GenericDTEUseCase`** es el struct central. Toda petición de creación de DTE — independientemente del tipo de documento — fluye a través de su método `Create`:

1. Lee `AuthClaims` del contexto de la petición.
2. Llama al `DTEMapper` para convertir el JSON crudo de la petición en un struct de input de dominio tipado.
3. Delega al servicio de dominio específico del tipo (`ports.DTEService.Create`).
4. Llama al `SequentialNumberManager` para asignar un número de control.
5. Llama a `BaseTransmitter.RetryTransmission` para enviar a Hacienda.
6. Persiste el resultado via `DTEManager`.
7. Llama a la `AdditionalOperationsFunc` opcional (ej. ajuste de saldo para notas de crédito/débito).
8. Mapea el resultado de dominio al formato de respuesta de Hacienda via `ResponseMapperFunc`.

**`DTEUseCaseFactory`** instancia variantes de `GenericDTEUseCase`, inyectando el mapper correcto y el hook de operaciones por tipo de DTE.

### 2.3 Infraestructura

**Ubicación:** `internal/infrastructure/`

Todo el I/O vive aquí. Los paquetes de infraestructura implementan las interfaces de puertos definidas en las capas de dominio y aplicación.

| Sub-paquete | Propósito |
|---|---|
| `infrastructure/api/` | Router Gorilla Mux, handlers HTTP, middleware (auth, CORS, métricas, timeout, BD), escritor de respuestas, helpers |
| `infrastructure/database/` | GORM + driver MySQL, `DbConnection`, structs de modelo GORM en `db_models/` |
| `infrastructure/adapters/repositories/` | Implementaciones basadas en GORM de todos los puertos de repositorio |
| `infrastructure/adapters/tokens/` | Emisión y validación de JWT (`HaciendaAuthManager`, `TokenManager`) |
| `infrastructure/adapters/cache/` | `CacheManager` respaldado por Redis |
| `infrastructure/adapters/signing/` | Adaptador cliente HTTP al servicio externo de firma Spring Boot |
| `infrastructure/adapters/transmitter/` | Implementación de `BaseTransmitter` con reintento + backoff; `hacienda_error/` parseo de errores; `processors/` procesadores de transmisión por tipo; `batch/` envío por lotes de contingencia |
| `infrastructure/adapters/circuit/` | `CircuitBreaker` que envuelve las llamadas a la API de Hacienda |
| `infrastructure/adapters/contingency/` | Manager de eventos de contingencia y almacenamiento |
| `infrastructure/adapters/crypt/` | Generación y cifrado de API keys (`CryptManager`) |
| `infrastructure/adapters/health/` | Adaptadores de health-check (BD, Redis, Hacienda, signer, filesystem, eventos de dominio, SMTP) |
| `infrastructure/adapters/metrics/` | Adaptador de métricas Prometheus |
| `infrastructure/adapters/test_endpoint/` | Adaptador de endpoint de pruebas/sandbox |
| `infrastructure/adapters/events/` | `InMemoryBus` — pub/sub in-process asíncrono que persiste eventos `PersistableEvent` en `domain_events` |
| `infrastructure/adapters/notifier/email/` | `SMTPMailer` (go-mail), `RedisCooldown` y `TemplateRenderer` con plantillas embebidas desde `assets/mails/` |
| `infrastructure/error/` | Tipos de error de la capa de infraestructura y mapeo a códigos HTTP |
| `infrastructure/mapper/` | Mappers de petición-a-dominio y dominio-a-respuesta para cada tipo de DTE |
| `infrastructure/jobs/` | Jobs en segundo plano (ej. programador de reintentos de contingencia) |

### 2.4 Bootstrap

**Ubicación:** `internal/bootstrap/`

Conecta todo el grafo de objetos e inicia el servidor HTTP. Usa **inyección manual de dependencias** — sin framework de DI.

Cinco structs de contenedor en `internal/bootstrap/containers/`:

| Contenedor | Gestiona |
|---|---|
| `RepositoryContainer` | Todos los adaptadores de repositorio GORM |
| `ServicesContainer` | Todos los servicios de dominio e infraestructura |
| `UseCaseContainer` | Todos los casos de uso de aplicación |
| `MiddlewareContainer` | Todo el middleware HTTP |
| `HandlerContainer` | Todos los handlers HTTP |

Orden de inicialización: `RepositoryContainer` → `ServicesContainer` → `UseCaseContainer` → `MiddlewareContainer` → `HandlerContainer`. Solo `ServicesContainer.Initialize()` puede devolver error (fallo de conexión Redis, JWT secret faltante, etc.).

---

## 3. Árbol de Directorios

```
.
├── cmd/
│   └── main.go                   # Punto de entrada
├── config/                       # Archivos de configuración YAML
├── assets/
│   ├── i18n/                     # Traducciones (es.yaml, en.yaml)
│   └── mails/                    # Plantillas HTML, SVG y texto plano embebidas
├── docs/
│   ├── api/                      # Documentación de endpoints de la API
│   └── internal/                 # Documentación interna de arquitectura
├── internal/
│   ├── application/
│   │   ├── auth/                 # Caso de uso de autenticación
│   │   ├── dte/                  # Casos de uso DTE y factory
│   │   ├── handlers/notification/ # AdminEmailHandler (suscriptor del bus)
│   │   └── ports/                # Interfaces de puertos de la aplicación
│   ├── bootstrap/
│   │   └── containers/           # Contenedores de inyección de dependencias
│   ├── domain/
│   │   ├── auth/                 # Modelos de autenticación y puerto del repositorio
│   │   ├── core/                 # Modelos de persistencia transversales
│   │   │   ├── event/            # Contratos del bus de eventos de dominio
│   │   │   └── notification/     # Puerto Mailer
│   │   ├── dte/
│   │   │   ├── common/           # Bloques reutilizables compartidos
│   │   │   │   ├── constants/
│   │   │   │   ├── dte_errors/
│   │   │   │   ├── interfaces/
│   │   │   │   ├── models/
│   │   │   │   ├── validator/
│   │   │   │   └── value_objects/
│   │   │   ├── ccf/
│   │   │   ├── contingency/
│   │   │   ├── credit_note/
│   │   │   ├── debit_note/
│   │   │   ├── dte_documents/
│   │   │   ├── fse/
│   │   │   ├── invalidation/
│   │   │   ├── invoice/
│   │   │   ├── remission_note/
│   │   │   ├── retention/
│   │   │   └── transmitter/
│   │   ├── health/
│   │   ├── metrics/
│   │   └── ports/                # Interfaces de puertos del dominio
│   └── infrastructure/
│       ├── adapters/             # Implementaciones de puertos
│       │   ├── events/           # InMemoryBus
│       │   ├── notifier/email/   # SMTPMailer, RedisCooldown, TemplateRenderer
│       │   └── ...               # repositories, cache, transmitter, contingency, health, etc.
│       ├── api/                  # Capa HTTP
│       ├── database/             # Driver BD + modelos GORM
│       ├── error/                # Errores de infraestructura
│       ├── jobs/                 # Jobs en segundo plano
│       └── mapper/               # Mappers de petición/respuesta
├── pkg/
│   └── shared/                   # Utilidades compartidas (logging, cargador de config)
├── scripts/
│   ├── docker-compose.yml
│   └── postman/                  # Colección y environment de Postman
└── tests/                        # Tests de integración y unitarios
```

---

## 4. Ciclo de Vida de una Petición

```
Cliente → Gorilla Mux
  → Cadena de Middleware (CORS → Auth → Timeout → Metrics → BD)
    → Handler (ej. DTEHandler.CreateInvoice)
      → DTEUseCaseFactory.CreateInvoiceUseCase()
        → GenericDTEUseCase.Create(ctx, rawRequest)
          → DTEMapper.Map(rawRequest) → InvoiceData
          → invoiceService.Create(ctx, InvoiceData, branchID)
              → ensambla agregado ElectronicInvoice
              → ejecuta InvoiceRulesValidator (cadena de estrategias)
              → ejecuta BaseValidator (reglas de reflexión por campo)
          → SequentialNumberManager.Reserve(ctx, branchID, dteType)
          → BaseTransmitter.RetryTransmission(ctx, document, token, nit)
              → SignerManager.Sign(document)
              → DTETransmitter.Transmit(ctx, signedDoc, token)  [+ circuit breaker]
          → DTEManager.SaveDTE(ctx, result)
          → AdditionalOperationsFunc (opcional; actualización de saldo crédito/débito)
          → ResponseMapper(document) → HaciendaResponse
        ← GenericDTEUseCase devuelve (HaciendaResponse, nil)
      ← Handler llama a ResponseWriter.WriteSuccess(w, response)
  ← Respuesta HTTP 200 JSON
```

---

## 5. Pipeline de Documentos DTE

Cada tipo de DTE soportado sigue el mismo patrón estructural en la capa de dominio:

```
internal/domain/dte/<tipo>/
├── <tipo>_models/
│   └── <tipo>.go          # Agregado que embebe *models.DTEDocument
├── <tipo>_service.go      # implementa ports.DTEService
└── validator/
    ├── <tipo>_validator.go # DTERulesValidator con estrategias específicas del tipo
    └── strategy/
        └── *.go            # Structs de estrategia de validación individual
```

Tipos de DTE soportados:

| Código | Tipo | Paquete |
|--------|------|---------|
| 01 | Factura Electrónica | `domain/dte/invoice/` |
| 03 | Comprobante de Crédito Fiscal | `domain/dte/ccf/` |
| 04 | Nota de Remisión | `domain/dte/remission_note/` |
| 05 | Nota de Crédito | `domain/dte/credit_note/` |
| 06 | Nota de Débito | `domain/dte/debit_note/` |
| 07 | Comprobante de Retención | `domain/dte/retention/` |
| 14 | Factura de Sujeto Excluido | `domain/dte/fse/` |
| — | Anulación | `domain/dte/invalidation/` |

---

## 6. Reglas de Dependencia

| Capa | Puede importar | No debe importar |
|---|---|---|
| Dominio | `shopspring/decimal`, `google/uuid`, stdlib | Infraestructura, HTTP, BD, Redis |
| Aplicación | Paquetes de dominio, `application/ports` | Paquetes de infraestructura directamente |
| Infraestructura | Dominio, Aplicación, todas las libs de terceros | Nada — es el anillo más externo |
| Bootstrap | Todos los paquetes internos | Código externo más allá del cableado |

Las interfaces de puerto actúan como límite de inversión de control:

- El dominio define `ports.DTEService`, `ports.SequentialNumberRepositoryPort`, etc.
- La aplicación define `application/ports.BaseTransmitter`, `HaciendaAuthManager`, etc.
- La infraestructura provee las implementaciones concretas.
- El bootstrap las conecta todas.

---

## 7. Patrones de Diseño Clave

### Patrón Estrategia — Validación

Cada tipo de DTE tiene un `DTERulesValidator` que contiene un slice de implementaciones de `ValidationStrategy`. Llamar a `Validate(document)` ejecuta cada estrategia en orden y acumula todos los errores antes de devolver. Las nuevas reglas de negocio se agregan implementando un nuevo struct de estrategia — las estrategias existentes nunca se modifican.

### Patrón Factory — Casos de Uso

`DTEUseCaseFactory` centraliza la construcción de variantes de `GenericDTEUseCase`. Los llamadores solicitan un caso de uso por tipo de documento; la factory inyecta el mapper correcto y el hook de operaciones adicionales. Agregar un nuevo tipo de DTE requiere: un nuevo mapper, un método de factory y una nueva ruta.

### Inyección Manual de Dependencias — Bootstrap

Sin framework de DI. Todo el cableado es código Go explícito en `internal/bootstrap/containers/`. El grafo de objetos completo es visible en cinco archivos. Los errores en tiempo de arranque se surfacean inmediatamente con contexto exacto.

### Circuit Breaker — API de Hacienda

Todas las llamadas al endpoint de transmisión de Hacienda están envueltas por un `CircuitBreaker` (implementado en `infrastructure/adapters/circuit/`). Si la API de Hacienda comienza a fallar, el breaker se abre y las llamadas subsiguientes fallan rápido, previniendo el agotamiento de threads.

### Modo de Contingencia

Cuando la API de Hacienda no está disponible, el sistema almacena los documentos localmente y transiciona al modo de contingencia. Un job en segundo plano (`infrastructure/jobs/`) reintenta la transmisión en lotes cuando la conectividad se restaura.

### Bus de Eventos de Dominio + Notificaciones por Correo

`InMemoryBus` (in-process, async vía goroutines) recibe eventos publicados por los puntos de emisión: `ContingencyService.StoreDocumentInContingency`, `BatchTransmitterService` (rechazos de Hacienda) y `RetransmissionJob`. El `AdminEmailHandler` los recibe, consulta el `RedisCooldown` (15 min default por `<event_name>:<aggregate_id>`) y envía un correo HTML al `ADMIN_EMAIL` usando plantillas embebidas en `assets/mails/`. Los eventos que implementan `PersistableEvent` se guardan también en la tabla `domain_events` para auditoría. Si SMTP no está configurado, el sistema degrada gracefully sin bloquear la emisión de DTE.

Detalle: `docs/internal/infrastructure/domain-events.md`.

### Value Objects

Los primitivos de dominio (`NIT`, `Amount`, `ControlNumber`, `EmissionDate`, etc.) son structs inmutables con campos privados. Solo pueden crearse via su constructor (`NewNIT(value)`, etc.), que valida el valor y devuelve un error si es inválido. Esto elimina la necesidad de validar la misma restricción en múltiples lugares.

---

## 8. Dependencias Externas

| Dependencia | Propósito |
|---|---|
| `gorilla/mux` | Router HTTP |
| `gorm.io/gorm` + `gorm.io/driver/mysql` | ORM + driver MySQL |
| `go-redis/redis` | Cliente Redis para caché de tokens |
| `golang-jwt/jwt` | Emisión y validación de JWT |
| `shopspring/decimal` | Aritmética decimal exacta (valores monetarios) |
| `google/uuid` | Generación de UUID para `GenerationCode` |
| `prometheus/client_golang` | Exposición de métricas |
| `wneessen/go-mail` | Cliente SMTP para el envío de correos al administrador |
| Servicio de firma | Servicio externo Spring Boot para firma digital de DTE (llamado via HTTP) |
| API de Hacienda | API de facturación electrónica del MH de El Salvador (llamada via HTTP) |

---

## 9. Configuración y Entorno

La configuración se carga desde archivos YAML en `config/` y se sobreescribe con variables de entorno. Configuraciones principales:

| Configuración | Descripción |
|---|---|
| DSN de base de datos | String de conexión MySQL |
| Dirección Redis | Caché de tokens |
| Secreto JWT | Usado para validación de API keys |
| URL del servicio de firma | URL base del servicio externo de firma |
| URL de la API de Hacienda | URL base del endpoint de transmisión del MH |
| Puerto | Puerto del listener HTTP (por defecto `8080`) |

Ver `pkg/shared/` para la implementación del cargador de configuración.

---

## 10. Lectura Adicional

| Documento | Ubicación |
|---|---|
| Referencia de endpoints de la API | `docs/api/README.md` |
| Análisis profundo de la capa de dominio | `docs/internal/domain/README.md` |
| Referencia de Value Objects | `docs/internal/domain/value-objects.md` |
| Servicios de Dominio | `docs/internal/domain/domain-services.md` |
| Capa de aplicación | `docs/internal/application/README.md` |
| Casos de uso paso a paso | `docs/internal/application/use-cases.md` |
| Puertos de aplicación | `docs/internal/application/ports.md` |
| Bootstrap / Inyección de dependencias | `docs/internal/bootstrap/README.md` |
| Eventos de dominio y notificaciones por correo | `docs/internal/infrastructure/domain-events.md` |
| Health checks y métricas | `docs/internal/infrastructure/health-metrics.md` |
| Colección Postman | `scripts/postman/Collection.postman_collection.json` |
