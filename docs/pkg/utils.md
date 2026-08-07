# Utilidades Compartidas

> **Paquete:** `pkg/shared/utils`

## Descripción General

El paquete `utils` contiene funciones utilitarias puras que se reutilizan a lo largo de todo el proyecto. Se organizan en 6 archivos por área funcional.

---

## time.go — Manejo de Zona Horaria

### Variables

```go
var timezone *time.Location  // Inicializado con "America/El_Salvador"
```

### Funciones

| Función | Descripción |
|---|---|
| `TimeInit()` | Se inicializa la zona horaria a `America/El_Salvador`. Se debe llamar una vez durante el bootstrap |
| `TimeNow()` | Se retorna la hora actual en la zona horaria de El Salvador |

### Uso

```go
// En bootstrap (una sola vez)
utils.TimeInit()

// En cualquier parte del código
now := utils.TimeNow()
// → 2024-01-15 14:30:05 -0600 CST (America/El_Salvador)
```

> **Importante:** Todas las fechas y horas del sistema (emisión de DTEs, timestamps de contingencia, etc.) usan `TimeNow()`, nunca `time.Now()` directamente.

---

## pointer.go — Conversiones de Punteros

### Funciones

| Función | Entrada | Salida | Comportamiento con zero-value |
|---|---|---|---|
| `ToStringPointer(s)` | `string` | `*string` | `""` → `nil` |
| `ToIntPointer(i)` | `int` | `*int` | `0` → `nil` |
| `ToFloat64Pointer(f)` | `float64` | `*float64` | `0.0` → `*float64(0)` (no nil) |
| `PointerToString(s)` | `*string` | `string` | `nil` → `""` |

### Uso Típico

Se usan extensivamente en los response mappers para convertir campos opcionales del dominio a punteros JSON:

```go
// Campo opcional en JSON Hacienda
receiver.NRC = utils.ToStringPointer(domain.GetNRC())
// → Si NRC = "" → nil (se omite en JSON con omitempty)
// → Si NRC = "12345" → &"12345" (se incluye en JSON)
```

---

## mount_in_letters.go — Números a Letras (Español)

### Función Principal

```go
func InLetters(n float64) string
```

Se convierte un monto numérico a su representación en palabras en español, formateado para documentos fiscales de El Salvador.

### Formato de Salida

```
InLetters(1250.50) → "MIL DOSCIENTOS CINCUENTA 50/100"
InLetters(0.00)    → "CERO 00/100"
InLetters(999999.99) → "NOVECIENTOS NOVENTA Y NUEVE MIL NOVECIENTOS NOVENTA Y NUEVE 99/100"
```

### Restricciones

| Restricción | Comportamiento |
|---|---|
| `NaN` | Se retorna mensaje de error |
| `±Infinity` | Se retorna mensaje de error |
| `>= 1,000,000,000` | Se retorna mensaje de error |
| Negativos | Se agrega prefijo `"MENOS "` |
| Decimales | Se redondea a 2 posiciones y se formatea como `XX/100` |

### Manejo Especial del Rango 20-29

Se maneja la particularidad del español para números como "veintiuno":

- `21` antes de "mil" → `"veintiun"` (sin acento por contexto)
- `21` al final → `"veintiuno"`

### Tablas Internas

```go
us  = ["cero", "uno", "dos", ..., "nueve"]           // 0-9
ds  = ["X", "y", "veinte", ..., "noventa"]           // Decenas
des = ["diez", "once", "doce", ..., "diecinueve"]    // 10-19
cs  = ["x", "cien", "doscientos", ..., "novecientos"] // Centenas
```

---

## find_root_project.go — Localización de la Raíz del Proyecto

### Función

```go
func FindProjectRoot() string
```

Se busca la raíz del proyecto recorriendo el árbol de directorios hacia arriba hasta encontrar un archivo `go.mod`.

### Algoritmo

```
FindProjectRoot()
  │
  ├── [1] Obtener directorio actual (os.Getwd())
  │
  ├── [2] ¿Existe go.mod en directorio actual?
  │     ├── Sí → retornar directorio
  │     └── No → subir un nivel (..)
  │
  ├── [3] Repetir hasta llegar a la raíz del filesystem
  │
  └── [4] Si no se encontró → retornar directorio original
```

Se usa durante la inicialización para localizar archivos de configuración (`.env`, `assets/i18n/`, etc.) independientemente del directorio de ejecución.

---

## extractors.go — Extracción de Datos desde JSON de DTEs

### Estructuras de Extracción

```go
type AuxiliarIdentificationExtractor struct {
    Identification struct {
        DTEType        string `json:"tipoDte"`
        ControlNumber  string `json:"numeroControl"`
        GenerationCode string `json:"codigoGeneracion"`
    } `json:"identificacion"`
    Issuer struct {
        NIT string `json:"nit"`
    } `json:"emisor"`
}

type AuxiliarReceiverExtractor struct {
    Receiver struct {
        NIT string `json:"nit"`
    } `json:"receptor"`
}

type AuxiliarTotalAmountsExtractor struct {
    Summary struct {
        TotalTaxed      float64 `json:"totalGravada"`
        TotalExempt     float64 `json:"totalExenta"`
        TotalNotSubject float64 `json:"totalNoSuj"`
    } `json:"resumen"`
}

type AuxiliarRelatedDocAndItemsExtractor struct {
    RelatedDocs []struct { ... } `json:"documentoRelacionado"`
    Items       []struct { ... } `json:"cuerpoDocumento"`
}
```

### Funciones

| Función | Entrada | Salida |
|---|---|---|
| `ExtractAuxiliarIdentification(doc)` | `interface{}` (Go object) | Identificación + NIT emisor |
| `ExtractAuxiliarIdentificationFromStringJSON(doc)` | `interface{}` (JSON string) | Identificación + NIT emisor |
| `ExtractSummaryTotalAmounts(doc)` | `interface{}` (Go object) | Totales financieros |
| `ExtractSummaryTotalAmountsFromStringJSON(doc)` | `interface{}` (JSON string) | Totales financieros |
| `ExtractRelatedDocAndItemsFromStringJSON(doc)` | `interface{}` (JSON string) | Docs relacionados + ítems |
| `ExtractDTEReceiverFromString(doc)` | `interface{}` (JSON string) | NIT del receptor |

### Patrón de Dos Variantes

Cada extractor tiene dos variantes:
- **Desde objeto Go**: Se serializa a JSON con `json.Marshal()` y luego se deserializa a la estructura extractora
- **Desde string JSON**: Se deserializa directamente desde el string

Se usan en el servicio de invalidación, contingencia y operaciones adicionales donde se necesita leer campos específicos de un DTE ya serializado.

---

## translater.go — Helpers de Traducción

### Funciones

| Función | Descripción |
|---|---|
| `TranslateMessage(key, code, params...)` | Se construye la clave `"key.code"` y se traduce con `config.Translate()` |
| `TranslateHealthUp(key)` | Se traduce mensaje de health check UP |
| `TranslateHealthDown(key)` | Se traduce mensaje de health check DOWN |
| `TranslateHealthError(key, params...)` | Se traduce error de health check |

### Uso

```go
msg := utils.TranslateMessage("validation", "required_field", "nombre")
// Clave: "validation.required_field"
// → "El campo nombre es requerido"

msg := utils.TranslateHealthUp("database")
// → "La base de datos está operativa"
```

---

## updates.go — Modificación de DTEs Serializados

### `UpdateContingencyIdentification`

```go
func UpdateContingencyIdentification(document interface{}, contiType *int8, reason *string) (map[string]interface{}, error)
```

Se modifica el JSON de un DTE para agregar información de contingencia:

```json
// Antes:
{ "identificacion": { "tipoModelo": 1, "tipoOperacion": 1 } }

// Después:
{ "identificacion": {
    "tipoModelo": 2,           // ModeloFacturacionDiferido
    "tipoOperacion": 2,        // TransmisionContingencia
    "tipoContingencia": 2,     // Tipo proporcionado
    "motivoContin": "Falla..." // Motivo proporcionado
  }
}
```

Se acepta el documento como `string`, `[]byte`, o `interface{}`.

### `SetReceptionStampIntoAppendix`

```go
func SetReceptionStampIntoAppendix(document string, receptionStamp *string) (string, error)
```

Se agrega el sello de recepción de Hacienda al apéndice del DTE:

```json
// Se agrega al array "apendice":
{
  "campo": "Datos del documento",
  "etiqueta": "Sello de recepción",
  "valor": "2024011514300500000..."
}
```

Si el apéndice no existe, se crea. Si ya existe, se agrega al final del array.

---

## Notas

1. **TimeNow() siempre**: Nunca usar `time.Now()` directamente. Siempre usar `utils.TimeNow()` para garantizar la zona horaria correcta.
2. **Punteros nil = omitir en JSON**: `ToStringPointer("")` retorna `nil`, lo que con `omitempty` omite el campo en la serialización JSON.
3. **InLetters para TotalLetras**: Se usa automáticamente en los mappers de factura cuando el campo `TotalEnLetras` no se proporciona en el request.
4. **Extractors para DTE serializado**: Los extractors se usan cuando ya se tiene un DTE en formato JSON (almacenado en BD o en cache de contingencia) y se necesitan campos específicos sin deserializar todo el documento.
5. **FindProjectRoot en tests**: Se usa en `TestMain` para localizar archivos de configuración y traducción desde cualquier directorio de ejecución de tests.
