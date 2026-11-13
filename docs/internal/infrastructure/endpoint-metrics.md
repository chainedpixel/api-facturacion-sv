# Métricas de Endpoints y Test de Sistema

> **Paquetes:**
> - `internal/infrastructure/adapters/metrics` — Recolección de métricas de rendimiento
> - `internal/infrastructure/adapters/test_endpoint` — Diagnóstico de componentes

## Descripción General

Dos herramientas de observabilidad complementarias:

1. **MetricManager** — Registra y consulta métricas de rendimiento (duración, contadores) por endpoint de la API, segmentadas por NIT del emisor.
2. **TestService** — Ejecuta un diagnóstico activo de todos los componentes del sistema bajo demanda.

---

## MetricManager

> **Archivo:** `adapters/metrics/metrics_manager.go`
> **Implementa:** `metricsPort.MetricsManager`

### Estructura

```go
type MetricManager struct {
    cache     ports.CacheManager
    endpoints []struct {
        path   string
        method string
    }
}
```

### Métodos

| Método | Descripción |
|---|---|
| `GetEndpointMetrics(systemNIT, method, endpoint)` | Se obtienen métricas de un endpoint específico |
| `GetAllMetricsEndpoint(systemNIT)` | Se obtienen métricas de todos los endpoints registrados |

### Endpoints Rastreados

`GetAllMetricsEndpoint` consulta únicamente esta lista fija (la clave del resultado es `{METODO}-{nombre}`):

| Nombre | Método | Clave en la respuesta |
|---|---|---|
| `invoices` | POST | `POST-invoices` |
| `ccf` | POST | `POST-ccf` |
| `invalidation` | POST | `POST-invalidation` |
| `retention` | POST | `POST-retention` |
| `credit_note` | POST | `POST-credit_note` |
| `dte` | GET | `GET-dte` |
| `dte/{id}` | GET | `GET-dte/{id}` |

> Discrepancia vigente en el código: el `MetricsMiddleware` registra las notas de crédito con el nombre `creditnote` (no `credit_note`) y las consultas por UUID con el nombre `consult` (no `dte/{id}`); además registra con su propio nombre (`fse`, `debitnote`, `remissionnote`, etc.) los documentos que no figuran en esta lista. Por ello, `GET /metrics` sin filtros no devuelve contadores para `credit_note` ni `dte/{id}`, y esos otros endpoints solo se obtienen consultando con `?endpoint=...&method=...` (ver [middleware.md](middleware.md)).

### Estructura de Métricas por Endpoint

```go
type EndpointMetrics struct {
    Path           string  `json:"path"`
    Method         string  `json:"method"`
    SystemNIT      string  `json:"system_nit"`
    TotalRequests  int64   `json:"total_requests"`
    SuccessCount   int64   `json:"success_count"`
    ErrorCount     int64   `json:"error_count"`
    CurrentAverage int64   `json:"current_average_ms"`
    MaxDuration    int64   `json:"max_duration_ms"`
    MinDuration    int64   `json:"min_duration_ms"`
    LastDurations  []int64 `json:"last_durations"`
}
```

Las duraciones son enteros en milisegundos.

### Cálculo de las Métricas

```
GetEndpointMetrics(nit, method, endpoint)
  │
  ├── LRange("metrics:{nit}:{method}:{endpoint}:durations", 0, 19)
  │   → lista de RequestMetric (JSON) con duration, timestamp y status_code
  │
  ├── Get("metrics:{nit}:{method}:{endpoint}:counters")
  │   → JSON {total_requests, success_count, error_count}
  │
  └── Sobre las duraciones leídas:
        last_durations = duraciones (más reciente primero)
        current_average_ms = suma / cantidad (división entera)
        min_duration_ms / max_duration_ms = mínimo y máximo de esas duraciones
```

El mínimo, el máximo y el promedio se calculan en cada consulta sobre la ventana de las últimas 20 mediciones; no se persisten.

### Estructura de Claves en Redis

```
metrics:{NIT}:{METHOD}:{ENDPOINT}:durations → List   (últimas 20 mediciones JSON, TTL 24h)
metrics:{NIT}:{METHOD}:{ENDPOINT}:counters  → String (JSON con los tres contadores, TTL 24h)
```

### Exposición HTTP

`GET /api/v1/metrics` (protegida) responde en formato JSON estándar (`{success, data}`), no en formato Prometheus. Se filtra por el NIT de los claims del token. Si se envían los query params `endpoint` y `method` se devuelve un único `EndpointMetrics`; si falta alguno, se devuelve un mapa con los endpoints rastreados.

---

## TestService

> **Archivo:** `adapters/test_endpoint/test_endpoint_service.go`
> **Implementa:** `test_endpoint.TestManager`

### Propósito

Ejecuta un conjunto de pruebas activas contra cada componente del sistema para verificar que el entorno completo está operativo. A diferencia del health check (que solo verifica conectividad), el test endpoint realiza operaciones reales.

### Estructura

```go
type testService struct {
    db         *gorm.DB
    authRepo   auth.AuthRepositoryPort
    httpClient *http.Client  // timeout: 5s
}
```

### Pruebas Ejecutadas

```
RunSystemTest(ctx)
  │
  ├── [1] database_connection
  │     db.DB().Ping()
  │     └── Fallo → se detienen las pruebas
  │
  ├── [2] dte_mapping
  │     Se mapea un DTE de prueba con ToMHInvoice
  │     └── Fallo → se detienen las pruebas
  │
  ├── [3] sequence_generation
  │     UPDATE de last_number + 1 en control_number_sequences
  │     (branch_id = 0, dte_type = "01") dentro de una transacción
  │
  ├── [4] signer_service
  │     Se obtiene el cliente por NIT del token y se hace POST al servicio de firma
  │     con un DTE de prueba. Éxito si responde 200
  │
  └── [5] hacienda_transmission
        POST del DTE de prueba (sin firmar ni autenticar) a la URL de recepción de Hacienda
        (timeout 5s). Éxito si responde 4xx
```

### Resultado

```go
type TestResult struct {
    Success  bool            `json:"success"`
    Tests    []ComponentTest `json:"tests"`
    Duration int64           `json:"duration_ms"`
}

type ComponentTest struct {
    Name     string `json:"name"`
    Success  bool   `json:"success"`
    Duration int64  `json:"duration_ms"`
}
```

`Success` global es verdadero solo si todas las pruebas ejecutadas fueron exitosas. No se reporta mensaje de error por prueba (el detalle queda en los logs).

### Comportamiento ante Fallos

- Si **database_connection** o **dte_mapping** fallan, se detienen las pruebas restantes y se devuelve el resultado parcial.
- Si falla otro componente, se continúa con los siguientes y se reporta el fallo individualmente.
- Se **espera** un `4xx` de Hacienda en la prueba de transmisión (el documento de prueba no está firmado ni autenticado); un `2xx` o `5xx` se reporta como fallo.
- El endpoint siempre responde `200` con el resultado, salvo error interno (`500`).

### Efecto colateral

La prueba `sequence_generation` incrementa `last_number` de la fila con `branch_id = 0` y `dte_type = "01"`, si existe. Con datos reales normalmente no existe una sucursal con id 0, por lo que el UPDATE no afecta filas y la prueba resulta exitosa.

---

## Notas

1. **Métricas segmentadas por NIT**: Cada emisor tiene sus propias claves en Redis, permitiendo monitoreo independiente por cliente.
2. **Ventana de 20 mediciones y TTL**: Los promedios se calculan solo sobre las últimas 20 mediciones. Cada escritura del middleware renueva un TTL de 24 horas; el `MetricsCleanupJob` aplica TTL a claves huérfanas sin expiración (ver [Jobs](jobs.md)).
3. **Test endpoint protegido**: Requiere autenticación Bearer porque ejecuta operaciones reales (firma con las credenciales del cliente, conecta a Hacienda).
4. **Hacienda test intencional**: El rechazo `4xx` de Hacienda es el resultado esperado; confirma que la comunicación HTTP funciona sin emitir un DTE real. Esta prueba usa `http.DefaultClient` con un contexto de 5 segundos.
