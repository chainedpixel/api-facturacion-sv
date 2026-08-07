# Punto de Entrada y Jobs — cmd/

**Ubicación:** `cmd/`

## Descripción General

El directorio `cmd/` contiene el punto de entrada de la aplicación (`main.go`) y la configuración de trabajos asíncronos (`setup/jobs.go`) que se ejecutan en segundo plano.

---

## main.go — Entry Point

> **Archivo:** `cmd/main.go`
> **Paquete:** `main`

```go
func main() {
    app := bootstrap.NewApplication()

    if err := app.Initialize(); err != nil {
        log.Fatal(err)
    }

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Flujo de Inicialización

```
main()
  │
  ├── [1] bootstrap.NewApplication()
  │     → Crear instancia vacía de Application
  │
  ├── [2] app.Initialize()
  │     │
  │     ├── [2.1] FindProjectRoot()
  │     │     → Localizar raíz del proyecto
  │     │
  │     ├── [2.2] config.InitEnvConfig(rootPath)
  │     │     ├── Leer archivo .env con Viper
  │     │     ├── Mapear variables → structs
  │     │     └── Validar toda la configuración
  │     │
  │     ├── [2.3] logs.InitLogger()
  │     │     → Configurar logger estructurado
  │     │
  │     ├── [2.4] utils.TimeInit()
  │     │     → Inicializar zona horaria (America/El_Salvador)
  │     │
  │     ├── [2.5] config.InitTranslations()
  │     │     → Cargar archivos YAML de traducción
  │     │
  │     ├── [2.6] initDatabaseConfigurations()
  │     │     ├── selectDatabaseDriver() → MySQL o PostgreSQL
  │     │     ├── NewDatabaseConnection(driver)
  │     │     ├── dbConnection.Open()
  │     │     └── RunMigrations() (si RUN_MIGRATION=true)
  │     │
  │     ├── [2.7] containers.NewContainer()
  │     │     → Crear contenedor de inyección de dependencias
  │     │
  │     ├── [2.8] container.Initialize()
  │     │     ├── RepositoryContainer.Initialize()
  │     │     ├── ServicesContainer.Initialize()
  │     │     ├── UseCaseContainer.Initialize()
  │     │     ├── MiddlewareContainer.Initialize()
  │     │     └── HandlerContainer.Initialize()
  │     │
  │     ├── [2.9] server.Initialize()
  │     │     → Configurar servidor HTTP con rutas y middlewares
  │     │
  │     └── [2.10] setup.SetupJobs()
  │           → Programar trabajos asíncronos
  │
  └── [3] app.Start()
        │
        ├── Configurar canal de señales OS (SIGINT, SIGTERM)
        ├── Iniciar servidor HTTP en goroutine
        ├── Esperar señal de terminación
        └── Graceful shutdown (timeout: 30s)
              ├── server.Shutdown(ctx)
              └── dbConnection.Close()
```

---

## setup/jobs.go — Trabajos Asíncronos

> **Archivo:** `cmd/setup/jobs.go`
> **Paquete:** `setup`

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

| Parámetro | Producción (`"01"`) | Testing (`"00"`) |
|---|---|---|
| StartTime | 22:00 | 08:00 |
| EndTime | 05:00 | 17:00 |
| Interval | 30 min | 5 min |

### Jobs Programados

#### 1. RetransmissionJob — Retransmisión de Contingencia

```
Cada {interval} minutos:
  │
  ├── Buscar documentos de contingencia pendientes
  ├── Agrupar por NIT y tipo de DTE
  ├── Preparar y enviar evento de contingencia
  ├── Transmitir lote a Hacienda
  └── Actualizar estados en BD
```

- **Producción**: Se ejecuta cada 30 minutos (ventana nocturna 22:00–05:00)
- **Testing**: Se ejecuta cada 5 minutos (horario laboral 08:00–17:00)

#### 2. ReservationCleanerJob — Limpieza de Reservaciones

```
Cada 10 minutos:
  │
  ├── Buscar reservaciones expiradas no contingentes
  └── Liberar números de control para reutilización
```

- Se ejecuta cada **10 minutos** independiente del ambiente
- Se liberan números de control que fueron reservados pero nunca confirmados ni enviados como contingencia

#### 3. MetricsCleanupJob — Limpieza de Métricas en Cache

```
Cada 6 horas:
  │
  ├── Escanear keys de tipo metrics:*:durations en Redis
  ├── Establecer TTL de 24 horas en keys sin expiración
  ├── Escanear keys de tipo metrics:*:counters en Redis
  └── Establecer TTL de 24 horas en keys sin expiración
```

- Se ejecuta cada **6 horas** independiente del ambiente
- Se resuelve el problema de keys de métricas que persisten indefinidamente en Redis sin TTL
- Las listas de duración (`metrics:{NIT}:{METHOD}:{ENDPOINT}:durations`) se almacenaban sin expiración, causando crecimiento descontrolado de memoria en Redis
- El job actúa como red de seguridad complementaria al TTL que ahora se establece en el middleware de métricas al registrar cada request

### Función Principal

```go
func SetupJobs(
    contingencyService contingency.ContingencyManager,
    reservedSequenceRepo dte_documents.ReservedSequenceRepositoryPort,
    ambientCode string,
    connection *drivers.DbConnection,
    cache ports.CacheManager,
) error
```

Se usa la librería `gocron` para la programación. El scheduler se inicia de forma asíncrona y los jobs continúan ejecutándose en segundo plano durante toda la vida de la aplicación.

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
func selectDatabaseDriver() drivers.DriverConfig
```

| Driver Config | `DB_DRIVER` |
|---|---|
| `MysqlDriver` | `"mysql"` |
| `PostgresDriver` | `"postgres"` |
| `nil` (error) | Cualquier otro valor |

---

## Notas

1. **Orden de inicialización importa**: La configuración debe cargarse antes que el logger, el logger antes que la BD, la BD antes que los contenedores, etc.
2. **Graceful shutdown**: El servidor espera hasta 30 segundos para que las peticiones en curso finalicen antes de cerrarse.
3. **Migraciones opcionales**: `RUN_MIGRATION=true` ejecuta AutoMigrate al iniciar. En producción se recomienda `false` después del deploy inicial.
4. **Jobs asíncronos**: Se ejecutan en goroutines independientes. No bloquean el servidor HTTP.
5. **Zona horaria**: El sistema se inicializa con `America/El_Salvador` como zona horaria por defecto para todas las operaciones de fecha/hora.
