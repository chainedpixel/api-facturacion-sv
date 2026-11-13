# InvalidationUseCase — Caso de Uso de Invalidación

> **Paquete:** `internal/application/dte`
> **Archivo:** `invalidation_use_case.go`

## Descripción General

El `InvalidationUseCase` gestiona la invalidación de documentos tributarios electrónicos previamente emitidos. A diferencia de los DTEs regulares, la invalidación tiene un flujo propio que **no usa el `GenericDTEUseCase`** porque:

- No genera número de control secuencial
- No soporta contingencia
- Requiere validación de estado y de dependencias del documento original
- Modifica el estado de un documento existente
- Registra un evento (`dte_events`) con su evidencia de transmisión, en lugar de un documento

---

## Estructura

```go
type InvalidationUseCase struct {
    dteManager          dteInterfaces.DTEManager
    authManager         authManager.AuthManager
    invalidationManager invalidation.InvalidationManager
    mapper              *request_mapper.InvalidationMapper
    transmitter         ports.BaseTransmitter
}
```

### Dependencias

| Campo | Tipo | Propósito |
|---|---|---|
| `dteManager` | `DTEManager` | Se accede al documento original y se registra el evento |
| `authManager` | `AuthManager` | Se obtiene información del emisor |
| `invalidationManager` | `InvalidationManager` | Se validan estado, dependencias y reglas, y se ejecuta la invalidación |
| `mapper` | `InvalidationMapper` | Se valida el request y se mapea a modelo de invalidación (se instancia dentro del constructor) |
| `transmitter` | `BaseTransmitter` | Se firma y transmite a Hacienda |

La interfaz `InvalidationManager` expone `Validate`, `ValidateStatus`, `ValidateDependencies` e `InvalidateDocument`.

---

## Método Principal: `InvalidateDocument`

```go
func (u *InvalidationUseCase) InvalidateDocument(
    ctx context.Context,
    request structs.CreateInvalidationRequest,
) (*dte.EventArchiveResponse, error)
```

### Flujo

```
CreateInvalidationRequest
  │
  ▼
[1] Extraer contexto (claims + token)
  │
  ▼
[2] Validar estructura del request
  │   mapper.ValidateInvalidationReRequest(&request)
  │
  ▼
[3] Validar estado del documento original
  │   invalidationManager.ValidateStatus(ctx, branchID, request)
  │   ├── Documento original: INVALIDATED / REJECTED / PENDING → error
  │   └── Documento de reemplazo (si tipo != 2 y viene el código): mismas reglas
  │
  ▼
[4] Cargar documento original y validar dependencias
  │   dteManager.GetByGenerationCode(ctx, branchID, request.GenerationCode)
  │   invalidationManager.ValidateDependencies(ctx, branchID, originalDTE)
  │   └── CCF o CR con NC/ND vigentes → PendingAdjustmentNotes (lista de notas a invalidar primero)
  │
  ▼
[5] Cargar información del emisor
  │   authManager.GetIssuer(ctx, branchID)
  │
  ▼
[6] Mapear a modelo de dominio de invalidación
  │   mapper.MapToInvalidationData(&request, issuer, originalDTE.Details, originalDTE.CreatedAt)
  │   ├── Exige sello de recepción del documento original (DocumentWithoutStamp)
  │   ├── Fecha de generación y del sello: fecEmi/horEmi y processing_date del DTE (CreatedAt como respaldo)
  │   └── Fecha del evento: actual para FE/FEXE/FSEE; la de generación para el resto (policy.EventDateTime)
  │
  ▼
[7] Validar documento de invalidación completo
  │   invalidationManager.Validate(ctx, branchID, invalidationDocument)
  │   (reglas de negocio y plazos de invalidación)
  │
  ▼
[8] Mapear a formato Hacienda
  │   response_mapper.ToMHInvalidation(invalidationDocument)
  │
  ▼
[9] Firmar y transmitir a Hacienda (SIN contingencia)
  │   transmitter.RetryTransmission(ctx, mhInvalidation, token, claims.NIT)
  │   ├── Si error → se devuelve tal cual
  │   ├── Si status != "PROCESADO" → error TransmissionFailed
  │   └── Si éxito → continuar
  │
  ▼
[10] Marcar documento como invalidado
  │   invalidationManager.InvalidateDocument(ctx, branchID, request.GenerationCode)
  │   ├── Actualizar estado → INVALIDATED
  │   └── Recuperar balance (si el documento invalidado es nota de crédito/débito)
  │
  ▼
[11] Registrar el evento y responder
      buildInvalidationEvent(...) → dteManager.RegisterEvent(ctx, event)
      return dte_documents.ToEventArchiveResponse(*event)
```

---

## Input: CreateInvalidationRequest

```go
type CreateInvalidationRequest struct {
    GenerationCode            string         `json:"generation_code"`                       // UUID del documento a invalidar
    Reason                    *ReasonRequest `json:"reason"`                                // Motivo de invalidación
    ReplacementGenerationCode *string        `json:"replacement_generation_code,omitempty"` // UUID del doc de reemplazo (tipos 1 y 3)
}

type ReasonRequest struct {
    Type               int     `json:"type"`                  // 1, 2 o 3
    ResponsibleName    string  `json:"responsible_name"`
    ResponsibleDocType string  `json:"responsible_doc_type"`
    ResponsibleNumDoc  string  `json:"responsible_num_doc"`
    RequestorName      string  `json:"requestor_name"`
    RequestorDocType   string  `json:"requestor_doc_type"`
    RequestorNumDoc    string  `json:"requestor_num_doc"`
    Reason             *string `json:"reason_field,omitempty"` // Obligatorio para tipo 3
}
```

Reglas de `ValidateInvalidationReRequest`:

| Regla | Error |
|---|---|
| `reason` ausente, o cualquier campo obligatorio vacío | `ValidationError` `RequiredField` |
| Tipo 1 o 3 sin `replacement_generation_code` | `InvalidReplacementCode1And3` |
| Tipo 2 con `replacement_generation_code` | `InvalidInvalidationType2` |
| Tipo 3 sin `reason_field` | `InvalidInvalidationType3` |

---

## Output: EventArchiveResponse

```go
type EventArchiveResponse struct {
    DTEEvent                                    // campos del evento
    HaciendaResponse json.RawMessage `json:"hacienda_response,omitempty"`
    Archive          json.RawMessage `json:"archivo_dte,omitempty"`
}
```

Campos del evento (`DTEEvent`, serializados en JSON):

| Campo JSON | Contenido |
|---|---|
| `generation_code` | Código de generación del evento de invalidación |
| `branch_id` | Sucursal |
| `event_type` | `INVALIDATION` |
| `related_dte_id` | Código de generación del documento invalidado |
| `replacement_dte_id` | Código de generación del documento de reemplazo (si aplica) |
| `reception_stamp` | Sello de recepción del evento |
| `processing_date` | `fhProcesamiento` informado por Hacienda |
| `status` | `RECEIVED` |
| `created_at` | Fecha de registro |

`hacienda_response` es la respuesta completa de Hacienda y `archivo_dte` es el Archivo DTE del evento (estructura firmada + `firmaElectronica` + `selloRecibido`), presente cuando el JWS guardado es válido. El JSON del evento y el JWS se guardan en la base de datos pero no se serializan en la respuesta (`json:"-"`).

Normativa de Cumplimiento DTE v2.0 (regla 11): los eventos de invalidación se entregan al receptor como Archivo DTE. Los eventos de un documento también se consultan con `GET /dte/{id}/eventos`.

---

## Formato Hacienda de la Invalidación

Estructura transmitida (`response_mapper.ToMHInvalidation`, versión 3), con los campos que serializa el código:

```json
{
  "identificacion": {
    "version": 3,
    "ambiente": "00",
    "codigoGeneracion": "UUID-NUEVO-EN-MAYUSCULAS",
    "fecEmi": "2024-03-15",
    "horEmi": "10:30:00",
    "fusion": null
  },
  "emisor": {
    "nit": "06142803101234",
    "nombre": "Empresa SA",
    "codEstableMH": "M001",
    "codEstable": "0001",
    "codPuntoVentaMH": "P001",
    "codPuntoVenta": "0001",
    "telefono": "22223333",
    "correo": "empresa@email.com"
  },
  "documento": {
    "tipoDte": "03",
    "codigoGeneracion": "UUID-ORIGINAL",
    "selloRecibido": "SELLO-DE-RECEPCION",
    "numeroControl": "DTE-03-M001P001-000000000000001",
    "fecEmi": "2024-03-10",
    "codigoGeneracionR": "UUID-REEMPLAZO-O-NULL",
    "tipoDocumento": "36",
    "numDocumento": "06141804941035",
    "nombre": "Receptor SA",
    "telefono": "22223333",
    "correo": "receptor@email.com"
  },
  "motivo": {
    "tipoAnulacion": 1,
    "motivoAnulacion": null,
    "nombreResponsable": "Juan Pérez",
    "tipDocResponsable": "13",
    "numDocResponsable": "01234567-8",
    "nombreSolicita": "María López",
    "tipDocSolicita": "13",
    "numDocSolicita": "01234567-8"
  }
}
```

Los datos de `documento` (receptor, teléfono, correo, tipo y número de documento) se leen del JSON del documento original (sección `receptor` o `sujetoExcluido`). El JSON de ejemplo es ilustrativo; la validación del evento contra el JSON Schema de invalidación se hace en las pruebas (ver [Tests](../../../test/README.md)).

---

## Diferencias con GenericDTEUseCase

| Aspecto | GenericDTEUseCase | InvalidationUseCase |
|---|---|---|
| Número de control | Se reserva y confirma/libera | No aplica |
| Contingencia | Soportada según el endpoint | **No soportada** |
| Validación de estado | No | Sí (documento original y reemplazo) |
| Validación de dependencias | No | Sí (`ValidateDependencies`) |
| Documento existente | No se modifica | Se actualiza estado a `INVALIDATED` |
| Operaciones adicionales | Balance (crédito/débito) | Recuperación de balance (si se invalida una nota) |
| Estado de transmisión | No se verifica en el caso de uso (el `BaseTransmitter` decide) | Verifica `PROCESADO` |
| Persistencia | Documento en `dte` | Evento en `dte_events` |
| Respuesta | Archivo DTE del documento | `EventArchiveResponse` |
| Error de transmisión | Contingencia o liberación | Error directo |

---

## Condiciones de Error

| Paso | Error | Causa |
|---|---|---|
| 2 | `ValidationError` / `InvalidReplacementCode1And3` / `InvalidInvalidationType2` / `InvalidInvalidationType3` | Request incompleto o inválido |
| 3 | `DocumentAlreadyInvalid` | Documento ya invalidado |
| 3 | `DocumentReject` | Documento rechazado |
| 3 | `DocumentPending` | Documento pendiente de retransmitir |
| 3 | `FailedToVerifyDTE` | No se pudo verificar el estado (por ejemplo, el documento no existe para esta sucursal) |
| 4 | `FailedToGetDTE` | No se pudo cargar el documento original |
| 4 | `PendingAdjustmentNotes` | CCF o CR con Notas de Crédito/Débito vigentes |
| 6 | `DocumentWithoutStamp` | El documento original no tiene sello de recepción |
| 7 | `DTEError` / `ServiceError` | Reglas de negocio o plazo no cumplidos (ver validaciones de invalidación) |
| 8 | `ErrorMapping` | No se pudo armar el modelo Hacienda |
| 9 | `TransmissionFailed` | Hacienda no devolvió el estado `PROCESADO` |
| 9 | `HaciendaResponseError` / `HTTPResponseError` / error del firmador | Rechazo de Hacienda o fallo técnico, devueltos sin contingencia |
| 10 | `FailedToInvalidatedDTE`, `FailedToRecoverInvalidatedAmounts` | Fallo al actualizar el estado o el balance |
| 11 | Error del repositorio de eventos | Fallo al registrar el evento (el documento ya quedó invalidado) |

---

## Notas

1. **Sin contingencia**: Si Hacienda no está disponible, la invalidación falla. No se almacena para reintento posterior.
2. **Verificación de status**: El caso de uso verifica explícitamente que `result.Status == "PROCESADO"`; el `GenericDTEUseCase` no lo verifica.
3. **Balance cascade**: Al invalidar una nota de crédito/débito, el `InvalidationManager` revierte automáticamente el efecto en el balance de los documentos electrónicos relacionados.
4. **Sin acciones recursivas**: Para invalidar un CCF o CR con notas vigentes, el servicio rechaza la petición y lista las notas pendientes; nunca las invalida de forma encadenada.
5. **Atomicidad parcial**: Los pasos 10 y 11 no forman una transacción con la transmisión del paso 9; un fallo en ellos se devuelve como error aunque Hacienda ya haya aceptado el evento.
6. **Estado `EXPIRED`**: `ValidateStatus` solo rechaza `INVALIDATED`, `REJECTED` y `PENDING`; cualquier otro estado (incluido `EXPIRED`) pasa esa validación.
