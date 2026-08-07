# Jobs — Trabajos Programados

> **Paquete:** `internal/infrastructure/jobs`

## Descripción General

Los jobs son trabajos de mantenimiento que se ejecutan periódicamente de forma programada. Cada job implementa protección contra ejecuciones concurrentes mediante `atomic.Bool` (`IsRunning`), lo que garantiza que una segunda invocación del mismo job sea ignorada si la primera todavía está en curso.

Todos los jobs se registran y se programan en la capa de bootstrap (`bootstrap/scheduler.go`).

---

## RetransmissionJob

> **Archivo:** `contingency_retransmission_job.go`

### Propósito

Retransmite a Hacienda los documentos que quedaron pendientes durante un período de contingencia. Es el mecanismo de recuperación automática que cierra el ciclo del flujo offline → online.

### Estructura

```go
type RetransmissionJob struct {
    connection         *drivers.DbConnection
    ContingencyService contingency.ContingencyManager
    IsRunning          atomic.Bool
    MaxExecutionTime   time.Duration  // default: 10 minutos
    Bus                event.Bus
}
```

### Flujo de Ejecución

```
Execute()
  │
  ├── [1] Verificar concurrencia
  │     IsRunning.CompareAndSwap(false, true)
  │     └── Ya en ejecución → salir sin error (log Warn)
  │
  ├── [2] Verificar conexión a base de datos
  │     connection.Db.DB().Ping()
  │
  ├── [3] Retransmitir documentos pendientes
  │     ContingencyService.RetransmitPendingDocuments(ctx)
  │     ├── Éxito → log Info "completed successfully"
  │     └── Error → handleExecutionError(err)
  │
  └── [4] Liberar flag de ejecución
        defer IsRunning.Store(false)
```

### Manejo de Errores

Cuando `RetransmitPendingDocuments` falla, el job distingue dos casos:

| Error | Comportamiento |
|---|---|
| `context.DeadlineExceeded` | Se registra timeout en logs + se publica `RetransmissionJobFailedEvent` |
| Cualquier otro error | Se registra fallo en logs + se publica `RetransmissionJobFailedEvent` |

La publicación del evento activa el notificador de email al administrador (ver [Eventos de Dominio](domain-events.md)).

### Configuración

| Campo | Valor por Defecto |
|---|---|
| `MaxExecutionTime` | 10 minutos |
| Frecuencia de ejecución | Configurable en bootstrap (cron) |

---

## MetricsCleanupJob

> **Archivo:** `metrics_cleanup_job.go`

### Propósito

Aplica TTL a las claves de métricas en Redis que no tienen expiración configurada. Esto previene el crecimiento indefinido de claves huérfanas en Redis cuando los endpoints son llamados durante mucho tiempo sin reinicio del servicio.

### Estructura

```go
type MetricsCleanupJob struct {
    cache     ports.CacheManager
    IsRunning atomic.Bool
}

const metricsTTL = 24 * time.Hour
```

### Flujo de Ejecución

```
Execute()
  │
  ├── [1] Verificar concurrencia (IsRunning)
  │
  ├── [2] Limpiar claves de duraciones
  │     cleanPattern("metrics:*:durations")
  │     └── ScanKeys → Expire(key, 24h) por cada clave
  │
  ├── [3] Limpiar claves de contadores
  │     cleanPattern("metrics:*:counters")
  │     └── ScanKeys → Expire(key, 24h) por cada clave
  │
  └── [4] Log con total de claves procesadas
```

### Patrones de Claves Gestionadas

| Patrón | Contenido |
|---|---|
| `metrics:*:durations` | Listas Redis con duraciones de respuesta por endpoint |
| `metrics:*:counters` | Hashes Redis con contadores (total, success, errors, min, max) |

---

## ReservationCleanerJob

> **Archivo:** `reservation_cleaner_job.go`

### Propósito

Libera reservas de números de control que expiraron sin ser utilizadas. Esto ocurre cuando una solicitud de DTE reserva un número secuencial pero falla antes de completar la emisión, dejando la reserva en estado `pending` indefinidamente.

### Estructura

```go
type ReservationCleanerJob struct {
    connection       *drivers.DbConnection
    repository       dte_documents.ReservedSequenceRepositoryPort
    IsRunning        atomic.Bool
    MaxExecutionTime time.Duration  // default: 5 minutos
}
```

### Flujo de Ejecución

```
Execute()
  │
  ├── [1] Verificar concurrencia (IsRunning)
  │
  ├── [2] Verificar conexión a base de datos
  │     connection.Db.DB().Ping()
  │
  ├── [3] Obtener reservas expiradas
  │     repository.GetExpiredNonContingencyReservations(ctx)
  │     └── Sin reservas → log Info + salir
  │
  ├── [4] Por cada reserva expirada:
  │     repository.UpdateStatus(
  │       branchID, dteType, sequenceNumber, year,
  │       status=Released, releasedAt=now
  │     )
  │     ├── Éxito → log Info "auto-released"
  │     └── Error → log Error + continuar con la siguiente
  │
  └── [5] Liberar flag de ejecución
```

### Estado de Reservas

| Estado | Descripción |
|---|---|
| `pending` | Reserva activa, número en uso |
| `Released` | Reserva liberada por expiración (acción de este job) |

### Configuración

| Campo | Valor por Defecto |
|---|---|
| `MaxExecutionTime` | 5 minutos |
| Frecuencia de ejecución | Configurable en bootstrap (cron) |

---

## Patrón Común: Protección Contra Concurrencia

Los tres jobs utilizan el mismo patrón de guardia:

```go
if !j.IsRunning.CompareAndSwap(false, true) {
    logs.Warn("Job already running, skipping execution")
    return
}
defer j.IsRunning.Store(false)
```

`atomic.Bool` garantiza que la verificación sea thread-safe sin necesidad de mutex. Si el scheduler dispara un job mientras el anterior todavía está corriendo, la segunda ejecución se descarta silenciosamente.

---

## Notas

1. **Sin estado compartido entre ejecuciones**: Cada llamada a `Execute()` es independiente. Los jobs no acumulan estado entre corridas.
2. **Errores no fatales**: Los jobs manejan errores individuales (por ejemplo, una reserva que no pudo liberarse) sin interrumpir el procesamiento del resto. Solo errores globales (como perder la conexión a BD) detienen la ejecución completa.
3. **Bus de eventos opcional**: El `RetransmissionJob` acepta un `event.Bus` pero funciona correctamente sin él — simplemente no publica el evento de fallo si el bus es `nil`.
4. **Registro en scheduler**: Los jobs no son autocontenidos — deben registrarse en el scheduler del bootstrap. Consultar `bootstrap/scheduler.go` para ver la frecuencia de cada uno.
