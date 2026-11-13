# Sistema de Transmisión — BaseTransmitter y DTETransmitter

> **Paquete aplicación:** `internal/application/dte` (`base_transmit.go`)
> **Paquete infraestructura:** `internal/infrastructure/adapters/transmitter`

## Descripción General

El sistema de transmisión gestiona la firma y el envío de documentos tributarios electrónicos a la API del Ministerio de Hacienda (MH). Se compone de dos niveles:

1. **BaseTransmitter** (aplicación) — Firma, lógica de reintentos y verificación de estado
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

func NewBaseTransmitter(transmitter ports.DTETransmitter, signer ports.SignerManager) ports.BaseTransmitter
```

`UseCaseContainer.Initialize` lo construye con `ServicesContainer.TransmitterManager()` (`MHTransmitter`) y `ServicesContainer.SignerManager()` (`DTESigner`).

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
  │   signer.SignDTE(ctx, jsonData, nit) → signedDoc (JWS)
  │   └── Error → return (nil, err)
  │
  ▼
[3] Primer intento de transmisión
  │   transmitter.Transmit(ctx, document, signedDoc, token)
  │   │
  │   ├── err == nil y Status == "PROCESADO" → return (resultado + signedDoc)
  │   ├── err != nil → return (result, err)          ← no hay verificación ni reintentos
  │   └── err == nil y Status != "PROCESADO" → continuar
  │
  ▼
[4] Verificar si ya fue procesado
  │   CheckStatus(ctx, document, nit)
  │   │
  │   ├── Status == "PROCESADO" → return (resultado de la consulta + signedDoc)
  │   └── Error o no procesado → continuar con reintentos
  │
  ▼
[5] Loop de reintentos (max 2)
      │
      ├── Intento 1: transmitter.Transmit(...)
      │     ├── err == nil y Status == "PROCESADO" → return (resultado + signedDoc)
      │     └── Otro caso → sleep 8 segundos
      │
      ├── Intento 2: transmitter.Transmit(...)
      │     ├── err == nil y Status == "PROCESADO" → return (resultado + signedDoc)
      │     └── Otro caso → sleep 8 segundos
      │
      ▼
      Return (result, err) ← del último intento
```

### Adjunto del JWS (`withSignedDocument`)

```go
func withSignedDocument(result *models.TransmitResult, signedDoc string) *models.TransmitResult
```

Cuando la transmisión (o la consulta de estado) devuelve `PROCESADO`, el JWS que se firmó y transmitió se asigna a `TransmitResult.SignedDocument`. Los casos de uso lo guardan como `firmaElectronica` del Archivo DTE (junto con el sello y la respuesta de Hacienda). El JWS se firma una sola vez: los reintentos reutilizan el mismo `signedDoc`.

### Comportamiento real de los reintentos

- Un error en el primer intento (red, timeout, rechazo de Hacienda, HTTP no 2xx) se devuelve de inmediato: la verificación de estado y el loop de reintentos solo se ejecutan cuando el primer intento no falla pero Hacienda no responde `PROCESADO`.
- El resultado de `CheckStatus` se ignora si falla; el flujo sigue con los reintentos.
- Dentro del loop, el error de un intento no interrumpe el flujo: se espera `MaxTimeout` segundos y se reintenta. Al terminar se devuelve el resultado y el error del último intento (puede ser `(result, nil)` con un estado distinto de `PROCESADO`).
- `GenericDTEUseCase` no revisa `result.Status` cuando el error es nulo; `InvalidationUseCase` sí lo revisa.

---

## SignerManager

```go
type SignerManager interface {
    SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
```

Se firma digitalmente el documento DTE antes de transmitirlo a Hacienda. La implementación (`DTESigner`) obtiene la contraseña de la llave privada del cliente por NIT (`AuthRepositoryPort.GetByNIT`) y llama al servicio externo de firma (`config.Signer.Path`, timeout HTTP de 2 segundos), que devuelve el JWS.

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

El `MHTransmitter` implementa `DTETransmitter` con la comunicación real a la API de Hacienda. Su cliente HTTP tiene un timeout de 8 segundos y se construye con `HaciendaAuthManager` y el repositorio de secuencias fallidas.

### `Transmit` — Flujo Interno

```
[0] Contingencia forzada
  │   Si FORCE_CONTINGENCY=true y MH_AMBIENT_CODE="00" → HTTPResponseError 503 simulado (no se llama a Hacienda)
  │
  ▼
[1] Identificar tipo de documento
  │   getProcessor(document)
  │   ├── Si es evento de invalidación → InvalidationProcessor
  │   └── Si no → DTEProcessor
  │
  ▼
[2] Procesar request
  │   processor.ProcessRequest(signedDoc, document)
  │   → Extraer: version, tipoDte, codigoGeneracion, correlativo del numeroControl
  │   → Construir HaciendaRequest con el JWS como `documento` y la URL apropiada
  │
  ▼
[3] Enviar a Hacienda
  │   SendToHacienda(ctx, haciendaRequest, token)
  │   └── Si falla con HaciendaResponseError → se registra la secuencia fallida
  │
  ▼
[4] Procesar response
      processor.ProcessResponse(haciendaResponse)
      → TransmitResult { Status, ReceptionStamp, ProcessingDate, ... }
      → TransmitResult.HaciendaResponse = respuesta completa serializada a JSON
```

La detección de invalidación (`isInvalidationDocument`) considera invalidación al documento que tiene `motivo` con `tipoAnulacion` y `nombreResponsable`; si no tiene `motivo`, al que no tiene `resumen` ni `cuerpoDocumento`.

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
  │   POST {request.URL}  (ReceptionURL si viene vacía)
  │   Headers:
  │     Authorization: {token de Hacienda tal como lo devolvió la autenticación}
  │     Content-Type: application/json
  │     User-Agent: HaciendaApp/1.0
  │
  ▼
[4] Interpretar la respuesta
      ├── Cuerpo vacío / JSON inválido / `estado` ausente → HTTPResponseError
      ├── estado "RECHAZADO" → HaciendaResponseError (rechazo)
      ├── HTTP no 2xx → HTTPResponseError
      └── En otro caso → HaciendaResponse
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
- Se extrae de `identificacion`: version, tipoDte, codigoGeneracion y el correlativo de `numeroControl` (usado como `idEnvio`)
- Se usa `ReceptionURL` como endpoint y se envían `tipoDte` y `codigoGeneracion`

#### InvalidationProcessor

Se procesan invalidaciones:
- Se extrae `version` de `identificacion` y el correlativo de `documento.numeroControl` (si no hay, `idEnvio` es 1); `tipoDte` se obtiene de `documento.tipoDte`
- Se usa `NullifyURL` como endpoint; la petición no incluye `tipoDte` ni `codigoGeneracion`

### `CheckDocumentStatus`

Consulta `ReceptionConsultURL` con `{nitEmisor, tdte, codigoGeneracion}`. Acepta HTTP 200 o 202. Devuelve un `TransmitResult` con `Status`, `ReceptionStamp`, `ProcessingDate`, mensajes y la respuesta completa serializada. Usa el token de Hacienda que el `MHTransmitter` guardó en su último `SendToHacienda`.

---

## TransmitResult

```go
type TransmitResult struct {
    Status           string   // "PROCESADO" o "RECHAZADO"
    ReceptionStamp   *string  // selloRecibido
    ProcessingDate   string   // fhProcesamiento
    MessageCode      string   // codigoMsg
    MessageDesc      string   // descripcionMsg
    Observations     []string // observaciones
    SignedDocument   string   // JWS transmitido (firmaElectronica del Archivo DTE)
    HaciendaResponse string   // respuesta completa de Hacienda en JSON
}
```

`SignedDocument` lo asigna `BaseTransmitter` (`withSignedDocument`); `HaciendaResponse` lo asigna `MHTransmitter`.

---

## HaciendaResponseError

```go
type HaciendaResponseError struct {
    Version        int      // version
    Ambient        string   // ambiente
    VersionApp     int      // versionApp
    Status         string   // estado ("RECHAZADO")
    StatusCode     int      // código HTTP (se llena desde la respuesta HTTP)
    Code           string   // codigoMsg
    Description    string   // descripcionMsg
    Classification string   // clasificaMsg
    Observations   []string // observaciones
    ProcessedAt    string   // fhProcesamiento
}
```

Su mensaje es `Hacienda said: [<codigo>] <descripcion>`. Cuando `SendToHacienda` devuelve este error, `MHTransmitter` registra la secuencia como fallida (`FailedSequenceNumberRepositoryPort.RegisterFailedSequence`) con el código, la descripción y la respuesta formateada. Además existen `HTTPResponseError` (código HTTP, cuerpo, URL y método) y `NetworkError`, ambos en `internal/infrastructure/adapters/transmitter/hacienda_error`.

---

## Circuit Breaker

El circuit breaker (`internal/infrastructure/adapters/circuit`) **no** forma parte de `MHTransmitter` ni de `BaseTransmitter`. Lo usa únicamente `BatchTransmitterService` (retransmisión de contingencia por lotes), con umbral de 3 fallos y 5 minutos de recuperación:

```go
type CircuitBreaker struct {
    failures    int32
    lastFailure time.Time
    threshold   int32          // Umbral de fallos
    resetTime   time.Duration  // Tiempo de recuperación
    state       State          // Closed, Open, HalfOpen
    mu          sync.Mutex
}
```

| Estado | Descripción | `AllowRequest()` |
|---|---|---|
| `Closed` | Operación normal | `true` |
| `Open` | Servicio no disponible | `false` (hasta superar `resetTime` desde el último fallo) |
| `HalfOpen` | Probando recuperación | `true` |

**Transiciones:**
```
Closed ──(failures >= threshold)──→ Open
Open   ──(resetTime expired, en AllowRequest)──→ HalfOpen
HalfOpen/Open ──(RecordSuccess)──→ Closed
HalfOpen ──(RecordFailure con failures >= threshold)──→ Open
```

Ver [Circuit Breaker](../../infrastructure/circuit-breaker.md).

---

## Diagrama Completo de Transmisión

```
GenericDTEUseCase.Create()  /  InvalidationUseCase.InvalidateDocument()
  │
  ▼
BaseTransmitter.RetryTransmission()
  │
  ├── SignerManager.SignDTE() ← Firma digital (JWS)
  │
  ├── MHTransmitter.Transmit() ← Primer intento
  │     │
  │     ├── getProcessor() ← DTE o Invalidación
  │     ├── ProcessRequest() ← Construir HaciendaRequest
  │     ├── SendToHacienda()
  │     │     ├── getHaciendaToken() ← token de Hacienda (caché Redis o autenticación)
  │     │     ├── HTTP POST ← Envío real
  │     │     └── Interpretar respuesta
  │     └── ProcessResponse() ← TransmitResult (+ HaciendaResponse)
  │
  ├── [Si respondió sin error pero sin PROCESADO] CheckStatus() ← Verificar si ya procesado
  │
  └── [Si sigue sin PROCESADO] Retry x2 (sleep 8s tras cada intento fallido)
        │
        ▼
      withSignedDocument() ← JWS en TransmitResult.SignedDocument
```

---

## Notas

1. **Firma antes de transmisión**: Todo documento se firma digitalmente con `SignDTE` dentro de `RetryTransmission`; el JWS firmado se conserva como evidencia.
2. **Hasta 3 transmisiones**: 1 intento inicial + 2 reintentos con 8 segundos de espera tras cada intento fallido del loop, pero los reintentos solo se activan si el primer intento no devuelve error (ver "Comportamiento real de los reintentos").
3. **Verificación pre-reintento**: Antes de reintentar se consulta si el documento ya fue procesado (evita duplicados).
4. **Circuit breaker**: Solo protege las llamadas del transmisor por lotes.
5. **Token de Hacienda**: `HaciendaAuthService` lo cachea en Redis por 24 horas (clave `hacienda:token:<JWT del sistema>`). `MHTransmitter` lo guarda además en su campo `HaciendaToken`, que se sobrescribe en cada transmisión.
6. **Contingencia forzada**: Con `FORCE_CONTINGENCY=true` y ambiente de pruebas (`00`) se simula un 503 de Hacienda para probar el flujo de contingencia.
