# Sistema de Autenticación — Dominio

> **Paquetes:**
> - `internal/domain/auth` — Interfaces y puerto de repositorio
> - `internal/domain/auth/models` — Modelos de credenciales y claims
> - `internal/domain/auth/constants` — Tipos de autenticación
> - `internal/domain/auth/service/strategies` — Implementación del servicio y estrategias

## Descripción General

El sistema de autenticación del dominio define las reglas y contratos para identificar a un sistema emisor y emitir un JWT de sesión. Se basa en el patrón **Strategy** para soportar múltiples mecanismos de autenticación de forma extensible.

Actualmente existe una única estrategia implementada: `STANDARD`, que valida las credenciales API key + API secret de una sucursal.

---

## Modelos

### `AuthCredentials`

> **Archivo:** `auth/models/auth_model.go`

Representa las credenciales que un sistema emisor envía para autenticarse.

```go
type AuthCredentials struct {
    MHCredentials *HaciendaCredentials `json:"credentials"`
    APIKey        string               `json:"api_key"`
    APISecret     string               `json:"api_secret"`
}
```

| Campo | Descripción |
|---|---|
| `APIKey` | Identificador público de la sucursal. Requerido. |
| `APISecret` | Clave secreta de la sucursal (comparación en tiempo constante). Requerido. |
| `MHCredentials` | Credenciales del usuario en Hacienda (se cachean en Redis tras el login). Requerido. |

El método `Validate()` verifica que los tres campos y sus subcampos no estén vacíos.

---

### `HaciendaCredentials`

```go
type HaciendaCredentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
}
```

Credenciales del usuario ante la API de Hacienda. Se almacenan cifradas en Redis con el JWT como clave, con TTL igual al lifetime del token.

---

### `AuthClaims`

```go
type AuthClaims struct {
    ClientID  uint      `json:"sub"`
    BranchID  uint      `json:"branch_sub"`
    AuthType  string    `json:"auth_type"`
    NIT       string    `json:"nit"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

Contenido del JWT generado tras una autenticación exitosa. Se inyectan en el contexto de cada request por el middleware de autenticación.

| Campo | Uso en el sistema |
|---|---|
| `ClientID` | Identifica al usuario propietario de la sucursal |
| `BranchID` | Identifica la sucursal que emite los DTEs |
| `NIT` | NIT del emisor; se usa en los mappers de DTE y en la generación de secuencias |
| `AuthType` | Tipo de autenticación; determina la estrategia usada en el login |

---

## Constantes

> **Archivo:** `auth/constants/user_auth_type.go`

```go
var StandardAuthType = "STANDARD"
```

Identificador del único tipo de autenticación activo. Se almacena en la base de datos por usuario y se usa para seleccionar la estrategia en tiempo de ejecución.

---

## Interfaces del Dominio

> **Archivo:** `auth/auth_repository_port.go`

### `AuthRepositoryPort`

Puerto que define todas las operaciones de lectura y escritura sobre usuarios y sucursales.

| Método | Descripción |
|---|---|
| `GetAuthTypeByApiKey(ctx, apiKey)` | Obtiene el tipo de auth del usuario que posee ese API key |
| `GetAuthTypeByNIT(ctx, nit)` | Obtiene el tipo de auth por NIT |
| `GetByNIT(ctx, nit)` | Busca un usuario completo por NIT |
| `GetByBranchApiKey(ctx, apiKey)` | Obtiene el usuario al que pertenece una sucursal por API key |
| `GetBranchByBranchApiKey(ctx, apiKey)` | Obtiene solo la sucursal por API key |
| `GetBranchByBranchID(ctx, branchID)` | Obtiene una sucursal por ID |
| `GetByBranchID(ctx, branchID)` | Obtiene el usuario propietario de una sucursal por ID |
| `GetIssuerInfoByBranchID(ctx, branchID)` | Obtiene los datos del emisor DTE para una sucursal |
| `GetMatrixBranch(ctx, userID)` | Obtiene la casa matriz del usuario |
| `Create(ctx, user)` | Crea un usuario con todas sus sucursales |
| `Update(ctx, user)` | Actualiza la información del usuario |
| `UpdateBranchOffices(ctx, userID, branches)` | Actualiza las sucursales del usuario |
| `DeleteBranchOffice(ctx, userID, branchID)` | Elimina una sucursal |

### `AuthStrategy`

Interface que cada estrategia de autenticación debe implementar.

```go
type AuthStrategy interface {
    GetAuthType() string
    Authenticate(ctx, credentials) (*AuthClaims, error)
    ValidateCredentials(credentials) error
    GetHaciendaCredentials(token) (*HaciendaCredentials, error)
    GetTokenLifetime(credentials) (time.Duration, error)
}
```

### `AuthManager`

Interface del servicio de autenticación que consume la capa de aplicación.

```go
type AuthManager interface {
    Login(ctx, credentials) (string, error)
    GetByNIT(ctx, nit) (*User, error)
    GetBranchByBranchID(ctx, branchID) (*BranchOffice, error)
    GetIssuer(ctx, branchID) (*IssuerDTE, error)
    GetHaciendaCredentials(ctx, nit, token) (*HaciendaCredentials, error)
    Create(ctx, user) error
}
```

---

## AuthService

> **Archivo:** `auth/service/strategies/auth_service.go`
> **Implementa:** `auth.AuthManager`

### Estructura

```go
type AuthService struct {
    strategies   map[string]AuthStrategy
    authRepo     AuthRepositoryPort
    tokenService ports.TokenManager
    cacheService ports.CacheManager
}
```

El mapa `strategies` se inicializa con una entrada: `"STANDARD" → StandardAuthStrategy`. Agregar un nuevo tipo de autenticación requiere únicamente registrar una nueva estrategia en este mapa.

### Flujo de Login

```
Login(ctx, credentials)
  │
  ├── [1] Verificar que las credenciales no están vacías
  │     credentialsExists(credentials)
  │     └── Vacíos → error "MissingCredentials"
  │
  ├── [2] Obtener el tipo de autenticación del usuario
  │     authRepo.GetAuthTypeByApiKey(ctx, credentials.APIKey)
  │     └── No encontrado → error "NotFound"
  │
  ├── [3] Seleccionar la estrategia correspondiente
  │     strategies[authType]
  │     └── No existe → error "ServerError" (tipo no soportado)
  │
  ├── [4] Validar credenciales con la estrategia
  │     strategy.ValidateCredentials(credentials)
  │
  ├── [5] Autenticar y obtener claims
  │     strategy.Authenticate(ctx, credentials)
  │     → AuthClaims { ClientID, BranchID, AuthType, NIT }
  │
  ├── [6] Obtener lifetime del token
  │     strategy.GetTokenLifetime(credentials)
  │     → user.TokenLifetime * 24h
  │
  ├── [7] Generar JWT
  │     tokenService.GenerateToken(claims, lifetime)
  │
  └── [8] Cachear credenciales de Hacienda
        cacheService.SetCredentials(token, MHCredentials, lifetime)
        → Almacenado en Redis, cifrado, con TTL = lifetime del token
```

### Otros Métodos

| Método | Descripción |
|---|---|
| `GetHaciendaCredentials(ctx, nit, token)` | Resuelve el tipo de auth por NIT y delega a la estrategia para obtener las creds de Hacienda del cache |
| `GetIssuer(ctx, branchID)` | Delega a `authRepo.GetIssuerInfoByBranchID` — usado por los mappers de DTE |
| `ValidateToken(token)` | Valida un JWT existente delegando en `tokenService` |
| `RevokeToken(token)` | Revoca un token delegando en `tokenService` |
| `Create(ctx, user)` | Crea un usuario; maneja errores de duplicidad (NIT, email, phone, NRC) |

### Manejo de Errores de Escritura

`handleGormError` normaliza los errores de GORM en errores de dominio:

| Condición | Error resultante |
|---|---|
| `gorm.ErrInvalidData` | "InvalidData" |
| Entrada duplicada en `nit` | "DuplicatedEntry: nit" |
| Entrada duplicada en `email` | "DuplicatedEntry: email" |
| Entrada duplicada en `phone` | "DuplicatedEntry: phone" |
| Entrada duplicada en `nrc` | "DuplicatedEntry: nrc" |

---

## StandardAuthStrategy

> **Archivo:** `auth/service/strategies/standard_auth.go`
> **Implementa:** `auth.AuthStrategy`

### Flujo de Autenticación

```
Authenticate(ctx, credentials)
  │
  ├── [1] Obtener la sucursal por API key
  │     authRepo.GetBranchByBranchApiKey(ctx, credentials.APIKey)
  │     └── No encontrada → error "NotFound"
  │
  ├── [2] Comparar API secret (tiempo constante)
  │     subtle.ConstantTimeCompare(credentials.APISecret, branch.APISecret)
  │     └── No coincide → error "InvalidCredentials"
  │
  ├── [3] Obtener el usuario propietario
  │     authRepo.GetByBranchApiKey(ctx, credentials.APIKey)
  │
  ├── [4] Verificar estado activo del usuario
  │     user.Status == true
  │     └── Inactivo → error "UserNotActive"
  │
  └── [5] Construir y retornar AuthClaims
        { ClientID: user.ID, BranchID: branch.ID, AuthType: user.AuthType, NIT: user.NIT }
```

La comparación del secret usa `crypto/subtle.ConstantTimeCompare` para prevenir ataques de timing.

`GetTokenLifetime` lee `user.TokenLifetime` (campo en días) y lo convierte a `time.Duration`: `days * 24 * time.Hour`.

---

## Notas

1. **Extensibilidad**: Para agregar un nuevo tipo de auth, se implementa `AuthStrategy` y se registra en el mapa `strategies` de `NewAuthService`. No se requiere modificar la lógica del servicio.
2. **Cache de credenciales MH**: Las credenciales de Hacienda se cachean con el JWT del sistema como clave. El TTL coincide con el lifetime del token para que expiren juntos.
3. **`subtle.ConstantTimeCompare`**: Obligatorio para comparar secrets. Las comparaciones de strings estándar son vulnerables a ataques de timing side-channel.
4. **`IssuerDTE` en el contexto DTE**: `GetIssuerInfoByBranchID` es el puente entre la capa de auth y los mappers de DTE. Devuelve todos los datos del emisor necesarios para construir el JSON de Hacienda.
