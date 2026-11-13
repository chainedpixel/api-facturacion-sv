# Servidor HTTP y Respuestas

> **Paquetes:**
> - `internal/infrastructure/api/server` — Configuración del servidor y rutas
> - `internal/infrastructure/api/response` — Escritor de respuestas estandarizado

## Descripción General

El servidor HTTP es el punto de entrada de todas las peticiones al sistema. Se configura con Gorilla Mux como router, aplica un stack de middlewares, y delega a handlers que invocan los casos de uso.

---

## Server

> **Archivo:** `api/server/server.go`

### Estructura

```go
type Server struct {
    router    *mux.Router
    container *containers.Container
    srv       *http.Server

    privatePath string  // "/api/v1"
    publicPath  string  // "/api/v1"
}
```

`Initialize(container)` crea el router Gorilla Mux con ambos prefijos en `/api/v1`.

### Inicialización

```go
func (s *Server) Start() error
```

```
Start()
  │
  ├── [1] ConfigureRoutes()
  │     ├── Middlewares globales: CORS, Error, Timeout
  │     ├── Handler global de OPTIONS (200)
  │     ├── Subrouters público y protegido (prefijo /api/v1)
  │     ├── Middlewares protegidos: Auth, TokenExtractor, Metrics
  │     ├── Middleware global DBConnection
  │     ├── Ruta /health (raíz)
  │     ├── Rutas públicas (auth)
  │     └── Rutas protegidas (DTE, metrics, test y, con DEBUG=true, debug)
  │
  ├── [2] Configurar servidor HTTP
  │     &http.Server{
  │       Addr:         ":" + config.Server.Port,
  │       Handler:      router,
  │       ReadTimeout:  15s,
  │       WriteTimeout: 15s,
  │       IdleTimeout:  60s,
  │     }
  │
  └── [3] srv.ListenAndServe()
        (http.ErrServerClosed no se considera error)
```

### Shutdown

```go
func (s *Server) Shutdown(ctx context.Context) error
```

Apagado graceful del servidor, esperando que las peticiones en curso terminen antes de cerrar. Si el servidor no fue iniciado, no se realiza ninguna acción.

---

## Rutas

### Stack de Middlewares (orden de ejecución)

Los middlewares globales se registran sobre el router raíz (`CORS`, `Error`, `Timeout` y, a continuación, `DBConnection`). Los de las rutas protegidas se registran sobre el subrouter protegido (`Auth`, `TokenExtractor`, `Metrics`).

```
Request HTTP entrante
  │
  ├── [1] CORS Middleware
  │     → Headers Access-Control-* según los orígenes configurados; responde 200 a OPTIONS
  │
  ├── [2] Error Middleware
  │     → Recupera panics y completa respuestas 4xx sin cuerpo
  │
  ├── [3] Timeout Middleware (14s)
  │     → Responde 408 si el handler excede el tiempo
  │
  ├── [4] DB Connection Middleware
  │     → Verifica la conexión a la BD (Ping) antes de continuar
  │
  ├── [5] Auth Middleware (solo rutas protegidas)
  │     → Valida JWT, inyecta claims en el contexto
  │
  ├── [6] Token Extraction Middleware (solo rutas protegidas)
  │     → Inyecta el token raw del header en el contexto
  │
  └── [7] Metrics Middleware (solo rutas protegidas)
        → Registra duración y status en Redis, por NIT, método y endpoint
```

Detalle de cada middleware en [middleware.md](middleware.md).

### Grupos de Rutas

Todas las rutas de negocio cuelgan del prefijo `/api/v1`, excepto `/health`, que se registra en la raíz del router (sin prefijo).

#### Rutas Públicas

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/health` | HealthHandler.CheckHealth | Estado de salud del sistema (sin prefijo `/api/v1`, sin autenticación) |

#### Rutas de Autenticación (públicas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | AuthHandler.Login | Autenticación y emisión de JWT |
| `POST` | `/api/v1/auth/register` | AuthHandler.Register | Registro de usuario con sus sucursales |

> No existe ruta de `logout`.

#### Rutas DTE (protegidas)

Las rutas de creación se registran dinámicamente en `RegisterDTERoutes`: se crea una ruta `POST` por cada documento registrado en `GenericCreatorDTEHandler` (ver `bootstrap/containers/handlers.go`). Todas se atienden con `GenericCreatorDTEHandler.HandleCreate`.

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `POST` | `/api/v1/dte/invoices` | HandleCreate | Crear factura (01) |
| `POST` | `/api/v1/dte/ccf` | HandleCreate | Crear CCF (03) |
| `POST` | `/api/v1/dte/remissionnote` | HandleCreate | Crear nota de remisión (04) |
| `POST` | `/api/v1/dte/creditnote` | HandleCreate | Crear nota de crédito (05) |
| `POST` | `/api/v1/dte/debitnote` | HandleCreate | Crear nota de débito (06) |
| `POST` | `/api/v1/dte/retention` | HandleCreate | Crear comprobante de retención (07) |
| `POST` | `/api/v1/dte/fse` | HandleCreate | Crear factura de sujeto excluido (14) |
| `POST` | `/api/v1/dte/invalidation` | DTEHandler.InvalidateDocument | Invalidar DTE |
| `GET` | `/api/v1/dte/{id}/archivo` | DTEHandler.GetArchive | Archivo DTE listo para entregar al receptor |
| `GET` | `/api/v1/dte/{id}/eventos` | DTEHandler.GetEvents | Eventos del documento con la respuesta de Hacienda |
| `GET` | `/api/v1/dte/{id}` | DTEHandler.GetByGenerationCode | Obtener DTE por código de generación (UUID) |
| `GET` | `/api/v1/dte` | DTEHandler.GetAll | Listar DTEs con filtros (query string) |

Notas de comportamiento:

- `GetArchive` no usa el envoltorio `{success, data}`: responde el JSON del Archivo DTE tal cual (`Content-Type: application/json`) con `Content-Disposition: attachment; filename="{codigoGeneracion}.json"`. Ante error se usa `HandleError`.
- `GetEvents` responde con el formato estándar de éxito (`200`).
- `HandleCreate` responde `201` en éxito. Si el caso de uso falla y el documento tiene `UsesContingency=true` (factura, CCF, nota de remisión, FSE y nota de débito), el error se evalúa en `ContingencyHandler`: si corresponde a contingencia, el documento se almacena y se responde `201` con el DTE actualizado con el tipo y motivo de contingencia; en caso contrario se responde el error. Nota de crédito y comprobante de retención tienen `UsesContingency=false` y siempre responden el error.
- `InvalidateDocument` responde `200` en éxito.
- La ruta se resuelve por coincidencia de subcadena del path contra las rutas registradas (`strings.Contains`).

#### Rutas de Métricas (protegidas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/api/v1/metrics` | MetricsHandler.GetEndpointMetrics | Métricas por endpoint del NIT autenticado, en JSON |

Con los query params `endpoint` y `method` se devuelven las métricas de un solo endpoint; si falta alguno, se devuelven las de todos los endpoints. No existe la ruta `/metrics/{endpoint}` ni formato Prometheus (ver [endpoint-metrics.md](endpoint-metrics.md)).

#### Rutas de Test (protegidas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/api/v1/test` | TestHandler.RunSystemTest | Prueba de componentes del sistema |

#### Rutas de Depuración (protegidas, condicionales)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `POST` | `/api/v1/debug/notify-test` | DebugNotifyHandler.SendTestAlert | Envía un correo de ejemplo al administrador (`?event=contingency\|emission\|job`, por defecto `contingency`) sin aplicar cooldown |

Solo se registra si la configuración `Server.Debug` está activa (`DEBUG=true`). Responde `503` si el mailer o el renderer no están inicializados.

---

## ResponseWriter

> **Archivo:** `api/response/writer.go`

### Estructura

```go
type ResponseWriter struct{}
```

> Servicio sin estado que estandariza el formato de todas las respuestas HTTP.

### Métodos

| Método | Descripción |
|---|---|
| `Success(rw, status, data, options)` | Se envía respuesta exitosa. Si `options` es nil se usa `{success, data}`; si no, se usa el formato DTE con sello de recepción y enlace QR |
| `Error(rw, status, message, details)` | Se envía respuesta de error con `code` derivado del status HTTP |
| `HandleError(rw, err)` | Se clasifica el error y se enruta al formato correcto |

### Formato de Respuesta Exitosa

Sin `options` (consultas, auth, métricas, etc.):

```json
{
  "success": true,
  "data": { ... }
}
```

Con `options` (creación de DTE vía `HandleCreate`):

```json
{
  "success": true,
  "reception_stamp": "2024ABC...",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=00&codGen=UUID&fechaEmi=2024-01-15",
  "data": { ... }
}
```

`reception_stamp` es el sello de recepción (`null` si no hay) y `qr_link` se arma con `GenerateQRLink` (ambiente, código de generación y fecha de emisión con formato `2006-01-02`).

### Formato de Respuesta de Error

Cuando el error se emite con `Error(...)` (middlewares, handlers que validan el cuerpo, etc.) no se incluye `success`:

```json
{
  "error": {
    "message": "Descripción del error",
    "details": [ ... ],
    "code": "BAD_REQUEST"
  }
}
```

Cuando se emite con `HandleError(...)` se incluye `success: false` y `data: null`:

```json
{
  "success": false,
  "data": null,
  "error": {
    "message": "...",
    "details": [ ... ],
    "code": "..."
  }
}
```

Códigos derivados del status en `Error(...)`: 400 `BAD_REQUEST`, 401 `UNAUTHORIZED`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 405 `METHOD_NOT_ALLOWED`, 408 `REQUEST_TIMEOUT`, 500 `INTERNAL_SERVER_ERROR`; cualquier otro `UNKNOWN_ERROR`.

### Clasificación de Errores

```
HandleError(rw, err)
  │
  ├── Validación: *DTEError o *ValidationError
  │   → 400
  │   → DTEError: message = GetMessage(), details = errores de validación, code = GetCode()
  │   → ValidationError: message = Error(), details = ["sin detalles disponibles" (service_errors.NoDetailsAvailable)],
  │     code = tipo en mayúsculas
  │
  ├── Negocio: *HaciendaResponseError
  │   → 400
  │   → message = descripción de Hacienda, code = "HACIENDA_{codigo}",
  │     details = ["State: {estado}", "Processed at: {fecha}", ...observaciones]
  │
  ├── Negocio: *ServiceError (shared_error)
  │   → 400
  │   → message = Message, details = GetErrError(), code = GetCode() en mayúsculas
  │
  ├── Negocio: *HTTPResponseError (Hacienda)
  │   → status HTTP original del error
  │   → message = error, details = [service_errors.ContingencyActiveTransmission], code = "HACIENDA_{status}"
  │
  ├── Negocio: cualquier otro tipo de negocio
  │   → 400, code "BUSINESS_ERROR", message = err.Error()
  │
  └── Cualquier otro error (sistema)
      → 500, message = validation_errors.ServerError, code "SYSTEM_ERROR"
        (no se expone el detalle interno)
```

La clasificación inicial se hace por tipo concreto del error (`switch err.(type)`), por lo que un error envuelto con `fmt.Errorf("%w")` se clasifica como sistema aunque contenga uno de los tipos anteriores.

### Generación de QR

Para DTEs exitosos, se genera un enlace de consulta pública de Hacienda:

```
https://admin.factura.gob.sv/consultaPublica?ambiente={amb}&codGen={uuid}&fechaEmi={fecha}
```

---

## Notas

1. **Timeouts**: Read/Write timeout del servidor de 15 segundos, idle timeout de 60 segundos y timeout de petición (TimeoutMiddleware) de 14 segundos.
2. **Gorilla Mux**: Se usa para routing con soporte de path parameters (`{id}`), métodos HTTP y subrouters. Las rutas más específicas (`/dte/{id}/archivo`, `/dte/{id}/eventos`) se registran antes que `/dte/{id}`.
3. **Middleware selectivo**: Auth, TokenExtractor y Metrics solo aplican a rutas protegidas, no a `/health` ni a auth.
4. **QR**: Solo las respuestas de creación de DTE (con `options`) incluyen `qr_link` y `reception_stamp`.
5. **Panic recovery**: El Error Middleware captura panics, registra el stack trace y responde 500.
6. **Orden de registro de middlewares**: `DBConnection` se agrega a los globales después de crear los subrouters; en Gorilla Mux los middlewares del router raíz se ejecutan antes que los del subrouter.
