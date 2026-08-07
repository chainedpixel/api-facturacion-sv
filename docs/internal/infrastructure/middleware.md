# Middlewares — Interceptores de Petición

> **Paquete:** `internal/infrastructure/api/middleware`

## Descripción General

Los middlewares interceptan cada petición HTTP antes de que llegue al handler. Se encadenan en un orden específico, donde cada middleware procesa la petición y opcionalmente la pasa al siguiente. El sistema implementa 7 middlewares.

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
  ├── [1] Extraer header Authorization
  │     ├── No existe → 401 "Missing authorization header"
  │     └── Existe → continuar
  │
  ├── [2] Validar formato "Bearer {token}"
  │     ├── Formato inválido → 401 "Invalid authorization format"
  │     └── Válido → extraer token
  │
  ├── [3] Validar token
  │     tokenService.ValidateToken(token)
  │     ├── Error → 401 "Invalid or expired token"
  │     └── Válido → obtener AuthClaims
  │
  └── [4] Inyectar claims en el contexto
        ctx = context.WithValue(ctx, "claims", claims)
        next.ServeHTTP(rw, req.WithContext(ctx))
```

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
  ├── [1] Crear statusWriter wrapper
  │     statusWriter{ResponseWriter, statusCode: 200, written: false}
  │
  ├── [2] Configurar panic recovery
  │     defer func() {
  │       if r := recover(); r != nil {
  │         log.Error(stackTrace)
  │         → 500 "Internal server error"
  │       }
  │     }
  │
  ├── [3] Ejecutar handler
  │     next.ServeHTTP(statusWriter, req)
  │
  └── [4] Verificar status code post-handler
        ├── 4xx → Error response (si no se escribió)
        ├── 5xx → Error response (si no se escribió)
        └── 2xx → No acción
```

### statusWriter

```go
type statusWriter struct {
    http.ResponseWriter
    statusCode int
    written    bool
}
```

Se envuelve el `ResponseWriter` original para:
- Capturar el código de status (`WriteHeader`)
- Rastrear si ya se escribió una respuesta (`written`)
- Prevenir escrituras duplicadas

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
  ├── [1] Registrar tiempo de inicio
  │     start := time.Now()
  │
  ├── [2] Ejecutar handler
  │     next.ServeHTTP(statusWriter, req)
  │
  ├── [3] Calcular duración
  │     duration := time.Since(start)
  │
  ├── [4] Mapear ruta a nombre canónico
  │     "/api/v1/dte/invoice" → "invoices"
  │     "/api/v1/dte/{id}"    → "dte_single"
  │
  └── [5] Actualizar métricas en Redis
        ├── RPush duration al historial
        ├── LTrim a últimos 20 registros
        ├── Incrementar contadores:
        │   ├── total_requests++
        │   ├── success_count++ (si 2xx)
        │   └── error_count++ (si 4xx/5xx)
        └── Actualizar min/max duración
```

### Mapeo de Endpoints

| Ruta | Nombre Canónico |
|---|---|
| `/api/v1/dte/invoice` | `invoices` |
| `/api/v1/dte/ccf` | `ccf` |
| `/api/v1/dte/invalidation` | `invalidation` |
| `/api/v1/dte/retention` | `retention` |
| `/api/v1/dte/credit-note` | `credit_note` |
| `/api/v1/dte` (GET) | `dte_list` |
| `/api/v1/dte/{id}` (GET) | `dte_single` |

### Datos Almacenados

```
Redis Keys por endpoint (prefijo: {NIT}:{method}:{endpoint}):
  ├── :durations    → Lista de últimas 20 duraciones (ms)
  ├── :total        → Contador total de requests
  ├── :success      → Contador de éxitos
  ├── :errors       → Contador de errores
  ├── :min_duration → Duración mínima registrada
  └── :max_duration → Duración máxima registrada
```

---

## TimeoutMiddleware

### Propósito

Se envuelve cada petición con un contexto con timeout de 15 segundos. Si el handler no responde antes del deadline, la petición se cancela automáticamente.

```
Handle(next, timeout=15s)
  │
  ├── ctx, cancel = context.WithTimeout(req.Context(), 15s)
  ├── defer cancel()
  └── next.ServeHTTP(rw, req.WithContext(ctx))
```

---

## DBConnectionMiddleware

### Propósito

Se inyecta la conexión de base de datos en el contexto de la petición para que los handlers y repositorios puedan acceder a ella.

```
Handle(next, dbConnection)
  │
  ├── ctx = context.WithValue(ctx, "db", dbConnection)
  └── next.ServeHTTP(rw, req.WithContext(ctx))
```

---

## CorsMiddleware

### Propósito

Se configuran los headers CORS para permitir peticiones desde clientes web:

```
Headers configurados:
  Access-Control-Allow-Origin: *
  Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
  Access-Control-Allow-Headers: Content-Type, Authorization
```

Se responde con `200 OK` a peticiones `OPTIONS` (preflight) sin ejecutar el handler.

---

## TokenExtractionMiddleware

### Propósito

Se extrae el token JWT raw del header `Authorization` y se almacena en el contexto para uso interno (por ejemplo, para descifrar credenciales de Hacienda del cache).

```
Handle(next)
  │
  ├── Extraer token del header "Authorization: Bearer {token}"
  ├── ctx = context.WithValue(ctx, "token", rawToken)
  └── next.ServeHTTP(rw, req.WithContext(ctx))
```

---

## Orden de Ejecución

```
Request entrante
  │
  ▼
[1] CORS           ← Headers de acceso
[2] Error          ← Panic recovery + captura de status
[3] Timeout        ← Deadline de 15s
[4] DB Connection  ← Conexión de BD en contexto
[5] Auth           ← Validación JWT + claims en contexto (solo protegidas)
[6] Token Extract  ← Token raw en contexto
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

1. **Orden importa**: Los middlewares se ejecutan en el orden registrado. El Error Middleware debe estar al inicio para capturar panics de cualquier middleware posterior.
2. **Selectividad**: Auth y Metrics solo se aplican a rutas protegidas, usando subrouters de Gorilla Mux.
3. **statusWriter**: El patrón de envolver `ResponseWriter` permite interceptar el status code sin modificar el comportamiento del handler.
4. **Métricas limitadas**: Solo se almacenan las últimas 20 duraciones por endpoint para evitar crecimiento ilimitado en Redis.
5. **Token dual**: El sistema almacena tanto los claims (parseados) como el token raw en el contexto, ya que el token raw se necesita para operaciones de cache (descifrado de credenciales).
