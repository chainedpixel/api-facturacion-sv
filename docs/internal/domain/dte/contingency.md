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
}
```

El servicio se compone internamente de dos sub-servicios especializados:

### `contingencyDocumentSvc` — Gestión de Documentos

Se encarga de almacenar, agrupar y consultar documentos de contingencia.

### `contingencyTransmissionSvc` — Transmisión

Se encarga de la autenticación con Hacienda, firma de documentos, y transmisión por lotes.

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
| `cfg` | Config | Configuración del sistema |

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
1. Serializar el documento a JSON
2. Crear registro ContingencyDocument con:
   - BranchID, ContingencyType, Reason
   - Estado: PENDING
   - JSON del documento
3. Persistir en base de datos
4. Registrar evento de dominio (log)
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
1. Obtener documentos pendientes (límite configurable)
2. Agrupar documentos por NIT del sistema y tipo de DTE
3. Por cada grupo (NIT + tipo):
   │
   ├── 3a. Enviar evento de contingencia a Hacienda
   │        ├── Si el evento ya existe → Continuar (no es error)
   │        └── Si falla → Detener procesamiento de este grupo
   │
   ├── 3b. Obtener token de autenticación de Hacienda
   │
   └── 3c. Procesar lotes por tipo de DTE
            │
            ├── Transmitir lote a Hacienda (BatchTransmitter)
            ├── Verificar estado de documentos ya existentes
            └── Actualizar estado de cada documento:
                 ├── RECEIVED → Confirmar reserva de número
                 ├── REJECTED → Liberar reserva de número
                 └── PENDING → Mantener para siguiente intento
```

### Agrupamiento de Documentos

```go
func (s *ContingencyService) groupBySystemAndType(
    docs []dte.ContingencyDocument,
) map[string]map[string][]dte.ContingencyDocument
```

Se agrupan los documentos en una estructura de dos niveles:

```
NIT del Sistema
  └── Tipo de DTE
        └── []ContingencyDocument
```

Se extrae el NIT del emisor del JSON almacenado:

```go
func (s *ContingencyService) extractNITFromDocument(jsonData string) (string, error)
```

### Evento de Contingencia

Antes de retransmitir documentos, se debe notificar a Hacienda que hubo una contingencia:

```go
contingencyEvents.PrepareAndSendContingencyEvent(ctx, docs, branchID)
```

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

1. Se obtienen credenciales de Hacienda por NIT
2. Se obtiene la versión del DTE
3. Se transmite el lote a Hacienda
4. Se actualiza el estado de cada documento según la respuesta

### Verificación de Documentos Existentes

```go
func (s *ContingencyService) verifyAndUpdateExistingDocuments(
    ctx context.Context,
    docs []dte.ContingencyDocument,
) error
```

Se verifica el estado en Hacienda de documentos que pueden haber sido procesados en intentos anteriores.

### Mapeo de Estados

```go
func (s *ContingencyService) mapHaciendaStatusToInternal(haciendaStatus string) string
```

Se mapean los estados de Hacienda a estados internos del sistema.

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
    ContingencyReason string  // Descripción del motivo
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

La retransmisión de documentos pendientes se ejecuta como un proceso de fondo con reintentos exponenciales:

```
Intento 1: espera InitialInterval
Intento 2: espera InitialInterval * BackoffFactor
Intento 3: espera InitialInterval * BackoffFactor²
...hasta MaxInterval o MaxAttempts
```

---

## Cuándo Se Activa la Contingencia

La contingencia se activa en la capa de aplicación (`shouldHandleAsContingency()`):

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

Determina si un tipo de DTE específico puede entrar en contingencia.

---

## DTEs Que Soportan Contingencia

| DTE | Código | Contingencia |
|---|---|---|
| Factura Electrónica | `01` | Sí |
| CCF Electrónico | `03` | Sí |
| Nota de Remisión | `04` | Sí |
| Nota de Crédito | `05` | Sí |
| Nota de Débito | `06` | Sí |
| Retención | `07` | Sí |
| Liquidación | `08` | No |
| Doc. Contable Liquidación | `09` | No |
| Factura Exportación | `11` | Sí |
| FSE | `14` | Sí |
| Donación | `15` | Sí |
| **Invalidación** | — | **No** |

---

## Diagrama de Flujo de Retransmisión

```
RetransmitPendingDocuments()
  │
  ▼
GetPending(limit) ──→ []ContingencyDocument
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
│   ├── Ya existe → continuar     │
│   └── Error → saltar grupo      │
│   │                             │
│   ▼                             │
│ Por cada tipo de DTE:           │
│   │                             │
│   ├── GetHaciendaToken()        │
│   ├── TransmitBatch()           │
│   ├── VerifyExistingDocs()      │
│   └── UpdateBatch()             │
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
| `internal/domain/dte/contingency/models/` | Modelos de contingencia |
| `internal/domain/ports/` | Puertos del dominio |

---

## Notas

1. **12 dependencias**: El servicio de contingencia tiene la mayor cantidad de dependencias de todo el dominio, ya que orquesta autenticación, firma, transmisión, persistencia, y gestión de números.
2. **Evento duplicado**: Si el evento de contingencia ya existe en Hacienda, se ignora el error y se continúa con la retransmisión. Esto es intencional para manejar reintentos.
3. **Agrupamiento por NIT**: El sistema puede manejar múltiples contribuyentes (multi-tenant), por lo que los documentos se agrupan por NIT del emisor.
4. **Sin intervención manual**: La retransmisión es automática y se ejecuta como proceso de fondo.
5. **Modelo de facturación diferido**: Los documentos en contingencia usan `ModeloFacturacionDiferido` (modelo 2) en lugar de `ModeloFacturacionPrevio` (modelo 1).
