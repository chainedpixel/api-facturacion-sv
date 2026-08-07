# AuthUseCase — Caso de Uso de Autenticación

> **Paquete:** `internal/application/auth`
> **Archivo:** `auth_use_case.go`

## Descripción General

El `AuthUseCase` gestiona los flujos de autenticación y registro de usuarios en el sistema. Proporciona dos operaciones principales: `Login` y `Register`.

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
    MHCredentials *HaciendaCredentials  // Credenciales de Hacienda
    APIKey        string                 // API key de la sucursal
    APISecret     string                 // API secret de la sucursal
}

type HaciendaCredentials struct {
    Username string  // Usuario de Hacienda
    Password string  // Contraseña de Hacienda
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
  │   Internamente:
  │   ├── Buscar sucursal por API key
  │   ├── Verificar API secret
  │   ├── Obtener tipo de autenticación
  │   ├── Ejecutar estrategia de autenticación
  │   ├── Autenticar con Hacienda (obtener token MH)
  │   ├── Generar JWT del sistema
  │   └── Cachear token de Hacienda en Redis (24h)
  │
  ▼
[3] Retornar token JWT
      return tokenJWT, nil
```

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
| `ValidationError` | Campos faltantes en credenciales |
| `AuthenticationError` | API key/secret inválidos |
| `HaciendaAuthError` | Credenciales de Hacienda rechazadas |

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
  │   ├── Al menos una sucursal
  │   ├── Exactamente una sucursal matriz
  │   └── Sucursal matriz con dirección
  │
  ▼
[2] Generar API keys y secrets
  │   cryptManager.GenerateBulkAPIKeys(len(user.BranchOffices))
  │   → (keys []string, secrets []string, error)
  │
  ▼
[3] Asignar keys a sucursales
  │   user.SetBranchesKeysAndSecrets(keys, secrets)
  │   → Cada sucursal recibe su par APIKey/APISecret único
  │
  ▼
[4] Crear usuario en BD
  │   authManager.Create(ctx, user)
  │   → Persiste usuario + sucursales (transaccional)
  │
  ▼
[5] Retornar lista de sucursales con keys
      user.ListBranches()
      → []ListBranchesResponse
```

### Output: ListBranchesResponse

Se retornan las sucursales creadas con sus API keys en **texto plano**. Esta es la **única vez** que las keys se muestran — después se almacenan hasheadas.

### Errores

| Error | Causa |
|---|---|
| `ValidationError` | Estructura de usuario inválida |
| `ErrAtLeastOneBranch` | Sin sucursales |
| `ErrDontHaveBranchMatrix` | Sin sucursal matriz |
| `ErrMoreThanOneBranchMatrix` | Múltiples matrices |
| `ErrBranchMatrixWithoutAddress` | Matriz sin dirección |
| `FailedToCreateUser` | Error al generar keys o al persistir |

---

## Diagrama del Flujo de Autenticación

```
Cliente HTTP
  │
  ├── POST /auth/login
  │     Body: { apiKey, apiSecret, mhCredentials: { user, pass } }
  │     │
  │     ▼
  │   AuthUseCase.Login()
  │     │
  │     ├── Validar credenciales
  │     ├── Buscar sucursal por APIKey
  │     ├── Verificar APISecret
  │     ├── Autenticar con Hacienda API
  │     ├── Generar JWT del sistema
  │     ├── Cachear token MH en Redis
  │     └── Retornar JWT
  │
  └── POST /auth/register
        Body: { nit, nrc, name, branchOffices: [...] }
        │
        ▼
      AuthUseCase.Register()
        │
        ├── Validar usuario y sucursales
        ├── Generar N pares APIKey/APISecret
        ├── Asignar a cada sucursal
        ├── Persistir en BD
        └── Retornar sucursales con keys (texto plano, única vez)
```

---

## Notas

1. **Keys de una sola vez**: Las API keys y secrets se muestran al usuario solo en el momento del registro. Después se almacenan cifradas/hasheadas.
2. **Token de Hacienda cacheado**: El token de Hacienda se cachea en Redis por 24 horas para evitar autenticaciones repetidas.
3. **Estrategia de autenticación**: El sistema soporta diferentes tipos de autenticación a través del patrón Strategy (`AuthStrategy`).
4. **Transaccional**: La creación de usuario y sucursales es transaccional — si falla alguna parte, se revierte todo.
