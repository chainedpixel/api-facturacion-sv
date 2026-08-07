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

Se crea un ServiceError con mensaje traducido automáticamente. Se usa el `code` como clave de traducción:

```go
err := shared_error.NewFormattedGeneralServiceError(
    "invoice",
    "Create",
    "invalid_receiver",     // Se traduce via config.TranslateServiceArgs()
)
// Message = "El receptor es inválido" (según i18n)
```

#### `NewFormattedGeneralServiceWithError`

```go
func NewFormattedGeneralServiceWithError(serviceType, op string, err error, code string, args ...interface{}) *ServiceError
```

Se crea un ServiceError con mensaje traducido **y** error subyacente. Se usa cuando se necesitan ambos:

```go
err := shared_error.NewFormattedGeneralServiceWithError(
    "auth",
    "Login",
    repositoryErr,          // Error subyacente
    "invalid_credentials",  // Código para traducción
)
```

### Método `Error()`

Se formatea el mensaje según el modo de la aplicación:

```
Modo normal (DEBUG=false):
  → "El receptor es inválido"

Modo debug (DEBUG=true):
  → "[ServiceError] Type: invoice | Operation: Create | Code: invalid_receiver | Message: El receptor es inválido | Cause: field 'NRC' is required"
```

### Método `GetErrError()`

```go
func (e *ServiceError) GetErrError() []string
```

Se extraen los detalles del error subyacente. Si el error contiene múltiples mensajes separados por `;`, se dividen:

```go
// Error subyacente: "campo1 inválido; campo2 faltante; campo3 fuera de rango"
details := err.GetErrError()
// → ["campo1 inválido", "campo2 faltante", "campo3 fuera de rango"]
```

Se limpian las comillas y se recortan espacios en cada detalle.

### Método `GetCode()`

```go
func (e *ServiceError) GetCode() string
```

Se extrae el código de error para clasificación:

1. Si el error subyacente es `ValidationError` → se retorna su código
2. Si `Code` no está vacío → se retorna `Code`
3. Fallback → se retorna `Type` en minúsculas

---

## Integración con ResponseWriter

```
ServiceError generado en dominio/aplicación
  │
  ▼
ResponseWriter.HandleError(rw, err)
  │
  ├── ¿Es ServiceError?
  │     ├── err.GetCode() → determina tipo de error
  │     ├── err.Error() → mensaje principal
  │     ├── err.GetErrError() → detalles adicionales
  │     └── → 400 Bad Request con detalles
  │
  └── ¿Es otro error?
        → 500 Internal Server Error
```

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
        ├── ServiceError → 400 + detalles
        ├── ValidationError → 400 + campos
        └── Otro → 500 genérico
```

---

## Notas

1. **Modo debug**: En producción (`DEBUG=false`), solo se muestra el mensaje traducido. En desarrollo (`DEBUG=true`), se incluye tipo, operación, código y causa raíz.
2. **Traducción automática**: Los constructores `NewFormatted*` traducen automáticamente el `code` usando el sistema i18n. El código debe existir en el archivo de traducciones con prefijo `service_errors.`.
3. **Errores centinela por capa**: Cada capa define sus propios errores centinela. Se comparan con `errors.Is()` para mantener el desacoplamiento.
4. **Semicolons como separador**: `GetErrError()` divide errores compuestos por `;`. Los servicios de validación usan este separador al concatenar múltiples errores.
5. **No envolver dos veces**: Si un error ya es `ServiceError`, no se debe envolver en otro `ServiceError`. Se propaga directamente.
