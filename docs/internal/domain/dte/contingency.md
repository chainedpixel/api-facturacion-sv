# Servicio de Contingencia - Contingency Service

> **Paquete:** `internal/domain/dte/contingency`
> **Servicio:** `ContingencyService`
> **Interfaz implementada:** `ContingencyManager`

## Descripción General

El servicio de contingencia maneja la situación en la que la API de Hacienda no está disponible. Cuando un DTE no puede ser transmitido por problemas de conectividad, tiempo de espera agotado, o errores del servidor de Hacienda, el documento se almacena localmente con estado `PENDING` y se retransmite automáticamente cuando el servicio se restablece.

Características principales:
- **Almacenamiento local** — Los documentos se persisten con estado `PENDING`
- **Retransmisión en lotes** — Los documentos pendientes se agrupan y envían por lotes
- **Evento de contingencia** — Se notifica a Hacienda del evento de contingencia antes de retransmitir
- **Agrupamiento inteligente** — Los documentos se agrupan por NIT del sistema y tipo de DTE
- **Verificación de estado** — Se consulta Hacienda para documentos que ya pueden haber sido procesados

---

## Estructura del Servicio

```go
type ContingencyService struct {
    contingencyEvents ContingencyEventSender
    timeProvider      ports.TimeProvider
    docSvc            *contingencyDocumentSvc
    txSvc             *contingencyTransmissionSvc
    bus               event.Bus
}
```

El bus de eventos se inyecta después de la construcción con `SetEventBus(bus event.Bus)`; si no se configura, no se publican eventos ni alertas.

El servicio se compone internamente de dos sub-servicios especializados:

### `contingencyDocumentSvc` — Gestión de Documentos

Se encarga de almacenar documentos de contingencia (registro del DTE en estado `PENDING`, registro de contingencia y marca de la reserva del número de control), consultarlos, actualizar su estado y marcarlos como `EXPIRED`.

### `contingencyTransmissionSvc` — Transmisión

Se encarga de la autenticación con Hacienda, firma de documentos (se guarda el JWS transmitido en `SignedDocument`, base de la `firmaElectronica` del Archivo DTE), transmisión por lotes y consulta del estado de un documento ya enviado.

### Dependencias Completas

| Dependencia | Tipo | Propósito |
|---|---|---|
| `authManager` | `AuthManager` | Se obtienen credenciales de Hacienda |
| `dteManager` | `DTEManager` | Se gestionan documentos DTE |
| `repo` | `ContingencyRepositoryPort` | Se persisten documentos de contingencia |
| `haciendaAuth` | `HaciendaAuthManager` | Se gestionan tokens de Hacienda |
| `cache` | `CacheManager` | Se cachean tokens |
| `tokenService` | `TokenManager` | Se gestionan tokens JWT |
| `signer` | `SignerManager` | Se firman documentos digitalmente |
| `batchTransmitter` | `BatchTransmitterPort` | Se transmiten lotes a Hacienda |
| `contingencyEvents` | `ContingencyEventSender` | Se envían eventos de contingencia |
| `sequentialManager` | `SequentialNumberManager` | Se gestionan números de control |
| `timeProvider` | `TimeProvider` | Se obtiene la hora actual |
| `cfg` | `*TransmissionConfig` | Configuración de transmisión (tamaño de lote, intervalos) |

### Constructor

```go
func NewContingencyManager(
    authManager, dteManager, repo, haciendaAuth,
    cache, tokenService, signer, batchTransmitter,
    contingencyEvents, sequentialManager, timeProvider, cfg,
) ContingencyManager
```

---

## Interfaz `ContingencyManager`

```go
type ContingencyManager interface {
    StoreDocumentInContingency(
        ctx context.Context,
        document interface{},
        dteType string,
        contingencyType int8,
        reason string,
    ) error
    RetransmitPendingDocuments(ctx context.Context) error
}
```

---

## Método: `StoreDocumentInContingency`

```go
func (s *ContingencyService) StoreDocumentInContingency(
    ctx context.Context,
    document interface{},
    dteType string,
    contingencyType int8,
    reason string,
) error
```

Se almacena un documento que no pudo ser transmitido a Hacienda:

### Flujo

```
1. Se toman los claims de autenticación del contexto (sucursal y NIT)
2. Se extrae la identificación del documento (código de generación y número de control);
   si falta alguno se retorna MissingDTEIdentification
3. Se crea el DTE con transmisión CONTINGENCY y estado PENDING, sin evidencia de transmisión
4. Se crea el registro ContingencyDocument (BranchID, ContingencyType, Reason, DocumentID)
5. Se marca la reserva del número de control como contingencia
   (un fallo en este paso solo se registra como advertencia)
6. Si hay bus de eventos, se publica ContingencyActivatedEvent
```

### Tipos de Contingencia

| Código | Motivo |
|---|---|
| `1` | No disponibilidad de Ministerio de Hacienda |
| `2` | Falla de conexión del sistema |
| `3` | Falla del servicio de Internet |
| `4` | Falla de energía eléctrica |
| `5` | Otro motivo |

---

## Método: `RetransmitPendingDocuments`

```go
func (s *ContingencyService) RetransmitPendingDocuments(ctx context.Context) error
```

Se retransmiten los documentos pendientes de contingencia. Este es el método más complejo del servicio.

### Flujo de Retransmisión

```
1. Obtener documentos pendientes (límite `config.Server.MaxBatchSize`), primero los que ya tienen sello del evento
   (event_stamped_at más antiguo) y después los que aún no están en un evento (por created_at)
1.1 Vencidos: si pasaron más de 72 horas desde event_stamped_at, el documento pasa a EXPIRED,
    no se reintenta y se publica RetransmissionJobFailedEvent (job contingency_72h_expired)
1.2 Si el documento pendiente más antiguo (por created_at) supera 3 días (TechnicalReportThreshold),
    se publica la alerta de Informe Técnico (job contingency_technical_report)
2. Agrupar documentos por NIT del sistema y tipo de DTE
3. Por cada NIT, con todos sus documentos:
   │
   ├── 3a. Enviar evento de contingencia a Hacienda (PrepareAndSendContingencyEvent)
   │        ├── Si el evento ya existe (ContingencyEventExistsError) → se consulta en Hacienda el estado
   │        │   de los documentos indicados (verifyAndUpdateExistingDocuments) y se continúa
   │        └── Si falla por otra causa → se omite este NIT en el ciclo
   │
   └── 3b. Por cada tipo de DTE del NIT (processSystemDocumentsByType):
            │
            ├── Obtener token y credenciales de la sucursal
            └── Por cada lote (tamaño de lote configurado):
                 ├── Firmar los documentos y transmitir el lote (BatchTransmitter)
                 └── Verificar el estado del lote; el transmisor actualiza cada documento:
                      ├── RECEIVED → Confirmar reserva de número
                      ├── REJECTED → Liberar reserva de número
                      └── PENDING → Mantener para siguiente intento
```

Un lote que falla se registra en el log y se continúa con el siguiente. El job que invoca `RetransmitPendingDocuments` se ubica en `internal/infrastructure/jobs/contingency_retransmission_job.go`.

### Agrupamiento de Documentos

```go
func groupBySystemAndType(
    docs []dte.ContingencyDocument,
) map[string]map[string][]dte.ContingencyDocument
```

Se agrupan los documentos en una estructura de dos niveles:

```
NIT del Sistema
  └── Tipo de DTE
        └── []ContingencyDocument
```

El NIT de agrupación es el del usuario de la sucursal del documento (`Branch.User.NIT`). Para la verificación de documentos existentes, el NIT se extrae del campo `emisor.nit` del JSON almacenado (`extractNITFromDocument`).

### Evento de Contingencia

Antes de retransmitir documentos, se debe notificar a Hacienda que hubo una contingencia:

```go
contingencyEvents.PrepareAndSendContingencyEvent(ctx, docs)
```

El adaptador (`internal/infrastructure/adapters/contingency`) arma el evento con la identificación (versión 4), el emisor, el detalle de los DTE (`noItem` correlativo) y el motivo (fecha y hora de inicio tomadas del primer documento pendiente de la sucursal, con holgura de 60 segundos antes del inicio y 10 segundos después del fin). Al recibir el sello, el evento se registra como `DTEEvent` de tipo `CONTINGENCY` con su evidencia de transmisión (ver [transmitter-models.md](transmitter-models.md)) y se guarda `event_stamped_at` en los documentos que aún no lo tenían, con la fecha de procesamiento de Hacienda (`ContingencyRepositoryPort.MarkEventStamped`). Un reenvío del evento no reinicia la ventana de 72 horas.

Si el evento ya fue registrado previamente (error `ContingencyEventExistsError`), se continúa con la retransmisión normalmente.

### Procesamiento por Tipo de DTE

```go
func (s *ContingencyService) processSystemDocumentsByType(
    ctx context.Context,
    systemNIT string,
    dteType string,
    docs []dte.ContingencyDocument,
) error
```

1. Se obtienen el token y las credenciales de Hacienda de la sucursal del primer documento
2. Se divide la lista en lotes del tamaño configurado
3. Por cada lote se firman y transmiten los documentos y se verifica el estado del lote
4. El estado de cada documento se actualiza según la respuesta de Hacienda (en el adaptador de transmisión por lotes)

### Verificación de Documentos Existentes

```go
func (s *ContingencyService) verifyAndUpdateExistingDocuments(
    ctx context.Context,
    docs []dte.ContingencyDocument,
)
```

Se consulta en Hacienda el estado de documentos que pueden haber sido procesados en intentos anteriores. Por cada documento se actualiza el DTE (estado, sello y fecha de procesamiento) y luego, agrupados por estado, los registros de contingencia (`batch_id = "verified"`) con su sello. La función no retorna error: los fallos por documento se registran en el log y se continúa.

### Mapeo de Estados

```go
func mapHaciendaStatusToInternal(haciendaStatus string) string
```

`PROCESADO` se mapea a `RECEIVED`, `RECHAZADO` a `REJECTED` y cualquier otro estado a `PENDING`.

### Vencimiento a 72 horas y alerta de 3 días

```go
func IsContingencyWindowExpired(eventStampedAt *time.Time, now time.Time) bool
```

Definido en `contingency_window.go`:

| Constante | Valor | Propósito |
|---|---|---|
| `TransmissionWindow` | 72 horas | Tiempo permitido para transmitir los documentos desde el sello del evento de contingencia |
| `TechnicalReportThreshold` | 72 horas | Duración de contingencia a partir de la cual se alerta la necesidad del Informe Técnico de Contingencia |
| `ContingencyExpiredCode` | `CONTINGENCY_72H_EXPIRED` | Código usado en las alertas al administrador por documentos vencidos |

Un documento vence cuando `event_stamped_at` existe y han transcurrido más de 72 horas. Si no tiene sello de evento (`EventStampedAt` nulo) nunca vence. Al vencer: se marca el registro de contingencia y el DTE como `EXPIRED` (`batch_id = "expired"`, con una observación que indica la fecha del sello), se excluye del reintento y se publica `RetransmissionJobFailedEvent` con el job `contingency_72h_expired` y el resumen de documentos (número de control y código `CONTINGENCY_72H_EXPIRED`). La alerta de 3 días se publica con el job `contingency_technical_report`.

---

## Modelo de Evento de Contingencia

### ContingencyEvent

```go
type ContingencyEvent struct {
    Identification ContingencyIdentification
    Issuer         ContingencyIssuer
    DTEDetails     []DTEDetail
    Reason         ContingencyReason
}
```

### ContingencyIdentification

```go
type ContingencyIdentification struct {
    Version          int       // Versión del esquema
    Ambient          string    // "00" (test) o "01" (producción)
    GenerationCode   string    // UUID del evento de contingencia
    TransmissionDate string    // Fecha de transmisión
    TransmissionTime string    // Hora de transmisión
}
```

### ContingencyIssuer

```go
type ContingencyIssuer struct {
    NIT                  string
    Name                 string
    ResponsibleName      string   // Nombre del responsable de la contingencia
    ResponsibleDocType   string   // Tipo de documento del responsable
    ResponsibleDocNumber string   // Número de documento del responsable
    EstablishmentType    string
    Phone                string
    Email                string
    EstablishmentCodeMH  *string
    POSCode              *string
}
```

### DTEDetail

```go
type DTEDetail struct {
    ItemNumber     int     // Número secuencial del detalle
    GenerationCode string  // UUID del documento DTE
    DocumentType   string  // Tipo de DTE (01, 03, etc.)
}
```

### ContingencyReason

```go
type ContingencyReason struct {
    StartDate         string  // Fecha de inicio de la contingencia
    EndDate           string  // Fecha de fin de la contingencia
    StartTime         string  // Hora de inicio
    EndTime           string  // Hora de fin
    ContingencyType   int8    // Código del tipo de contingencia
    ContingencyReason *string // Descripción del motivo (nulo si el documento no tiene motivo)
}
```

---

## Política de Reintento

```go
type RetryPolicy struct {
    MaxAttempts     int            // Número máximo de intentos
    InitialInterval time.Duration  // Intervalo inicial entre intentos
    MaxInterval     time.Duration  // Intervalo máximo
    BackoffFactor   float64        // Factor de retroceso exponencial
}
```

La política la construye `TransmissionConfig.GetRetryPolicy()` (con `MaxAttempts` fijo en 3) y la consume el transmisor de lotes de infraestructura, no el servicio de contingencia. Los reintentos son exponenciales:

```
Intento 1: espera InitialInterval
Intento 2: espera InitialInterval * BackoffFactor
Intento 3: espera InitialInterval * BackoffFactor²
...hasta MaxInterval o MaxAttempts
```

---

## Cuándo Se Activa la Contingencia

La contingencia se evalúa en la capa de aplicación (`GenericDTEUseCase.shouldHandleAsContingency()`, que conserva la reserva del número de control) y en infraestructura (`ContingencyHandler.shouldHandleAsContingency()`, que almacena el documento con `StoreDocumentInContingency`), con el mismo criterio:

### Se activa para:
- Errores de red (timeouts, conexión rechazada)
- HTTP 5xx del servidor de Hacienda
- Errores de timeout

### NO se activa para:
- `ValidationError` — Errores de validación del documento
- `HaciendaResponseError` con estado `RECHAZADO` — Hacienda rechazó el documento
- `ServiceError` — Errores internos del servicio
- `DTEError` — Errores del dominio

### Flag de Configuración

```go
DocumentConfig.UsesContingency = true/false
```

Determina si un tipo de DTE específico puede entrar en contingencia. Se define al registrar cada documento en `internal/bootstrap/containers/handlers.go`.

---

## DTEs Que Soportan Contingencia

| DTE | Código | `UsesContingency` (endpoint registrado) | En `ValidDTETypesForContingency` |
|---|---|---|---|
| Factura Electrónica | `01` | Sí | Sí |
| CCF Electrónico | `03` | Sí | Sí |
| Nota de Remisión | `04` | Sí | Sí |
| Nota de Crédito | `05` | No | Sí |
| Nota de Débito | `06` | Sí | Sí |
| Retención | `07` | No | No |
| Factura Exportación | `11` | Sin endpoint registrado | Sí |
| FSE | `14` | Sí | Sí |
| **Invalidación** | — | **No** | — |

Los tipos `08`, `09` y `15` no tienen endpoint de creación registrado ni están en `ValidDTETypesForContingency`. Actualmente `ValidDTETypesForContingency` no se consulta fuera de su definición.

---

## Diagrama de Flujo de Retransmisión

```
RetransmitPendingDocuments()
  │
  ▼
GetPending(limit) ──→ []ContingencyDocument
  │
  ▼
expireOverdueDocuments() ──→ EXPIRED (más de 72 h desde el sello del evento)
alertLongContingency()   ──→ alerta de Informe Técnico (más de 3 días)
  │
  ▼
groupBySystemAndType()
  │
  ▼
┌─────────────────────────────────┐
│ Por cada NIT del sistema:       │
│   │                             │
│   ▼                             │
│ PrepareAndSendContingencyEvent()│
│   │                             │
│   ├── Éxito → continuar         │
│   ├── Ya existe → verificar     │
│   │   estado y continuar        │
│   └── Error → saltar NIT        │
│   │                             │
│   ▼                             │
│ Por cada tipo de DTE:           │
│   │                             │
│   ├── getTokenAndCreds()        │
│   ├── signDocuments()           │
│   ├── TransmitBatch()           │
│   └── VerifyContingencyBatch    │
│       Status()                  │
│         │                       │
│         ├── RECEIVED:           │
│         │   ConfirmReservation()│
│         │                       │
│         ├── REJECTED:           │
│         │   ReleaseReservation()│
│         │                       │
│         └── PENDING:            │
│             (siguiente intento) │
└─────────────────────────────────┘
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/contingency/contingency_service.go` | Servicio principal |
| `internal/domain/dte/contingency/contingency_window.go` | Ventana de 72 horas y constantes de alerta |
| `internal/domain/dte/contingency/contingency_repository_port.go` | Puerto del repositorio (`Create`, `GetPending`, `UpdateBatch`, `GetFirstContingencyTimestamp`, `MarkEventStamped`) |
| `internal/domain/dte/contingency/contingency_event_sender_port.go` | Puerto `ContingencyEventSender` |
| `internal/domain/dte/contingency/models/` | Modelos de contingencia |
| `internal/domain/core/dte/contingency_document_model.go` | `ContingencyDocument` (incluye `EventStampedAt`) |
| `internal/domain/ports/` | Puertos del dominio |

---

## Notas

1. **12 dependencias**: El servicio de contingencia tiene la mayor cantidad de dependencias de todo el dominio, ya que orquesta autenticación, firma, transmisión, persistencia, y gestión de números.
2. **Evento duplicado**: Si el evento de contingencia ya existe en Hacienda, se ignora el error y se continúa con la retransmisión. Esto es intencional para manejar reintentos.
3. **Agrupamiento por NIT**: El sistema puede manejar múltiples contribuyentes (multi-tenant), por lo que los documentos se agrupan por NIT del emisor.
4. **Sin intervención manual**: La retransmisión es automática y se ejecuta como proceso de fondo.
5. **Modelo de facturación diferido**: Los documentos en contingencia usan `ModeloFacturacionDiferido` (modelo 2) en lugar de `ModeloFacturacionPrevio` (modelo 1).
6. **Plazo de 72 horas**: `contingency_documents.event_stamped_at` guarda la fecha del sello del evento de contingencia (solo la primera vez). Desde ahí corren 72 horas (`contingency.TransmissionWindow`); al vencer, el documento queda `EXPIRED` y se considera no emitido. Los documentos registrados antes de esta columna no tienen fecha y no vencen. La alerta del Informe Técnico se calcula con `created_at` del documento pendiente más antiguo.
