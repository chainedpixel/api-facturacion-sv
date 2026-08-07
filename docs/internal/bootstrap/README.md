# Capa de Bootstrap

El paquete `internal/bootstrap` es responsable de iniciar la aplicación: conectar cada dependencia, inicializar las conexiones de infraestructura y entregar el grafo de objetos compuesto al servidor HTTP.

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
package main

import (
    "github.com/chainedpixel/ordo-factus/internal/bootstrap"
    "github.com/chainedpixel/ordo-factus/pkg/shared/logs"
    "os"
)

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

`bootstrap.NewApplication()` crea el struct `Application`. `Initialize()` abre la conexión a la base de datos y llama a `Container.Initialize()`. `Start()` registra las rutas e inicia el listener HTTP.

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

El `sync.RWMutex` protege los campos del contenedor ante lecturas concurrentes durante el apagado controlado, aunque en la práctica `Initialize()` se llama una sola vez al arranque antes de que cualquier goroutine acceda a los contenedores.

## Orden de Inicialización (Grafo de Dependencias)

`Container.Initialize()` impone un orden de construcción estricto que refleja el grafo de dependencias:

```
Conexión BD
    │
    └── RepositoryContainer
            │  authRepo, sequentialNumberRepo, failedSequentialNumberRepo,
            │  reservedSequenceRepo, dteRepo, contingencyRepo
            │
            └── ServicesContainer
                    │  cryptManager → cacheManager → tokenManager → authManager
                    │  signerManager → haciendaAuthManager → transmitterManager
                    │  dteManager, sequentialManager
                    │  invoiceManager, ccfManager, retentionManager,
                    │  creditNoteManager, remissionNoteManager, fseManager, debitNoteManager
                    │  invalidationManager
                    │  transmitterBatchManager → contingencyEventManager → contingencyManager
                    │  testManager, metricsManager, healthManager
                    │
                    ├── UseCaseContainer
                    │       authUseCase, baseTransmitter, dteConsult
                    │       dteUseCaseFactory → invoiceUseCase, ccfUseCase, …
                    │
                    ├── MiddlewareContainer
                    │       corsMid, tokenMid, errorMid, authMid,
                    │       metricMid, dbMid, timeoutMid
                    │
                    └── HandlerContainer
                            authHandler, dteHandler, healthHandler,
                            testHandler, metricsHandler, contingencyHandler
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
        return err   // Fallo de Redis, JWT secret faltante, etc.
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

Solo `ServicesContainer.Initialize()` devuelve un error porque realiza conexiones externas (Redis). Todos los demás contenedores se inicializan sincrónicamente sin I/O.

## Cómo Arranca la Aplicación

1. `main()` llama a `app.Initialize()`.
2. `Initialize()` abre una conexión a la base de datos mediante `drivers.NewDbConnection()`.
3. Se llama a `Container.Initialize()` — ver orden arriba.
4. `app.Start()` crea el router de Gorilla Mux, registra todas las rutas del `HandlerContainer`, adjunta todo el middleware del `MiddlewareContainer` y llama a `http.ListenAndServe`.

Si algo falla en los pasos 2–3, el proceso registra un mensaje fatal y sale con código 1. No existe inicio parcial ni lógica de reintento en el arranque.

## Decisiones de Diseño Clave

**Sin variables globales.** Cada dependencia se pasa explícitamente. No hay funciones `init()` que muten estado global (excepto la inicialización del logger en `pkg/shared/logs`).

**Surfacing de errores.** `ServicesContainer.Initialize()` es el único lugar donde un error puede escapar durante el arranque. Todos los demás constructores entran en pánico ante errores de programador (ej. puntero nulo por mala configuración) o simplemente tienen éxito.

**Accessors de contenedor.** Cada sub-contenedor expone sus servicios construidos mediante métodos de acceso tipados. Los llamadores nunca acceden a los campos directamente:

```go
func (c *Container) Services() *ServicesContainer { return c.services }
func (c *Container) UseCases() *UseCaseContainer  { return c.useCases }
func (c *Container) Handlers() *HandlerContainer   { return c.handlers }
func (c *Container) Middleware() *MiddlewareContainer { return c.middleware }
```

Esto mantiene el código de registro de rutas legible mientras previene la mutación accidental de los internos del contenedor.
