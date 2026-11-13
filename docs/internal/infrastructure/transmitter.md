# Transmisores — Comunicación con Hacienda

> **Paquetes:**
> - `internal/infrastructure/adapters/transmitter` — Transmisión individual de DTEs y eventos
> - `internal/infrastructure/adapters/transmitter/batch` — Transmisión por lotes
> - `internal/infrastructure/adapters/signing` — Autenticación con Hacienda y firma

## Descripción General

El sistema de transmisión de infraestructura gestiona la comunicación HTTP real con la API del Ministerio de Hacienda (MH). Se compone de cuatro componentes:

1. **MHTransmitter** — Envío individual de DTEs y de eventos de invalidación
2. **BatchTransmitterService** — Envío de lotes de contingencia
3. **HaciendaAuthService** — Autenticación con Hacienda
4. **DTESigner** — Firma digital de documentos

El envío del evento de contingencia lo realiza `ContingencyHTTPService` (ver [contingency.md](contingency.md)).

---

## MHTransmitter

> **Archivo:** `adapters/transmitter/mh_transmitter.go`
> **Implementa:** `ports.DTETransmitter`

### Estructura

```go
type MHTransmitter struct {
    HaciendaToken      string
    haciendaAuth       ports.HaciendaAuthManager
    failedSequenceRepo ports.FailedSequenceNumberRepositoryPort
    httpClient         *http.Client  // timeout: 8s
    processors         map[string]DocumentProcessor
}
```

### Resultado de la Transmisión

```go
type TransmitResult struct {
    Status           string
    ReceptionStamp   *string
    ProcessingDate   string   // fhProcesamiento
    MessageCode      string   // codigoMsg
    MessageDesc      string   // descripcionMsg
    Observations     []string
    SignedDocument   string   // JWS transmitido (lo asigna la capa de aplicación tras firmar)
    HaciendaResponse string   // respuesta completa de Hacienda en JSON
}
```

`Transmit` y `CheckDocumentStatus` llenan `HaciendaResponse` serializando la respuesta completa de Hacienda (`marshalHaciendaResponse`). `SignedDocument` no lo asigna el transmisor: lo completa `base_transmit.go` (capa de aplicación) con el JWS enviado. Ambos campos se persisten luego como evidencia (ver [database.md](database.md)): `signed_document`, `processing_date`, `hacienda_response` y `reception_stamp`.

### Procesadores de Documento

Se registran dos procesadores que determinan cómo construir el request según el tipo de documento:

| Procesador | Detección | Endpoint |
|---|---|---|
| `DTEProcessor` | Cualquier documento que no sea evento de invalidación | `ReceptionURL` |
| `InvalidationProcessor` | Evento de invalidación | `NullifyURL` |

La detección (`isInvalidationDocument`) examina el JSON del documento: es invalidación si trae un objeto `motivo` con las claves `tipoAnulacion` y `nombreResponsable`; si no hay `motivo`, se considera invalidación cuando el documento no tiene `resumen` ni `cuerpoDocumento`.

#### DTEProcessor

> **Archivo:** `adapters/transmitter/processors/dte_processor.go`

Construye el `HaciendaRequest` para la transmisión normal de DTEs.

```
ProcessRequest(signedDoc, document)
  │
  ├── GetDocumentRequestData(document)
  │   → version, tipoDte, codigoGeneracion, correlativo (4.º segmento del numeroControl; 1 si no hay)
  │
  └── HaciendaRequest{
        Ambient:        config.Server.AmbientCode,
        SendID:         correlativo,
        Version:        version,
        Document:       signedDoc,
        DTEType:        tipoDte,
        GenerationCode: codigoGeneracion,
        URL:            ReceptionURL,
      }
```

Cuerpo JSON enviado a Hacienda para un DTE:

```json
{
  "ambiente": "00",
  "idEnvio": 1,
  "version": 3,
  "documento": "<JWS>",
  "tipoDte": "01",
  "codigoGeneracion": "UUID"
}
```

#### InvalidationProcessor

> **Archivo:** `adapters/transmitter/processors/invalidation_processor.go`

Construye el request de invalidación apuntando a `config.MHPaths.NullifyURL`. No incluye `tipoDte` ni `codigoGeneracion` (campos con `omitempty`):

```json
{
  "ambiente": "00",
  "idEnvio": 1,
  "version": 3,
  "documento": "<JWS del evento de invalidación>"
}
```

`version` es la versión de la estructura del evento (`identificacion.version`) y `idEnvio` el correlativo extraído del número de control del documento (si no existe, `1`).

### `Transmit` — Flujo

```
Transmit(ctx, document, signedDoc, systemToken)
  │
  ├── [0] Contingencia forzada
  │     Si FORCE_CONTINGENCY=true y el ambiente es "00" (pruebas):
  │     → HTTPResponseError 503 simulado, sin llamar a Hacienda
  │
  ├── [1] Identificar tipo de documento
  │     getProcessor(document) → InvalidationProcessor o DTEProcessor
  │
  ├── [2] processor.ProcessRequest(signedDoc, document)
  │     → HaciendaRequest con la URL apropiada
  │
  ├── [3] SendToHacienda(ctx, request, systemToken)
  │     └── Error → handleFailedSequence (solo si es HaciendaResponseError) y se devuelve el error
  │
  └── [4] processor.ProcessResponse(response)
        → TransmitResult { Status, ReceptionStamp, ProcessingDate, MessageCode,
                           MessageDesc, Observations }
        + HaciendaResponse = respuesta completa serializada
```

### `SendToHacienda` — Comunicación HTTP

```
SendToHacienda(ctx, request, systemToken)
  │
  ├── [1] haciendaAuth.GetOrCreateHaciendaToken(ctx, systemToken)
  │
  ├── [2] POST {request.URL} (ReceptionURL si viene vacía)
  │     Headers:
  │       Authorization: {haciendaToken}
  │       Content-Type: application/json
  │       User-Agent: HaciendaApp/1.0
  │     Timeout del cliente: 8 s
  │
  └── [3] Evaluar respuesta
        ├── Error de red / timeout          → se devuelve el error tal cual
        ├── Cuerpo vacío                    → HTTPResponseError
        ├── JSON inválido                   → HTTPResponseError (con el status HTTP recibido)
        ├── Campo "estado" vacío            → HTTPResponseError
        ├── estado == "RECHAZADO"           → HaciendaResponseError (cualquier status HTTP)
        ├── Status HTTP fuera de 2xx        → HTTPResponseError
        └── En otro caso                    → HaciendaResponse (éxito)
```

El token se envía sin prefijo `Bearer`: se usa el valor devuelto por la autenticación de Hacienda tal cual.

### `CheckDocumentStatus` — Consulta de Estado

```
CheckDocumentStatus(ctx, document, nit)
  │
  ├── Extrae tipoDte y codigoGeneracion del documento
  ├── POST {ReceptionConsultURL}  { nitEmisor, tdte, codigoGeneracion }
  │     Authorization: token de Hacienda del transmisor (HaciendaToken)
  ├── Acepta HTTP 200 y 202; otro status devuelve error
  └── Retorna TransmitResult con estado, sello, fecha, mensaje, observaciones y HaciendaResponse
```

### Registro de Fallos

Cuando Hacienda rechaza un DTE (`HaciendaResponseError`), `MHTransmitter` registra el correlativo fallido mediante `FailedSequenceNumberRepository` (tabla `failed_sequence_numbers`):

```go
failedSequenceRepo.RegisterFailedSequence(
    ctx, branchID, dteType,
    sequenceNumber, year,
    failureReason,       // Descripción del rechazo
    responseCode,        // Código de Hacienda
    originalRequestData, // Documento original
    mhResponse,          // JSON con estado, código, descripción, clasificación, observaciones y fecha
)
```

Los errores de transporte o HTTP (`HTTPResponseError`, timeouts) no se registran: son candidatos a contingencia.

---

## BatchTransmitterService

> **Archivo:** `adapters/transmitter/batch/batch_transmitter_service.go`
> **Implementa:** `batchPorts.BatchTransmitterPort`

### Estructura

```go
type BatchTransmitterService struct {
    haciendaAuth      authPorts.HaciendaAuthManager
    signer            authPorts.SignerManager
    contingencyRepo   contingency.ContingencyRepositoryPort
    sequentialManager dte_documents.SequentialNumberManager
    timeProvider      ports.TimeProvider
    config            *models.TransmissionConfig
    httpClient        *http.Client             // timeout: 30s, connection pooling
    circuitBreaker    *circuit.CircuitBreaker  // threshold: 3, reset: 5min
    connection        *drivers.DbConnection
    bus               event.Bus
}
```

### Parámetros de Reintento

`TransmissionConfig` se construye en `bootstrap/containers/services.go` con `NewTransmissionConfig(5s, 2min, 2.0)`: intervalo inicial 5 s, máximo 2 minutos, factor 2.0 y `MaxAttempts = 3`. El tamaño del lote es `MH_MAX_BATCH_SIZE` (validado entre 1 y 100).

La espera entre intentos es `InitialInterval * (attempt * BackoffFactor)` con tope `MaxInterval`: nula en el primer reintento inmediato (`attempt = 0`), 10 s tras el segundo intento y 20 s tras el tercero.

### `TransmitBatch` — Flujo

```
TransmitBatch(ctx, systemNIT, dteType, signedDocs, token, creds)
  │
  ├── [1] Construir BatchRequest
  │     {
  │       ambiente:   config.Ambient,
  │       idEnvio:    UUID en mayúsculas,
  │       version:    GetDTEVersion(dteType),
  │       nitEmisor:  systemNIT,
  │       documentos: signedDocs   (JWS)
  │     }
  │
  ├── [2] getHaciendaTokenWithRetry (hasta 3 intentos)
  │     haciendaAuth.GetOrCreateHaciendaTokenWithCreds(ctx, token, creds)
  │
  └── [3] sendBatchWithRetry (hasta 3 intentos)
        transmitToHacienda:
          ├── circuitBreaker.AllowRequest()
          │     └── Bloqueado → error "service temporarily unavailable due to consecutive failures"
          ├── POST {LoteReceptionURL}   (Authorization: token, Content-Type: application/json)
          ├── Error de red / status != 200 / JSON inválido → circuitBreaker.RecordFailure()
          └── Éxito → circuitBreaker.RecordSuccess()  → BatchResponse
```

`shouldRetry` no reintenta si el circuito está abierto, ni ante `HTTPResponseError` no reintentables (400, 401, 403, 404...), ni ante errores de Hacienda cuya descripción contenga "validaci" o "autoriza". Reintenta ante errores de red, 5xx, 408, 429, errores de contexto y errores sin clasificar.

### `VerifyContingencyBatchStatus` — Polling

```
VerifyContingencyBatchStatus(ctx, batchID, mhBatchID, token, branchID, docsMap)
  │
  ├── Intervalo: cada 10 s (ticker)
  ├── Deadline: 2 minutos
  │
  └── Loop:
        ├── GET {LoteReceptionConsultURL}/{codigoLote}
        ├── Cuerpo vacío → aún en proceso; si supera el deadline, error "batch processing timeout"
        └── Con respuesta {procesados, rechazados}:
              ├── Procesados:
              │     contingencyRepo.UpdateBatch(status = RECEIVED) con sello y fecha de
              │     procesamiento por documento; ConfirmReservationByDocumentID por cada uno
              └── Rechazados:
                    contingencyRepo.UpdateBatch(status = REJECTED)
                    publica EmissionFailureEvent (uno por documento) y un
                    RetransmissionJobFailedEvent con el resumen
                    ReleaseReservationByDocumentID por cada uno
```

Para los rechazados, el código de error usado es `clasificaMsg` (o `codigoMsg` si el primero está vacío).

### Versionado de DTEs en lotes

```go
func GetDTEVersion(dteType string) int {
    switch dteType {
    case "01", "07", "14":        return 2
    case "11":                    return 3
    case "03", "04", "05", "06":  return 4
    default:                      return 2
    }
}
```

---

## HaciendaAuthService

> **Archivo:** `adapters/signing/hacienda_auth_service.go`
> **Implementa:** `ports.HaciendaAuthManager`

### Estructura

```go
type HaciendaAuthService struct {
    client      *http.Client   // timeout: 8s
    cache       ports.CacheManager
    authService auth.AuthManager
}
```

### Métodos

| Método | Descripción |
|---|---|
| `GetOrCreateHaciendaToken(ctx, systemToken)` | Se obtiene un token cacheado o se obtienen las credenciales del usuario (por NIT de los claims) y se crea uno nuevo |
| `GetOrCreateHaciendaTokenWithCreds(ctx, systemToken, creds)` | Igual, pero con las credenciales ya resueltas (usado por contingencia y lotes) |

### Flujo de Autenticación

```
GetOrCreateHaciendaToken(ctx, systemToken)
  │
  ├── [1] Buscar en cache
  │     cache.Get("hacienda:token:{systemToken}")
  │     ├── Existe → retornar token cacheado
  │     └── No → continuar
  │
  ├── [2] Obtener credenciales de Hacienda
  │     authService.GetHaciendaCredentials(ctx, claims.NIT, systemToken)
  │
  ├── [3] Autenticar con Hacienda
  │     POST {AuthURL}
  │     Content-Type: application/x-www-form-urlencoded
  │     Body: user={usuario}&pwd={clave}
  │
  ├── [4] Parsear respuesta
  │     { "status": "OK", "body": { "token": "...", "tokenType": "Bearer" } }
  │     Status HTTP != 200 o status != "OK" → error de servicio
  │
  └── [5] Cachear token
        cache.Set("hacienda:token:{systemToken}", token, 24h)
```

---

## DTESigner

> **Archivo:** `adapters/signing/signer/dte_signer.go`
> **Implementa:** `ports.SignerManager`

### Estructura

```go
type DTESigner struct {
    clientRepo auth.AuthRepositoryPort
    client     *http.Client  // timeout: 2s
}
```

### Método Principal

```go
func (s *DTESigner) SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
```

### Flujo de Firma

```
SignDTE(ctx, dte, nit)
  │
  ├── [1] clientRepo.GetByNIT(nit) → password de la llave privada (password_pri)
  │
  ├── [2] Construir request de firma
  │     { "nit", "activo": true, "passwordPri", "dteJson": <documento> }
  │
  ├── [3] POST {config.Signer.Path}  (Content-Type: application/json, timeout 2 s)
  │
  └── [4] Respuesta
        ├── 200 → { "status", "body": "<JWS>" }  → se retorna body (JWS)
        └── Otro status → se interpreta el error en este orden:
              1. Formato Spring Boot (timestamp, status, error, message, path)
              2. Formato del firmador ({ status, body: { codigo, mensaje[] } })
              3. Cuerpo crudo con el status HTTP
```

El JWS devuelto es el valor que se transmite a Hacienda y se guarda en `signed_document`.

---

## Modelos de Request/Response

### HaciendaRequest (individual)

```go
type HaciendaRequest struct {
    Ambient        string      `json:"ambiente"`
    SendID         int         `json:"idEnvio"`
    Version        int         `json:"version"`
    Document       interface{} `json:"documento"`
    DTEType        string      `json:"tipoDte,omitempty"`
    GenerationCode string      `json:"codigoGeneracion,omitempty"`
    URL            string      `json:"-"`
}
```

### BatchRequest

```go
type BatchRequest struct {
    Version   int      `json:"version"`
    Ambient   string   `json:"ambiente"`
    SendID    string   `json:"idEnvio"`
    NIT       string   `json:"nitEmisor"`
    Documents []string `json:"documentos"`
}
```

### HaciendaResponse

```go
type HaciendaResponse struct {
    Version            int      `json:"version"`
    Ambient            string   `json:"ambiente"`
    VersionApp         int      `json:"versionApp"`
    Status             string   `json:"estado"`
    GenerationCode     string   `json:"codigoGeneracion"`
    ReceptionStamp     string   `json:"selloRecibido"`
    ProcessingDate     string   `json:"fhProcesamiento"`
    ClassifyMessage    string   `json:"clasificaMsg"`
    MessageCode        string   `json:"codigoMsg"`
    DescriptionMessage string   `json:"descripcionMsg"`
    Observations       []string `json:"observaciones,omitempty"`
}
```

### HaciendaResponseError y HTTPResponseError

```go
type HaciendaResponseError struct {      // rechazo con respuesta de Hacienda
    Version, VersionApp int
    Ambient        string
    Status         string   // "RECHAZADO"
    StatusCode     int      // status HTTP
    Code           string   // codigoMsg
    Description    string   // descripcionMsg
    Classification string   // clasificaMsg
    Observations   []string
    ProcessedAt    string   // fhProcesamiento
}

type HTTPResponseError struct {          // respuesta HTTP no utilizable
    StatusCode int
    Body       []byte
    URL        string
    Method     string
}
```

La API responde ambos como `HACIENDA_{codigo}` y `HACIENDA_{status}` respectivamente (ver [api-server.md](api-server.md)).

---

## Endpoints de Hacienda

| Endpoint | Variable de entorno | Uso |
|---|---|---|
| Recepción DTE | `MH_RECEPTION_URL` (`MHPaths.ReceptionURL`) | Transmisión de DTEs individuales |
| Invalidación | `MH_NULLIFY_URL` (`MHPaths.NullifyURL`) | Invalidación de DTEs |
| Consulta | `MH_RECEPTION_CONSULT_URL` (`MHPaths.ReceptionConsultURL`) | Consulta de estado por documento |
| Lote | `MH_LOTE_RECEPTION_URL` (`MHPaths.LoteReceptionURL`) | Transmisión por lotes (contingencia) |
| Consulta de lote | `MH_RECEPTION_CONSULT_LOTE_URL` (`MHPaths.LoteReceptionConsultURL`) | Estado del lote |
| Contingencia | `MH_CONTINGENCY_URL` (`MHPaths.ContingencyURL`) | Envío del evento de contingencia |
| Autenticación | `MH_AUTH_URL` (`MHPaths.AuthURL`) | Login con Hacienda |
| Firma | `config.Signer.Path` (variable del firmador) | Servicio de firma digital |

---

## Timeouts

| Cliente | Timeout |
|---|---|
| `MHTransmitter` (recepción y consulta individual) | 8 s |
| `HaciendaAuthService` | 8 s |
| `ContingencyHTTPService` (evento de contingencia) | 8 s |
| `BatchTransmitterService` (envío y consulta de lote) | 30 s |
| `DTESigner` | 2 s |
| Polling de lote (`VerifyContingencyBatchStatus`) | ticker 10 s, deadline 2 min |

---

## Notas

1. **Timeouts diferenciados**: El firmador usa 2 s (servicio local), las transmisiones individuales y la autenticación 8 s, y los lotes 30 s.
2. **Connection pooling**: El `BatchTransmitterService` y `ContingencyHTTPService` configuran pooling HTTP (`MaxIdleConns` 100, `IdleConnTimeout` 90 s).
3. **Circuit breaker**: Solo el `BatchTransmitterService` integra circuit breaker (threshold=3, reset=5min). El `MHTransmitter` no reintenta por sí mismo: la consulta de estado y los reintentos tras un fallo los realiza `BaseTransmitter` en la capa de aplicación, y si no se logra la recepción el documento puede pasar a contingencia.
4. **Registro de fallos**: Cada rechazo de Hacienda de un DTE se persiste en `failed_sequence_numbers` con el documento y la respuesta originales.
5. **Contingencia forzada**: `FORCE_CONTINGENCY=true` solo tiene efecto con `MH_AMBIENT_CODE=00`; simula un 503 sin invocar a Hacienda.
6. **Evidencia**: Cada transmisión exitosa conserva el JWS y la respuesta completa de Hacienda para armar el Archivo DTE (`GET /api/v1/dte/{id}/archivo`).
