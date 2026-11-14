# Utilidades Compartidas

> **Paquete:** `pkg/shared/utils`

## Descripción General

El paquete `utils` contiene funciones utilitarias puras que se reutilizan a lo largo de todo el proyecto. Se organizan en 7 archivos por área funcional.

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
| `ParseProcessingDate(value)` | Se interpreta la fecha de procesamiento devuelta por Hacienda en la zona horaria configurada. Retorna `(time.Time, bool)`; `false` si el valor está vacío o su formato es desconocido |
| `ExtractGenerationDateTime(jsonData)` | Se obtiene la fecha y hora de generación (`identificacion.fecEmi` + `identificacion.horEmi`) de un JSON de DTE en la zona horaria configurada. Retorna `(time.Time, bool)`; `false` si el JSON no es válido o no trae `fecEmi`. Si falta `horEmi` se usa `00:00:00` |
| `GenerationDateOrDefault(jsonData, fallback)` | Se retorna la fecha de generación del JSON o `fallback` cuando no se puede extraer |

### Formatos de `ParseProcessingDate`

Se prueban en orden los formatos `dd/MM/yyyy HH:mm:ss`, `yyyy-MM-dd HH:mm:ss` y RFC 3339, todos interpretados en la zona horaria de El Salvador.

### Consumidores

| Función | Uso |
|---|---|
| `ParseProcessingDate` | Mapper de invalidación (fecha del sello) y servicio de eventos de contingencia (evidencia de transmisión) |
| `GenerationDateOrDefault` | Mapper de invalidación y servicios de NC y ND (fecha de emisión de los documentos relacionados, usando `CreatedAt` como respaldo) |

### Uso

```go
// En bootstrap (una sola vez)
utils.TimeInit()

// En cualquier parte del código
now := utils.TimeNow()
// → 2024-01-15 14:30:05 -0600 CST (America/El_Salvador)
```

> **Importante:** Todas las fechas y horas del sistema (emisión de DTEs, timestamps de contingencia, etc.) usan `TimeNow()`, nunca `time.Now()` directamente. `TimeInit()` debe haberse ejecutado antes: sin zona horaria cargada `TimeNow()` no es válido.

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
// → Si NRC = "" → nil (se serializa como null)
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
        NIT          string `json:"nit"`
        NumDocumento string `json:"numDocumento"`
    } `json:"receptor"`
}

// ReceiverID devuelve el NIT del receptor: "nit" (CCF) o, si está vacío, "numDocumento" (Comprobante de Retención)
func (a AuxiliarReceiverExtractor) ReceiverID() string

type AuxiliarTotalAmountsExtractor struct {
    Summary struct {
        TotalTaxed      float64 `json:"totalGravada"`
        TotalExempt     float64 `json:"totalExenta"`
        TotalNotSubject float64 `json:"totalNoSuj"`

        DiscountNotSubject float64 `json:"descuNoSuj"`
        DiscountExempt     float64 `json:"descuExenta"`
        DiscountTaxed      float64 `json:"descuGravada"`
    } `json:"resumen"`
}

// Total menos el descuento global de la categoría, redondeado a centavos y nunca negativo
func (a AuxiliarTotalAmountsExtractor) NetTaxed() float64
func (a AuxiliarTotalAmountsExtractor) NetExempt() float64
func (a AuxiliarTotalAmountsExtractor) NetNotSubject() float64

// Mismos totales, pero leídos del formato interno (claves en inglés)
type AuxiliarDTETotalAmountsExtractor struct {
    Summary struct {
        TotalTaxed      float64 `json:"total_taxed"`
        TotalExempt     float64 `json:"total_exempt"`
        TotalNotSubject float64 `json:"total_non_subject"`
    } `json:"summary"`
}

type AuxiliarRelatedDocAndItemsExtractor struct {
    RelatedDocs []struct { GenerationType int; DocumentNumber string } `json:"documentoRelacionado"`
    Items       []struct { TaxedAmount, ExemptAmount, NotSubjectAmount float64; RelatedDoc string } `json:"cuerpoDocumento"`
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
| `ExtractDTEReceiverFromString(doc)` | `interface{}` (JSON string) | NIT del receptor (`nit` o `numDocumento`, ver `ReceiverID()`) |

### Patrón de Dos Variantes

Cada extractor tiene dos variantes:
- **Desde objeto Go**: Se serializa a JSON con `json.Marshal()` y luego se deserializa a la estructura extractora
- **Desde string JSON**: Se deserializa directamente desde el string

Las variantes `FromStringJSON` hacen type assertion a `string` y entran en pánico si reciben otro tipo. `ExtractRelatedDocAndItemsFromStringJSON` ignora errores de deserialización y retorna la estructura vacía.

Se usan en el servicio de invalidación, notas de crédito y débito (receptor del CCF o CR relacionado), contingencia y operaciones adicionales donde se necesita leer campos específicos de un DTE ya serializado.

---

## translater.go — Helpers de Traducción

### Funciones

| Función | Descripción |
|---|---|
| `TranslateMessage(key, code, params...)` | Se construye la clave `"key.code"` y se traduce con `config.Translate()` |
| `TranslateHealthUp(key)` | Se traduce mensaje de health check UP |
| `TranslateHealthDown(key)` | Se traduce mensaje de health check DOWN |
| `TranslateHealthError(key, params...)` | Se traduce error de health check |
| `TranslateHealthNotConfigured(key)` | Se traduce mensaje de health check no configurado |

### Uso

```go
msg := utils.TranslateHealthUp("database")
// Clave: "health.up.database" (se pasa a minúsculas)
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

---

## Notas

1. **TimeNow() siempre**: Nunca usar `time.Now()` directamente. Siempre usar `utils.TimeNow()` para garantizar la zona horaria correcta.
2. **Punteros nil**: `ToStringPointer("")` retorna `nil`; en los response mappers los campos sin `omitempty` se serializan como `null`.
3. **InLetters para TotalLetras**: Se usa automáticamente en los request mappers de todos los DTE con resumen (Factura, CCF, NC, ND, Nota de Remisión y FSE) cuando `TotalInWords` no se proporciona en el request. Ver [Request Mappers](request-mappers.md).
4. **Extractors para DTE serializado**: Los extractors se usan cuando ya se tiene un DTE en formato JSON (almacenado en BD o en cache de contingencia) y se necesitan campos específicos sin deserializar todo el documento.
5. **FindProjectRoot en tests**: Se usa en `TestMain` para localizar archivos de configuración y traducción desde cualquier directorio de ejecución de tests.
