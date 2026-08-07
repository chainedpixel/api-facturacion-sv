# Transmisores — Comunicación con Hacienda

> **Paquetes:**
> - `internal/infrastructure/adapters/transmitter` — Transmisión individual de DTEs
> - `internal/infrastructure/adapters/transmitter/batch` — Transmisión por lotes
> - `internal/infrastructure/adapters/signing` — Autenticación y firma con Hacienda

## Descripción General

El sistema de transmisión de infraestructura gestiona la comunicación HTTP real con la API del Ministerio de Hacienda (MH). Se compone de cuatro componentes:

1. **MHTransmitter** — Envío individual de DTEs
2. **BatchTransmitterService** — Envío de lotes de contingencia
3. **HaciendaAuthService** — Autenticación OAuth con Hacienda
4. **DTESigner** — Firma digital de documentos

---

## MHTransmitter

> **Archivo:** `adapters/transmitter/mh_transmitter.go`
> **Implementa:** `ports.DTETransmitter`

### Estructura

```go
type MHTransmitter struct {
    HaciendaToken     string
    haciendaAuth      ports.HaciendaAuthManager
    failedSequenceRepo ports.FailedSequenceNumberRepositoryPort
    httpClient        *http.Client  // timeout: 30s
    processors        map[string]DocumentProcessor
}
```

### Procesadores de Documento

Se registran dos procesadores que determinan cómo construir el request según el tipo de documento:

| Procesador | Detección | Endpoint |
|---|---|---|
| `DTEProcessor` | Sin campo `"documento"` | `ReceptionURL` |
| `InvalidationProcessor` | Con campo `"documento"` | `NullifyURL` |

#### DTEProcessor

> **Archivo:** `adapters/transmitter/processors/dte_processor.go`

Construye el `HaciendaRequest` para la transmisión normal de DTEs apuntando al endpoint de recepción (`config.MHPaths.ReceptionURL`).

```
ProcessRequest(signedDoc, document)
  │
  ├── GetDocumentRequestData(document)
  │   → version, dteType, generationCode, sequenceNumber
  │
  └── HaciendaRequest{
        Ambient:        config.Server.AmbientCode,
        SendID:         sequenceNumber,
        Version:        version,
        Document:       signedDoc,
        DTEType:        dteType,
        GenerationCode: generationCode,
        URL:            ReceptionURL,
      }
```

#### InvalidationProcessor

> **Archivo:** `adapters/transmitter/processors/invalidation_processor.go`

Idéntico a `DTEProcessor` en estructura pero apunta al endpoint de invalidación (`config.MHPaths.NullifyURL`). Se activa cuando el documento serializado contiene el campo `"documento"`, que es exclusivo del JSON de anulación de Hacienda.

```
ProcessRequest(signedDoc, document)
  │
  └── HaciendaRequest{ ..., URL: NullifyURL }
```

### `Transmit` — Flujo

```
Transmit(ctx, document, signedDoc, systemToken)
  │
  ├── [1] Identificar tipo de documento
  │     getProcessor(document)
  │     ├── ¿Tiene campo "documento"? → InvalidationProcessor
  │     └── ¿No tiene? → DTEProcessor
  │
  ├── [2] Procesar request
  │     processor.ProcessRequest(document, signedDoc)
  │     → Extraer: version, dteType, generationCode, sequenceNumber
  │     → Construir HaciendaRequest con URL apropiada
  │
  ├── [3] Enviar a Hacienda
  │     SendToHacienda(ctx, haciendaRequest, systemToken)
  │
  └── [4] Procesar respuesta
        processor.ProcessResponse(haciendaResponse)
        → TransmitResult { Status, ReceptionStamp, ... }
```

### `SendToHacienda` — Comunicación HTTP

```
SendToHacienda(ctx, request, systemToken)
  │
  ├── [1] Obtener token de Hacienda
  │     haciendaAuth.GetOrCreateHaciendaToken(ctx, systemToken)
  │
  ├── [2] Serializar request a JSON
  │     json.Marshal(request)
  │
  ├── [3] Enviar HTTP POST
  │     POST {request.URL}
  │     Headers:
  │       Authorization: Bearer {haciendaToken}
  │       Content-Type: application/json
  │       User-Agent: HaciendaApp/1.0
  │
  └── [4] Parsear respuesta
        ├── Status "PROCESADO" → HaciendaResponse (éxito)
        └── Status "RECHAZADO" → HaciendaResponseError (rechazo)
              → RegisterFailedSequence() si hay número secuencial
```

### `CheckDocumentStatus` — Consulta de Estado

```
CheckDocumentStatus(ctx, document, nit)
  │
  ├── Construir URL de consulta (ReceptionConsultURL)
  ├── HTTP POST con código de generación y NIT
  └── Retornar TransmitResult con estado actual
```

### Registro de Fallos

Cuando un DTE es rechazado por Hacienda, el `MHTransmitter` registra el fallo a través del `FailedSequenceNumberRepository`:

```go
failedSequenceRepo.RegisterFailedSequence(
    ctx, branchID, dteType,
    sequenceNumber, year,
    failureReason,     // Descripción del rechazo
    responseCode,      // Código de Hacienda
    originalRequestData, // JSON del request original
    mhResponse,        // JSON de la respuesta de Hacienda
)
```

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
    config            *models.TransmissionConfig
    httpClient        *http.Client      // timeout: 30s, connection pooling
    circuitBreaker    *circuit.CircuitBreaker  // threshold: 3, reset: 5min
    connection        *drivers.DbConnection
}
```

### `TransmitBatch` — Flujo

```
TransmitBatch(ctx, systemNIT, dteType, signedDocs, token, creds)
  │
  ├── [1] Verificar circuit breaker
  │     circuitBreaker.AllowRequest()
  │     ├── Allowed → continuar
  │     └── Blocked → return error (servicio no disponible)
  │
  ├── [2] Obtener token de Hacienda
  │     haciendaAuth.GetOrCreateHaciendaTokenWithCreds(ctx, token, creds)
  │
  ├── [3] Construir BatchRequest
  │     {
  │       NIT:       systemNIT,
  │       DTEType:   dteType,
  │       Version:   GetDTEVersion(dteType),
  │       Documents: signedDocs
  │     }
  │
  ├── [4] Enviar HTTP POST al endpoint de lotes
  │     POST {batchURL}
  │     Authorization: Bearer {haciendaToken}
  │
  ├── [5] Parsear respuesta
  │     ├── Éxito → circuitBreaker.RecordSuccess()
  │     │          → VerifyContingencyBatchStatus()
  │     └── Error → circuitBreaker.RecordFailure()
  │                → return error
  │
  └── [6] Verificar estado del lote
        VerifyContingencyBatchStatus(ctx, batchID, mhBatchID, ...)
```

### `VerifyContingencyBatchStatus` — Polling

```
VerifyContingencyBatchStatus(ctx, batchID, mhBatchID, ...)
  │
  ├── Deadline: 2 minutos
  │
  └── Loop:
        ├── HTTP GET estado del lote
        ├── ¿Procesado? → Mapear resultados
        │     ├── Documentos procesados → stamps de recepción
        │     ├── Documentos rechazados → motivos de rechazo
        │     └── Actualizar contingencyRepo.UpdateBatch()
        ├── ¿Pendiente? → Esperar intervalo configurado
        └── ¿Timeout? → return error
```

### Versionado de DTEs

```go
func GetDTEVersion(dteType string) int {
    switch dteType {
    case "01":  return 1  // Factura: versión JSON 1
    default:    return 2  // Todos los demás: versión JSON 2
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
    client      *http.Client
    cache       ports.CacheManager
    authService auth.AuthManager
}
```

### Métodos

| Método | Descripción |
|---|---|
| `GetOrCreateHaciendaToken(ctx, systemToken)` | Se obtiene un token cacheado o se crea uno nuevo |
| `GetOrCreateHaciendaTokenWithCreds(ctx, systemToken, creds)` | Se crea un token con credenciales proporcionadas |

### Flujo de Autenticación

```
GetOrCreateHaciendaToken(ctx, systemToken)
  │
  ├── [1] Buscar en cache
  │     cache.GetCredentials(systemToken) → haciendaCreds
  │     ├── Si existe creds.Token → retornar token cacheado
  │     └── Si no → continuar
  │
  ├── [2] Obtener credenciales del usuario
  │     authService.GetCredentials(systemToken) → (username, password)
  │
  ├── [3] Autenticar con Hacienda
  │     POST {authURL}
  │     Content-Type: application/x-www-form-urlencoded
  │     Body: user={username}&pwd={password}
  │
  ├── [4] Parsear respuesta
  │     {
  │       "status": "OK",
  │       "body": {
  │         "token": "eyJ...",
  │         "token_type": "Bearer"
  │       }
  │     }
  │
  └── [5] Cachear token
        cache.SetCredentials(systemToken, creds, 24h)
        → Token de Hacienda cifrado en Redis con TTL de 24h
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
func (s *DTESigner) SignDTE(ctx, dte json.RawMessage, nit string) (string, error)
```

### Flujo de Firma

```
SignDTE(ctx, dte, nit)
  │
  ├── [1] Obtener credenciales de firma
  │     clientRepo.GetByNIT(nit) → user
  │     → Extraer password de firma digital
  │
  ├── [2] Construir request de firma
  │     {
  │       "nit":      nit,
  │       "activo":   true,
  │       "passwordPri": privateKeyPassword,
  │       "dteJson":  dte  // DTE completo como JSON
  │     }
  │
  ├── [3] Enviar al servicio de firma
  │     POST {signerURL}
  │     Content-Type: application/json
  │     Timeout: 2 segundos
  │
  └── [4] Parsear respuesta
        ├── Éxito → retornar documento firmado (string)
        └── Error → parsear error de Spring Boot o Hacienda
```

### Manejo de Errores

El firmador maneja dos formatos de error:

1. **Errores Spring Boot** — El servicio de firma corre sobre Java/Spring
2. **Errores personalizados de Hacienda** — Formato específico de la API

---

## Modelos de Request/Response

### HaciendaRequest

```go
type HaciendaRequest struct {
    URL            string           // Endpoint destino
    Version        int              // Versión del formato (1 o 2)
    Ambiente       string           // "00" (pruebas) o "01" (producción)
    DTEType        string           // Tipo de DTE (01, 03, etc.)
    GenerationCode string           // UUID del documento
    Document       string           // Documento firmado
    SequenceNumber int              // Número de control
}
```

### HaciendaResponse

```go
type HaciendaResponse struct {
    Status         string    // "PROCESADO"
    ReceptionStamp *string   // Sello de recepción (40 chars)
    Code           string    // Código de resultado
    Description    string    // Descripción
    Observations   []string  // Observaciones adicionales
}
```

### HaciendaResponseError

```go
type HaciendaResponseError struct {
    Status         string   // "RECHAZADO"
    Code           string   // Código de error
    Description    string   // Mensaje descriptivo
    Classification string   // Clasificación del error
    Observations   []string // Detalles
    ProcessedAt    string   // Timestamp
    StatusCode     int      // HTTP status code
}
```

---

## Endpoints de Hacienda

| Endpoint | Variable de Config | Uso |
|---|---|---|
| Recepción DTE | `config.MHPaths.ReceptionURL` | Transmisión de DTEs individuales |
| Invalidación | `config.MHPaths.NullifyURL` | Invalidación de DTEs |
| Consulta | `config.MHPaths.ReceptionConsultURL` | Consulta de estado |
| Autenticación | `config.MHPaths.AuthURL` | Login OAuth con Hacienda |
| Firma | `config.SignerURL` | Servicio de firma digital |
| Lotes | `config.MHPaths.BatchURL` | Transmisión por lotes (contingencia) |

---

## Notas

1. **Timeout diferenciados**: El firmador usa 2s (servicio local rápido), mientras que los transmisores usan 30s (API externa con latencia variable).
2. **Connection pooling**: El `BatchTransmitterService` configura pooling HTTP para optimizar transmisiones de lotes.
3. **Circuit breaker**: Solo el `BatchTransmitterService` integra circuit breaker (threshold=3, reset=5min). El `MHTransmitter` depende del retry logic del `BaseTransmitter` de la capa de aplicación.
4. **Registro de fallos**: Cada rechazo se persiste con el request y response original para auditoría y debugging.
5. **Contingencia forzada**: El `MHTransmitter` soporta un modo de contingencia forzada para testing del flujo de contingencia.
