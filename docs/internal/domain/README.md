# Capa de Dominio

La capa de dominio es el núcleo más interno de la arquitectura en capas de la aplicación. Contiene la lógica de negocio pura del sistema de documentos tributarios electrónicos (DTE): cómo luce una factura válida, cómo se estructuran los números NIT, cómo se calcula el IVA, y qué tipos de documentos pueden referenciar a otros. Nada en esta capa sabe que existen HTTP, una base de datos, Redis ni ningún servicio externo.

## La Regla del "Núcleo Puro"

Todos los paquetes bajo `internal/domain/` deben cumplir una invariante: **ninguna dependencia de infraestructura**. En concreto:

- Sin imports de `net/http` ni de frameworks (sin Gin, Echo, etc.)
- Sin imports de ORM ni drivers SQL (sin GORM, sin `database/sql`)
- Sin imports de clientes de caché (sin `go-redis`, sin `memcache`)
- Sin clientes de APIs de terceros

Los únicos módulos externos de Go permitidos son:

| Módulo | Propósito |
|--------|---------|
| `github.com/shopspring/decimal` | Aritmética decimal exacta para montos monetarios — evita errores de punto flotante en cálculos de impuestos |
| `github.com/google/uuid` | Generación de UUIDs para los value objects `GenerationCode` |

Todo lo demás que el dominio necesita del mundo exterior se expresa como una **interfaz de puerto** en `internal/domain/ports/`. La infraestructura implementa esas interfaces; el dominio solo conoce la interfaz.

## Árbol de Directorios

```
internal/domain/
├── auth/                       Modelos de dominio de autenticación y puerto del repositorio
│   └── models/                 AuthClaims, HaciendaCredentials
├── core/                       Modelos de persistencia transversales (sin lógica de negocio)
│   ├── dte/                    DTEDocument, DTEDetails, BalanceControl, BalanceTransaction
│   ├── error/                  Errores centinela (ErrBranchMatrixNotFound, etc.)
│   ├── event/                  Modelo DomainEvent para event sourcing
│   ├── notification/           Modelos NotifiableUser, UserNotification
│   └── user/                   User, BranchOffice, Address — agregado raíz del tenant
├── ports/                      Interfaces de puertos (límite de inversión de dependencias)
│   ├── circuit_breaker_port.go CircuitManager — circuit breaker para la API de MH
│   ├── crypt_manager_port.go   CryptManager — generación/cifrado de API keys
│   ├── dte_services_port.go    DTEService — el único método Create que implementan todos los servicios DTE
│   ├── failed_secuence_number_repository_port.go FailedSequenceNumberRepositoryPort
│   ├── secuence_number_repository_port.go        SequentialNumberRepositoryPort
│   ├── time_provider_service_interface.go        TimeProvider
│   └── token_cache_port.go     CacheManager, TokenManager
└── dte/                        Toda la lógica de negocio de documentos electrónicos
    ├── common/                 Bloques reutilizables compartidos por todos los tipos de DTE
    │   ├── constants/          Constantes con nombre: códigos de tipo DTE, códigos de impuesto, helpers de código de error
    │   ├── dte_errors/         DTEError, ValidationError, CompositeError
    │   ├── interfaces/         Todas las interfaces del dominio (documento, ítem, resumen, …)
    │   ├── models/             Structs de modelos concretos que implementan las interfaces
    │   ├── validator/          BaseValidator (reflexión) + DTERulesValidator (cadena de estrategias)
    │   └── value_objects/      Primitivos inmutables y auto-validantes
    │       ├── base/           Email, Phone
    │       ├── document/       DTEType, ModelType, ContingencyType, Version, …
    │       ├── financial/      Amount, Discount, Currency, Tax, PaymentCondition, …
    │       ├── identification/ NIT, NRC, ActivityCode, ControlNumber, GenerationCode, DocumentNumber
    │       ├── item/           ItemNumber, ItemType, Quantity, UnitMeasure, ItemCode
    │       ├── location/       Department, Municipality, Address
    │       └── temporal/       EmissionDate, EmissionTime
    ├── dte_documents/          SequentialNumberManager + puerto DTEManager + servicio DTE compartido
    ├── invoice/                Factura Electrónica (01)
    │   ├── invoice_models/     ElectronicInvoice, InvoiceItem, InvoiceSummary, InvoiceData
    │   ├── invoice_service.go  Servicio de dominio — flujo Create
    │   └── validator/          InvoiceRulesValidator + 4 estrategias específicas de factura
    ├── ccf/                    Comprobante de Crédito Fiscal Electrónico (03)
    ├── fse/                    Factura de Sujeto Excluido (14)
    ├── credit_note/            Nota de Crédito Electrónica (05)
    ├── debit_note/             Nota de Débito Electrónica (06)
    ├── retention/              Comprobante de Retención Electrónico (07)
    ├── remission_note/         Nota de Remisión Electrónica (04)
    ├── invalidation/           Documentos de anulación
    ├── contingency/            Modelos de documentos en contingencia
    └── transmitter/            Modelos del lado del transmisor
```

## Cómo Navegar la Capa de Dominio

**Partiendo de una pregunta de negocio:**

- "¿Qué es un NIT válido?" → `internal/domain/dte/common/value_objects/identification/nit.go`
- "¿Cómo se valida el IVA en una factura?" → `internal/domain/dte/invoice/validator/strategy/invoice_tax_strategy.go`
- "¿Qué campos tiene un documento de factura?" → `internal/domain/dte/invoice/invoice_models/`
- "¿Qué errores puede emitir el dominio?" → `internal/domain/dte/common/dte_errors/`
- "¿Qué interfaz implementa el adaptador de base de datos?" → `internal/domain/ports/`
- "¿Cómo se genera un número de control?" → `internal/domain/dte/dte_documents/sequential_number_service.go`

**Puntos de entrada principales para cada tipo de DTE:**

Cada tipo de DTE sigue el mismo patrón:
1. `<tipo>_models/` — el modelo agregado que embebe `*models.DTEDocument`
2. `<tipo>_service.go` — el servicio de dominio que implementa `ports.DTEService`
3. `validator/<tipo>_validator.go` — el `DTERulesValidator` específico del tipo
4. `validator/strategy/` — estrategias de validación específicas del tipo

Consulta los archivos de documentación individuales para mayor detalle:

### Documentación de Referencia

- [value-objects.md](value-objects.md) — todos los Value Objects, constructores y validaciones
- [domain-services.md](domain-services.md) — patrones de servicios, SequentialNumberManager, DTEManager
- [models.md](models.md) — modelos del dominio (DTEDocument, User, BranchOffice, etc.)
- [ports.md](ports.md) — puertos e interfaces del dominio
- [errors.md](errors.md) — sistema de errores (DTEError, ValidationError, errores centinela)
- [constants.md](constants.md) — constantes y enumeraciones (tipos de DTE, impuestos, pagos, etc.)
- [sequential-numbers.md](sequential-numbers.md) — sistema de números de control secuenciales

### Autenticación de Dominio

- [auth/auth-system.md](auth/auth-system.md) — AuthCredentials, AuthClaims, AuthService, StandardAuthStrategy, Strategy Pattern

### Núcleo del Sistema (core/)

- [core/event-system.md](core/event-system.md) — Bus de eventos, Event, Handler, PersistableEvent, Repository, los 3 eventos de dominio

### Transmisión — Modelos

- [dte/transmitter-models.md](dte/transmitter-models.md) — HaciendaRequest/Response, TransmitResult, BatchRequest/Response, TransmissionConfig, BatchTransmitterPort

### Documentación por Tipo de DTE

| DTE | Código | Servicio | Validación |
|-----|--------|----------|------------|
| Factura Electrónica | `01` | [dte/invoice.md](dte/invoice.md) | [validation/invoice-validation.md](validation/invoice-validation.md) |
| CCF Electrónico | `03` | [dte/ccf.md](dte/ccf.md) | [validation/ccf-validation.md](validation/ccf-validation.md) |
| Nota de Remisión | `04` | [dte/remission-note.md](dte/remission-note.md) | [validation/remission-note-validation.md](validation/remission-note-validation.md) |
| Nota de Crédito | `05` | [dte/credit-note.md](dte/credit-note.md) | [validation/credit-note-validation.md](validation/credit-note-validation.md) |
| Nota de Débito | `06` | [dte/debit-note.md](dte/debit-note.md) | [validation/debit-note-validation.md](validation/debit-note-validation.md) |
| Retención | `07` | [dte/retention.md](dte/retention.md) | [validation/retention-validation.md](validation/retention-validation.md) |
| FSE | `14` | [dte/fse.md](dte/fse.md) | [validation/fse-validation.md](validation/fse-validation.md) |
| Invalidación | — | [dte/invalidation.md](dte/invalidation.md) | [validation/invalidation-validation.md](validation/invalidation-validation.md) |
| Contingencia | — | [dte/contingency.md](dte/contingency.md) | — |

### Sistema de Validación

- [validation/README.md](validation/README.md) — arquitectura del sistema de validación (Strategy Pattern)
