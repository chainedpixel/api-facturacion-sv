# Punto de Entrada y Jobs — cmd/

**Ubicación:** `cmd/`

## Descripción General

El directorio `cmd/` contiene el punto de entrada de la aplicación (`main.go`) y la configuración de trabajos asíncronos (`setup/jobs.go`) que se ejecutan en segundo plano.

```
cmd/
├── main.go          # Punto de entrada
└── setup/
    └── jobs.go      # Programación de jobs (gocron)
```

---

## main.go — Entry Point

> **Archivo:** `cmd/main.go`
> **Paquete:** `main`

```go
func main() {
    app := bootstrap.NewApplication()
    if err := app.Initialize(); err != nil {
        logs.Fatal("Failed to initialize application", map[string]interface{}{"error": err.Error()})
        os.Exit(1)
    }

    if err := app.Start(); err != nil {
        logs.Fatal("Application error", map[string]interface{}{"error": err.Error()})
        os.Exit(1)
    }
}
```

Un error en cualquiera de las dos fases se registra con `logs.Fatal` y el proceso termina con código 1.

### Flujo de Inicialización

```
main()
  │
  ├── [1] bootstrap.NewApplication()
  │     → Crear instancia vacía de Application
  │
  ├── [2] app.Initialize()
  │     │
  │     ├── [2.1] utils.FindProjectRoot()
  │     │     → Localizar raíz del proyecto
  │     │
  │     ├── [2.2] config.InitEnvConfig(rootPath)
  │     │     ├── Leer archivo .env con Viper
  │     │     ├── Mapear variables → structs
  │     │     └── Validar toda la configuración
  │     │
  │     ├── [2.3] logs.InitLogger(config.Log.Level, config.Log.Path)
  │     │     → Configurar logger estructurado
  │     │
  │     ├── [2.4] utils.TimeInit()
  │     │     → Inicializar zona horaria (America/El_Salvador)
  │     │
  │     ├── [2.5] config.InitTranslations(<raíz>/<APP_LANG_PATH>, <APP_LANG>)
  │     │     → Cargar archivos YAML de traducción (assets/i18n)
  │     │
  │     ├── [2.6] initDatabaseConfigurations()
  │     │     ├── selectDatabaseDriver() → MySQL o PostgreSQL
  │     │     ├── NewDatabaseConnection(driver)
  │     │     ├── dbConnection.Open()
  │     │     └── RunMigrations() (si RUN_MIGRATION=true)
  │     │
  │     ├── [2.7] containers.NewContainer(dbConnection)
  │     │     → Crear contenedor de inyección de dependencias
  │     │
  │     ├── [2.8] container.Initialize()
  │     │     ├── RepositoryContainer.Initialize()
  │     │     ├── ServicesContainer.Initialize()
  │     │     ├── UseCaseContainer.Initialize()
  │     │     ├── MiddlewareContainer.Initialize()
  │     │     └── HandlerContainer.Initialize()
  │     │
  │     ├── [2.9] server.Initialize(container)
  │     │     → Crear el servidor HTTP (el router y los middlewares se configuran en Start)
  │     │
  │     └── [2.10] setup.SetupJobs(...)
  │           → Programar trabajos asíncronos
  │
  └── [3] app.Start()
        │
        ├── Configurar canal de señales OS (SIGINT, SIGTERM)
        ├── Iniciar servidor HTTP en goroutine (server.Start: ConfigureRoutes + ListenAndServe)
        ├── Esperar señal de terminación o error del servidor
        └── Graceful shutdown (timeout: 30s)
              ├── server.Shutdown(ctx)
              └── dbConnection.Close()
```

El detalle de cada contenedor está en [Bootstrap](../internal/bootstrap/README.md).

---

## setup/jobs.go — Trabajos Asíncronos

> **Archivo:** `cmd/setup/jobs.go`
> **Paquete:** `setup`

Los jobs se programan con la librería `gocron` (`gocron.NewScheduler(time.UTC)`) y se inician con `scheduler.StartAsync()`. Cada job corre en goroutines del scheduler y no bloquea el servidor HTTP. Los tres jobs viven en `internal/infrastructure/jobs/`.

### Configuración de Jobs

```go
type JobConfig struct {
    StartTime   string    // Hora de inicio (HH:MM)
    EndTime     string    // Hora de fin (HH:MM)
    Interval    int       // Intervalo en minutos
    Environment string    // Nombre del ambiente
}
```

### Configuración por Ambiente

La selección depende de `config.Server.AmbientCode` (`MH_AMBIENT_CODE`): `"01"` usa la configuración de producción y cualquier otro valor la de pruebas.

| Parámetro | Producción (`"01"`) | Pruebas (cualquier otro valor, p. ej. `"00"`) |
|---|---|---|
| StartTime | 22:00 | 08:00 |
| EndTime | 05:00 | 17:00 |
| Interval | 30 min | 5 min |
| Environment | `production` | `testing` |

`StartTime` y `EndTime` solo se escriben en el log al programar los jobs: `ScheduleContingencyJob` programa la ejecución únicamente con `Interval` (`scheduler.Every(Interval).Minutes()`), de modo que la retransmisión corre a lo largo de todo el día y no se limita a una ventana horaria.

### Jobs Programados

#### 1. RetransmissionJob — Retransmisión de Contingencia

```
Cada {Interval} minutos:
  │
  ├── Si ya hay una ejecución en curso → se omite (IsRunning)
  ├── Contexto con tiempo máximo de 10 minutos
  └── ContingencyManager.RetransmitPendingDocuments(ctx):
        ├── Obtener documentos de contingencia pendientes (hasta MH_MAX_BATCH_SIZE)
        ├── Marcar como EXPIRED los que superan las 72 horas desde el sello del evento
        ├── Alertar si la contingencia supera 3 días (reporte técnico requerido)
        ├── Agrupar por NIT del sistema y tipo de DTE
        ├── Preparar y enviar el evento de contingencia por NIT
        │     └── Si el evento ya existe: verificar el estado de los documentos en Hacienda
        └── Transmitir los documentos por lotes por tipo de DTE y actualizar estados
```

- **Producción**: Se ejecuta cada 30 minutos
- **Pruebas**: Se ejecuta cada 5 minutos
- Si la ejecución falla (incluido el timeout de 10 minutos), registra el error y publica un `RetransmissionJobFailedEvent` (`job_name: contingency_retransmission`) en el bus de eventos, que el `AdminEmailHandler` convierte en un correo al administrador.

#### 2. ReservationCleanerJob — Limpieza de Reservaciones

```
Cada 10 minutos:
  │
  ├── Si ya hay una ejecución en curso → se omite
  ├── Contexto con tiempo máximo de 5 minutos
  ├── Buscar reservaciones expiradas no contingentes
  └── Marcar cada una como liberada (ReservationStatusReleased)
```

- Se ejecuta cada **10 minutos** independiente del ambiente
- Se liberan números de control que fueron reservados pero nunca confirmados ni enviados como contingencia
- Un fallo al liberar una reservación se registra y se continúa con la siguiente

#### 3. MetricsCleanupJob — Limpieza de Métricas en Cache

```
Cada 6 horas:
  │
  ├── Si ya hay una ejecución en curso → se omite
  ├── Escanear keys con el patrón metrics:*:durations en Redis
  ├── Establecer TTL de 24 horas en cada key encontrada
  ├── Escanear keys con el patrón metrics:*:counters en Redis
  └── Establecer TTL de 24 horas en cada key encontrada
```

- Se ejecuta cada **6 horas** independiente del ambiente
- Actúa como red de seguridad para las keys de métricas que pudieran quedar sin expiración; el job asigna el TTL de 24 horas a todas las keys que coinciden con los patrones, tengan o no expiración previa
- Las listas de duración siguen el formato `metrics:{NIT}:{METHOD}:{ENDPOINT}:durations`

### Función Principal

```go
func SetupJobs(
    contingencyService contingency.ContingencyManager,
    reservedSequenceRepo dte_documents.ReservedSequenceRepositoryPort,
    ambientCode string,
    connection *drivers.DbConnection,
    cache ports.CacheManager,
    bus event.Bus,
) error
```

Si `bus` no es nulo, se inyecta en el `RetransmissionJob` (`SetEventBus`). `Initialize` la invoca con `ContingencyManager`, `ReservedSequenceRepo`, `config.Server.AmbientCode`, la conexión a la base de datos, `CacheManager` y `EventBus` del contenedor. Si la programación de cualquier job falla, devuelve el error y la aplicación no arranca.

---

## Application Struct

> **Archivo:** `internal/bootstrap/app.go`

```go
type Application struct {
    server       *server.Server
    container    *containers.Container
    dbConnection *drivers.DbConnection
}
```

| Campo | Propósito |
|---|---|
| `server` | Servidor HTTP (Gorilla Mux) |
| `container` | Contenedor de inyección de dependencias |
| `dbConnection` | Conexión a la base de datos (GORM) |

### Selección de Driver de BD

```go
func (app *Application) selectDatabaseDriver() drivers.DriverConfig
```

| Driver Config | `DB_DRIVER` |
|---|---|
| `MysqlDriver` | `"mysql"` |
| `PostgresDriver` | `"postgres"` |
| `nil` (error `ErrUnrecognizedDriver`) | Cualquier otro valor |

---

## Notas

1. **Orden de inicialización importa**: La configuración debe cargarse antes que el logger, el logger antes que la BD, la BD antes que los contenedores, etc.
2. **Graceful shutdown**: Con `SIGINT` o `SIGTERM`, el servidor espera hasta 30 segundos para que las peticiones en curso finalicen antes de cerrarse; luego se cierra la conexión a la base de datos. El scheduler de jobs no se detiene explícitamente.
3. **Migraciones opcionales**: `RUN_MIGRATION=true` ejecuta AutoMigrate al iniciar. En producción se recomienda `false` después del deploy inicial.
4. **Jobs asíncronos**: Se ejecutan en goroutines independientes. No bloquean el servidor HTTP. Cada job impide ejecuciones solapadas con un indicador atómico (`IsRunning`).
5. **Zona horaria**: El sistema se inicializa con `America/El_Salvador` como zona horaria por defecto para todas las operaciones de fecha/hora (`utils.TimeNow`). El scheduler de jobs usa UTC.
