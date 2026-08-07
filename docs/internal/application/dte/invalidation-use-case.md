# InvalidationUseCase — Caso de Uso de Invalidación

> **Paquete:** `internal/application/dte`
> **Archivo:** `invalidation_use_case.go`

## Descripción General

El `InvalidationUseCase` gestiona la invalidación de documentos tributarios electrónicos previamente emitidos. A diferencia de los DTEs regulares, la invalidación tiene un flujo propio que **no usa el `GenericDTEUseCase`** porque:

- No genera número de control secuencial
- No soporta contingencia
- Requiere validación de estado del documento original
- Modifica el estado de un documento existente

---

## Estructura

```go
type InvalidationUseCase struct {
    dteManager          DTEManager
    authManager         AuthManager
    invalidationManager invalidation.InvalidationManager
    mapper              *request_mapper.InvalidationMapper
    transmitter         ports.BaseTransmitter
}
```

### Dependencias

| Campo | Tipo | Propósito |
|---|---|---|
| `dteManager` | `DTEManager` | Se accede al documento original |
| `authManager` | `AuthManager` | Se obtiene información del emisor |
| `invalidationManager` | `InvalidationManager` | Se valida y ejecuta la invalidación |
| `mapper` | `InvalidationMapper` | Se mapea request → modelo de invalidación |
| `transmitter` | `BaseTransmitter` | Se transmite a Hacienda |

---

## Método Principal: `InvalidateDocument`

```go
func (u *InvalidationUseCase) InvalidateDocument(
    ctx context.Context,
    request structs.CreateInvalidationRequest,
) (*structs2.InvalidationResponse, error)
```

### Flujo de 10 Pasos

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
  │   ├── Documento original: debe estar en RECEIVED
  │   └── Documento reemplazo (tipo 1): debe existir y estar en RECEIVED
  │
  ▼
[4] Cargar documento original
  │   dteManager.GetByGenerationCode(ctx, branchID, request.GenerationCode)
  │
  ▼
[5] Cargar información del emisor
  │   authManager.GetIssuer(ctx, branchID)
  │
  ▼
[6] Mapear a modelo de dominio de invalidación
  │   mapper.MapToInvalidationData(&request, issuer, originalDTE.Details, originalDTE.CreatedAt)
  │
  ▼
[7] Validar documento de invalidación completo
  │   invalidationManager.Validate(ctx, branchID, invalidationDocument)
  │   (reglas de negocio: fechas, tipos, campos)
  │
  ▼
[8] Mapear a formato Hacienda
  │   response_mapper.ToMHInvalidation(invalidationDocument)
  │
  ▼
[9] Transmitir a Hacienda (SIN contingencia)
  │   transmitter.RetryTransmission(ctx, mhInvalidation, token, claims.NIT)
  │   ├── Si status != "PROCESADO" → error TransmissionFailed
  │   └── Si éxito → continuar
  │
  ▼
[10] Marcar documento como invalidado
      invalidationManager.InvalidateDocument(ctx, branchID, request.GenerationCode)
      ├── Actualizar estado → INVALIDATED
      └── Recuperar balance (si nota crédito/débito)
```

---

## Input: CreateInvalidationRequest

```go
type CreateInvalidationRequest struct {
    GenerationCode            string                 // UUID del documento a invalidar
    Reason                    InvalidationReasonReq  // Motivo de invalidación
    ReplacementGenerationCode *string                // UUID del doc de reemplazo (tipo 1 y 3)
}

type InvalidationReasonReq struct {
    Type                int     // 1=Reemplazo, 2=Anulación, 3=Otro
    ResponsibleName     string  // Nombre del responsable
    ResponsibleDocType  string  // Tipo de doc del responsable
    ResponsibleDocNum   string  // Número de doc del responsable
    RequesterName       string  // Nombre del solicitante
    RequesterDocType    string  // Tipo de doc del solicitante
    RequesterDocNum     string  // Número de doc del solicitante
    Reason              string  // Motivo (requerido para tipo 3)
}
```

---

## Output: Formato Hacienda de Invalidación

```json
{
  "identificacion": {
    "version": 2,
    "ambiente": "01",
    "codigoGeneracion": "UUID-nuevo",
    "fecAnula": "2024-03-15",
    "horAnula": "10:30:00"
  },
  "emisor": {
    "nit": "06142803101234",
    "nombre": "Empresa SA",
    "tipoEstablecimiento": "01",
    "correo": "empresa@email.com"
  },
  "documento": {
    "tipoDte": "03",
    "codigoGeneracion": "UUID-original",
    "selloRecibido": "SELLO40CARACTERES...",
    "numeroControl": "DTE-03-00010001-2024000000001",
    "fecEmi": "2024-03-10"
  },
  "motivo": {
    "tipoAnulacion": 1,
    "nombreResponsable": "Juan Pérez",
    "tipDocResponsable": "36",
    "numDocResponsable": "06142803101234",
    "nombreSolicita": "María López",
    "tipDocSolicita": "13",
    "numDocSolicita": "01234567-8"
  }
}
```

---

## Diferencias con GenericDTEUseCase

| Aspecto | GenericDTEUseCase | InvalidationUseCase |
|---|---|---|
| Número de control | Se reserva y confirma/libera | No aplica |
| Contingencia | Soportada | **No soportada** |
| Validación de estado | No | Sí (documento original) |
| Documento existente | No se modifica | Se actualiza estado |
| Operaciones adicionales | Balance (crédito/débito) | Recuperación de balance |
| Estado de transmisión | Verifica `ReceivedStatus` | Verifica `PROCESADO` |
| Error de transmisión | Contingencia o liberación | Error directo |

---

## Condiciones de Error

| Paso | Error | Causa |
|---|---|---|
| 2 | `ValidationError` | Request incompleto o inválido |
| 3 | `DocumentAlreadyInvalid` | Documento ya invalidado |
| 3 | `DocumentReject` | Documento rechazado |
| 3 | `DocumentPending` | Documento pendiente |
| 4 | Not found | Documento no existe para esta sucursal |
| 7 | `DTEError` | Reglas de negocio no cumplidas (fechas, tipos) |
| 9 | `TransmissionFailed` | Hacienda no aceptó la invalidación |

---

## Notas

1. **Sin contingencia**: Si Hacienda no está disponible, la invalidación falla. No se almacena para reintento posterior.
2. **Verificación de status**: Se verifica explícitamente que `result.Status == "PROCESADO"`, diferente al `GenericDTEUseCase` que verifica en el transmitter.
3. **Balance cascade**: Al invalidar una nota de crédito/débito, el `InvalidationManager` revierte automáticamente el efecto en el balance del documento original.
