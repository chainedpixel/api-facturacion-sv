# Sistema de Configuración — config/

**Ubicación:** `config/`

## Descripción General

El paquete `config` gestiona toda la configuración del sistema: lectura de variables de entorno, validación, drivers de base de datos, conexión a Redis, traducciones i18n, y errores centinela del sistema de configuración.

---

## Estructura de Configuración

> **Archivo:** `config/env_structs.go`

```go
type envConfig struct {
    Server   server
    Database database
    Redis    redis
    Log      log
    Signer   signer
    MHPaths  mhPaths
    SMTP     smtp
}
```

Se exponen como variables globales después de la inicialización:

```go
var EnvConfig *envConfig
var Server    *server
var Database  *database
var Redis     *redis
var Log       *log
var Signer    *signer
var MHPaths   *mhPaths
var SMTP      *smtp
```

---

## Variables de Entorno

### Servidor

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `SERVER_PORT` | string | Sí | Puerto HTTP (0–65535) |
| `JWT_SECRET` | string | Sí | Clave secreta para firma JWT |
| `MH_AMBIENT_CODE` | string | Sí | Código de ambiente (`"01"` producción, `"00"` pruebas) |
| `MH_MAX_BATCH_SIZE` | int | Sí | Tamaño máximo de lote (1–100) |
| `DEBUG` | bool | Sí | Modo debug |
| `FORCE_CONTINGENCY` | bool | Sí | Forzar modo contingencia |
| `RUN_MIGRATION` | bool | Sí | Ejecutar migraciones al iniciar |
| `ADMIN_EMAIL` | string | No | Email del administrador (destinatario de notificaciones de eventos de dominio) |
| `API_VERSION` | string | No | Versión de la API mostrada en el footer de los correos (default `3.0.0`) |
| `NOTIFY_COOLDOWN_MINUTES` | int | No | TTL del cooldown de notificaciones por correo (default `15`) |
| `APP_LANG` | string | Sí | Código de idioma (`"es"`, `"en"`) |
| `APP_LANG_PATH` | string | Sí | Ruta a archivos de traducción |

### SMTP (notificaciones al admin)

Todas las claves son opcionales. Si `SMTP_HOST` está vacío el sistema degrada gracefully y no envía correos. Detalles en [Eventos de Dominio y Notificaciones](../internal/infrastructure/domain-events.md).

| Variable | Tipo | Requerida | Descripción                                              |
|---|---|---|----------------------------------------------------------|
| `SMTP_HOST` | string | No | Host SMTP. Vacío = envío deshabilitado                   |
| `SMTP_PORT` | int | No | Puerto SMTP (default `587`)                              |
| `SMTP_USERNAME` | string | No | Usuario SMTP                                             |
| `SMTP_PASSWORD` | string | No | Clave SMTP                                               |
| `SMTP_FROM` | string | Sí si `SMTP_HOST` | Remitente, ej. `Ordo <noreply@example.com>`              |
| `SMTP_TLS` | bool | No | `true` aplica STARTTLS, `false` desactiva TLS (solo dev) |

### Base de Datos

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `DB_DRIVER` | string | Sí | Driver (`"mysql"` o `"postgres"`) |
| `DB_HOST` | string | Sí | Hostname/IP del servidor |
| `DB_PORT` | string | Sí | Puerto del servidor |
| `DB_DATABASE` | string | Sí | Nombre de la base de datos |
| `DB_USERNAME` | string | Sí | Usuario |
| `DB_PASSWORD` | string | Sí | Contraseña |
| `DB_CHARSET` | string | Sí | Charset (`"utf8mb4"`) |

### Redis

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `REDIS_HOST` | string | Sí | Hostname/IP de Redis |
| `REDIS_PORT` | string | Sí | Puerto de Redis |
| `REDIS_PASSWORD` | string | No | Contraseña (opcional) |

### Logging

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `LOG_LEVEL` | string | Sí | Nivel (`"debug"`, `"info"`, `"warn"`, `"error"`) |
| `LOG_PATH` | string | Sí | Directorio de archivos de log |
| `LOG_FILE_LOGGING` | bool | Sí | Habilitar logging a archivo |

### Servicio de Firma

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `SIGNER_PATH` | string | Sí | URL del servicio de firma digital |
| `SIGNER_HEALTH` | string | Sí | URL del health check del firmador |

### Endpoints de Hacienda

| Variable | Tipo | Requerida | Descripción |
|---|---|---|---|
| `MH_AUTH_URL` | string | Sí | Autenticación OAuth |
| `MH_RECEPTION_URL` | string | Sí | Recepción de DTEs |
| `MH_LOTE_RECEPTION_URL` | string | Sí | Recepción de lotes |
| `MH_RECEPTION_CONSULT_URL` | string | Sí | Consulta de DTEs |
| `MH_RECEPTION_CONSULT_LOTE_URL` | string | Sí | Consulta de lotes |
| `MH_CONTINGENCY_URL` | string | Sí | Envío de contingencia |
| `MH_NULLIFY_URL` | string | Sí | Invalidación de DTEs |

---

## Carga y Validación

> **Archivo:** `config/env.go`

### Flujo de Inicialización

```
InitEnvConfig(rootPath)
  │
  ├── [1] Crear instancia de Viper
  │     SetConfigName(".env")
  │     SetConfigType("env")
  │     AddConfigPath(rootPath)
  │
  ├── [2] Leer archivo .env
  │     viper.ReadInConfig()
  │
  ├── [3] Habilitar variables de ambiente
  │     viper.AutomaticEnv()
  │
  ├── [4] Mapeo automático con reflection
  │     autoMapEnvKeys(viper, envConfig)
  │     → Itera sobre los campos del struct
  │     → Lee el valor del tag `map-structure`
  │     → Asigna según tipo (string, bool, int, float)
  │
  ├── [5] Deserializar a struct
  │     viper.Unmarshal(&envConfig)
  │
  ├── [6] Validar toda la configuración
  │     ValidateConfig()
  │     ├── validateServerFields()
  │     ├── validateDatabaseFields()
  │     ├── validateRedisFields()
  │     ├── validateLogFields()
  │     ├── validateMHConfigFields()
  │     └── validateSignerFields()
  │
  └── [7] Asignar punteros globales
        Server = &envConfig.Server
        Database = &envConfig.Database
        ...
```

### Patrones de Validación

```go
URLPattern  = `https?://...`     // Se validan todas las URLs de servicios
PortPattern = `\b(0|...|65535)\b` // Se valida rango de puertos
HostPattern = `^(localhost|IP|domain)$` // Se valida formato de host
```

### Validaciones Específicas

| Validación | Regla |
|---|---|
| `SERVER_PORT` | Debe coincidir con PortPattern (0–65535) |
| `MH_MAX_BATCH_SIZE` | Debe estar entre 1 y 100 |
| `DB_DRIVER` | Debe ser `"mysql"` o `"postgres"` |
| `DB_HOST`, `REDIS_HOST` | Deben coincidir con HostPattern |
| `DB_PORT`, `REDIS_PORT` | Deben coincidir con PortPattern |
| URLs de MH y Signer | Deben coincidir con URLPattern |
| Campos booleanos | Deben ser `"true"` o `"false"` |
| `REDIS_PASSWORD` | Excepción: puede estar vacío |

---

## Drivers de Base de Datos

> **Directorio:** `config/drivers/`

### Interfaz

```go
type DriverConfig interface {
    GetDSN() gorm.Dialector
    GetDriverName() string
    GetHost() string
    GetStringConnection() string
}
```

### MysqlDriver

> **Archivo:** `drivers/mysql_driver.go`

```
DSN Format:
  user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=False&loc=America/El_Salvador
```

### PostgresDriver

> **Archivo:** `drivers/postgres_driver.go`

```
DSN Format:
  host=host user=user password=password dbname=database port=port
  sslmode=disable TimeZone=America/El_Salvador options='-c client_encoding=charset'
```

### DbConnection

> **Archivo:** `drivers/database_connector.go`

```go
type DbConnection struct {
    Db     *gorm.DB
    Config *gorm.Config
    Driver DriverConfig
    Err    error
}
```

| Método | Descripción |
|---|---|
| `NewDatabaseConnection(driver)` | Se crea una nueva conexión con el driver indicado |
| `Open()` | Se abre la conexión usando `gorm.Open()` |
| `Close()` | Se cierra la conexión obteniendo el `*sql.DB` subyacente |

---

## Configuración de Redis

> **Archivo:** `config/redis.go`

```go
type RedisConfig struct {
    Host     string
    Port     string
    Password string
}
```

| Método | Descripción |
|---|---|
| `NewRedisConfig()` | Se crea una instancia desde las variables globales |
| `GetURL()` | Se construye la URL: `redis://:password@host:port` o `redis://host:port` |

---

## Sistema de Traducciones

> **Archivo:** `config/translations.go`

### Inicialización

```
InitTranslations(configPath, lang)
  │
  ├── Cargar archivo {lang}.yaml desde configPath
  ├── Procesar claves recursivamente
  └── Almacenar en mapa global translations[lang][key]
```

### Funciones

| Función | Descripción |
|---|---|
| `InitTranslations(path, lang)` | Se cargan traducciones desde archivo YAML |
| `Translate(code, params...)` | Se traduce un código a mensaje legible |
| `TranslateServiceArgs(code, params...)` | Se traduce un error de servicio (prefijo `service_errors.`) |
| `ForceReload(path)` | Se recargan traducciones (inglés por defecto) |
| `normalizeLanguage(lang)` | Se normaliza idioma: `"es"` → español, cualquier otro → `"en"` |

### Thread Safety

Se usa `sync.RWMutex` para acceso concurrente:
- **Escritura**: `InitTranslations`, `ForceReload` (lock exclusivo)
- **Lectura**: `Translate`, `TranslateServiceArgs` (lock compartido)

### Uso

```go
// Traducción simple
msg := config.Translate("validation.required_field")
// → "El campo es requerido"

// Traducción con parámetros
msg := config.Translate("validation.min_length", "nombre", 3)
// → "El campo nombre debe tener al menos 3 caracteres"

// Error de servicio
msg := config.TranslateServiceArgs("auth.invalid_credentials")
// → Busca en "service_errors.auth.invalid_credentials"
```

---

## Errores Centinela

> **Archivo:** `config/error/setinels_err.go`

```go
var (
    ErrFailedToConnectDb         = errors.New("failed to connect to database")
    ErrFailedToCloseDbConnection = errors.New("failed to close database connection")
    ErrFailedToGetDBInstance     = errors.New("failed to get database instance")
    ErrEnvFileNotFound           = errors.New("env file not found")
    ErrFailedToLoadEnv           = errors.New("failed to load env file")
    ErrUnrecognizedDriver        = errors.New("unrecognized database driver")
)
```

---

## Notas

1. **Viper + reflection**: El sistema combina Viper para lectura de `.env` con reflection para mapeo automático a structs tipados, usando los tags `map-structure`.
2. **Validación al inicio**: Toda configuración se valida al arrancar. Si falta un campo requerido o un valor es inválido, la aplicación no inicia.
3. **Zona horaria hardcodeada**: Ambos drivers usan `America/El_Salvador`. No es configurable via `.env`.
4. **Variables globales**: Las configuraciones se exponen como punteros globales por conveniencia. Se inicializan una sola vez y no cambian durante la ejecución.
5. **Archivo de ejemplo**: Se incluye `.env.example` con todas las variables documentadas y valores de ejemplo para ambiente de pruebas.
