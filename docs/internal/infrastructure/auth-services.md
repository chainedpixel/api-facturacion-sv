# Servicios de Autenticación — JWT, Cache y Criptografía

> **Paquetes:**
> - `internal/infrastructure/adapters/tokens`
> - `internal/infrastructure/adapters/cache`
> - `internal/infrastructure/adapters/crypt`

## Descripción General

Los servicios de autenticación en la capa de infraestructura gestionan tres responsabilidades:

1. **JWT** — Generación, validación y revocación de tokens del sistema
2. **Cache (Redis)** — Almacenamiento temporal de tokens, credenciales y métricas
3. **Criptografía** — Generación de API keys/secrets y cifrado de credenciales

---

## JWTService

> **Archivo:** `adapters/tokens/jwt_service.go`
> **Implementa:** `ports.TokenManager`

### Estructura

```go
type JWTService struct {
    SecretKey    string
    cacheService ports.CacheManager
}
```

### Métodos

| Método | Descripción |
|---|---|
| `GenerateToken(claims, tokenLifetime)` | Se genera un JWT firmado con HS256 |
| `ValidateToken(tokenString)` | Se recuperan los claims del cache y además se verifica la firma y vigencia del JWT |
| `RevokeToken(token)` | Se invalida un token eliminándolo del cache |
| `SaveTimestampsForContingency(issuedAt, expiresAt, tokenLifetime, claims)` | Se cachean las marcas de emisión y expiración del usuario, que contingencia reutiliza para regenerar un token equivalente |
| `GetSecretKey()` | Se retorna la clave de firma |

### Claims del Token

El JWT se firma con `jwt.MapClaims` con estas claves:

| Claim | Origen |
|---|---|
| `sub` | `ClientID` |
| `branch_sub` | `BranchID` |
| `auth_type` | `AuthType` |
| `nit` | `NIT` |
| `exp` | emisión + `tokenLifetime` (Unix) |
| `iat` | momento de emisión (Unix) |

Los `AuthClaims` (`ClientID`, `BranchID`, `AuthType`, `NIT`) se serializan a JSON y se guardan en cache; el middleware de autenticación los inyecta en el contexto de la petición.

### Flujo de Generación

```
GenerateToken(claims, lifetime)
  │
  ├── 1. Crear JWT (HS256) con sub, branch_sub, auth_type, nit, exp, iat
  │
  ├── 2. Firmar con SecretKey → tokenString
  │
  ├── 3. SaveTimestampsForContingency
  │      Key: token:timestamps:{ClientID}
  │      Valor: { IssuedAt, ExpiresAt } (Unix)
  │      TTL: tokenLifetime
  │
  ├── 4. Cachear claims en Redis
  │      Key: token:{tokenString}
  │      TTL: tokenLifetime
  │
  └── 5. Retornar tokenString
```

### Flujo de Validación

```
ValidateToken(tokenString)
  │
  ├── 1. Buscar claims en cache (Redis)
  │      Key: token:{tokenString}
  │      └── No existe → error (FailedToLogin)
  │
  ├── 2. Deserializar AuthClaims
  │      └── Error → AuthServiceUnavailable
  │
  ├── 3. jwt.Parse con SecretKey (firma y expiración)
  │      └── Error → AuthServiceUnavailable
  │
  └── 4. token.Valid → retornar AuthClaims (si no, Unauthorized)
```

> **Nota:** La validación exige las dos condiciones: que el token exista en cache (lo que permite la revocación inmediata) y que su firma y expiración sean válidas.

> **Observación del código:** `GenerateToken` imprime el token firmado en la salida estándar con `fmt.Printf`, y `ValidateToken` y `RevokeToken` registran el token completo en los logs.

---

## RedisTokenCache

> **Archivo:** `adapters/cache/redis_token_cache.go`
> **Implementa:** `ports.CacheManager`

### Estructura

```go
type RedisTokenCache struct {
    client       *redis.Client
    cryptService ports.CryptManager
    ctx          context.Context
}
```

### Métodos — Operaciones Básicas

| Método | Descripción |
|---|---|
| `Set(key, value, ttl)` | Se almacena un valor con tiempo de expiración |
| `Get(key)` | Se recupera un valor por clave |
| `Delete(token)` | Se elimina la entrada `token:{token}` |
| `Expire(key, ttl)` | Se asigna expiración a una clave |
| `ScanKeys(pattern)` | Se listan las claves que coinciden con un patrón (iteración por cursor) |
| `Close()` | Se cierra la conexión Redis |

### Métodos — Credenciales Cifradas

| Método | Descripción |
|---|---|
| `SetCredentials(token, creds, ttl)` | Se cifran y almacenan credenciales de Hacienda en `hacienda:credentials:{token}` |
| `GetCredentials(token)` | Se descifran y retornan credenciales de `hacienda:credentials:{token}` |

Otras claves que se escriben vía `Set`: `token:{jwt}`, `token:timestamps:{clientID}`, `hacienda:token:{jwt}` (token de Hacienda, 24 h) y `metrics:...` (ver [endpoint-metrics.md](endpoint-metrics.md)).

### Flujo de Cifrado de Credenciales

```
SetCredentials(token, creds, ttl)
  │
  ├── 1. Serializar HaciendaCredentials a JSON
  ├── 2. Derivar clave AES del token (SHA256)
  ├── 3. Cifrar JSON con cryptopasta (AES-GCM)
  ├── 4. Codificar en base64 (CryptService)
  └── 5. Almacenar en Redis con TTL

GetCredentials(token)
  │
  ├── 1. Recuperar valor cifrado de Redis
  ├── 2. Decodificar base64
  ├── 3. Derivar clave AES del token (SHA256)
  ├── 4. Descifrar con cryptopasta
  └── 5. Deserializar a HaciendaCredentials
```

### Métodos — Listas (para Métricas)

| Método | Descripción |
|---|---|
| `RPush(key, value)` | Se agrega un elemento al final de la lista |
| `LPush(key, value)` | Se agrega un elemento al inicio de la lista (lo usa `MetricsMiddleware`) |
| `LRange(key, start, stop)` | Se obtiene un rango de elementos |
| `LLen(key)` | Se obtiene la longitud de la lista |
| `LTrim(key, start, stop)` | Se recorta la lista al rango especificado |

### Inicialización

```
NewRedisTokenCache(redisConfig, cryptService)
  │
  ├── 1. Parsear la URL construida con config.RedisConfig.GetURL()
  ├── 2. Crear cliente go-redis
  ├── 3. Verificar conexión (Ping)
  └── 4. Retornar instancia configurada
```

---

## CryptService

> **Archivo:** `adapters/crypt/crypt_service.go`
> **Implementa:** `ports.CryptManager`

### Estructura

```go
type CryptService struct{}
```

> Servicio sin estado — todas las operaciones son funciones puras.

### Métodos

| Método | Descripción |
|---|---|
| `GenerateAPIKey()` | Se genera una clave API de 32 bytes (hex-encoded, 64 chars) |
| `GenerateAPISecret()` | Se genera un secret de 48 bytes (base64 URL-safe) |
| `GenerateBulkAPIKeys(amount)` | Se generan N pares de key/secret en lote |
| `EncryptStruct(token, data)` | Se cifran credenciales con clave derivada del token |
| `DecryptStruct(token, data)` | Se descifran credenciales |

### Generación de API Keys

```
GenerateAPIKey()
  → crypto/rand.Read(32 bytes) → hex.EncodeToString → "a1b2c3d4..."

GenerateAPISecret()
  → crypto/rand.Read(48 bytes) → base64.URLEncoding → "dGVzdC..."

GenerateBulkAPIKeys(n)
  → keys[0..n-1] = GenerateAPIKey()
  → secrets[0..n-1] = GenerateAPISecret()
  → return (keys, secrets, error)
```

### Cifrado de Estructuras

```
EncryptStruct(token, data)
  │
  ├── 1. SHA256(token) → clave AES de 32 bytes
  ├── 2. json.Marshal(data) → plaintext
  ├── 3. cryptopasta.Encrypt(plaintext, key) → ciphertext
  └── 4. base64.Encode(ciphertext) → string cifrado

DecryptStruct(token, encryptedData)
  │
  ├── 1. SHA256(token) → clave AES de 32 bytes
  ├── 2. base64.Decode(encryptedData) → ciphertext
  ├── 3. cryptopasta.Decrypt(ciphertext, key) → plaintext
  └── 4. json.Unmarshal(plaintext) → HaciendaCredentials
```

---

## Diagrama de Interacción

```
Login Request
  │
  ▼
AuthMiddleware
  ├── JWTService.ValidateToken(token)
  │     ├── RedisTokenCache.Get("token:" + token) → claims
  │     └── jwt.Parse (firma y expiración)
  │
  ▼
AuthUseCase.Login()
  ├── AuthManager.Login() → genera claims
  ├── JWTService.GenerateToken(claims, tokenLifetime)
  │     ├── Firmar JWT
  │     ├── RedisTokenCache.Set("token:timestamps:{clientID}", ..., tokenLifetime)
  │     └── RedisTokenCache.Set("token:" + token, claims, tokenLifetime)
  │
  └── RedisTokenCache.SetCredentials(token, mhCreds, tokenLifetime)
        └── CryptService.EncryptStruct(token, creds)
```

---

## Notas

1. **Doble verificación del token**: Un token es válido solo si existe en Redis y su firma/expiración son correctas. Eliminar la entrada del cache revoca el token de inmediato.
2. **Credenciales cifradas en reposo**: Las credenciales de Hacienda se almacenan cifradas en Redis usando AES-GCM (via cryptopasta) con una clave derivada del token del usuario.
3. **TTL sincronizado**: El TTL del cache coincide con la vida útil del token (`token_lifetime` del usuario expresado en días, 14 por defecto en el modelo de BD; la estrategia estándar lo multiplica por 24 horas; ver [database.md](database.md)), garantizando limpieza automática.
4. **Crypto seguro**: Se usa `crypto/rand` para generación aleatoria (no `math/rand`), asegurando claves criptográficamente seguras.
5. **Derivación de clave**: Se utiliza SHA256 para derivar claves AES de 32 bytes a partir de tokens de longitud variable.
