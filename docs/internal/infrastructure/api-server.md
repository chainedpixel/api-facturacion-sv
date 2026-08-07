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
    router      *mux.Router
    container   *containers.Container
    srv         *http.Server
    privatePath string  // "/api/v1"
    publicPath  string  // "/api/v1"
}
```

### Inicialización

```go
func (s *Server) Start() error
```

```
Start()
  │
  ├── [1] Crear router Gorilla Mux
  │
  ├── [2] Configurar rutas y middlewares
  │     ConfigureRoutes()
  │
  ├── [3] Configurar servidor HTTP
  │     &http.Server{
  │       Addr:         ":{port}",
  │       Handler:      router,
  │       ReadTimeout:  15s,
  │       WriteTimeout: 15s,
  │       IdleTimeout:  60s,
  │     }
  │
  └── [4] Iniciar servidor
        srv.ListenAndServe()
```

### Shutdown

```go
func (s *Server) Shutdown(ctx context.Context) error
```

Se realiza un apagado graceful del servidor, esperando que las peticiones en curso terminen antes de cerrar.

---

## Rutas

### Stack de Middlewares (orden de ejecución)

```
Request HTTP entrante
  │
  ├── [1] CORS Middleware
  │     → Configura headers Access-Control-*
  │
  ├── [2] Error Middleware
  │     → Recupera panics, captura errores no manejados
  │
  ├── [3] Timeout Middleware (15s)
  │     → Cancela la petición si excede el timeout
  │
  ├── [4] DB Connection Middleware
  │     → Inyecta conexión de BD en el contexto
  │
  ├── [5] Auth Middleware (solo rutas protegidas)
  │     → Valida JWT, inyecta claims en el contexto
  │
  ├── [6] Token Extraction Middleware
  │     → Extrae el token raw del header para uso interno
  │
  └── [7] Metrics Middleware (solo rutas protegidas)
        → Captura duración, status code, endpoint
```

### Grupos de Rutas

#### Rutas Públicas

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/api/v1/health` | HealthHandler | Estado de salud del sistema |

#### Rutas de Autenticación (públicas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | AuthHandler.Login | Autenticación y emisión de JWT |
| `POST` | `/api/v1/auth/register` | AuthHandler.Register | Registro de usuario |
| `POST` | `/api/v1/auth/logout` | AuthHandler.Logout | Revocación de token |

#### Rutas DTE (protegidas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `POST` | `/api/v1/dte/invoice` | DTEHandler.CreateInvoice | Crear factura (01) |
| `POST` | `/api/v1/dte/ccf` | DTEHandler.CreateCCF | Crear CCF (03) |
| `POST` | `/api/v1/dte/credit-note` | DTEHandler.CreateCreditNote | Crear nota de crédito (05) |
| `POST` | `/api/v1/dte/debit-note` | DTEHandler.CreateDebitNote | Crear nota de débito (06) |
| `POST` | `/api/v1/dte/retention` | DTEHandler.CreateRetention | Crear retención (07) |
| `POST` | `/api/v1/dte/remission-note` | DTEHandler.CreateRemissionNote | Crear nota de remisión (04) |
| `POST` | `/api/v1/dte/fse` | DTEHandler.CreateFSE | Crear FSE (14) |
| `POST` | `/api/v1/dte/invalidation` | DTEHandler.Invalidate | Invalidar DTE |
| `GET` | `/api/v1/dte` | DTEHandler.GetAll | Listar DTEs con filtros |
| `GET` | `/api/v1/dte/{id}` | DTEHandler.GetByID | Obtener DTE por UUID |

#### Rutas de Métricas (protegidas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/api/v1/metrics` | MetricsHandler.GetAll | Métricas de todos los endpoints |
| `GET` | `/api/v1/metrics/{endpoint}` | MetricsHandler.GetByEndpoint | Métricas de un endpoint |

#### Rutas de Test (protegidas)

| Método | Ruta | Handler | Descripción |
|---|---|---|---|
| `GET` | `/api/v1/test` | TestHandler.RunSystemTest | Prueba de componentes del sistema |

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
| `Success(rw, status, data, options)` | Se envía respuesta exitosa con datos |
| `Error(rw, status, message, details)` | Se envía respuesta de error |
| `HandleError(rw, err)` | Se clasifica y enruta el error al formato correcto |

### Formato de Respuesta Exitosa

```json
{
  "success": true,
  "data": { ... },
  "links": {
    "qr": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=UUID&fechaEmi=2024-01-15"
  }
}
```

### Formato de Respuesta de Error

```json
{
  "success": false,
  "error": {
    "message": "Descripción del error",
    "details": [ ... ]
  }
}
```

### Clasificación de Errores

```
HandleError(rw, err)
  │
  ├── ValidationError (DTEError)
  │   → 400 Bad Request
  │   → Incluye detalles de campos inválidos
  │
  ├── HaciendaResponseError
  │   → 400 Bad Request
  │   → Incluye código y observaciones de Hacienda
  │
  ├── ServiceError
  │   → 400 Bad Request
  │   → Incluye servicio y método que falló
  │
  └── Error genérico
      → 500 Internal Server Error
      → Mensaje genérico
```

### Generación de QR

Para DTEs exitosos, se genera un enlace de consulta pública de Hacienda:

```
https://admin.factura.gob.sv/consultaPublica?ambiente={amb}&codGen={uuid}&fechaEmi={fecha}
```

---

## Notas

1. **Timeouts**: Read/Write timeout de 15 segundos, idle timeout de 60 segundos. Las transmisiones a Hacienda (30s) pueden acercarse al límite.
2. **Gorilla Mux**: Se usa para routing con soporte de path parameters (`{id}`), métodos HTTP, y subrouters.
3. **Middleware selectivo**: Auth y Metrics solo aplican a rutas protegidas, no a health checks ni login.
4. **QR automático**: Toda respuesta exitosa de creación de DTE incluye el enlace QR de consulta pública.
5. **Panic recovery**: El Error Middleware captura panics con stack trace para prevenir crashes del servidor.
