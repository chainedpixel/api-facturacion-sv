# Sistema de Errores — ServiceError y Errores Centinela

> **Paquetes:**
> - `pkg/shared/shared_error` — ServiceError (wrapper principal)
> - `pkg/error` — Errores centinela de logging
> - `internal/domain/core/error` — Errores centinela de dominio
> - `internal/infrastructure/error` — Errores centinela de infraestructura
> - `config/error` — Errores centinela de configuración

## Descripción General

El sistema de errores se organiza en dos componentes:

1. **ServiceError** — Wrapper de errores con contexto, traducción y soporte de modo debug
2. **Errores centinela** — Constantes de error predefinidas distribuidas por capa

---

## ServiceError

> **Archivo:** `pkg/shared/shared_error/service_error.go`

### Estructura

```go
type ServiceError struct {
    Type      string    // Tipo de servicio/componente (ej. "invoice", "auth")
    Operation string    // Operación que falló (ej. "Create", "Validate")
    Message   string    // Mensaje legible para el usuario
    Code      string    // Código de error para clasificación
    Err       error     // Error subyacente (causa raíz)
}
```

### Constructores

#### `NewGeneralServiceError`

```go
func NewGeneralServiceError(serviceType, op, msg string, err error) *ServiceError
```

Se crea un ServiceError con mensaje raw y error subyacente. Se usa cuando el mensaje ya está traducido o es técnico.

```go
err := shared_error.NewGeneralServiceError(
    "transmitter",
    "SendToHacienda",
    "Failed to send DTE",
    originalErr,
)
```

#### `NewFormattedGeneralServiceError`

```go
func NewFormattedGeneralServiceError(serviceType, op, code string, args ...interface{}) *ServiceError
```

Se crea un ServiceError con mensaje traducido automáticamente. El `code` se convierte a minúsculas y se usa como clave bajo el prefijo `service_errors.` en `assets/i18n/es.yaml`; `args` se aplican como parámetros de la plantilla. El `code` también se guarda en `ServiceError.Code`:

```go
err := shared_error.NewFormattedGeneralServiceError(
    "CommonMapper",
    "MapCommonRequestAddress",
    "AddressWithReceiver",  // clave service_errors.addresswithreceiver
)
```

#### `NewFormattedGeneralServiceWithError`

```go
func NewFormattedGeneralServiceWithError(serviceType, op string, err error, code string, args ...interface{}) *ServiceError
```

Se crea un ServiceError con mensaje traducido **y** error subyacente. Es el constructor usado por los request mappers para envolver errores de cada componente:

```go
err := shared_error.NewFormattedGeneralServiceWithError(
    "InvoiceMapper",
    "MapToInvoiceData",
    err,                     // Error subyacente
    "ErrorMapping",          // Código para traducción
    "Invoice->Items",        // Parámetro de la plantilla
)
```

Si la clave no existe en el idioma activo se busca en inglés; si tampoco existe, el mensaje es `[clave] <mensaje de error desconocido>` (ver `config.Translate`).

### Método `Error()`

El formato depende de `config.Server.Debug`:

```
Sin causa (Err == nil):
  DEBUG=false → "<Message>"
  DEBUG=true  → "[<Code>] <Message>"

Con causa:
  DEBUG=false → "<Message> -> <Err>"
  DEBUG=true  → "[<Type>] <Operation>: <Message> | cause: <Err>"
```

### Método `GetErrError()`

```go
func (e *ServiceError) GetErrError() []string
```

Se extraen los detalles del error subyacente (`Err`). El texto de `Err.Error()` se divide por `;`, se recorta cada fragmento y se descartan los vacíos:

```go
// Error subyacente: "campo1 inválido; campo2 faltante; campo3 fuera de rango"
details := err.GetErrError()
// → ["campo1 inválido", "campo2 faltante", "campo3 fuera de rango"]
```

Si `Err` es `nil` se retorna un único elemento con la traducción de `service_errors.NoDetailsAvailable` ("No hay más detalles disponibles").

### Método `GetCode()`

```go
func (e *ServiceError) GetCode() string
```

Se extrae el código de error para clasificación:

1. Si el error subyacente es (o contiene, con `errors.As`) un `ValidationError` → se retorna su tipo (`GetType()`, por ejemplo `RequiredField`)
2. Si `Code` no está vacío → se retorna `Code`
3. Fallback → se retorna `Type` en minúsculas

---

## Integración con ResponseWriter

> **Archivo:** `internal/infrastructure/api/response/writer.go`

`ResponseWriter.HandleError` clasifica el error por tipo concreto (`getErrorType`) y construye el cuerpo `APIResponse` con `success: false` y un objeto `error` con `message`, `details` y `code`:

| Tipo de error | Estado HTTP | `error.message` | `error.details` | `error.code` |
|---|---|---|---|---|
| `DTEError` | 400 | `GetMessage()` | `GetValidationErrorsString()` | `GetCode()` |
| `ValidationError` | 400 | `Error()` | `["No hay más detalles disponibles"]` | tipo en mayúsculas (`GetType()`) |
| `HaciendaResponseError` | 400 | descripción de Hacienda | `State: <estado>`, `Processed at: <fecha>` y las observaciones | `HACIENDA_<código>` |
| `ServiceError` | 400 | `ServiceError.Message` | `GetErrError()` | `GetCode()` en mayúsculas |
| `HTTPResponseError` | código HTTP del error | `Error()` | traducción de `service_errors.ContingencyActiveTransmission` | `HACIENDA_<código HTTP>` |
| Otro error de negocio no clasificado | 400 | `err.Error()` | (sin detalles) | `BUSINESS_ERROR` |
| Cualquier otro error | 500 | traducción de `validation_errors.ServerError` | (sin detalles) | `SYSTEM_ERROR` |

Los errores `DTEError` y `ValidationError` se clasifican como validación; `ServiceError`, `HaciendaResponseError` y `HTTPResponseError` como negocio; el resto como sistema. Como `getErrorType` compara por tipo concreto, un `ServiceError` que envuelve un `ValidationError` se responde como `ServiceError`: el `message` es el texto traducido del `ServiceError`, el `code` es el tipo del `ValidationError` interno en mayúsculas (por ejemplo `REQUIREDFIELD`) y los `details` provienen del texto del error interno.

`ResponseWriter.Error(rw, status, message, details)` genera respuestas con la forma `{"error": {message, details, code}}` (sin `success`) y deriva `code` del estado HTTP (`BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `INTERNAL_SERVER_ERROR`, `REQUEST_TIMEOUT`; cualquier otro, `UNKNOWN_ERROR`).

---

## Errores Centinela por Capa

### Capa pkg (logging)

> **Archivo:** `pkg/error/sentinels_error.go`

```go
ErrFailedToCreateLogFiles  // No se pudieron crear archivos de log
ErrLogDirectoryNotFound    // Directorio de logs no encontrado
```

### Capa de Dominio (core)

> **Archivo:** `internal/domain/core/error/sentinels_error.go`

```go
ErrBranchMatrixNotFound        // Casa matriz no encontrada
ErrAtLeastOneBranch            // Se requiere al menos una sucursal
ErrDontHaveBranchMatrix        // Usuario sin casa matriz
ErrMoreThanOneBranchMatrix     // Más de una casa matriz
ErrBranchMatrixWithoutAddress  // Casa matriz sin dirección
```

### Capa de Infraestructura

> **Archivo:** `internal/infrastructure/error/sentinels_error.go`

```go
ErrTokenNotFound               // Token no encontrado en cache
ErrBranchOfficeNotFound        // Sucursal no encontrada o inactiva
ErrUserNotFound                // Usuario no encontrado o inactivo
ErrBranchDoesNotBelong         // Sucursal no pertenece al usuario
ErrExpiredToken                // Token expirado
ErrDTEDocumentNotFound         // Documento DTE no encontrado
ErrHaciendaTokenGeneration     // Fallo al generar token de Hacienda
ErrInvalidDocumentJSON         // JSON del documento inválido
```

### Capa de Configuración

> **Archivo:** `config/error/setinels_err.go`

```go
ErrFailedToConnectDb           // Fallo al conectar a la BD
ErrFailedToCloseDbConnection   // Fallo al cerrar conexión
ErrFailedToGetDBInstance       // Fallo al obtener instancia de BD
ErrEnvFileNotFound             // Archivo .env no encontrado
ErrFailedToLoadEnv             // Fallo al cargar .env
ErrUnrecognizedDriver          // Driver de BD no reconocido
```

---

## Diagrama de Flujo de Errores

```
Servicio de Dominio
  │
  ├── Validación falla → DTEError / ValidationError
  │     → Se envuelve en ServiceError (NewFormattedGeneralServiceWithError)
  │
  ├── Regla de negocio falla → error específico
  │     → Se crea ServiceError (NewFormattedGeneralServiceError)
  │
  └── Error de infraestructura → error centinela
        → Se propaga al caso de uso → ServiceError

Caso de Uso (Aplicación)
  │
  ├── Error de dominio → Se propaga como ServiceError
  ├── Error de transmisión → Se crea nuevo ServiceError
  └── Error de contingencia → Se maneja internamente o ServiceError

Handler HTTP
  │
  └── ResponseWriter.HandleError(err)
        ├── DTEError / ValidationError → 400 + detalles de validación
        ├── ServiceError / HaciendaResponseError → 400 + detalles
        ├── HTTPResponseError → código HTTP del error
        └── Otro → 500 genérico (SYSTEM_ERROR)
```

---

## Notas

1. **Modo debug**: En producción (`DEBUG=false`), `Error()` muestra el mensaje traducido (y la causa, si existe). En desarrollo (`DEBUG=true`), se incluye tipo, operación y causa raíz (o código, si no hay causa). El cuerpo HTTP usa `ServiceError.Message` y `GetErrError()`, no `Error()`.
2. **Traducción automática**: Los constructores `NewFormatted*` traducen automáticamente el `code` usando el sistema i18n. El código debe existir en `assets/i18n/es.yaml` con prefijo `service_errors.` (la clave se busca en minúsculas).
3. **Errores centinela por capa**: Cada capa define sus propios errores centinela. Se comparan con `errors.Is()` para mantener el desacoplamiento.
4. **Semicolons como separador**: `GetErrError()` divide errores compuestos por `;`. Los servicios de validación usan este separador al concatenar múltiples errores.
5. **No envolver dos veces**: Si un error ya es `ServiceError`, no se debe envolver en otro `ServiceError`. Se propaga directamente.
