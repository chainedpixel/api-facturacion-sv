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
  ├── [1] Determinar nivel de log
  │     determineLogLevel(logLevel)
  │     → "debug" → DebugLevel
  │     → "info"  → InfoLevel
  │     → "warn"  → WarnLevel
  │     → "error" → ErrorLevel
  │     → "fatal" → FatalLevel
  │     → default → InfoLevel
  │
  ├── [2] Configurar formatter personalizado
  │     CustomFormatter{
  │       TextFormatter{ DisableColors: true }
  │     }
  │
  ├── [3] Configurar salida a stdout
  │     Logger.SetOutput(os.Stdout)
  │
  ├── [4] Crear directorio de logs (si no existe)
  │     os.MkdirAll(rootPath + logPath)
  │
  ├── [5] Crear/abrir archivo de log
  │     os.OpenFile("dte_microservice.log", O_CREATE|O_WRONLY|O_APPEND)
  │
  └── [6] Agregar hook de escritura a archivo
        Logger.AddHook(WriteHook{
          Writer: logFile,
          Formatter: CustomFormatter
        })
```

### Archivo de Log

- **Nombre:** `dte_microservice.log`
- **Ubicación:** `{projectRoot}{LOG_PATH}/dte_microservice.log`
- **Modo:** Append (no sobreescribe al reiniciar)

---

## Funciones de Log

| Función | Nivel | Uso |
|---|---|---|
| `Debug(msg, fields...)` | DEBUG | Información detallada para desarrollo |
| `Info(msg, fields...)` | INFO | Eventos normales del sistema |
| `Warn(msg, fields...)` | WARN | Situaciones inesperadas pero recuperables |
| `Error(msg, fields...)` | ERROR | Errores que requieren atención |
| `Fatal(msg, fields...)` | FATAL | Errores irrecuperables (termina el proceso) |

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

Se personaliza la salida de cada entrada de log con colores ANSI y formato estructurado.

### Formato de Salida

```
[LEVEL] 2024-01-15 14:30:05 | Mensaje del log
         Field Details
         ├── key1: value1
         └── key2: value2
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
- Se usa la hora del sistema (ya configurada con zona `America/El_Salvador`)

---

## WriteHook

```go
type WriteHook struct {
    Writer    io.Writer
    Formatter logrus.Formatter
}
```

Se implementa la interfaz `logrus.Hook` para escribir logs a destinos adicionales (archivo).

| Método | Descripción |
|---|---|
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

Se configura el nivel con la variable de entorno `LOG_LEVEL`. Los mensajes con nivel inferior al configurado se ignoran.

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
3. **Fatal termina el proceso**: `logs.Fatal()` llama a `os.Exit(1)` internamente. Solo se usa para errores irrecuperables durante el arranque.
4. **Archivo append**: El archivo de log no se rota automáticamente. Se recomienda configurar rotación a nivel de sistema operativo o contenedor.
5. **Dependencia logrus**: Se usa `github.com/sirupsen/logrus` como backend. El paquete `logs` actúa como wrapper para centralizar la configuración.
