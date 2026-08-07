# Capa de Aplicación

**Ubicación:** `internal/application/`

## Qué es la Capa de Aplicación

La capa de aplicación es el nivel de orquestación del sistema. Coordina las llamadas entre la capa de dominio (que contiene las reglas de negocio) y la capa de infraestructura (que gestiona el I/O externo). El código de la capa de aplicación no debe contener lógica de negocio ni emitir consultas SQL o llamadas HTTP directamente.

La capa agrupa dos categorías de paquetes:

| Paquete | Propósito |
|---|---|
| `internal/application/dte/` | Casos de uso para creación, consulta e invalidación de documentos DTE |
| `internal/application/auth/` | Caso de uso para autenticación de sistemas y registro de usuarios |
| `internal/application/ports/` | Interfaces de puertos que la capa de aplicación necesita de la infraestructura |
| `internal/application/handlers/notification/` | Handler de eventos de dominio para notificaciones por correo al administrador |

## Qué Hacen los Casos de Uso

Un método de caso de uso:

1. Lee los claims de autenticación del contexto de la petición (`ctx.Value("claims")`)
2. Delega en los servicios de dominio para validación y aplicación de reglas de negocio
3. Llama a la infraestructura a través de puertos específicos (transmisor, firmador, repositorio)
4. Traduce los errores a tipos que el escritor de respuestas entiende
5. Devuelve un valor de resultado o un error — nunca escribe respuestas HTTP directamente

## Qué NO Hacen los Casos de Uso

- No implementan lógica de validación. Eso corresponde a los servicios de dominio.
- No emiten consultas SQL. Eso corresponde a los adaptadores de repositorio.
- No formatean respuestas HTTP. Eso corresponde a los handlers y al escritor de respuestas.
- No conocen Gorilla Mux, `http.ResponseWriter` ni la codificación JSON.

## Lista Completa de Casos de Uso

| Tipo | Struct | Método | Descripción |
|---|---|---|---|
| Creación genérica de DTE | `GenericDTEUseCase` | `Create(ctx, req)` | Orquesta la creación de cualquier tipo de DTE a través de un pipeline compartido |
| Anulación de documento | `InvalidationUseCase` | `InvalidateDocument(ctx, req)` | Valida, mapea, transmite y registra una anulación de DTE |
| Consulta de DTE | `DTEConsultUseCase` | `GetByGenerationCode(ctx, id)` | Obtiene un DTE por su código de generación (UUID) |
| Listado de DTEs | `DTEConsultUseCase` | `GetAllDTEs(ctx, r)` | Devuelve un listado paginado y filtrado de DTEs de una sucursal |
| Login | `AuthUseCase` | `Login(ctx, credentials)` | Valida las credenciales de API y emite un JWT |
| Registro | `AuthUseCase` | `Register(ctx, user)` | Crea un nuevo usuario con sucursales y devuelve las API keys |

## El DTEUseCaseFactory

Los casos de uso para tipos individuales de DTE no se instancian directamente. En su lugar, `DTEUseCaseFactory` los crea bajo demanda, inyectando el mapper correcto y el hook de operaciones adicionales para cada tipo de documento.

```go
// internal/application/dte/dte_use_case_factory.go

type DTEUseCaseFactory struct {
    authService       auth.AuthManager
    dteService        dte_documents.DTEManager
    transmitter       ports.BaseTransmitter
    sequentialManager dte_documents.SequentialNumberManager
    mapperFactory     *mapper.MapperFactory
    operationsFactory *DTEOperations
}
```

La factory expone un método por tipo de DTE. Cada método llama a `NewGenericDTEUseCase` con:

- Un `DTEMapper` específico del tipo (ej. `CreateInvoiceMapperAdapter()`)
- Una `ResponseMapperFunc` específica del tipo (ej. `GetInvoiceResponseMapper()`)
- Una `AdditionalOperationsFunc` opcional (no nula solo para notas de crédito y débito, que deben ajustar los controles de saldo)

Métodos soportados por la factory:

| Método | Tipo de DTE | Tiene Ops Adicionales |
|---|---|---|
| `CreateInvoiceUseCase` | Factura Electrónica (01) | No |
| `CreateCCFUseCase` | Comprobante de Crédito Fiscal (03) | No |
| `CreateCreditNoteUseCase` | Nota de Crédito (05) | Sí — ajusta saldo |
| `CreateDebitNoteUseCase` | Nota de Débito (06) | Sí — ajusta saldo |
| `CreateRetentionUseCase` | Comprobante de Retención (07) | No |
| `CreateRemissionNoteUseCase` | Nota de Remisión (04) | No |
| `CreateFSEUseCase` | Factura Sujeto Excluido (14) | No |
| `CreateInvalidationUseCase` | Anulación | N/A |

## GenericDTEUseCase como Patrón Central

`GenericDTEUseCase` es el struct más importante de la capa de aplicación. Toda creación de DTE — independientemente del tipo — fluye a través de su método `Create`. Los campos del struct representan cada dependencia externa que necesita:

```go
// internal/application/dte/generic_dte_use_case.go

type GenericDTEUseCase struct {
    authService       auth.AuthManager               // obtiene información del emisor
    dteService        transmissionPorts.DTEManager   // guarda/lee de la BD
    transmitter       appPorts.BaseTransmitter       // envía a Hacienda (con reintento)
    service           ports.DTEService               // servicio de dominio específico del tipo
    sequentialManager transmissionPorts.SequentialNumberManager // gestiona números de control
    mapper            mapper.DTEMapper               // mapea la petición → modelo de dominio
    responseMapper    mapper.ResponseMapperFunc      // mapea modelo de dominio → formato MH
    additionalOps     AdditionalOperationsFunc       // hook opcional post-éxito
}
```

El método `Create` es el pipeline completo de creación de DTE. Consulta la documentación detallada en la sección de navegación a continuación.

---

## Documentación Detallada

Se organizan los documentos de la capa de aplicación por área funcional. Cada archivo `.md` contiene diagramas de flujo, estructuras, dependencias y notas.

### Casos de Uso DTE

| Documento | Descripción |
|---|---|
| [GenericDTEUseCase](dte/generic-dte-use-case.md) | Pipeline central de 9 pasos para la creación de cualquier DTE |
| [DTEUseCaseFactory](dte/factory.md) | Factory que instancia el caso de uso correcto por tipo de DTE |
| [InvalidationUseCase](dte/invalidation-use-case.md) | Flujo de 10 pasos para invalidación de documentos |
| [DTEConsultUseCase](dte/consult-use-case.md) | Consulta por código de generación y listado paginado con filtros |

### Componentes de Soporte DTE

| Documento | Descripción |
|---|---|
| [Transmitter](dte/transmitter.md) | BaseTransmitter (reintentos) + MHTransmitter (HTTP a Hacienda) + Circuit Breaker |
| [Mappers](dte/mappers.md) | Sistema de mapeo: Request → Dominio → Formato Hacienda |
| [Operaciones Adicionales](dte/additional-operations.md) | Hooks post-transmisión: transacciones de balance para notas de crédito/débito |

### Autenticación

| Documento | Descripción |
|---|---|
| [AuthUseCase](auth/auth-use-case.md) | Login (JWT) y Register (generación de API keys) |

### Handlers de Eventos

| Documento | Descripción |
|---|---|
| [AdminEmailHandler](handlers/admin-email-handler.md) | Handler de eventos que envía alertas por correo con cooldown Redis |

---

## Dirección de las Dependencias

```
Handler HTTP
    → GenericDTEUseCase (aplicación)
        → auth.AuthManager (interfaz de dominio)
        → ports.DTEService (interfaz de dominio)
        → appPorts.BaseTransmitter (puerto de aplicación → infraestructura)
        → dte_documents.DTEManager (interfaz de dominio → infraestructura)
        → dte_documents.SequentialNumberManager (interfaz de dominio → infraestructura)
```

La capa de aplicación solo depende hacia adentro (interfaces del dominio) y en sus propias definiciones de puertos. Nunca importa paquetes de infraestructura directamente.
