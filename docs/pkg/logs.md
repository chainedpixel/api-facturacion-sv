# Sistema de Logging

> **Paquete:** `pkg/shared/logs`
> **Archivo:** `logger.go`

## Descripción General

El sistema de logging se basa en la librería `logrus` con un formatter personalizado que agrega colores, timestamps estructurados y soporte para campos contextuales. Se escribe simultáneamente a stdout y a un archivo de log.

---

## Inicialización

```go
func InitLogger(logLevel, logPath string) error
```

```
InitLogger("debug", "/assets/logs/")
  │
  ├── [1] Crear un nuevo Logger (logrus.New())
  │
  ├── [2] Configurar formatter personalizado
  │     CustomFormatter{
  │       TextFormatter{ DisableColors: false, FullTimestamp: true }
  │     }
  │
  ├── [3] Determinar nivel de log
  │     determineLogLevel(logLevel)
  │     → "debug" → DebugLevel
  │     → "info"  → InfoLevel
  │     → "warn"  → WarnLevel
  │     → "error" → ErrorLevel
  │     → "fatal" → FatalLevel
  │     → cualquier otro valor → InfoLevel
  │
  ├── [4] Configurar salida a stdout
  │     Logger.SetOutput(os.Stdout)
  │
  ├── [5] ¿logPath vacío? → terminar (solo stdout)
  │
  ├── [6] Crear directorio de logs (si no existe)
  │     os.MkdirAll(FindProjectRoot() + logPath)
  │
  ├── [7] Crear/abrir archivo de log
  │     os.OpenFile("dte_microservice.log", O_APPEND|O_CREATE|O_WRONLY)
  │
  └── [8] Salida dual
        Logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
```

Si no se puede crear el directorio o abrir el archivo, `InitLogger` retorna `nil` y el logger sigue escribiendo solo a stdout (el error no se propaga).

### Archivo de Log

- **Nombre:** `dte_microservice.log`
- **Ubicación:** `{projectRoot}{LOG_PATH}/dte_microservice.log` (`LOG_PATH` por defecto `/assets/logs/`)
- **Modo:** Append (no sobreescribe al reiniciar)

---

## Funciones de Log

| Función | Nivel | Uso |
|---|---|---|
| `Debug(msg, fields...)` | DEBUG | Información detallada para desarrollo |
| `Info(msg, fields...)` | INFO | Eventos normales del sistema |
| `Warn(msg, fields...)` | WARN | Situaciones inesperadas pero recuperables |
| `Error(msg, fields...)` | ERROR | Errores que requieren atención |
| `Fatal(msg, fields...)` | FATAL | Errores irrecuperables. Solo registra la entrada: no termina el proceso |

### Uso con Campos Contextuales

```go
// Log simple
logs.Info("Servidor iniciado")

// Log con campos
logs.Info("DTE transmitido", map[string]interface{}{
    "dteType":        "01",
    "generationCode": "UUID-1234",
    "status":         "PROCESADO",
    "duration":       "250ms",
})

// Log de error con contexto
logs.Error("Fallo transmisión", map[string]interface{}{
    "endpoint": "ReceptionURL",
    "statusCode": 500,
    "error": err.Error(),
})
```

---

## CustomFormatter

```go
type CustomFormatter struct {
    logrus.TextFormatter
}
```

Se personaliza la salida de cada entrada de log con colores ANSI y formato estructurado. El código de color se antepone a la línea y se restablece (`[0m`) después del nivel; los campos (si existen) se imprimen en un bloque `Details:`, uno por línea, sin orden garantizado (iteración de mapa).

### Formato de Salida

```
 2024-01-15 14:30:05 [LEVEL] Mensaje del log
	Details:
	key1      : value1
	key2      : value2
```

### Colores por Nivel

| Nivel | Color | Código ANSI |
|---|---|---|
| DEBUG | Blanco | `\033[37m` |
| INFO | Verde | `\033[32m` |
| WARN | Amarillo | `\033[33m` |
| ERROR | Rojo | `\033[31m` |
| FATAL | Magenta | `\033[35m` |

### Timestamp

- **Formato:** `2006-01-02 15:04:05`
- Se usa `entry.Time` de logrus (hora local del proceso); no se convierte explícitamente con `utils.TimeNow()`

---

## WriteHook

```go
type WriteHook struct {
    Writer    io.Writer
    Formatter logrus.Formatter
}
```

Se implementa la interfaz `logrus.Hook` para escribir logs a destinos adicionales.

| Método | Descripción |
|---|---|
| `Fire(entry)` | Se formatea la entrada y se escribe al Writer |
| `Levels()` | Se retornan todos los niveles de log |

> **Nota:** `WriteHook` está definido pero `InitLogger` no lo registra; la escritura al archivo se hace con `io.MultiWriter(os.Stdout, logFile)`.

### Flujo de Escritura Dual

```
logs.Info("mensaje")
  │
  └── Logger (CustomFormatter) → io.MultiWriter
        ├── stdout
        └── dte_microservice.log
```

---|---|
| `Fire(entry)` | Se formatea la entrada y se escribe al Writer |
| `Levels()` | Se retornan todos los niveles de log (se escribe todo al archivo) |

### Flujo de Escritura Dual

```
logs.Info("mensaje")
  │
  ├── stdout (formatter con colores)
  │
  └── WriteHook → archivo (formatter con colores)
```

---

## Niveles de Log Disponibles

```go
var logsLevel = map[string]logrus.Level{
    "debug": logrus.DebugLevel,
    "info":  logrus.InfoLevel,
    "warn":  logrus.WarnLevel,
    "error": logrus.ErrorLevel,
    "fatal": logrus.FatalLevel,
}
```

Se configura el nivel con la variable de entorno `LOG_LEVEL` (`config.Log.Level`). Los mensajes con nivel inferior al configurado se ignoran.

```
fatal > error > warn > info > debug
  │       │       │      │      │
  │       │       │      │      └── Solo si LOG_LEVEL=debug
  │       │       │      └────────── Si LOG_LEVEL=info o inferior
  │       │       └───────────────── Si LOG_LEVEL=warn o inferior
  │       └───────────────────────── Si LOG_LEVEL=error o inferior
  └───────────────────────────────── Siempre visible
```

---

## Notas

1. **InitLogger obligatorio**: Se debe llamar a `InitLogger()` durante el bootstrap antes de usar cualquier función de log.
2. **Campos opcionales**: El segundo parámetro `fields` es variádico — si no se necesitan campos, se omite.
3. **Fatal no termina el proceso**: `logs.Fatal()` usa `Logger.Log(FatalLevel, ...)`, que registra la entrada pero no invoca `os.Exit`. Quien lo llama debe terminar el proceso (por ejemplo, `cmd/main.go` ejecuta `os.Exit(1)` después de cada `logs.Fatal`).
4. **Archivo append**: El archivo de log no se rota automáticamente. Se recomienda configurar rotación a nivel de sistema operativo o contenedor.
5. **Dependencia logrus**: Se usa `github.com/sirupsen/logrus` como backend. El paquete `logs` actúa como wrapper para centralizar la configuración.
