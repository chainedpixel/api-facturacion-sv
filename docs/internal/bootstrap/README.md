# Capa de Bootstrap

El paquete `internal/bootstrap` es responsable de iniciar la aplicación: conectar cada dependencia, inicializar las conexiones de infraestructura y entregar el grafo de objetos compuesto al servidor HTTP.

## Estructura

```
internal/bootstrap/
├── app.go                      # Application: ciclo de vida (Initialize / Start / apagado)
└── containers/
    ├── container.go            # Container: agrupa los cinco contenedores
    ├── repositories.go         # RepositoryContainer
    ├── services.go             # ServicesContainer
    ├── use_cases.go            # UseCaseContainer
    ├── handlers.go             # HandlerContainer
    └── middlewares.go          # MiddlewareContainer
```

## Por Qué Inyección Manual de Dependencias

El proyecto no utiliza ningún framework de DI (ni Wire, ni Fx, ni Dig). Cada dependencia se construye y conecta explícitamente dentro de los cinco structs de contenedor. Esta decisión fue deliberada:

- El grafo de dependencias completo cabe en una pantalla por archivo de contenedor — sin magia de anotaciones que rastrear.
- Los errores en tiempo de compilación detectan argumentos faltantes o con tipo incorrecto de inmediato.
- Agregar un nuevo servicio significa escribir una línea; no hay archivos generados que regenerar ni conjuntos de proveedores que actualizar.
- Los mensajes de error en tiempo de arranque son claros porque el error se devuelve exactamente desde el constructor que falló.

La contrapartida es que `services.go` crece con el proyecto, pero dado el dominio acotado (una API de facturación electrónica para El Salvador) ese costo es manejable.

## Punto de Entrada

La aplicación arranca en `cmd/main.go`:

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

`bootstrap.NewApplication()` crea el struct `Application` vacío. `Initialize()` carga la configuración y el entorno, abre la conexión a la base de datos, construye el `Container`, crea el servidor HTTP y programa los jobs. `Start()` arranca el servidor HTTP en una goroutine y espera una señal de apagado.

## La Estructura `Application`

```go
type Application struct {
    server       *server.Server
    container    *containers.Container
    dbConnection *drivers.DbConnection
}

var SupportedDrivers = map[string]drivers.DriverConfig{
    "mysql":    drivers.NewMysqlDriver(),
    "postgres": drivers.NewPostgresDriver(),
}
```

### `Initialize()`

Orden real de inicialización:

| Paso | Acción | Fallo |
|---|---|---|
| 1 | `utils.FindProjectRoot()` | — |
| 2 | `config.InitEnvConfig(rootPath)`: lee `.env`, mapea y valida las variables (incluye `MH_MAX_BATCH_SIZE` entre 1 y 100 y `MH_EXTRA_HOLIDAYS` con fechas `YYYY-MM-DD`) | Devuelve error |
| 3 | `logs.InitLogger(config.Log.Level, config.Log.Path)` | Devuelve error |
| 4 | `utils.TimeInit()`: zona horaria `America/El_Salvador` | Registra fatal y devuelve error |
| 5 | `config.InitTranslations(<raíz>/<APP_LANG_PATH>, <APP_LANG>)`: mensajes de `assets/i18n` | `log.Fatalf` (termina el proceso) |
| 6 | `initDatabaseConfigurations()`: selecciona el driver (`SupportedDrivers[config.Database.Driver]`), crea y abre la conexión y, si `RunMigration` está activo, ejecuta `database.RunMigrations` | Registra fatal y devuelve error (driver desconocido: `ErrUnrecognizedDriver`) |
| 7 | `containers.NewContainer(dbConnection)` y `container.Initialize()` | Devuelve error |
| 8 | `server.Initialize(container)`: crea el servidor (el router se configura al arrancar) | — |
| 9 | `setup.SetupJobs(ContingencyManager, ReservedSequenceRepo, AmbientCode, dbConnection, CacheManager, EventBus)` | Devuelve error |

### `Start()`

1. Registra `SIGINT` y `SIGTERM`.
2. Lanza `server.Start()` en una goroutine; esta llama a `ConfigureRoutes()` (middlewares globales, rutas públicas y protegidas, rutas de debug si `DEBUG=true`) y a `ListenAndServe` en `:<SERVER_PORT>` (timeouts de lectura y escritura de 15 s e inactividad de 60 s).
3. Espera un error del servidor o una señal. Con una señal ejecuta el apagado controlado con un contexto de 30 segundos: `server.Shutdown(ctx)` y luego `dbConnection.Close()`.

## La Jerarquía de Contenedores

Todos los contenedores viven en `internal/bootstrap/containers/`. Un único struct `Container` de nivel superior los agrupa a todos:

```go
// internal/bootstrap/containers/container.go
type Container struct {
    connection   *drivers.DbConnection
    repositories *RepositoryContainer
    services     *ServicesContainer
    useCases     *UseCaseContainer
    handlers     *HandlerContainer
    middleware   *MiddlewareContainer
    mu           sync.RWMutex
}
```

El `sync.RWMutex` solo se toma en `Initialize()` (bloqueo de escritura). Los métodos de acceso (`Repositories()`, `Services()`, etc.) no lo usan; en la práctica `Initialize()` se llama una sola vez al arranque, antes de que el servidor atienda peticiones.

## Orden de Inicialización (Grafo de Dependencias)

`Container.Initialize()` impone un orden de construcción estricto que refleja el grafo de dependencias:

```
Conexión BD
    │
    └── RepositoryContainer
            │  authRepo, sequentialNumberRepo, reservedSequenceRepo,
            │  dteRepo, dteEventRepo, contingencyRepo, failedSequentialNumberRepo
            │
            └── ServicesContainer
                    │  cryptManager → cacheManager (Redis) → tokenManager → authManager
                    │  signerManager → haciendaAuthManager → transmitterManager
                    │  dteManager, sequentialManager
                    │  invoiceManager, ccfManager, invalidationManager (con plazos y calendario),
                    │  retentionManager, creditNoteManager, remissionNoteManager,
                    │  fseManager, debitNoteManager
                    │  testManager, metricsManager
                    │  eventBus → healthManager
                    │  transmitterBatchManager → contingencyEventManager → contingencyManager
                    │  initNotificationStack (mailer, renderer, cooldown, AdminEmailHandler)
                    │  attachEventBusToPublishers
                    │
                    ├── UseCaseContainer
                    │       authUseCase, baseTransmitter, dteConsult
                    │       dteUseCaseFactory → invoiceUseCase, ccfUseCase, retentionUseCase,
                    │       creditNoteUseCase, remissionNoteUseCase, fseUseCase, debitNoteUseCase,
                    │       invalidationUseCase
                    │
                    ├── MiddlewareContainer
                    │       corsMid, tokenMid, errorMid, authMid,
                    │       metricMid, dbMid, timeoutMid
                    │
                    └── HandlerContainer
                            contingencyHandler, healthHandler, testHandler, authHandler,
                            metricsHandler, dteHandler (con el handler genérico de creación),
                            debugNotifyHandler
```

La secuencia de llamadas dentro de `Container.Initialize()` es:

```go
func (c *Container) Initialize() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.repositories = NewRepositoryContainer(c.connection)
    c.repositories.Initialize()

    c.services = NewServicesContainer(c.repositories)
    if err := c.services.Initialize(); err != nil {
        return err   // Fallo de Redis al crear la caché
    }

    c.useCases = NewUseCaseContainer(c.services)
    c.useCases.Initialize()

    c.middleware = NewMiddlewareContainer(c.services, c.connection)
    c.middleware.Initialize()

    c.handlers = NewHandlerContainer(c.useCases, c.services)
    c.handlers.Initialize()

    return nil
}
```

Solo `ServicesContainer.Initialize()` devuelve un error: lo produce `cache.NewRedisTokenCache`. Los demás contenedores no devuelven errores.

## Detalle de Cada Contenedor

### RepositoryContainer

Construye los repositorios GORM (`repositories.New*`) con `connection.Db`:

| Campo | Constructor | Puerto |
|---|---|---|
| `authRepo` | `NewAuthRepository` | `auth.AuthRepositoryPort` |
| `sequentialNumberRepo` | `NewControlNumberRepository` | `ports.SequentialNumberRepositoryPort` |
| `reservedSequenceRepo` | `NewReservedSequenceRepository` | `dte_documents.ReservedSequenceRepositoryPort` |
| `dteRepo` | `NewDTERepository` | `dte_documents.DTERepositoryPort` |
| `dteEventRepo` | `NewDTEEventRepository` | `dte_documents.DTEEventRepositoryPort` |
| `contingencyRepo` | `NewContingencyRepository` | `contingency.ContingencyRepositoryPort` |
| `failedSequentialNumberRepo` | `NewFailedSequenceNumberRepository` | `ports.FailedSequenceNumberRepositoryPort` |

### ServicesContainer

Cableado real en `Initialize()`:

| Servicio | Construcción |
|---|---|
| `cryptManager` | `crypt.NewCryptService()` |
| `cacheManager` | `cache.NewRedisTokenCache(config.NewRedisConfig(), cryptManager)` (único punto que puede fallar) |
| `tokenManager` | `tokens.NewJWTService(config.Server.JWTSecret, cacheManager)` |
| `authManager` | `strategies.NewAuthService(tokenManager, authRepo, cacheManager)` |
| `signerManager` | `signer.NewDTESigner(authRepo)` |
| `haciendaAuthManager` | `signing.NewHaciendaAuthService(cacheManager, authManager)` |
| `transmitterManager` | `adapterTransmitter.NewMHTransmitter(haciendaAuthManager, failedSequentialNumberRepo)` |
| `dteManager` | `dte_documents.NewDTEService(dteRepo, dteEventRepo)` |
| `sequentialManager` | `dte_documents.NewSequentialNumberService(sequentialNumberRepo, authRepo, reservedSequenceRepo)` |
| `invoiceManager`, `ccfManager`, `fseManager` | `NewInvoiceService`, `NewCCFService`, `NewFSEService` con `sequentialManager` |
| `retentionManager`, `creditNoteManager`, `remissionNoteManager`, `debitNoteManager` | Servicios correspondientes con `sequentialManager` y `dteManager` |
| `invalidationManager` | `invalidation.NewInvalidationService(dteManager, newInvalidationDeadlines())` |
| `testManager` | `adapterTest.NewTestService(db, authRepo)` |
| `metricsManager` | `adapterMetric.NewMetricService(cacheManager)` |
| `eventBus` | `adapterEvents.NewInMemoryBus(NewEventRepository(db))` |
| `healthManager` | `adapterHealth.NewHealthService` con la BD y el bus |
| `transmitterBatchManager` | `batch.NewBatchTransmitterService(haciendaAuthManager, signerManager, contingencyRepo, sequentialManager, config, RealTimeProvider, connection)` con `TransmissionConfig` de 5 s de reintento inicial, 2 min máximo y factor 2.0 |
| `contingencyEventManager` | `adapterContingecy.NewContingencyEventService(...)` |
| `contingencyManager` | `contingency.NewContingencyManager(...)` |

Después de construir los servicios:

- `initNotificationStack()` crea el mailer SMTP, el renderer de plantillas y el cooldown (Redis, con `NOTIFY_COOLDOWN_MINUTES`) y suscribe el `AdminEmailHandler` a `contingency.activated`, `emission.failure` y `retransmission_job.failed`. Si las plantillas no cargan, las notificaciones quedan deshabilitadas.
- `attachEventBusToPublishers()` inyecta el bus en `ContingencyService` y `BatchTransmitterService` (`SetEventBus`).

#### Calendario de días hábiles y plazos de invalidación

`newInvalidationDeadlines()` construye la política de plazos de invalidación:

```go
func newInvalidationDeadlines() *policy.Deadlines {
    holidays, err := calendar.ParseHolidays(config.Server.ExtraHolidays)
    if err != nil {
        holidays = nil
    }
    return policy.NewDeadlines(calendar.NewBusinessCalendar(holidays))
}
```

- `MH_EXTRA_HOLIDAYS` es una lista de fechas `YYYY-MM-DD` separadas por comas (vacía por defecto). `config.InitEnvConfig` ya la valida al arranque, por lo que un valor inválido detiene el inicio antes de llegar aquí; el `holidays = nil` es solo un respaldo.
- `calendar.BusinessCalendar` considera día hábil de lunes a viernes, excluyendo los asuetos nacionales fijos (1 de enero, 1 y 10 de mayo, 17 de junio, 6 de agosto, 15 de septiembre, 2 de noviembre y 25 de diciembre), la Semana Santa (de Lunes Santo a Lunes de Pascua) y los asuetos adicionales configurados.
- `policy.Deadlines` usa ese calendario para calcular el décimo día hábil del mes siguiente al sello (plazo de invalidación de los DTE distintos de FE, FEXE y FSEE). Ver [Invalidación](../domain/dte/invalidation.md).
- El calendario se crea una sola vez al arranque: cambiar `MH_EXTRA_HOLIDAYS` requiere reiniciar el servicio.

### UseCaseContainer

```go
c.authUseCase = auth.NewAuthUseCase(AuthManager(), CryptManager())
c.baseTransmitter = dte.NewBaseTransmitter(TransmitterManager(), SignerManager())
c.dteConsult = dte.NewDTEConsultUseCase(DTEManager())
c.dteUseCaseFactory = dte.NewDTEUseCaseFactory(AuthManager(), DTEManager(), baseTransmitter, SequentialManager(), EventBus())
```

La factory produce los casos de uso genéricos (factura, CCF, retención, nota de crédito, nota de remisión, FSE y nota de débito) y el de invalidación. Ver [Casos de uso](../application/README.md).

### MiddlewareContainer

Construye los middlewares CORS (orígenes `*`), extractor de token, errores, autenticación (con `TokenManager`), métricas (con `CacheManager`), conexión a BD y timeout.

### HandlerContainer

Construye los handlers y registra los endpoints de creación en el handler genérico (`initializeGenericCreatorHandler`):

| Endpoint | Caso de uso | `UsesContingency` |
|---|---|---|
| `/dte/invoices` | `InvoiceUseCase` | `true` |
| `/dte/ccf` | `CCFUseCase` | `true` |
| `/dte/creditnote` | `CreditNoteUseCase` | `false` |
| `/dte/retention` | `RetentionUseCase` | `false` |
| `/dte/remissionnote` | `RemissionNoteUseCase` | `true` |
| `/dte/fse` | `FSEUseCase` | `true` |
| `/dte/debitnote` | `DebitNoteUseCase` | `true` |

`DTEHandler` recibe además `DTEConsultUseCase` e `InvalidationUseCase`. `DebugNotifyHandler` se construye siempre, pero su ruta solo se registra si `DEBUG=true`.

## Cómo Arranca la Aplicación

1. `main()` llama a `app.Initialize()`.
2. `Initialize()` ejecuta los pasos de la tabla anterior (configuración, logger, zona horaria, traducciones, base de datos, contenedores, servidor, jobs).
3. `main()` llama a `app.Start()`, que lanza el servidor HTTP y espera una señal.

Si algo falla durante `Initialize()`, el proceso registra un mensaje fatal y sale con código 1. No existe inicio parcial ni lógica de reintento en el arranque.

## Decisiones de Diseño Clave

**Estado global acotado.** Las dependencias de negocio se pasan explícitamente por constructor. Existe estado global de proceso en la configuración (`config.Server`, `config.Redis`, `config.MHPaths`, etc.), en el logger (`pkg/shared/logs`), en la zona horaria (`utils.TimeNow`) y en el sistema de traducciones; esos paquetes se inicializan en `Application.Initialize` antes de construir los contenedores.

**Surfacing de errores.** `ServicesContainer.Initialize()` es el único constructor de contenedor que devuelve un error durante el arranque. Los demás pasos fatales (configuración, logger, zona horaria, base de datos, jobs) ocurren en `Application.Initialize`.

**Accessors de contenedor.** Cada sub-contenedor expone sus servicios construidos mediante métodos de acceso tipados. Los llamadores nunca acceden a los campos directamente:

```go
func (c *Container) Repositories() *RepositoryContainer { return c.repositories }
func (c *Container) Services() *ServicesContainer       { return c.services }
func (c *Container) UseCases() *UseCaseContainer        { return c.useCases }
func (c *Container) Handlers() *HandlerContainer        { return c.handlers }
func (c *Container) Middleware() *MiddlewareContainer   { return c.middleware }
```

Esto mantiene el código de registro de rutas legible mientras previene la mutación accidental de los internos del contenedor.
