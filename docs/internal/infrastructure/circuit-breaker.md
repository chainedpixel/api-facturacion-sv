# Circuit Breaker — Protección contra Fallos en Cascada

> **Paquete:** `internal/infrastructure/adapters/circuit`
> **Archivo:** `breaker.go`

## Descripción General

El Circuit Breaker protege al sistema contra llamadas excesivas a servicios externos (Hacienda API) cuando estos están caídos o degradados. Se implementa el patrón estándar de tres estados con transiciones basadas en conteo de fallos y tiempo de recuperación.

---

## Estructura

```go
type CircuitBreaker struct {
    failures    int32             // Contador de fallos consecutivos
    lastFailure time.Time         // Timestamp del último fallo
    threshold   int32             // Umbral de fallos para abrir (ej. 3)
    resetTime   time.Duration     // Tiempo de espera para semi-abrir (ej. 5min)
    state       constants.State   // Estado actual
    mu          sync.Mutex        // Protección contra acceso concurrente
}
```

### Configuración por Defecto

| Parámetro | Valor | Descripción |
|---|---|---|
| `threshold` | `3` | Se abre el circuito después de 3 fallos consecutivos |
| `resetTime` | `5 minutos` | Se intenta recuperar después de 5 minutos |

---

## Estados

```
┌──────────┐    failures >= threshold    ┌──────────┐
│          │ ──────────────────────────→ │          │
│  CLOSED  │                             │   OPEN   │
│ (Normal) │ ←────────────────────────── │ (Bloq.)  │
│          │    success en Half-Open     │          │
└──────────┘                             └──────────┘
                                              │
                                    resetTime expired
                                              │
                                              ▼
                                        ┌───────────┐
                                        │ HALF-OPEN │
                                        │ (Prueba)  │
                                        └───────────┘
                                         │         │
                                    success      failure
                                         │         │
                                         ▼         ▼
                                      CLOSED     OPEN
```

| Estado | `AllowRequest()` | Comportamiento |
|---|---|---|
| **Closed** | `true` | Operación normal. Todas las peticiones se permiten. |
| **Open** | `false` | Servicio no disponible. Se rechazan todas las peticiones inmediatamente. |
| **Half-Open** | `true` (una vez) | Se permite una sola petición de prueba para verificar recuperación. |

---

## Métodos

### `AllowRequest()`

```go
func (cb *CircuitBreaker) AllowRequest() bool
```

Se determina si una petición puede pasar:

```
AllowRequest()
  │
  ├── State == Closed
  │   → return true (permitir)
  │
  ├── State == Open
  │   ├── ¿Ha pasado resetTime desde lastFailure?
  │   │   ├── Sí → Transicionar a Half-Open
  │   │   │       → return true (permitir prueba)
  │   │   └── No → return false (rechazar)
  │
  └── State == Half-Open
      → return true (permitir prueba)
```

### `RecordSuccess()`

```go
func (cb *CircuitBreaker) RecordSuccess()
```

Se registra un éxito. Si el circuito estaba en Half-Open, se cierra:

```
RecordSuccess()
  │
  ├── failures = 0
  ├── state = Closed
  └── (mutex protected)
```

### `RecordFailure()`

```go
func (cb *CircuitBreaker) RecordFailure()
```

Se registra un fallo. Si se alcanza el threshold, se abre el circuito:

```
RecordFailure()
  │
  ├── failures++
  ├── lastFailure = time.Now()
  │
  ├── ¿failures >= threshold?
  │   ├── Sí → state = Open
  │   └── No → state permanece (Closed)
  │
  └── (mutex protected)
```

### `GetState()` / `GetFailureCount()`

```go
func (cb *CircuitBreaker) GetState() constants.State
func (cb *CircuitBreaker) GetFailureCount() int32
```

Se consulta el estado actual y el conteo de fallos (thread-safe).

---

## Integración con el Sistema

El circuit breaker se usa exclusivamente en el `BatchTransmitterService` para proteger las transmisiones de lotes de contingencia:

```
BatchTransmitterService.TransmitBatch()
  │
  ├── circuitBreaker.AllowRequest()
  │   ├── false → return error inmediato ("servicio no disponible")
  │   └── true → continuar con transmisión
  │
  ├── [Transmitir a Hacienda]
  │
  └── Resultado:
      ├── Éxito → circuitBreaker.RecordSuccess()
      └── Error → circuitBreaker.RecordFailure()
```

### Escenario Ejemplo

```
t=0:00  Transmisión 1 → Error    (failures=1, state=Closed)
t=0:05  Transmisión 2 → Error    (failures=2, state=Closed)
t=0:10  Transmisión 3 → Error    (failures=3, state=Open)
t=0:15  Transmisión 4 → Rechazada inmediatamente (state=Open)
t=0:20  Transmisión 5 → Rechazada inmediatamente (state=Open)
...
t=5:10  Transmisión N → Permitida (state=Half-Open, resetTime expirado)
        └── Éxito → state=Closed
```

---

## Thread Safety

Todas las operaciones que modifican estado están protegidas con `sync.Mutex`:

- `AllowRequest()` — Lectura y posible transición de estado
- `RecordSuccess()` — Reset de contadores y estado
- `RecordFailure()` — Incremento y posible transición

Esto garantiza consistencia cuando múltiples goroutines intentan transmitir concurrentemente.

---

## Notas

1. **Solo en lotes**: El circuit breaker se aplica solo a transmisiones por lotes (contingencia). Las transmisiones individuales dependen del retry logic del `BaseTransmitter`.
2. **Threshold conservador**: Con threshold=3, se toleran 3 fallos antes de bloquear. Esto previene abrir el circuito por errores transitorios.
3. **Reset automático**: Después de 5 minutos, el circuito pasa a Half-Open automáticamente, permitiendo una petición de prueba sin intervención manual.
4. **Fail fast**: Cuando el circuito está abierto, las peticiones fallan inmediatamente sin intentar la conexión, reduciendo la carga sobre el servicio externo caído.
5. **Sin persistencia**: El estado del circuit breaker es in-memory. Si el servicio se reinicia, el circuito vuelve a Closed.
