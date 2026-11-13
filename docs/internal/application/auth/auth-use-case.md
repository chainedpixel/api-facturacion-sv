# AuthUseCase — Caso de Uso de Autenticación

> **Paquete:** `internal/application/auth`
> **Archivo:** `auth_use_case.go`

## Descripción General

El `AuthUseCase` gestiona los flujos de autenticación y registro de usuarios en el sistema. Proporciona dos operaciones principales: `Login` y `Register`. Ambas se exponen como rutas públicas (`POST /api/v1/auth/login` y `POST /api/v1/auth/register`).

---

## Estructura

```go
type AuthUseCase struct {
    authManager  auth.AuthManager
    cryptManager ports.CryptManager
}
```

### Dependencias

| Campo | Tipo | Propósito |
|---|---|---|
| `authManager` | `AuthManager` | Se gestiona el flujo de autenticación y creación de usuarios |
| `cryptManager` | `CryptManager` | Se generan API keys y secrets para las sucursales |

---

## Login

```go
func (a *AuthUseCase) Login(
    ctx context.Context,
    credentials *models.AuthCredentials,
) (string, error)
```

### Input: AuthCredentials

```go
type AuthCredentials struct {
    MHCredentials *HaciendaCredentials `json:"credentials"`  // Credenciales de Hacienda
    APIKey        string               `json:"api_key"`      // API key de la sucursal
    APISecret     string               `json:"api_secret"`   // API secret de la sucursal
}

type HaciendaCredentials struct {
    Username string `json:"username"` // Usuario de Hacienda
    Password string `json:"password"` // Contraseña de Hacienda
}
```

### Flujo

```
AuthCredentials
  │
  ▼
[1] Validar credenciales
  │   credentials.Validate()
  │   ├── APIKey no vacío
  │   ├── APISecret no vacío
  │   ├── MHCredentials no nil
  │   ├── Username no vacío
  │   └── Password no vacío
  │
  ▼
[2] Autenticar
  │   authManager.Login(ctx, credentials)
  │
  │   Internamente (AuthService.Login):
  │   ├── Verificar que existan las credenciales (MissingCredentials)
  │   ├── Obtener el tipo de autenticación por API key (NotFound si no existe)
  │   ├── Seleccionar la estrategia de autenticación (StandardAuthStrategy)
  │   ├── Validar credenciales con la estrategia
  │   ├── Autenticar: buscar sucursal por API key y comparar API secret (tiempo constante)
  │   ├── Verificar que el usuario esté activo (UserNotActive)
  │   ├── Calcular la vigencia del token (token_lifetime del usuario, en días)
  │   ├── Generar el JWT del sistema con AuthClaims
  │   └── Guardar las credenciales de Hacienda cifradas en Redis
  │       (clave hacienda:credentials:<JWT>, con la misma vigencia del token)
  │
  ▼
[3] Retornar token JWT
      return tokenJWT, nil
```

El login no consulta a Hacienda: las credenciales de Hacienda no se validan en este paso. Se recuperan después, cuando el primer documento necesita un token de Hacienda (`HaciendaAuthManager.GetOrCreateHaciendaToken`), que se guarda en caché 24 horas.

### Output

Se retorna un **token JWT** como string que contiene los claims:

```go
type AuthClaims struct {
    ClientID  uint      `json:"sub"`           // ID del usuario
    BranchID  uint      `json:"branch_sub"`    // ID de la sucursal
    AuthType  string    `json:"auth_type"`     // Tipo de autenticación
    NIT       string    `json:"nit"`           // NIT del contribuyente
    ExpiresAt time.Time `json:"expires_at"`    // Fecha de expiración
}
```

### Errores

| Error | Causa |
|---|---|
| `ValidationError` (`RequiredField`) | Campos faltantes en credenciales |
| `ServiceError` `MissingCredentials` | Credenciales ausentes en `AuthService.Login` |
| `ServiceError` `NotFound` | API key inexistente |
| `ServiceError` `InvalidCredentials` | API secret incorrecto |
| `ServiceError` `UserNotActive` | Cuenta de usuario inactiva |
| `ServiceError` `ServerError` | Estrategia de autenticación no registrada o fallo al leer el usuario |
| Error de caché (`FailedToSetCache`) | No se pudieron guardar las credenciales de Hacienda en Redis |

Los errores de credenciales de Hacienda inválidas no se producen en el login: aparecen al transmitir el primer documento (`AuthHacienda`).

---

## Register

```go
func (a *AuthUseCase) Register(
    ctx context.Context,
    user *user.User,
) ([]user.ListBranchesResponse, error)
```

### Flujo

```
User + BranchOffices
  │
  ▼
[1] Validar usuario
  │   user.Validate()
  │   ├── NIT, NRC, actividad económica, teléfono y correo válidos
  │   ├── auth_type, password_pri, nombre comercial, razón social y
  │   │   descripción de actividad no vacíos
  │   ├── branch_offices presente y token_lifetime >= 0
  │   └── ValidateBranchOffices(): al menos una sucursal, exactamente una
  │       sucursal matriz y la matriz con dirección
  │
  ▼
[2] Generar API keys y secrets
  │   cryptManager.GenerateBulkAPIKeys(len(user.BranchOffices))
  │   → (keys []string, secrets []string, error)
  │
  ▼
[3] Asignar keys a sucursales
  │   user.SetBranchesKeysAndSecrets(keys, secrets)
  │   → Cada sucursal recibe su par APIKey/APISecret único y queda activa
  │
  ▼
[4] Crear usuario en BD
  │   authManager.Create(ctx, user)
  │   → Persiste usuario + sucursales + direcciones (transaccional)
  │
  ▼
[5] Retornar lista de sucursales con keys
      user.ListBranches()
      → []ListBranchesResponse
```

### Generación de keys

| Valor | Generación |
|---|---|
| API key | 32 bytes aleatorios (`crypto/rand`) en hexadecimal |
| API secret | 48 bytes aleatorios (`crypto/rand`) en base64 URL-safe |

### Output: ListBranchesResponse

```go
type ListBranchesResponse struct {
    BranchNumber      int     `json:"branch_number"`
    EstablishmentType string  `json:"establishment_type"`
    EstablishmentCode *string `json:"establishment_code,omitempty"`
    APIKey            string  `json:"api_key"`
    APISecret         string  `json:"api_secret"`
}
```

Se retornan las sucursales creadas con sus API keys y secrets. El `api_secret` se guarda sin hash en la columna `api_secret` de la sucursal (el login lo compara directamente), por lo que no es una entrega de una sola vez desde el punto de vista del almacenamiento.

### Errores

| Error | Causa |
|---|---|
| `ValidationError` | Estructura de usuario inválida |
| `ErrAtLeastOneBranch` | Sin sucursales |
| `ErrDontHaveBranchMatrix` | Sin sucursal matriz |
| `ErrMoreThanOneBranchMatrix` | Múltiples matrices |
| `ErrBranchMatrixWithoutAddress` | Matriz sin dirección |
| `FailedToCreateUser` | Error al generar keys o al persistir (en este último caso envuelve el error original) |

---

## Diagrama del Flujo de Autenticación

```
Cliente HTTP
  │
  ├── POST /api/v1/auth/login
  │     Body: { api_key, api_secret, credentials: { username, password } }
  │     │
  │     ▼
  │   AuthUseCase.Login()
  │     │
  │     ├── Validar credenciales
  │     ├── Buscar sucursal por api_key
  │     ├── Verificar api_secret
  │     ├── Generar JWT del sistema
  │     ├── Guardar credenciales de Hacienda cifradas en Redis
  │     └── Retornar JWT
  │
  └── POST /api/v1/auth/register
        Body: { nit, nrc, auth_type, password_pri, commercial_name, business_name,
                economic_activity, economic_activity_desc, email, phone,
                token_lifetime, branch_offices: [...] }
        │
        ▼
      AuthUseCase.Register()
        │
        ├── Validar usuario y sucursales
        ├── Generar N pares api_key/api_secret
        ├── Asignar a cada sucursal
        ├── Persistir en BD
        └── Retornar sucursales con keys (HTTP 201)
```

---

## Notas

1. **Secret sin hash**: Las API keys y secrets se almacenan tal como se generan y se devuelven en el registro. El secret se compara en tiempo constante, pero no se almacena hasheado.
2. **Credenciales de Hacienda**: Se guardan cifradas (cifrado simétrico con clave derivada del JWT) en Redis con la vigencia del token; el token de Hacienda se obtiene y se cachea 24 horas en la primera transmisión.
3. **Estrategia de autenticación**: El sistema soporta diferentes tipos de autenticación a través del patrón Strategy (`AuthStrategy`); actualmente solo existe `StandardAuthStrategy`.
4. **Transaccional**: La creación de usuario y sucursales es transaccional — si falla alguna parte, se revierte todo.
5. **Rutas públicas**: Ni `login` ni `register` pasan por el middleware de autenticación (`RegisterPublicAuthRoutes`).
