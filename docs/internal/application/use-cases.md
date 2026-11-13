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

`claims` lleva `ClientID`, `BranchID`, `AuthType`, `NIT` y `ExpiresAt`. El string JWT crudo se almacena en `token` y se reenvía al transmisor para que pueda intercambiarlo por un token de Hacienda. La aserción de tipo es directa: si el middleware no cargó los valores en el contexto, el caso de uso entra en pánico.

**Paso 2 — Cargar información del emisor**

```go
issuer, err := u.authService.GetIssuer(ctx, claims.BranchID)
```

Esto llama a `AuthManager.GetIssuer`, que a su vez llama a `AuthRepository.GetIssuerInfoByBranchID`. El resultado es un struct `dte.IssuerDTE` que contiene NIT, NRC, códigos de establecimiento, códigos de POS, dirección e información de contacto. Estos datos se inyectan en el documento mapeado como la sección `emisor`.

**Paso 3 — Mapear petición → modelo de dominio**

```go
domainModel, err := u.mapper.MapToDomainModel(req, issuer)
```

El mapper es específico del tipo (producido por `MapperFactory`, ej. `CreateInvoiceMapperAdapter()`). Verifica que la petición sea del tipo esperado y la convierte en el modelo de entrada del dominio (ej. `InvoiceData`), con los datos del emisor incluidos.

**Paso 4 — El servicio de dominio crea y valida**

```go
result, err := u.service.Create(ctx, domainModel, claims.BranchID)
```

El servicio de dominio (ej. el de factura) arma el documento, aplica las validaciones de reglas de negocio y genera el número de control (usando `SequentialNumberManager`, que reserva el correlativo) y el código de generación (UUID). Devuelve el modelo de dominio validado.

**Paso 5 — Mapear modelo de dominio → formato JSON de Hacienda**

```go
mhModel := u.responseMapper(result)
```

El `responseMapper` es una `mapper.ResponseMapperFunc` — una función simple que convierte el modelo de dominio a la estructura JSON exacta que espera Hacienda (ej. `response_mapper.ToMHInvoice(result)`).

**Paso 6 — Extraer identificadores del modelo MH**

```go
generationCode, err := extractGenerationCode(mhModel)   // lee Identification.GenerationCode
controlNumber, err  := extractControlNumber(mhModel)    // lee Identification.ControlNumber
```

Ambas funciones usan `utils.ExtractAuxiliarIdentification`, que serializa el modelo y lee el objeto `identificacion` independientemente del tipo concreto. Si el valor está vacío se devuelve `ValidationError` `RequiredField`. En este punto se arma también `response.SuccessOptions` con `Ambient` (`config.Server.AmbientCode`), `GenerationCode` y `EmissionDate`.

**Paso 7 — Transmitir a Hacienda**

```go
transmitResult, err := u.transmitter.RetryTransmission(ctx, mhModel, token, claims.NIT)
```

`RetryTransmission` está definido en `appPorts.BaseTransmitter`. La implementación (`dte.BaseTransmitter`, en `base_transmit.go`) firma el documento con `SignerManager.SignDTE`, lo transmite mediante `MHTransmitter.Transmit` y adjunta el JWS transmitido al resultado (`withSignedDocument`). Ver [transmitter.md](dte/transmitter.md) para el detalle de reintentos.

**Paso 8a — Transmisión exitosa**

```go
options.ReceptionStamp = transmitResult.ReceptionStamp

confirmErr := u.sequentialManager.ConfirmReservation(ctx, controlNumber, generationCode, claims.BranchID)

err = u.dteService.Create(ctx, mhModel, constants.TransmissionNormal, constants.DocumentReceived, &dte.TransmissionEvidence{
    ReceptionStamp:   transmitResult.ReceptionStamp,
    SignedDocument:   transmitResult.SignedDocument,
    ProcessingDate:   transmitResult.ProcessingDate,
    HaciendaResponse: transmitResult.HaciendaResponse,
})
```

En caso de éxito, el caso de uso:
1. Adjunta el `selloRecibido` (sello de recepción) a las opciones de respuesta.
2. Confirma la reserva de secuencia para que el número de control quede permanentemente confirmado. Si la confirmación falla, se devuelve el error (con `mhModel` y `options`) y el documento no se guarda.
3. Persiste el documento con estado `RECEIVED`, transmisión `NORMAL` y su `TransmissionEvidence`: JWS firmado (`firmaElectronica`), sello de recepción, `fhProcesamiento` y respuesta completa de Hacienda. `DTEService.Create` exige un sello no vacío para transmisiones normales (`MissingReceptionStamp`).

**Paso 8b — Operaciones adicionales (solo notas de crédito/débito)**

```go
if u.additionalOps != nil {
    err = u.additionalOps(ctx, result, claims.BranchID, mhModel)
}
```

Las notas de crédito y débito registran una transacción de balance sobre cada documento electrónico relacionado. Este hook se inyecta en tiempo de factory via `DTEOperations.GetCreditNoteOperations` o `GetDebitNoteOperations`; los demás tipos reciben `GetNoOperation()`. Si el hook falla, se devuelve el error junto con `mhModel` y `options`, aunque el documento ya fue transmitido y guardado.

**Paso 9 — Responder el Archivo DTE**

```go
archive, archiveErr := dte_documents.BuildArchive(transmitResult.SignedDocument, transmitResult.ReceptionStamp)
if archiveErr != nil {
    return mhModel, options, nil   // respaldo: se responde la estructura
}
return archive, options, nil
```

Normativa de Cumplimiento DTE v2.0 (Anexo I, regla 4.1): el Archivo DTE contiene la estructura de datos firmada, íntegra y sin alteraciones, más `firmaElectronica` y `selloRecibido`. `BuildArchive` toma la estructura del payload del JWS (lo que realmente se firmó y transmitió) y agrega ambos atributos al final. Si el JWS no es válido, solo se registra una advertencia y se responde `mhModel` (la estructura en formato Hacienda).

**Paso 10 — Transmisión fallida**

Cuando `RetryTransmission` devuelve error, el caso de uso llama a `shouldHandleAsContingency(err)` para decidir qué hacer. En ambos casos devuelve `(mhModel, options, err)`.

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

Los errores del firmador (`fmt.Errorf` de `DTESigner`) y los errores `HTTPResponseError` de Hacienda (respuestas no 2xx o sin cuerpo válido) no coinciden con ningún tipo anterior, por lo que se tratan como contingencia.

**Si `shouldHandleAsContingency` devuelve false (fallo de transmisión, no contingencia):**

La reserva de secuencia se libera para que el número de control pueda reutilizarse. Si el error es un `HaciendaResponseError`, el código de error de Hacienda y la descripción (más las observaciones, separadas por `" | Observaciones: "`) se almacenan en el registro de reserva para auditoría:

```go
releaseErr := u.sequentialManager.ReleaseReservation(ctx, controlNumber, rejectionReason, haciendaCode, claims.BranchID)
```

Si el error no es de Hacienda, `rejectionReason` es el texto del error y `haciendaCode` queda vacío. Un fallo al liberar solo se registra en el log; el error original se devuelve igual.

**Si `shouldHandleAsContingency` devuelve true:**

La reserva se mantiene y el caso de uso devuelve el modelo MH y el error de transmisión al handler sin cambios. El handler (no el caso de uso) es responsable de llamar a `ContingencyHandler.HandleContingency`. Esta separación significa que el caso de uso nunca sabe si el procesamiento de contingencia tuvo éxito.

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

Valores registrados en `HandlerContainer.initializeGenericCreatorHandler`:

| Endpoint | `UsesContingency` |
|---|---|
| `/dte/invoices` | `true` |
| `/dte/ccf` | `true` |
| `/dte/remissionnote` | `true` |
| `/dte/fse` | `true` |
| `/dte/debitnote` | `true` |
| `/dte/creditnote` | `false` |
| `/dte/retention` | `false` |

Los documentos con `UsesContingency: false` (nota de crédito y retención) siempre devuelven un error al cliente si la transmisión falla.

---

## InvalidationUseCase

**Archivo:** `internal/application/dte/invalidation_use_case.go`

### Flujo Completo

```go
func (u *InvalidationUseCase) InvalidateDocument(ctx context.Context, request structs.CreateInvalidationRequest) (*dte.EventArchiveResponse, error)
```

**Paso 1 — Extraer claims y token** (mismo patrón que `GenericDTEUseCase`)

**Paso 2 — Validar la estructura de la petición de invalidación**

```go
if err := u.mapper.ValidateInvalidationReRequest(&request); err != nil {
    return nil, err
}
```

El mapper valida: `reason` presente; `replacement_generation_code` obligatorio para los tipos 1 y 3 y prohibido para el tipo 2; `generation_code`, `type`, nombre/tipo/número de documento del responsable y del solicitante no vacíos; y `reason_field` obligatorio para el tipo 3.

**Paso 3 — Validar el estado del documento original**

```go
if err := u.invalidationManager.ValidateStatus(ctx, claims.BranchID, request); err != nil {
    return nil, err
}
```

El servicio de dominio verifica el estado del documento a invalidar y, si la petición no es de tipo 2 y trae `replacement_generation_code`, también el del documento de reemplazo. Los estados `INVALIDATED`, `REJECTED` y `PENDING` se rechazan con `DocumentAlreadyInvalid`, `DocumentReject` y `DocumentPending`; cualquier otro estado pasa esta validación.

**Paso 4 — Cargar el DTE original y validar dependencias**

```go
originalDTE, err := u.dteManager.GetByGenerationCode(ctx, claims.BranchID, request.GenerationCode)
err = u.invalidationManager.ValidateDependencies(ctx, claims.BranchID, originalDTE)
```

`ValidateDependencies` rechaza la invalidación de un CCF o un Comprobante de Retención que todavía tenga Notas de Crédito o Débito vigentes (`PendingAdjustmentNotes`, que lista los códigos de las notas). El servicio nunca invalida las notas de forma automática: el consumidor debe invalidar cada nota y volver a solicitar la invalidación del documento.

**Paso 5 — Cargar información del emisor**

```go
issuer, err := u.authManager.GetIssuer(ctx, claims.BranchID)
```

**Paso 6 — Mapear al modelo de dominio de invalidación**

```go
invalidationDocument, err := u.mapper.MapToInvalidationData(&request, issuer, originalDTE.Details, originalDTE.CreatedAt)
```

Esto produce un `invalidation_models.InvalidationDocument` con la identificación del evento (versión 3, nuevo código de generación en mayúsculas), el emisor, el motivo y los datos del documento invalidado. Exige que el documento original tenga sello de recepción (`DocumentWithoutStamp`). La fecha de generación se toma de `fecEmi`/`horEmi` del JSON del documento, y la fecha del sello de `processing_date` (con `CreatedAt` como respaldo). La fecha y hora del evento se calculan con `policy.EventDateTime`: la actual para FE, FEXE y FSEE; para el resto, la de generación del documento (o la actual si se generó hoy).

**Paso 7 — Validar el documento de invalidación completo**

```go
if err = u.invalidationManager.Validate(ctx, claims.BranchID, invalidationDocument); err != nil {
    return nil, err
}
```

El servicio de dominio aplica las reglas de negocio y los plazos (ver [invalidation.md](../domain/dte/invalidation.md)):
- FE, FEXE y FSEE: plazo de 3 meses desde el sello (2 años para FE y FEXE de emisores de actividad farmacéutica)
- Resto de DTE: plazo hasta el décimo día hábil del mes siguiente al sello

**Paso 8 — Mapear al formato Hacienda**

```go
mhInvalidation := response_mapper.ToMHInvalidation(invalidationDocument)
```

Si el resultado es nulo se devuelve `ErrorMapping` (`MH model`).

**Paso 9 — Transmitir a Hacienda**

```go
result, err := u.transmitter.RetryTransmission(ctx, mhInvalidation, token, claims.NIT)
if result.Status != ReceivedStatus { return nil, dte_errors.NewDTEErrorSimple("TransmissionFailed") }
```

La invalidación NO pasa por el flujo de contingencia. Si la transmisión falla, el error se devuelve directamente al cliente. El procesador de invalidación envía el documento a `NullifyURL`.

**Paso 10 — Actualizar el estado del documento original**

```go
if err := u.invalidationManager.InvalidateDocument(ctx, claims.BranchID, request.GenerationCode); err != nil {
    return nil, err
}
```

El manager de dominio actualiza el estado del DTE original a `INVALIDATED`. Si el documento invalidado es una nota de crédito o débito, además revierte su efecto en el balance de los documentos electrónicos relacionados (`GenerateBalanceTransactionWithAmounts`). El caso de uso de aplicación no necesita saber sobre esto.

**Paso 11 — Registrar el evento y responder**

```go
event, err := buildInvalidationEvent(mhInvalidation, request, claims.BranchID, result)
err = u.dteManager.RegisterEvent(ctx, event)
response := dte_documents.ToEventArchiveResponse(*event)
```

Normativa de Cumplimiento DTE v2.0 (regla 11): el evento de invalidación se entrega al receptor como Archivo DTE. Se registra un `DTEEvent` (tabla `dte_events`) de tipo `INVALIDATION`, con `RelatedDTEID` (documento invalidado), `ReplacementDTEID`, `JSONData`, el JWS (`SignedDocument`), `ReceptionStamp`, `ProcessingDate`, la respuesta de Hacienda y estado `RECEIVED`. La respuesta es un `EventArchiveResponse`: el evento, `hacienda_response` y `archivo_dte` (si el JWS es válido). Si el registro del evento falla, se devuelve el error aunque el documento ya quedó invalidado en Hacienda y en la base de datos.

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

El método limita la consulta al `claims.BranchID` para que una sucursal solo pueda acceder a sus propios documentos. `GetByGenerationCodeConsult` devuelve un `dte.DTEResponse` con código de generación, número de control, estado, tipo de transmisión, sello de recepción, el JSON del documento y fechas de creación y actualización.

### `GetArchive()`

```go
func (u *DTEConsultUseCase) GetArchive(ctx context.Context, id string) (json.RawMessage, error)
```

Delega en `DTEManager.GetArchive` con el `BranchID` del claim. Devuelve el Archivo DTE del documento: la estructura firmada tomada del JWS guardado, más `firmaElectronica` y `selloRecibido`. Si el documento no tiene JWS guardado (emitido antes de conservar la firma) se devuelve `ArchiveNotAvailable`; si el JWS no es válido, también.

### `GetEvents()`

```go
func (u *DTEConsultUseCase) GetEvents(ctx context.Context, id string) ([]dte.EventArchiveResponse, error)
```

Delega en `DTEManager.GetEvents`. Primero valida que el documento exista en la sucursal (un código inexistente responde no encontrado, no una lista vacía) y luego lista los eventos relacionados (`dte_events`). Cada elemento es un `EventArchiveResponse`: datos del evento, `hacienda_response` y `archivo_dte`.

### `GetAllDTEs()` con `parseDTEFilters()`

```go
func (u *DTEConsultUseCase) GetAllDTEs(ctx context.Context, r *http.Request) (*dte.DTEListResponse, error) {
    filters, err := parseDTEFilters(r)
    // ...
    return u.dteService.GetAllDTEs(ctx, filters)
}
```

`parseDTEFilters` lee los parámetros de query y puebla un struct `dte.DTEFilters`. Los valores inválidos en `status`, `transmission`, `type`, `startDate` y `endDate` devuelven un error `InvalidQueryParam`; `page` y `page_size` inválidos se reemplazan por sus valores por defecto.

**Detalles del parseo de filtros:**

| Parámetro de Query | Campo | Validación |
|---|---|---|
| `all=true` | `IncludeAll` | Si es true, `BranchID` no se establece (consulta entre todas las sucursales); en caso contrario se toma del claim |
| `status` | `Status` | Se convierte a mayúsculas y debe estar en `constants.ValidReceiverDocumentStates`: `RECEIVED`, `PENDING`, `REJECTED`, `INVALIDATED`, `EXPIRED` |
| `transmission` | `Transmission` | Se convierte a mayúsculas y debe estar en `constants.ValidTransmissionTypes`: `NORMAL`, `CONTINGENCY` |
| `type` | `DTEType` / `DTETypes` | Se permite lista separada por comas; cada valor se valida contra `constants.ValidDTETypes`: 01, 03, 04, 05, 06, 07, 08, 09, 11, 14, 15 |
| `startDate` | `StartDate` | Formato RFC3339 requerido |
| `endDate` | `EndDate` | Formato RFC3339 requerido |
| `page` | `Page` | Por defecto 1 si falta o es inválido |
| `page_size` | `PageSize` | Por defecto 5 si falta o es inválido |

El mensaje de `InvalidQueryParam` para `status` indica `'received', 'invalidated', 'rejected'` y para `type` indica `01-15`, aunque los valores aceptados son más (ver tabla).

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

`Validate()` verifica que `api_key`, `api_secret`, `credentials` y `credentials.username/password` no estén vacíos. Cualquier campo faltante devuelve un `ValidationError` `RequiredField`.

**Paso 2 — Delegar al AuthManager**

```go
token, err := a.authManager.Login(ctx, credentials)
```

El `AuthManager.Login` del dominio (implementado por `AuthService`) realiza lo siguiente:
1. Busca el tipo de autenticación por `api_key` (`NotFound` si no existe)
2. Selecciona la estrategia de autenticación (actualmente `StandardAuthStrategy`) y valida las credenciales
3. La estrategia busca la sucursal por `api_key` y compara `api_secret` con el valor almacenado (comparación en tiempo constante)
4. Verifica que el usuario esté activo (`UserNotActive`)
5. Emite un JWT firmado con `AuthClaims`, con vigencia de `token_lifetime` días del usuario
6. Guarda las credenciales de Hacienda cifradas en Redis (clave `hacienda:credentials:<JWT>`) con el mismo tiempo de vida del token

El login no consulta a Hacienda: las credenciales de Hacienda no se validan en este paso. Se usan después, cuando el primer documento necesita un token de Hacienda (`HaciendaAuthManager.GetOrCreateHaciendaToken`), que se guarda en caché 24 horas.

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

`Validate()` comprueba NIT, NRC, actividad económica, teléfono, correo, `auth_type`, `password_pri`, nombre comercial, razón social, descripción de actividad, `branch_offices` y `token_lifetime`, y finalmente `ValidateBranchOffices` (al menos una sucursal, exactamente una casa matriz y la matriz con dirección).

**Paso 2 — Generar API keys en masa**

```go
keys, secrets, err := a.cryptManager.GenerateBulkAPIKeys(len(user.BranchOffices))
```

`CryptManager.GenerateBulkAPIKeys` genera un par `(key, secret)` por sucursal. La key son 32 bytes aleatorios en hexadecimal; el secret son 48 bytes aleatorios en base64 URL-safe. Un fallo devuelve `FailedToCreateUser`.

**Paso 3 — Asignar keys a las sucursales**

```go
user.SetBranchesKeysAndSecrets(keys, secrets)
```

Esto muta el slice `user.BranchOffices`, emparejando cada sucursal con su propia API key y secret y marcándola activa.

**Paso 4 — Persistir en la base de datos**

```go
if err = a.authManager.Create(ctx, user); err != nil {
    return nil, ...
}
```

`AuthManager.Create` delega en el repositorio, que inserta usuario, sucursales y direcciones dentro de una única transacción de base de datos. Si cualquier inserción falla, toda la transacción se revierte. El error se envuelve en `FailedToCreateUser`.

**Paso 5 — Devolver listado de sucursales**

```go
return user.ListBranches(), nil
```

`ListBranches()` devuelve un `[]user.ListBranchesResponse`, cada uno con número de sucursal, tipo y código de establecimiento, API key y API secret. El `api_secret` se guarda sin hash en la columna `api_secret` de la sucursal.
