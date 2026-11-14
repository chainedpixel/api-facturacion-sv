# Middlewares — Interceptores de Petición

> **Paquete:** `internal/infrastructure/api/middleware`

## Descripción General

Los middlewares interceptan cada petición HTTP antes de que llegue al handler. Se encadenan en un orden específico, donde cada middleware procesa la petición y opcionalmente la pasa al siguiente. El sistema implementa 7 middlewares, instanciados en `bootstrap/containers/middlewares.go` y registrados en `api/server/server.go`.

---

## AuthMiddleware

> **Archivo:** `auth_middleware.go`

### Estructura

```go
type AuthMiddleware struct {
    tokenService ports.TokenManager
    respWriter   *response.ResponseWriter
}
```

### Flujo

```
Handle(next http.Handler)
  │
  ├── [1] Leer header Authorization
  │     ├── Vacío → 401 "Authorization header required"
  │     └── Existe → continuar
  │
  ├── [2] Validar formato "Bearer {token}" (dos partes separadas por espacio)
  │     ├── Formato inválido → 401 "Invalid authorization format"
  │     └── Válido → extraer token
  │
  ├── [3] tokenService.ValidateToken(token)
  │     ├── Error → 401 message "error", details [mensaje del error]
  │     └── Válido → obtener AuthClaims
  │
  └── [4] Inyectar claims en el contexto
        ctx = context.WithValue(ctx, "claims", claims)
        next.ServeHTTP(rw, req.WithContext(ctx))
```

Las respuestas 401 usan `ResponseWriter.Error`, por lo que llevan `error.code = "UNAUTHORIZED"`.

---

## ErrorMiddleware

> **Archivo:** `error_handler_middleware.go`

### Estructura

```go
type ErrorMiddleware struct {
    responseWriter *response.ResponseWriter
}
```

### Flujo

```
Handler(next http.Handler)
  │
  ├── [1] defer: recuperación de panics
  │     recover() → se registra error y stack trace
  │     → 500 "An unexpected error occurred", details [valor del panic]
  │
  ├── [2] Envolver el ResponseWriter en statusWriter
  │
  ├── [3] next.ServeHTTP(statusWriter, req)
  │
  └── [4] Evaluar el status resultante
        ├── status >= 400 y sin respuesta escrita → se escribe respuesta de error
        │     404 "Resource not found", 405 "Method not allowed",
        │     400 "Bad request", otros: texto estándar del status
        └── status >= 500 → solo se registra en log
```

### statusWriter

```go
type statusWriter struct {
    http.ResponseWriter
    status  int
    written bool
}
```

Se envuelve el `ResponseWriter` original para capturar el código de status (`WriteHeader`), rastrear si ya se escribió una respuesta (`written`) y soportar `Hijack`. Nota: `WriteHeader` marca `written = true`, por lo que la respuesta de error de este middleware solo se escribe cuando el handler no invocó `WriteHeader` ni `Write` (por ejemplo, el 404 que genera Gorilla Mux sin pasar por un handler propio).

---

## MetricsMiddleware

> **Archivo:** `metrics_middleware.go`

### Estructura

```go
type MetricsMiddleware struct {
    cache      ports.CacheManager
    maxMetrics int  // 20
}
```

### Flujo

```
Handle(next http.Handler)
  │
  ├── [1] Obtener el NIT de los claims del contexto (requiere AuthMiddleware previo)
  │
  ├── [2] Ejecutar el handler midiendo la duración
  │
  ├── [3] Resolver el nombre del endpoint: extractEndpoint(method, path)
  │
  ├── [4] Historial de duraciones (clave :durations)
  │     ├── LPush de RequestMetric {duración en ms, timestamp, status_code}
  │     ├── LTrim a los últimos 20 registros (índices 0..19)
  │     └── Expire 24h
  │
  └── [5] Contadores (clave :counters, JSON)
        ├── Se lee el valor actual, se incrementa y se guarda con TTL 24h
        ├── total_requests++
        ├── success_count++ si el status es 2xx
        └── error_count++ en cualquier otro caso
```

Los errores de Redis solo se registran en log; no afectan la respuesta.

### Mapeo de Endpoints

`extractEndpoint` consulta primero `endpointMappings` (clave `METODO:ruta`):

| Método y ruta | Nombre |
|---|---|
| `GET /api/v1/dte` | `dte` |
| `GET /api/v1/dte/{id}` | `dte/{id}` |
| `POST /api/v1/dte/invoices` | `invoices` |
| `POST /api/v1/dte/ccf` | `ccf` |
| `POST /api/v1/dte/invalidation` | `invalidation` |
| `POST /api/v1/dte/retention` | `retention` |
| `POST /api/v1/dte/creditnote` | `creditnote` |

Para el resto de rutas:

- Si el primer segmento tras `/api/v1/dte/` es un UUID, se busca la plantilla `/api/v1/dte/{id}[/sufijo]` en el mapa; si no existe y el método es `GET`, el nombre es `consult` (así, `/dte/{id}/archivo` y `/dte/{id}/eventos` se registran bajo `consult`). Nótese que la ruta `GET /api/v1/dte/{uuid}` con un UUID real no coincide con la clave literal `{id}`, por lo que también cae en `consult`.
- En otro caso se usa el primer segmento del path tras `/api/v1/dte/` (por ejemplo `fse`, `debitnote`, `remissionnote`).
- Si no hay segmento, el nombre es `dte`.

### Datos Almacenados

```
Claves Redis por endpoint:
  ├── metrics:{NIT}:{METODO}:{endpoint}:durations → Lista (LPush) de las últimas 20 mediciones (JSON)
  └── metrics:{NIT}:{METODO}:{endpoint}:counters  → JSON con total_requests, success_count, error_count
```

No se almacenan duraciones mínima ni máxima como claves: se calculan al consultar (ver [endpoint-metrics.md](endpoint-metrics.md)).

---

## TimeoutMiddleware

> **Archivo:** `timeout_middleware.go`

Se envuelve cada petición con un contexto con timeout de **14 segundos** (valor fijo en el código). El handler se ejecuta en una goroutine; si el contexto expira antes de que termine, se responde `408 Request Timeout` (título y detalle `RequestTimeOutTitle` y `RequestTimeOut` de i18n) siempre que el handler no haya escrito ya una respuesta.

```
Handler(next)
  │
  ├── ctx, cancel = context.WithTimeout(req.Context(), 14s)
  ├── goroutine: next.ServeHTTP(rw, req.WithContext(ctx)); close(done)
  └── select
        ├── done       → fin
        └── ctx.Done() → log + 408 (si no se escribió respuesta)
```

---

## DBConnectionMiddleware

> **Archivo:** `dbconnection_middleware.go`

Verifica la disponibilidad de la base de datos antes de ejecutar el handler. No inyecta la conexión en el contexto.

```
Handler(next)
  │
  ├── sqlDB, err = connection.Db.DB()
  │     └── Error → 500 "Error connecting to database" (http.Error)
  ├── sqlDB.Ping()
  │     └── Error → log Warn y retorna sin escribir cuerpo (la petición termina con 200 vacío)
  └── next.ServeHTTP(rw, req)
```

---

## CorsMiddleware

> **Archivo:** `cors_middleware.go`

Se instancia con `allowedOrigins = ["*"]` y con los métodos y headers por defecto:

```
Métodos por defecto: GET, POST, PUT, PATCH, DELETE, OPTIONS
Headers por defecto: Accept, Authorization, Content-Type, X-CSRF-Token
```

Si la petición trae header `Origin` y el origen está permitido (`*` o coincidencia exacta) se configuran:

```
Access-Control-Allow-Origin: {origen de la petición}
Access-Control-Allow-Methods: ...
Access-Control-Allow-Headers: ...
Access-Control-Allow-Credentials: true
X-Content-Type-Options: nosniff
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

Se responde `200 OK` a peticiones `OPTIONS` sin ejecutar el handler.

---

## TokenExtractor

> **Archivo:** `auth_token_middleware.go`

Se extrae el token JWT raw del header `Authorization` y se almacena en el contexto (`"token"`) para uso interno (por ejemplo, para descifrar credenciales de Hacienda del cache). Se toma `parts[1]` del header dividido por espacios, sin validación propia: depende de que `AuthMiddleware` haya validado antes el formato.

---

## Orden de Ejecución

```
Request entrante
  │
  ▼
[1] CORS           ← Headers de acceso
[2] Error          ← Panic recovery + respuestas 4xx sin cuerpo
[3] Timeout        ← Deadline de 14s
[4] DB Connection  ← Ping a la BD
[5] Auth           ← Validación JWT + claims en contexto (solo protegidas)
[6] Token Extract  ← Token raw en contexto (solo protegidas)
[7] Metrics        ← Captura de rendimiento (solo protegidas)
  │
  ▼
Handler (DTE, Auth, Health, etc.)
  │
  ▼
Response
```

---

## Notas

1. **Orden importa**: Los middlewares se ejecutan en el orden registrado. El Error Middleware está antes que Timeout, DBConnection y los protegidos para capturar sus panics (no cubre el panic dentro de la goroutine que lanza el Timeout, porque corre fuera de la goroutine de la petición).
2. **Selectividad**: Auth, TokenExtractor y Metrics solo se aplican a rutas protegidas, usando subrouters de Gorilla Mux. `/health` y `/auth/*` no pasan por ellos.
3. **Wrappers de status**: Error, Metrics y Timeout envuelven `ResponseWriter` para interceptar el status code sin modificar el comportamiento del handler.
4. **Métricas limitadas**: Solo se almacenan las últimas 20 mediciones por endpoint, con TTL de 24 horas.
5. **Token dual**: El contexto contiene los claims parseados (`"claims"`) y el token raw (`"token"`).
