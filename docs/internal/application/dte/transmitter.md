# Sistema de Transmisión — BaseTransmitter y DTETransmitter

> **Paquete aplicación:** `internal/application/dte`
> **Paquete infraestructura:** `internal/infrastructure/adapters/transmitter`

## Descripción General

El sistema de transmisión gestiona el envío de documentos tributarios electrónicos a la API del Ministerio de Hacienda (MH). Se compone de dos niveles:

1. **BaseTransmitter** (aplicación) — Lógica de reintentos y verificación de estado
2. **DTETransmitter/MHTransmitter** (infraestructura) — Comunicación HTTP con Hacienda

---

## BaseTransmitter (Capa de Aplicación)

### Interfaz

```go
type BaseTransmitter interface {
    RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*TransmitResult, error)
    CheckStatus(ctx context.Context, document interface{}, nit string) (*TransmitResult, error)
}
```

### Implementación

```go
type BaseTransmitter struct {
    transmitter ports.DTETransmitter
    signer      ports.SignerManager
}
```

### Constantes

```go
const (
    MaxRetries     = 2          // Máximo de reintentos
    MaxTimeout     = 8          // Segundos entre reintentos
    ReceivedStatus = "PROCESADO" // Estado exitoso de Hacienda
)
```

### `RetryTransmission` — Flujo Completo

```
Documento DTE (formato Hacienda)
  │
  ▼
[1] Serializar a JSON
  │   json.Marshal(document) → jsonData
  │
  ▼
[2] Firmar documento
  │   signer.SignDTE(ctx, jsonData, nit) → signedDoc
  │
  ▼
[3] Primer intento de transmisión
  │   transmitter.Transmit(ctx, document, signedDoc, token)
  │   │
  │   ├── Status == "PROCESADO" → return (éxito)
  │   └── Error → continuar
  │
  ▼
[4] Verificar si ya fue procesado
  │   CheckStatus(ctx, document, nit)
  │   │
  │   ├── Status == "PROCESADO" → return (ya recibido)
  │   └── No procesado → continuar con reintentos
  │
  ▼
[5] Loop de reintentos (max 2)
      │
      ├── Intento 1:
      │     transmitter.Transmit(ctx, document, signedDoc, token)
      │     ├── Status == "PROCESADO" → return
      │     └── Error → sleep 8 segundos
      │
      ├── Intento 2:
      │     transmitter.Transmit(ctx, document, signedDoc, token)
      │     ├── Status == "PROCESADO" → return
      │     └── Error → return último error
      │
      ▼
      Return (result, error) ← Todos los intentos fallaron
```

---

## SignerManager

```go
type SignerManager interface {
    SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
```

Se firma digitalmente el documento DTE antes de transmitirlo a Hacienda. La firma se genera usando el NIT del emisor como referencia para las credenciales de firma.

---

## DTETransmitter (Capa de Infraestructura)

### Interfaz

```go
type DTETransmitter interface {
    Transmit(ctx context.Context, document interface{}, signedDoc string, token string) (*TransmitResult, error)
    CheckDocumentStatus(ctx context.Context, document interface{}, nit string) (*TransmitResult, error)
    SendToHacienda(ctx context.Context, request *HaciendaRequest, token string) (*HaciendaResponse, error)
}
```

### MHTransmitter — Implementación

El `MHTransmitter` implementa `DTETransmitter` con la comunicación real a la API de Hacienda.

### `Transmit` — Flujo Interno

```
[1] Identificar tipo de documento
  │   getProcessor(document)
  │   ├── Si tiene campo "documento" → InvalidationProcessor
  │   └── Si no → DTEProcessor
  │
  ▼
[2] Procesar request
  │   processor.ProcessRequest(document, signedDoc)
  │   → Extraer: version, dteType, generationCode, sequenceNumber
  │   → Construir HaciendaRequest con URL apropiada
  │
  ▼
[3] Enviar a Hacienda
  │   SendToHacienda(ctx, haciendaRequest, token)
  │
  ▼
[4] Procesar response
      processor.ProcessResponse(haciendaResponse)
      → TransmitResult { Status, ReceptionStamp, ... }
```

### `SendToHacienda` — Comunicación HTTP

```
[1] Obtener token de Hacienda
  │   getHaciendaToken(ctx, systemToken)
  │   → HaciendaAuthManager.GetOrCreateHaciendaToken()
  │
  ▼
[2] Serializar request
  │   json.Marshal(request)
  │
  ▼
[3] Enviar HTTP POST
  │   POST {request.URL}
  │   Headers:
  │     Authorization: Bearer {haciendaToken}
  │     Content-Type: application/json
  │     User-Agent: HaciendaApp/1.0
  │
  ▼
[4] Parsear respuesta
      ├── Status "PROCESADO" → return HaciendaResponse (éxito)
      └── Status "RECHAZADO" → return HaciendaResponseError (rechazo)
```

### Endpoints de Hacienda

| Endpoint | Uso | Configuración |
|---|---|---|
| `ReceptionURL` | Transmisión de DTEs | `config.MHPaths.ReceptionURL` |
| `NullifyURL` | Invalidación de DTEs | `config.MHPaths.NullifyURL` |
| `ReceptionConsultURL` | Consulta de estado | `config.MHPaths.ReceptionConsultURL` |

### Procesadores de Documentos

#### DTEProcessor

Se procesan DTEs estándar (factura, CCF, notas, retención, etc.):
- Se extrae de `identificacion`: version, tipoDte, codigoGeneracion
- Se usa `ReceptionURL` como endpoint

#### InvalidationProcessor

Se procesan invalidaciones:
- Se extrae de `documento.tipoDte`
- Se usa `NullifyURL` como endpoint

---

## TransmitResult

```go
type TransmitResult struct {
    Status         string   // "PROCESADO" o "RECHAZADO"
    ReceptionStamp *string  // Sello de recepción (40 chars alfanuméricos)
    Code           string   // Código de resultado
    Description    string   // Descripción
    Observations   []string // Observaciones adicionales
}
```

---

## HaciendaResponseError

```go
type HaciendaResponseError struct {
    Status         string     // "RECHAZADO"
    Code           string     // Código de error de Hacienda
    Description    string     // Mensaje descriptivo
    Classification string     // Clasificación del error
    Observations   []string   // Detalles adicionales
    ProcessedAt    string     // Timestamp del procesamiento
    StatusCode     int        // HTTP status code
}
```

---

## Circuit Breaker

Se integra un circuit breaker para proteger contra llamadas excesivas a Hacienda cuando el servicio está caído:

```go
type CircuitBreaker struct {
    failures    int32
    lastFailure time.Time
    threshold   int32          // Umbral de fallos
    resetTime   time.Duration  // Tiempo de recuperación
    state       State          // Closed, Open, HalfOpen
}
```

| Estado | Descripción | `AllowRequest()` |
|---|---|---|
| `Closed` | Operación normal | `true` |
| `Open` | Servicio no disponible | `false` (hasta `resetTime`) |
| `HalfOpen` | Probando recuperación | `true` (una vez) |

**Transiciones:**
```
Closed ──(failures >= threshold)──→ Open
Open   ──(resetTime expired)──→ HalfOpen
HalfOpen ──(success)──→ Closed
HalfOpen ──(failure)──→ Open
```

---

## Diagrama Completo de Transmisión

```
GenericDTEUseCase.Create()
  │
  ▼
BaseTransmitter.RetryTransmission()
  │
  ├── SignerManager.SignDTE() ← Firma digital
  │
  ├── MHTransmitter.Transmit() ← Primer intento
  │     │
  │     ├── getProcessor() ← DTE o Invalidación
  │     ├── ProcessRequest() ← Construir HaciendaRequest
  │     ├── SendToHacienda()
  │     │     ├── getHaciendaToken() ← OAuth con Hacienda
  │     │     ├── HTTP POST ← Envío real
  │     │     └── Parse response
  │     └── ProcessResponse() ← TransmitResult
  │
  ├── [Si error] CheckStatus() ← Verificar si ya procesado
  │
  └── [Si error] Retry x2 (sleep 8s entre intentos)
```

---

## Notas

1. **Firma antes de transmisión**: Todo documento se firma digitalmente con `SignDTE` antes de enviarse.
2. **3 intentos totales**: 1 intento inicial + 2 reintentos con 8 segundos de espera entre cada uno.
3. **Verificación pre-reintento**: Antes de reintentar, se verifica si el documento ya fue procesado (evita duplicados).
4. **Circuit breaker**: Protege contra cascadas de fallos. Si Hacienda está caído, el circuit breaker se abre y las solicitudes fallan rápido.
5. **Token de Hacienda**: Se cachea y se reutiliza. Se renueva automáticamente cuando expira.
