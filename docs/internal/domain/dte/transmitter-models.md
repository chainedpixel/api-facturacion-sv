# Modelos de Transmisión — Dominio

> **Paquetes:**
> - `internal/domain/dte/transmitter/models` — Modelos de request/response para Hacienda
> - `internal/domain/dte/transmitter` — Interface `BatchTransmitterPort` y `TimeProvider`

## Descripción General

Este paquete define los modelos de datos que representan la comunicación entre el sistema y la API del Ministerio de Hacienda. Son modelos de dominio puros — sin lógica de negocio — que actúan como contratos de serialización para transmisiones individuales y por lotes.

---

## Modelos de Transmisión Individual

### `HaciendaRequest`

> **Archivo:** `models/hacienda_request_model.go`

Request enviado a Hacienda para la transmisión de un DTE individual.

```go
type HaciendaRequest struct {
    Ambient        string      `json:"ambiente"`
    SendID         int         `json:"idEnvio"`
    Version        int         `json:"version"`
    Document       interface{} `json:"documento"`
    DTEType        string      `json:"tipoDte"`
    GenerationCode string      `json:"codigoGeneracion"`
    URL            string      `json:"-"`
}
```

| Campo | Descripción |
|---|---|
| `Ambient` | Código de ambiente: `"00"` (pruebas) o `"01"` (producción) |
| `SendID` | Número secuencial de envío (número de control del DTE) |
| `Version` | Versión del formato JSON del tipo de DTE |
| `Document` | Documento firmado (string) o estructura JSON del DTE |
| `DTEType` | Código del tipo de DTE (`"01"`, `"03"`, etc.) |
| `GenerationCode` | UUID del documento (código de generación) |
| `URL` | Endpoint destino; excluido del JSON (`json:"-"`) |

---

### `HaciendaResponse`

> **Archivo:** `models/hacienda_response_model.go`

Respuesta de Hacienda ante una transmisión exitosa (`"PROCESADO"`).

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

El campo `ReceptionStamp` (`selloRecibido`) es el sello de recepción de 40 caracteres que acredita que Hacienda procesó el documento. Se almacena en la tabla `dte_details`.

---

### `TransmitResult`

> **Archivo:** `models/transmit_result_model.go`

Resultado normalizado de una transmisión, independiente del formato raw de Hacienda.

```go
type TransmitResult struct {
    Status         string
    ReceptionStamp *string
    ProcessingDate string
    MessageCode    string
    MessageDesc    string
    Observations   []string
}
```

Este struct es el que retornan tanto `MHTransmitter.Transmit` como `BaseTransmitter.RetryTransmission`. La capa de aplicación solo trabaja con `TransmitResult`, nunca con `HaciendaResponse` directamente.

| `Status` | Significado |
|---|---|
| `"PROCESADO"` | DTE aceptado por Hacienda — `ReceptionStamp` contiene el sello |
| `"RECHAZADO"` | DTE rechazado — `MessageCode` y `Observations` contienen el motivo |
| `"CONTINGENCIA"` | DTE almacenado en contingencia — no hubo respuesta de Hacienda |

---

## Modelos de Transmisión por Lotes (Contingencia)

### `BatchRequest`

> **Archivo:** `models/batch_request_model.go`

Request para enviar un lote de DTEs de contingencia.

```go
type BatchRequest struct {
    Version   int      `json:"version"`
    Ambient   string   `json:"ambiente"`
    SendID    string   `json:"idEnvio"`
    NIT       string   `json:"nitEmisor"`
    Documents []string `json:"documentos"`
}
```

`Documents` contiene los strings de los DTEs firmados. El `BatchTransmitterService` los agrupa por tipo de DTE y NIT antes de construir este request.

---

### `BatchResponse`

> **Archivo:** `models/batch_response_model.go`

Respuesta inmediata de Hacienda al recibir el lote. Contiene un `BatchCode` (`codigoLote`) que se usa para consultar el estado de procesamiento posteriormente.

```go
type BatchResponse struct {
    Version         int     `json:"version"`
    Ambient         string  `json:"ambiente"`
    VersionApp      int     `json:"versionApp"`
    Status          string  `json:"estado"`
    SendID          string  `json:"idEnvio"`
    BatchCode       string  `json:"codigoLote"`
    ProcessingDate  string  `json:"fhProcesamiento"`
    ReceptionStamp  *string `json:"selloRecibido"`
    ClassifyMessage string  `json:"clasificaMsg"`
    MessageCode     string  `json:"codigoMsg"`
    Description     string  `json:"descripcionMsg"`
}
```

---

### `ConsultBatchResponse`

Respuesta al consultar el estado de un lote ya enviado. Contiene los documentos procesados y rechazados separadamente.

```go
type ConsultBatchResponse struct {
    Processed []HaciendaResponse `json:"procesados"`
    Rejected  []HaciendaResponse `json:"rechazados"`
}
```

El `BatchTransmitterService` hace polling a Hacienda hasta obtener esta respuesta (máximo 2 minutos). Ver [Transmisores](../../infrastructure/transmitter.md).

---

## Configuración de Transmisión

### `TransmissionConfig`

> **Archivo:** `models/transmission_config.go`

Configuración del servicio de retransmisión por lotes. Define los parámetros del backoff exponencial.

```go
type TransmissionConfig struct {
    Ambient       string
    BatchSize     int
    RetryInterval time.Duration
    MaxInterval   time.Duration
    BackoffFactor float64
}
```

| Campo | Origen | Descripción |
|---|---|---|
| `Ambient` | `config.Server.AmbientCode` | Ambiente actual (`"00"` o `"01"`) |
| `BatchSize` | `config.Server.MaxBatchSize` | Máximo de documentos por lote |
| `RetryInterval` | Parámetro de constructor | Intervalo inicial entre reintentos |
| `MaxInterval` | Parámetro de constructor | Intervalo máximo (techo del backoff) |
| `BackoffFactor` | Parámetro de constructor | Factor de crecimiento exponencial |

El método `GetRetryPolicy()` construye un `models.RetryPolicy` con `MaxAttempts: 3` fijo, usado por el `ContingencyService`.

---

## Interface de Puerto

### `BatchTransmitterPort`

> **Archivo:** `batch_transmitter_interface.go`

Puerto que define la transmisión por lotes. Lo implementa `BatchTransmitterService` en la capa de infraestructura.

```go
type BatchTransmitterPort interface {
    TransmitBatch(ctx, systemNIT, dteType string, documents []string, token string, credentials HaciendaCredentials) (*BatchResponse, string, error)
    VerifyContingencyBatchStatus(ctx, batchID, mhBatchID, token string, branchID uint, docsMap map[string]ContingencyDocument) error
    GetDTEVersion(dteType string) int
}
```

| Método | Descripción |
|---|---|
| `TransmitBatch` | Envía el lote y retorna `BatchResponse` y el `batchID` interno |
| `VerifyContingencyBatchStatus` | Polling del estado hasta confirmación o timeout (2 min) |
| `GetDTEVersion` | Retorna la versión JSON del tipo: `1` para factura (`01`), `2` para el resto |

---

## Relación entre Modelos

```
GenericDTEUseCase
  │
  └── BaseTransmitter.RetryTransmission()
        │
        └── ports.DTETransmitter.Transmit()
              │
              ├── Construye HaciendaRequest
              ├── HTTP POST a Hacienda
              ├── Parsea HaciendaResponse / HaciendaResponseError
              └── Retorna TransmitResult
                    │
                    └── La capa de aplicación evalúa TransmitResult.Status
```

Para contingencia:

```
ContingencyService
  │
  └── BatchTransmitterPort.TransmitBatch()
        │
        ├── Construye BatchRequest (agrupa por NIT + DTE type)
        ├── HTTP POST → BatchResponse (codigoLote)
        └── VerifyContingencyBatchStatus()
              ├── HTTP GET polling
              └── ConsultBatchResponse { Processed, Rejected }
```

---

## Notas

1. **`URL` excluido del JSON**: El campo `URL` en `HaciendaRequest` usa `json:"-"` porque es información de enrutamiento interno, no parte del payload que Hacienda espera recibir.
2. **`TransmitResult` como capa de abstracción**: Normaliza las respuestas de Hacienda para que la capa de aplicación no dependa del formato de la API externa. Si Hacienda cambia su formato de respuesta, solo cambia el parser en infraestructura.
3. **`BatchCode` y polling**: El envío por lotes en Hacienda es asíncrono — la respuesta inmediata solo acusa recibo. El estado real se consulta por polling con el `BatchCode` hasta un máximo de 2 minutos.
