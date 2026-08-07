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
| `ValidateToken(tokenString)` | Se valida el token y se recuperan claims del cache |
| `RevokeToken(token)` | Se invalida un token eliminándolo del cache |
| `SaveTimestampsForContingency(issuedAt, expiresAt, tokenLifetime, claims)` | Se cachean timestamps para contingencia |
| `GetSecretKey()` | Se retorna la clave de firma |

### Claims del Token

```go
type AuthClaims struct {
    ClientID  uint      `json:"sub"`
    BranchID  uint      `json:"branch_sub"`
    AuthType  string    `json:"auth_type"`
    NIT       string    `json:"nit"`
    ExpiresAt time.Time `json:"expires_at"`
    IssuedAt  time.Time `json:"iat"`
}
```

### Flujo de Generación

```
GenerateToken(claims, lifetime)
  │
  ├── 1. Crear JWT con claims estándar (sub, exp, iat)
  │      Algoritmo: HS256
  │      Firma: SecretKey
  │
  ├── 2. Firmar token → tokenString
  │
  ├── 3. Cachear claims en Redis
  │      Key: tokenString
  │      TTL: tokenLifetime
  │
  └── 4. Retornar tokenString
```

### Flujo de Validación

```
ValidateToken(tokenString)
  │
  ├── 1. Buscar claims en cache (Redis)
  │      Key: tokenString
  │
  ├── 2. Si no existe → token inválido o expirado
  │
  └── 3. Si existe → retornar AuthClaims deserializados
```

> **Nota:** La validación se basa en la existencia del token en cache, no en la firma JWT. Esto permite la revocación inmediata.

---

## RedisTokenCache

> **Archivo:** `adapters/cache/redis_token_cache.go`
> **Implementa:** `ports.CacheManager`

### Estructura

```go
type RedisTokenCache struct {
    client      *redis.Client
    cryptService ports.CryptManager
    ctx         context.Context
}
```

### Métodos — Operaciones Básicas

| Método | Descripción |
|---|---|
| `Set(key, value, ttl)` | Se almacena un valor con tiempo de expiración |
| `Get(key)` | Se recupera un valor por clave |
| `Delete(token)` | Se elimina una entrada |
| `Close()` | Se cierra la conexión Redis |

### Métodos — Credenciales Cifradas

| Método | Descripción |
|---|---|
| `SetCredentials(token, creds, ttl)` | Se cifran y almacenan credenciales de Hacienda |
| `GetCredentials(token)` | Se descifran y retornan credenciales |

### Flujo de Cifrado de Credenciales

```
SetCredentials(token, creds, ttl)
  │
  ├── 1. Serializar HaciendaCredentials a JSON
  ├── 2. Derivar clave AES del token (SHA256)
  ├── 3. Cifrar JSON con cryptopasta (AES-GCM)
  ├── 4. Codificar en base64
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
| `LPush(key, value)` | Se agrega un elemento al inicio de la lista |
| `LRange(key, start, stop)` | Se obtiene un rango de elementos |
| `LLen(key)` | Se obtiene la longitud de la lista |
| `LTrim(key, start, stop)` | Se recorta la lista al rango especificado |

### Inicialización

```
NewRedisTokenCache(redisURL, cryptService)
  │
  ├── 1. Parsear URL de Redis
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
| `GenerateAPISecret()` | Se genera un secret de 48 bytes (base64-encoded) |
| `GenerateBulkAPIKeys(amount)` | Se generan N pares de key/secret en lote |
| `EncryptStruct(token, data)` | Se cifran credenciales con clave derivada del token |
| `DecryptStruct(token, data)` | Se descifran credenciales |

### Generación de API Keys

```
GenerateAPIKey()
  → crypto/rand.Read(32 bytes) → hex.EncodeToString → "a1b2c3d4..."

GenerateAPISecret()
  → crypto/rand.Read(48 bytes) → base64.StdEncoding → "dGVzdC..."

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
  │     └── RedisTokenCache.Get(token) → claims
  │
  ▼
AuthUseCase.Login()
  ├── AuthManager.Login() → genera claims
  ├── JWTService.GenerateToken(claims, 24h)
  │     ├── Firmar JWT
  │     └── RedisTokenCache.Set(token, claims, 24h)
  │
  └── RedisTokenCache.SetCredentials(token, mhCreds, 24h)
        └── CryptService.EncryptStruct(token, creds)
```

---

## Notas

1. **Token en cache = token válido**: La validación de JWT se basa en la existencia en Redis, no en la verificación criptográfica de la firma. Esto permite la revocación instantánea al eliminar la entrada del cache.
2. **Credenciales cifradas en reposo**: Las credenciales de Hacienda se almacenan cifradas en Redis usando AES-GCM (via cryptopasta) con una clave derivada del token del usuario.
3. **TTL sincronizado**: El TTL del cache coincide con la vida útil del token (24 horas), garantizando limpieza automática.
4. **Crypto seguro**: Se usa `crypto/rand` para generación aleatoria (no `math/rand`), asegurando claves criptográficamente seguras.
5. **Derivación de clave**: Se utiliza SHA256 para derivar claves AES de 32 bytes a partir de tokens de longitud variable.
