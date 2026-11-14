# Jobs — Trabajos Programados

> **Paquete:** `internal/infrastructure/jobs`

## Descripción General

Los jobs son trabajos de mantenimiento que se ejecutan periódicamente de forma programada. Cada job implementa protección contra ejecuciones concurrentes mediante `atomic.Bool` (`IsRunning`), lo que garantiza que una segunda invocación del mismo job sea ignorada si la primera todavía está en curso.

Todos los jobs se instancian y se programan en `cmd/setup/jobs.go` (`SetupJobs`), con el scheduler `gocron` en zona horaria UTC.

### Frecuencia de Ejecución

| Job | Frecuencia |
|---|---|
| `RetransmissionJob` | Cada 30 minutos en producción (`MH_AMBIENT_CODE=01`); cada 5 minutos en cualquier otro ambiente |
| `ReservationCleanerJob` | Cada 10 minutos |
| `MetricsCleanupJob` | Cada 6 horas |

`JobConfig` también define `StartTime` y `EndTime` (22:00-05:00 en producción, 08:00-17:00 en pruebas), pero solo se usan para el log de arranque: el scheduler ejecuta el job de contingencia en cualquier hora del día con el intervalo indicado.

---

## RetransmissionJob

> **Archivo:** `contingency_retransmission_job.go`

### Propósito

Retransmite a Hacienda los documentos que quedaron pendientes durante un período de contingencia. Es el mecanismo de recuperación automática que cierra el ciclo del flujo offline → online. El job solo orquesta la ejecución: la lógica está en `ContingencyService.RetransmitPendingDocuments` (ver [contingency.md](contingency.md)).

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
  ├── [2] Contexto con timeout de MaxExecutionTime (10 min)
  │
  ├── [3] Obtener la conexión a base de datos y ejecutar Ping
  │     (si no se obtiene la conexión se registra el error y se sale; el resultado del Ping no se evalúa)
  │
  ├── [4] Retransmitir documentos pendientes
  │     ContingencyService.RetransmitPendingDocuments(ctx)
  │     ├── Éxito → log Info "completed successfully"
  │     └── Error → handleExecutionError(err)
  │
  └── [5] Liberar flag de ejecución
        defer IsRunning.Store(false)

RetransmitPendingDocuments(ctx)
  ├── Lee hasta MH_MAX_BATCH_SIZE documentos PENDING, ordenados por vencimiento
  ├── Marca como EXPIRED los documentos con más de 72 horas desde el sello del evento
  │     y avisa al administrador (RetransmissionJobFailedEvent, JobName contingency_72h_expired)
  ├── Alerta de contingencia mayor a 3 días (JobName contingency_technical_report)
  └── Envía el evento de contingencia por NIT y transmite los lotes por tipo de DTE
```

### Ventana de 72 horas y Avisos al Administrador

Dentro de cada ejecución, `RetransmitPendingDocuments` aplica la ventana de transmisión de 72 horas contada desde el sello del evento de contingencia (`contingency_documents.event_stamped_at`):

| Situación | Efecto |
|---|---|
| Documento con `event_stamped_at` de más de 72 horas | Pasa a estado `EXPIRED` (observación "Not transmitted within 72 hours after the contingency event stamp"), se considera no emitido y se publica `RetransmissionJobFailedEvent` con `JobName = contingency_72h_expired` y código `CONTINGENCY_72H_EXPIRED` por documento |
| Documento sin `event_stamped_at` (el evento aún no se envió) | Nunca vence; se intenta enviar el evento |
| Documento pendiente más antiguo (`created_at`) con más de 72 horas | Se registra una advertencia y se publica `RetransmissionJobFailedEvent` con `JobName = contingency_technical_report`: la contingencia supera los 3 días y se requiere un informe técnico |

Ambos avisos llegan por correo al administrador (ver [domain-events.md](domain-events.md)) y están sujetos al cooldown por evento y agregado (`job:{JobName}`).

### Manejo de Errores

Cuando `RetransmitPendingDocuments` falla (error global, no por documento), el job distingue dos casos:

| Error | Comportamiento |
|---|---|
| `context.DeadlineExceeded` | Se registra timeout en logs + se publica `RetransmissionJobFailedEvent` (`JobName = contingency_retransmission`) |
| Cualquier otro error | Se registra fallo en logs + se publica `RetransmissionJobFailedEvent` (`JobName = contingency_retransmission`) |

Los errores por documento (un NIT, un lote o un tipo de DTE que fallan) se registran en logs y no abortan el resto de la ejecución.

La publicación del evento activa el notificador de email al administrador (ver [Eventos de Dominio](domain-events.md)).

### Configuración

| Campo | Valor por Defecto |
|---|---|
| `MaxExecutionTime` | 10 minutos |
| Frecuencia de ejecución | 30 min (producción) o 5 min (otros ambientes), ver arriba |

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
| `metrics:*:counters` | Cadena JSON con los contadores (`total_requests`, `success_count`, `error_count`) |

---

## ReservationCleanerJob

> **Archivo:** `reservation_cleaner_job.go`

### Propósito

Libera reservas de números de control que expiraron sin ser utilizadas. Esto ocurre cuando una solicitud de DTE reserva un número secuencial pero falla antes de completar la emisión, dejando la reserva en estado `RESERVED` indefinidamente.

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
  ├── [2] Contexto con timeout de MaxExecutionTime (5 min)
  │
  ├── [3] Obtener la conexión a base de datos y ejecutar Ping
  │
  ├── [4] Obtener reservas expiradas
  │     repository.GetExpiredNonContingencyReservations(ctx)
  │     (status RESERVED, is_contingency = false, expires_at vencido)
  │     └── Sin reservas → log Info + salir
  │
  ├── [5] Por cada reserva expirada:
  │     repository.UpdateStatus(
  │       branchID, dteType, sequenceNumber, year,
  │       status=RELEASED, releasedAt=now
  │     )
  │     ├── Éxito → log Info "auto-released"
  │     └── Error → log Error + continuar con la siguiente
  │
  └── [6] Liberar flag de ejecución
```

Un error global (por ejemplo, al consultar las reservas) se registra; si es un timeout se registra como tal. No se publica ningún evento.

### Estado de Reservas

| Estado | Descripción |
|---|---|
| `RESERVED` | Reserva activa, número en uso |
| `CONFIRMED` | Reserva usada por un documento aceptado |
| `RELEASED` | Reserva liberada por expiración (acción de este job) o por rechazo; el correlativo puede reutilizarse |

### Configuración

| Campo | Valor por Defecto |
|---|---|
| `MaxExecutionTime` | 5 minutos |
| Frecuencia de ejecución | Cada 10 minutos |

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
4. **Registro en scheduler**: Los jobs no son autocontenidos — deben registrarse en el scheduler de `cmd/setup/jobs.go`, donde se define la frecuencia de cada uno.
5. **Documentos vencidos**: Un documento en `EXPIRED` no vuelve a ser candidato de retransmisión: `GetPending` solo toma documentos con estado `PENDING`.
