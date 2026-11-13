# GenericDTEUseCase — Caso de Uso Central

> **Paquete:** `internal/application/dte`
> **Archivo:** `generic_dte_use_case.go`

## Descripción General

El `GenericDTEUseCase` es el orquestador central de la creación de documentos tributarios electrónicos. Implementa un flujo genérico que aplica a **todos los tipos de DTE emitidos por la API** (01, 03, 04, 05, 06, 07 y 14), diferenciándose únicamente por las dependencias inyectadas (mapper, response mapper, operaciones adicionales). Tras la transmisión exitosa guarda la evidencia de transmisión (`TransmissionEvidence`) y responde el Archivo DTE.

---

## Estructura

```go
type GenericDTEUseCase struct {
    authService       auth.AuthManager
    dteService        transmissionPorts.DTEManager
    transmitter       appPorts.BaseTransmitter
    service           ports.DTEService
    sequentialManager transmissionPorts.SequentialNumberManager
    mapper            mapper.DTEMapper
    responseMapper    mapper.ResponseMapperFunc
    additionalOps     AdditionalOperationsFunc
}
```

### Dependencias

| Campo | Tipo | Propósito |
|---|---|---|
| `authService` | `AuthManager` | Se obtiene información del emisor (IssuerDTE) |
| `dteService` | `DTEManager` | Se persiste el DTE con su evidencia y se gestionan transacciones de balance |
| `transmitter` | `BaseTransmitter` | Se firma y transmite el documento a Hacienda (con reintentos) y se obtiene la evidencia |
| `service` | `DTEService` | Se ejecuta la lógica de dominio específica del DTE |
| `sequentialManager` | `SequentialNumberManager` | Se confirma/libera la reserva de número de control |
| `mapper` | `DTEMapper` | Se mapea request HTTP → modelo de dominio |
| `responseMapper` | `ResponseMapperFunc` | Se mapea modelo de dominio → formato Hacienda |
| `additionalOps` | `AdditionalOperationsFunc` | Se ejecutan operaciones post-transmisión (balance) |

---

## Método Principal: `Create`

```go
func (u *GenericDTEUseCase) Create(
    ctx context.Context,
    req interface{},
) (interface{}, *response.SuccessOptions, error)
```

### Flujo Completo

```
Request HTTP
  │
  ▼
[1] Extraer contexto de autenticación
  │   claims := ctx.Value("claims").(*AuthClaims)
  │   token  := ctx.Value("token").(string)
  │
  ▼
[2] Obtener información del emisor
  │   issuer := authService.GetIssuer(ctx, claims.BranchID)
  │
  ▼
[3] Mapear request → modelo de dominio
  │   domainModel := mapper.MapToDomainModel(req, issuer)
  │
  ▼
[4] Crear DTE en capa de dominio
  │   result := service.Create(ctx, domainModel, claims.BranchID)
  │   (validaciones + número de control con reserva + código de generación)
  │
  ▼
[5] Mapear modelo de dominio → formato Hacienda
  │   mhModel := responseMapper(result)
  │
  ▼
[6] Extraer identificadores (código de generación + número de control)
  │   generationCode := extractGenerationCode(mhModel)
  │   controlNumber  := extractControlNumber(mhModel)
  │
  ▼
[7] Firmar y transmitir a Hacienda
  │   transmitResult := transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
  │
  ├── [Si éxito]:
  │     │
  │     ▼
  │   [8a] Confirmar reserva + Persistir DTE con evidencia
  │     │   sequentialManager.ConfirmReservation(controlNumber, generationCode, branchID)
  │     │   dteService.Create(mhModel, "NORMAL", "RECEIVED", &TransmissionEvidence{
  │     │       sello, JWS firmado, fhProcesamiento, respuesta de Hacienda})
  │     │
  │     ▼
  │   [8b] Operaciones adicionales
  │     │   additionalOps(ctx, result, branchID, mhModel)
  │     │   (Balance para notas de crédito/débito)
  │     │
  │     ▼
  │   [9] Responder el Archivo DTE
  │       archive := dte_documents.BuildArchive(JWS, sello)
  │       return (archive, options, nil)      // respaldo: (mhModel, options, nil)
  │
  └── [Si error]:
        │
        ▼
      shouldHandleAsContingency(err)?
        │
        ├── TRUE (red/timeout/HTTP no 2xx/firmador):
        │     → Mantener reserva (el handler aplica la contingencia)
        │     → return (mhModel, options, err)
        │
        └── FALSE (validación/rechazo/error de servicio o de negocio):
              → Liberar reserva: ReleaseReservation()
              → return (mhModel, options, err)
```

---

## Paso 1: Extracción del Contexto

Se extraen los claims de autenticación y el token JWT del contexto HTTP:

```go
claims := ctx.Value("claims").(*models.AuthClaims)
token  := ctx.Value("token").(string)
```

La aserción de tipo es directa: si los middlewares de autenticación no cargaron ambos valores, el caso de uso entra en pánico.

**AuthClaims contiene:**

| Campo | Tipo | Uso |
|---|---|---|
| `ClientID` | `uint` | ID del usuario |
| `BranchID` | `uint` | ID de la sucursal (filtra documentos) |
| `AuthType` | `string` | Tipo de autenticación |
| `NIT` | `string` | NIT del emisor (usado para firma) |
| `ExpiresAt` | `time.Time` | Fecha de expiración del token |

---

## Paso 2: Información del Emisor

```go
issuer, err := u.authService.GetIssuer(ctx, claims.BranchID)
```

Se obtiene `*dte.IssuerDTE` que contiene toda la información fiscal del emisor: NIT, NRC, nombre comercial, dirección, códigos de establecimiento, etc.

---

## Paso 3: Mapeo Request → Dominio

```go
domainModel, err := u.mapper.MapToDomainModel(req, issuer)
```

El mapper específico del DTE transforma:
- Request HTTP (structs de la API) → Modelo de entrada del dominio con Value Objects
- Se inyecta la información del emisor
- Se valida el tipo del request y los valores de sus campos (los value objects rechazan valores inválidos)

---

## Paso 4: Creación en Dominio

```go
result, err := u.service.Create(ctx, domainModel, claims.BranchID)
```

Se ejecuta el servicio de dominio específico del DTE:
- Validación de reglas de negocio
- Reserva de número de control secuencial (`SequentialNumberManager`)
- Generación de UUID (código de generación)
- Retorna el modelo validado con identificadores

---

## Paso 5: Mapeo Dominio → Hacienda

```go
mhModel := u.responseMapper(result)
```

Se transforma el modelo de dominio al formato JSON que espera la API de Hacienda (ver [mappers.md](./mappers.md)).

---

## Paso 6: Extracción de Identificadores

```go
generationCode, err := extractGenerationCode(mhModel)
controlNumber, err  := extractControlNumber(mhModel)
```

Se usa `utils.ExtractAuxiliarIdentification(mhModel)`, que serializa el modelo a JSON y lo deserializa en un extractor, para leer:
- `codigoGeneracion` (UUID del documento)
- `numeroControl` (número secuencial)

Estos se necesitan para la confirmación/liberación de la reserva. Si alguno está vacío se devuelve `ValidationError` `RequiredField`.

---

## Paso 7: Firma y Transmisión a Hacienda

```go
transmitResult, err := u.transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
```

`BaseTransmitter` firma el documento y lo transmite a Hacienda con lógica de reintentos (ver [transmitter.md](./transmitter.md)). El resultado trae, además del sello, el JWS transmitido (`SignedDocument`) y la respuesta completa de Hacienda (`HaciendaResponse`).

### Construcción del SuccessOptions

Antes de transmitir se arma:

```go
options := &response.SuccessOptions{
    Ambient:        config.Server.AmbientCode,
    GenerationCode: generationCode,
    EmissionDate:   utils.TimeNow(),
}
```

Si la transmisión es exitosa, se agrega el sello de recepción:
```go
options.ReceptionStamp = transmitResult.ReceptionStamp
```

---

## Paso 8a: Confirmación y Persistencia

**Confirmar reserva:**
```go
sequentialManager.ConfirmReservation(ctx, controlNumber, generationCode, claims.BranchID)
```

Si falla, se devuelve el error y el documento no se persiste.

**Persistir DTE con su evidencia:**
```go
dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, &dte.TransmissionEvidence{
    ReceptionStamp:   transmitResult.ReceptionStamp,
    SignedDocument:   transmitResult.SignedDocument,
    ProcessingDate:   transmitResult.ProcessingDate,
    HaciendaResponse: transmitResult.HaciendaResponse,
})
```

`TransmissionEvidence` (`internal/domain/core/dte`) agrupa lo que Hacienda devuelve y lo que se transmitió:

| Campo | Contenido |
|---|---|
| `ReceptionStamp` | `selloRecibido` otorgado por Hacienda |
| `SignedDocument` | JWS transmitido (`firmaElectronica` del Archivo DTE) |
| `ProcessingDate` | `fhProcesamiento` informado por Hacienda |
| `HaciendaResponse` | Respuesta completa de Hacienda en JSON |

Para transmisiones normales, `DTEService.Create` exige un sello no vacío (`MissingReceptionStamp`).

---

## Paso 8b: Operaciones Adicionales

```go
if u.additionalOps != nil {
    err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
}
```

Solo tiene efecto para Nota de Crédito y Nota de Débito — genera transacciones de balance contra los documentos relacionados (ver [additional-operations.md](./additional-operations.md)). Los demás tipos reciben una función vacía. Un error aquí se devuelve junto con `mhModel` y `options`, aunque el documento ya fue transmitido y guardado.

---

## Paso 9: Respuesta con el Archivo DTE

```go
archive, archiveErr := dte_documents.BuildArchive(transmitResult.SignedDocument, transmitResult.ReceptionStamp)
if archiveErr != nil {
    return mhModel, options, nil
}
return archive, options, nil
```

Normativa de Cumplimiento DTE v2.0 (Anexo I, regla 4.1): el Archivo DTE contiene la estructura de datos firmada de forma íntegra, más `firmaElectronica` y `selloRecibido`. `BuildArchive` extrae la estructura del payload del JWS y agrega ambos atributos al final sin reordenar el resto. Si el JWS no es válido solo se registra una advertencia y se responde `mhModel` (la estructura en formato Hacienda). El mismo Archivo DTE puede consultarse después con `GET /dte/{id}/archivo`.

---

## Manejo de Errores de Transmisión

### `shouldHandleAsContingency`

```go
func (u *GenericDTEUseCase) shouldHandleAsContingency(err error) bool
```

| Tipo de Error | Contingencia | Acción |
|---|---|---|
| `ValidationError` | No | Liberar reserva |
| `HaciendaResponseError` (RECHAZADO) | No | Liberar reserva con código y motivo |
| `ServiceError` | No | Liberar reserva |
| `DTEError` | No | Liberar reserva |
| Error de red / timeout | **Sí** | Mantener reserva |
| `HTTPResponseError` (HTTP 5xx, respuesta no 2xx, cuerpo vacío o inválido) | **Sí** | Mantener reserva |
| Error del firmador | **Sí** | Mantener reserva |
| Error desconocido | **Sí** | Mantener reserva |

En ambos casos el caso de uso devuelve `(mhModel, options, err)`; la decisión de aplicar contingencia corresponde al handler (`DocumentConfig.UsesContingency`, ver [use-cases.md](../use-cases.md)).

### Liberación de Reserva (cuando NO es contingencia)

```go
rejectionReason := err.Error()
haciendaCode := ""

var haciendaErr *hacienda_error.HaciendaResponseError
if errors.As(err, &haciendaErr) {
    haciendaCode = haciendaErr.Code
    rejectionReason = haciendaErr.Description
    if len(haciendaErr.Observations) > 0 {
        rejectionReason += " | Observaciones: " + strings.Join(haciendaErr.Observations, "; ")
    }
}

sequentialManager.ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode, claims.BranchID)
```

Si el error no es de Hacienda, el motivo es el texto del error y el código queda vacío. Un fallo al liberar solo se registra en el log; el error de transmisión se devuelve igual.

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/application/dte/generic_dte_use_case.go` | Caso de uso genérico |
| `internal/application/dte/dte_use_case_factory.go` | Factory que crea variantes |
| `internal/application/dte/additional_operations.go` | Operaciones post-transmisión |
| `internal/application/dte/base_transmit.go` | `BaseTransmitter`: firma, reintentos y adjunto del JWS |
| `internal/domain/core/dte/transmission_evidence.go` | `TransmissionEvidence` |
| `internal/domain/dte/dte_documents/archive.go` | `BuildArchive` (Archivo DTE) |
| `pkg/mapper/` | Mappers de request y response |
