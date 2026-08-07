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

| Nombre | Método | Ruta |
|---|---|---|
| `invoices` | POST | `/api/v1/dte/invoice` |
| `ccf` | POST | `/api/v1/dte/ccf` |
| `invalidation` | POST | `/api/v1/dte/invalidation` |
| `retention` | POST | `/api/v1/dte/retention` |
| `credit_note` | POST | `/api/v1/dte/credit-note` |
| `dte_list` | GET | `/api/v1/dte` |
| `dte_single` | GET | `/api/v1/dte/{id}` |

### Estructura de Métricas por Endpoint

```go
type EndpointMetrics struct {
    Endpoint      string
    Method        string
    TotalRequests int64
    SuccessCount  int64
    ErrorCount    int64
    MinDuration   float64  // ms
    MaxDuration   float64  // ms
    AvgDuration   float64  // ms (promedio de las últimas 20 duraciones)
}
```

### Cálculo del Promedio

```
GetEndpointMetrics(nit, method, endpoint)
  │
  ├── LRange("{nit}:{method}:{endpoint}:durations", 0, -1)
  │   → [12.5, 15.3, 10.8, ...]
  │
  ├── Calcular promedio de la lista
  │
  ├── HGet counters (total, success, errors)
  │
  └── HGet min/max duration
```

### Estructura de Claves en Redis

```
{NIT}:{METHOD}:{ENDPOINT}:durations     → List  (últimas 20 duraciones en ms)
{NIT}:{METHOD}:{ENDPOINT}:total         → String (contador total de requests)
{NIT}:{METHOD}:{ENDPOINT}:success       → String (contador de éxitos)
{NIT}:{METHOD}:{ENDPOINT}:errors        → String (contador de errores)
{NIT}:{METHOD}:{ENDPOINT}:min_duration  → String (duración mínima registrada)
{NIT}:{METHOD}:{ENDPOINT}:max_duration  → String (duración máxima registrada)
```

---

## TestService

> **Archivo:** `adapters/test_endpoint/test_endpoint_service.go`
> **Implementa:** `test_endpoint.TestManager`

### Propósito

Ejecuta un conjunto de pruebas activas contra cada componente del sistema para verificar que el entorno completo está operativo. A diferencia del health check (que solo verifica conectividad), el test endpoint realiza operaciones reales.

### Estructura

```go
type TestService struct {
    db         *gorm.DB
    authRepo   auth.AuthRepositoryPort
    httpClient *http.Client  // timeout: 5s
}
```

### Pruebas Ejecutadas

```
RunSystemTest(ctx)
  │
  ├── [1] Database Connection
  │     db.Raw("SELECT 1").Scan()
  │     ├── Éxito → continuar
  │     └── Fallo → detener todas las pruebas (dependencia crítica)
  │
  ├── [2] DTE Mapping
  │     Crear request de prueba → mapear a dominio → verificar estructura
  │
  ├── [3] Sequence Generation
  │     Generar siguiente número de control → verificar incremento correcto
  │
  ├── [4] Signer Service
  │     HTTP POST al servicio de firma → verificar respuesta del servidor
  │
  └── [5] Hacienda Transmission
        HTTP POST al endpoint de recepción → esperar 4xx (comportamiento esperado con datos de prueba)
```

### Resultado

```go
type TestResult struct {
    Success        bool
    ComponentTests []ComponentTest
    TotalDuration  time.Duration
}

type ComponentTest struct {
    Name     string
    Success  bool
    Duration time.Duration
    Error    string  // vacío si exitoso
}
```

### Comportamiento ante Fallos

- Si **Database** falla → se detienen todas las pruebas restantes.
- Si cualquier otro componente falla → se continúa con los siguientes y se reporta el fallo individualmente.
- Se **espera** un `4xx` de Hacienda en la prueba de transmisión — datos de prueba inválidos es el comportamiento correcto; un `2xx` indicaría un problema.

---

## Notas

1. **Métricas segmentadas por NIT**: Cada emisor tiene sus propias claves en Redis, permitiendo monitoreo independiente por cliente.
2. **Ventana de 20 duraciones**: El promedio se calcula solo sobre las últimas 20 duraciones para evitar crecimiento ilimitado de listas en Redis. El `MetricsCleanupJob` aplica TTL a estas claves periódicamente (ver [Jobs](jobs.md)).
3. **Test endpoint protegido**: Requiere autenticación Bearer porque ejecuta operaciones reales (genera secuencias, conecta a Hacienda).
4. **Hacienda test intencional**: El rechazo `4xx` de Hacienda es el resultado esperado — confirma que la comunicación HTTP funciona y que Hacienda responde, sin emitir un DTE real.
