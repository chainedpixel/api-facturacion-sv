# GenericDTEUseCase — Caso de Uso Central

> **Paquete:** `internal/application/dte`
> **Archivo:** `generic_dte_use_case.go`

## Descripción General

El `GenericDTEUseCase` es el orquestador central de la creación de documentos tributarios electrónicos. Implementa un flujo genérico de 9 pasos que aplica a **todos los tipos de DTE**, diferenciándose únicamente por las dependencias inyectadas (mapper, response mapper, operaciones adicionales).

---

## Estructura

```go
type GenericDTEUseCase struct {
    authService       auth.AuthManager
    dteService        DTEManager
    transmitter       appPorts.BaseTransmitter
    service           ports.DTEService
    sequentialManager SequentialNumberManager
    mapper            mapper.DTEMapper
    responseMapper    mapper.ResponseMapperFunc
    additionalOps     AdditionalOperationsFunc
}
```

### Dependencias

| Campo | Tipo | Propósito |
|---|---|---|
| `authService` | `AuthManager` | Se obtiene información del emisor (IssuerDTE) |
| `dteService` | `DTEManager` | Se persiste el DTE y se gestionan transacciones de balance |
| `transmitter` | `BaseTransmitter` | Se transmite el documento a Hacienda con reintentos |
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

### Flujo Completo de 9 Pasos

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
  │   (validaciones + generación de UUID y número de control)
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
[7] Transmitir a Hacienda
  │   transmitResult := transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
  │
  ├── [Si éxito]:
  │     │
  │     ▼
  │   [8a] Confirmar reserva + Persistir DTE
  │     │   sequentialManager.ConfirmReservation(controlNumber, generationCode, branchID)
  │     │   dteService.Create(mhModel, "NORMAL", "RECEIVED", receptionStamp)
  │     │
  │     ▼
  │   [8b] Operaciones adicionales
  │     │   additionalOps(ctx, result, branchID, mhModel)
  │     │   (Balance para notas de crédito/débito)
  │     │
  │     ▼
  │   [9] Retornar resultado exitoso
  │       return (mhModel, options, nil)
  │
  └── [Si error]:
        │
        ▼
      shouldHandleAsContingency(err)?
        │
        ├── TRUE (error de red/timeout):
        │     → Mantener reserva (flujo de contingencia)
        │     → return (mhModel, options, err)
        │
        └── FALSE (validación/rechazo):
              → Liberar reserva: ReleaseReservation()
              → return (nil, nil, err)
```

---

## Paso 1: Extracción del Contexto

Se extraen los claims de autenticación y el token JWT del contexto HTTP:

```go
claims := ctx.Value("claims").(*models.AuthClaims)
token  := ctx.Value("token").(string)
```

**AuthClaims contiene:**

| Campo | Tipo | Uso |
|---|---|---|
| `ClientID` | `uint` | ID del usuario |
| `BranchID` | `uint` | ID de la sucursal (filtra documentos) |
| `AuthType` | `string` | Tipo de autenticación |
| `NIT` | `string` | NIT del emisor (usado para firma) |

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
- Request HTTP (structs de la API) → Modelo de dominio con Value Objects
- Se inyecta la información del emisor
- Se valida la estructura del request

---

## Paso 4: Creación en Dominio

```go
result, err := u.service.Create(ctx, domainModel, claims.BranchID)
```

Se ejecuta el servicio de dominio específico del DTE:
- Validación de reglas de negocio
- Generación de UUID (código de generación)
- Reserva de número de control secuencial
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
generationCode := extractGenerationCode(mhModel)
controlNumber  := extractControlNumber(mhModel)
```

Se usa `utils.ExtractAuxiliarIdentification(mhModel)` que utiliza **reflexión** para extraer:
- `codigoGeneracion` (UUID del documento)
- `numeroControl` (número secuencial)

Estos se necesitan para la confirmación/liberación de la reserva.

---

## Paso 7: Transmisión a Hacienda

```go
transmitResult, err := u.transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
```

Se transmite el documento firmado a Hacienda con lógica de reintentos (ver [transmitter.md](./transmitter.md)).

### Construcción del SuccessOptions

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

**Persistir DTE:**
```go
dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, transmitResult.ReceptionStamp)
```

---

## Paso 8b: Operaciones Adicionales

```go
err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
```

Solo aplica para Nota de Crédito y Nota de Débito — genera transacciones de balance contra los documentos relacionados (ver [additional-operations.md](./additional-operations.md)).

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
| HTTP 5xx | **Sí** | Mantener reserva |
| Error desconocido | **Sí** | Mantener reserva |

### Liberación de Reserva (cuando NO es contingencia)

```go
// Se extrae información del error de Hacienda
haciendaErr := err.(*HaciendaResponseError)
rejectionReason := haciendaErr.Description
haciendaCode := haciendaErr.Code

sequentialManager.ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode, claims.BranchID)
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/application/dte/generic_dte_use_case.go` | Caso de uso genérico |
| `internal/application/dte/dte_use_case_factory.go` | Factory que crea variantes |
| `internal/application/dte/additional_operations.go` | Operaciones post-transmisión |
| `pkg/mapper/` | Mappers de request y response |
