# Casos de Uso — Análisis Detallado

**Ubicación:** `internal/application/dte/`, `internal/application/auth/`

---

## GenericDTEUseCase

**Archivo:** `internal/application/dte/generic_dte_use_case.go`

Este es el motor de todo el pipeline de creación de DTE. Cada tipo de documento — factura, CCF, nota de crédito, nota de débito, retención, nota de remisión, factura de sujeto excluido — usa el mismo método `Create`.

### Flujo Completo de `Create()`

```go
func (u *GenericDTEUseCase) Create(ctx context.Context, req interface{}) (interface{}, *response.SuccessOptions, error)
```

**Paso 1 — Extraer contexto de autenticación**

```go
claims := ctx.Value("claims").(*models.AuthClaims)
token  := ctx.Value("token").(string)
```

`claims` lleva `BranchID`, `NIT`, `ClientID` y `AuthType`. El string JWT crudo se almacena en `token` y se reenvía al transmisor para que pueda intercambiarlo por un token de Hacienda.

**Paso 2 — Cargar información del emisor**

```go
issuer, err := u.authService.GetIssuer(ctx, claims.BranchID)
```

Esto llama a `AuthManager.GetIssuer`, que a su vez llama a `AuthRepository.GetIssuerInfoByBranchID`. El resultado es un struct `dte.IssuerDTE` completamente poblado que contiene NIT, NRC, códigos de establecimiento, códigos de POS, dirección e información de contacto. Estos datos se inyectan en el documento mapeado como la sección `emisor`.

**Paso 3 — Mapear petición → modelo de dominio**

```go
domainModel, err := u.mapper.MapToDomainModel(req, issuer)
```

El mapper es específico del tipo (ej. `InvoiceMapperAdapter`). Deserializa el struct de petición crudo en un modelo de dominio fuertemente tipado (ej. `invoice.Invoice`) y poblado con todos los campos del emisor.

**Paso 4 — El servicio de dominio crea y valida**

```go
result, err := u.service.Create(ctx, domainModel, claims.BranchID)
```

El servicio de dominio (ej. `InvoiceService`) realiza toda la validación de reglas de negocio (totales de ítems, cálculos de impuestos, rangos de fechas, campos requeridos), asigna un número de control secuencial via `SequentialNumberManager.Reserve`, genera un código de generación UUID y devuelve el modelo de dominio completamente validado.

**Paso 5 — Mapear modelo de dominio → formato JSON de Hacienda**

```go
mhModel := u.responseMapper(result)
```

El `responseMapper` es una `mapper.ResponseMapperFunc` — una función simple que convierte el modelo de dominio a la estructura JSON exacta que espera Hacienda (ej. `response_mapper.ToMHInvoice(result)`).

**Paso 6 — Extraer identificadores del modelo MH**

```go
generationCode, _ := extractGenerationCode(mhModel)   // lee Identification.GenerationCode
controlNumber, _  := extractControlNumber(mhModel)     // lee Identification.ControlNumber
```

Ambas funciones usan `utils.ExtractAuxiliarIdentification`, que usa reflexión para navegar al struct `identificacion` embebido independientemente del tipo concreto del modelo MH.

**Paso 7 — Transmitir a Hacienda**

```go
transmitResult, err := u.transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
```

`RetryTransmission` está definido en `appPorts.BaseTransmitter`. La implementación (`RetryTransmitterService`) envuelve `MHTransmitter.Transmit` con lógica de reintento.

**Paso 8a — Transmisión exitosa**

```go
options.ReceptionStamp = transmitResult.ReceptionStamp

confirmErr := u.sequentialManager.ConfirmReservation(ctx, controlNumber, generationCode)

err = u.dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, transmitResult.ReceptionStamp)
```

En caso de éxito, el caso de uso:
1. Adjunta el `selloRecibido` (sello de recepción) a las opciones de respuesta — esto aparecerá en la respuesta de la API.
2. Confirma la reserva de secuencia para que el número de control quede permanentemente confirmado.
3. Persiste el documento en la base de datos con estado `RECEIVED` y tipo de transmisión `NORMAL`.

**Paso 8b — Operaciones adicionales (solo notas de crédito/débito)**

```go
if u.additionalOps != nil {
    err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
}
```

Las notas de crédito y débito deben actualizar el control de saldo del documento original referenciado. Este hook se inyecta en tiempo de factory via `DTEOperations.GetCreditNoteOperations` o `GetDebitNoteOperations`.

**Paso 9 — Transmisión fallida**

Cuando `RetryTransmission` devuelve error, el caso de uso llama a `shouldHandleAsContingency(err)` para decidir qué hacer.

### Lógica de `shouldHandleAsContingency(err)`

```go
func (u *GenericDTEUseCase) shouldHandleAsContingency(err error) bool {
    // ValidationError: fallo de validación de campo — no es contingencia
    var validationErr *dte_errors.ValidationError
    if errors.As(err, &validationErr) {
        return false
    }

    // HaciendaResponseError con estado RECHAZADO: Hacienda rechazó explícitamente — no es contingencia
    var haciendaErr *hacienda_error.HaciendaResponseError
    if errors.As(err, &haciendaErr) {
        if haciendaErr.Status == "RECHAZADO" {
            return false
        }
    }

    // ServiceError: error de servicio interno — no es contingencia
    var generalErr *shared_error.ServiceError
    if errors.As(err, &generalErr) {
        return false
    }

    // DTEError: error de regla de negocio del dominio — no es contingencia
    var businessErr *dte_errors.DTEError
    if errors.As(err, &businessErr) {
        return false
    }

    // Todo lo demás (errores de red, HTTP 5xx, timeouts) — SÍ es contingencia
    return true
}
```

**Si `shouldHandleAsContingency` devuelve false (fallo de transmisión, no contingencia):**

La reserva de secuencia se libera para que el número de control pueda reutilizarse. Si el error es un `HaciendaResponseError`, el código de error de Hacienda y la descripción se almacenan en el registro de reserva para auditoría:

```go
releaseErr := u.sequentialManager.ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode)
```

**Si `shouldHandleAsContingency` devuelve true:**

El caso de uso devuelve el modelo MH y el error de transmisión al handler sin cambios. El handler (no el caso de uso) es responsable de llamar a `ContingencyHandler.HandleContingency`. Esta separación significa que el caso de uso nunca sabe si el procesamiento de contingencia tuvo éxito.

### El Flag `DocumentConfig.UsesContingency`

En `GenericCreatorDTEHandler`, cada tipo de documento registrado tiene un `helpers.DocumentConfig.UsesContingency` booleano. Cuando un error llega al handler:

```go
if config.UsesContingency {
    err = h.handleErrorForContingency(r.Context(), resp, config.DocumentType, options, err, w)
    // Si la contingencia tiene éxito: devuelve 201 con metadatos de contingencia
    // Si la contingencia también falla: devuelve el error original
} else {
    h.respWriter.HandleError(w, err)
}
```

Los documentos configurados con `UsesContingency: false` (ej. retención en algunas configuraciones) siempre devolverán un error al cliente si la transmisión falla.

---

## InvalidationUseCase

**Archivo:** `internal/application/dte/invalidation_use_case.go`

### Flujo Completo

```go
func (u *InvalidationUseCase) InvalidateDocument(ctx context.Context, request structs.CreateInvalidationRequest) (*structs2.InvalidationResponse, error)
```

**Paso 1 — Extraer claims y token** (mismo patrón que `GenericDTEUseCase`)

**Paso 2 — Validar la estructura de la petición de invalidación**

```go
if err := u.mapper.ValidateInvalidationReRequest(&request); err != nil {
    return nil, err
}
```

El mapper valida los campos requeridos (código de generación, tipo de invalidación, nombre/documento del responsable, identidad del solicitante).

**Paso 3 — Validar el estado del documento original**

```go
if err := u.invalidationManager.ValidateStatus(ctx, claims.BranchID, request); err != nil {
    return nil, err
}
```

El servicio de dominio verifica que el documento referenciado existe y está en un estado que permite la invalidación (`RECEIVED`). Los documentos que ya están `INVALIDATED` o `REJECTED` no pueden invalidarse de nuevo.

**Paso 4 — Cargar el DTE original**

```go
originalDTE, err := u.dteManager.GetByGenerationCode(ctx, claims.BranchID, request.GenerationCode)
```

El documento original es necesario para poblar la sección `documento` de la invalidación con el tipo DTE original, número de control y fecha de emisión.

**Paso 5 — Cargar información del emisor**

```go
issuer, err := u.authManager.GetIssuer(ctx, claims.BranchID)
```

**Paso 6 — Mapear al modelo de dominio de invalidación**

```go
invalidationDocument, err := u.mapper.MapToInvalidationData(&request, issuer, originalDTE.Details, originalDTE.CreatedAt)
```

Esto produce un struct de dominio `invalidation.InvalidationDocument` poblado con los datos de identificación del DTE original y los campos específicos de razón de invalidación.

**Paso 7 — Validar el documento de invalidación completo**

```go
if err = u.invalidationManager.Validate(ctx, claims.BranchID, invalidationDocument); err != nil {
    return nil, err
}
```

El servicio de dominio aplica las reglas de negocio:
- Para facturas electrónicas (tipo 01): la solicitud de invalidación debe estar dentro de los 90 días de la emisión
- Para otros tipos de DTE: la invalidación debe estar dentro de las 24 horas de la emisión

**Paso 8 — Mapear al formato Hacienda**

```go
mhInvalidation := response_mapper.ToMHInvalidation(invalidationDocument)
```

**Paso 9 — Transmitir a Hacienda**

```go
result, err := u.transmitter.RetryTransmission(ctx, mhInvalidation, token, claims.NIT)
```

Nota: la invalidación NO pasa por el flujo de contingencia. Si la transmisión falla, el error se devuelve directamente al cliente.

**Paso 10 — Actualizar el estado del documento original**

```go
if err := u.invalidationManager.InvalidateDocument(ctx, claims.BranchID, request.GenerationCode); err != nil {
    return nil, err
}
```

El manager de dominio actualiza el estado del DTE original a `INVALIDATED` en la base de datos.

**Recuperación de Saldo**

Las notas de crédito y débito que referencian una factura tendrán su impacto en el saldo revertido cuando la factura original sea invalidada. Esto se maneja dentro de `InvalidationManager.InvalidateDocument` en la capa de dominio — el caso de uso de aplicación no necesita saber sobre esto.

---

## DTEConsultUseCase

**Archivo:** `internal/application/dte/dte_consult_use_case.go`

### `GetByGenerationCode()`

```go
func (u *DTEConsultUseCase) GetByGenerationCode(ctx context.Context, id string) (interface{}, error) {
    claims := ctx.Value("claims").(*models.AuthClaims)
    dte, err := u.dteService.GetByGenerationCodeConsult(ctx, claims.BranchID, id)
    return dte, err
}
```

El método limita la consulta al `claims.BranchID` para que una sucursal solo pueda acceder a sus propios documentos. `GetByGenerationCodeConsult` (distinto de `GetByGenerationCode`) devuelve los detalles completos del DTE incluyendo el JSON crudo, estado y número de control.

### `GetAllDTEs()` con `parseDTEFilters()`

```go
func (u *DTEConsultUseCase) GetAllDTEs(ctx context.Context, r *http.Request) (*dte.DTEListResponse, error) {
    filters, err := parseDTEFilters(r)
    // ...
    return u.dteService.GetAllDTEs(ctx, filters)
}
```

`parseDTEFilters` lee los parámetros de query y puebla un struct `dte.DTEFilters`. Cada parámetro es validado; los valores inválidos devuelven un error en lugar de ser silenciosamente ignorados.

**Detalles del parseo de filtros:**

| Parámetro de Query | Campo | Validación |
|---|---|---|
| `all=true` | `IncludeAll` | Si es true, `BranchID` no se establece (consulta de administrador entre todas las sucursales) |
| `status` | `Status` | Debe estar en `constants.ValidReceiverDocumentStates`: `RECEIVED`, `INVALIDATED`, `REJECTED` |
| `transmission` | `Transmission` | Debe estar en `constants.ValidTransmissionTypes`: `NORMAL`, `CONTINGENCY` |
| `type` | `DTEType` / `DTETypes` | Se permite lista separada por comas; cada valor validado contra `constants.ValidDTETypes` (01–15) |
| `startDate` | `StartDate` | Formato RFC3339 requerido |
| `endDate` | `EndDate` | Formato RFC3339 requerido |
| `page` | `Page` | Por defecto 1 si falta o es inválido |
| `page_size` | `PageSize` | Por defecto 5 si falta o es inválido |

Ejemplo de combinación de filtros:

```
GET /api/v1/dte?status=RECEIVED&type=01,03&page=2&page_size=10&startDate=2024-01-01T00:00:00Z
```

Esto devuelve la página 2 (registros 11–20) de facturas y CCFs recibidos emitidos desde el 1 de enero de 2024.

---

## AuthUseCase

**Archivo:** `internal/application/auth/auth_use_case.go`

### Flujo de Login

```go
func (a *AuthUseCase) Login(ctx context.Context, credentials *models.AuthCredentials) (string, error)
```

El struct `AuthCredentials` lleva tres piezas de información:

```go
type AuthCredentials struct {
    MHCredentials *HaciendaCredentials `json:"credentials"`  // usuario + contraseña de Hacienda
    APIKey        string               `json:"api_key"`
    APISecret     string               `json:"api_secret"`
}
```

**Paso 1 — Validación estructural**

```go
if err := credentials.Validate(); err != nil {
    return "", err
}
```

`Validate()` verifica que `api_key`, `api_secret` y `credentials.username/password` no estén vacíos. Cualquier campo faltante devuelve un `ValidationError`.

**Paso 2 — Delegar al AuthManager**

```go
token, err := a.authManager.Login(ctx, credentials)
```

El `AuthManager.Login` del dominio (implementado por `AuthService`) realiza lo siguiente:
1. Busca la sucursal por `api_key`
2. Verifica `api_secret` contra el secreto almacenado (hasheado)
3. Valida las credenciales de Hacienda llamando realmente al endpoint de autenticación de Hacienda
4. Si todo pasa, emite un JWT firmado con `AuthClaims`
5. Almacena el token de Hacienda en Redis con clave del string JWT

El string JWT se devuelve al cliente y debe incluirse en todas las peticiones posteriores como `Authorization: Bearer <token>`.

### Flujo de Registro

```go
func (a *AuthUseCase) Register(ctx context.Context, user *user.User) ([]user.ListBranchesResponse, error)
```

**Paso 1 — Validación estructural**

```go
if err := user.Validate(); err != nil {
    return nil, err
}
```

**Paso 2 — Generar API keys en masa**

```go
keys, secrets, err := a.cryptManager.GenerateBulkAPIKeys(len(user.BranchOffices))
```

`CryptManager.GenerateBulkAPIKeys` genera un par `(key, secret)` por sucursal. La key es un string URL-safe aleatorio; el secret se hashea antes del almacenamiento.

**Paso 3 — Asignar keys a las sucursales**

```go
user.SetBranchesKeysAndSecrets(keys, secrets)
```

Esto muta el slice `user.BranchOffices`, emparejando cada sucursal con su propia API key y secret.

**Paso 4 — Persistir en la base de datos**

```go
if err = a.authManager.Create(ctx, user); err != nil {
    return nil, ...
}
```

`AuthManager.Create` envuelve toda la inserción (usuario + sucursales + direcciones) en una única transacción de base de datos. Si cualquier inserción falla, toda la transacción se revierte.

**Paso 5 — Devolver listado de sucursales**

```go
return user.ListBranches(), nil
```

`ListBranches()` devuelve un `[]user.ListBranchesResponse`, cada uno con el ID de sucursal, código de establecimiento, API key y API secret en texto plano. Esta es la única vez que el secret en texto plano se devuelve al llamador; no se almacena y no puede recuperarse después.
