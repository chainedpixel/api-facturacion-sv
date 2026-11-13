# Comprobante de Retención Electrónico (DTE 07) - Retention Service

> **Código DTE:** `07`
> **Paquete:** `internal/domain/dte/retention`
> **Servicio:** `retentionService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

El Comprobante de Retención es un documento tributario que registra retenciones de IVA aplicadas en transacciones comerciales. A diferencia de otros DTEs, **no tiene ítems de venta** sino ítems de retención que referencian documentos previamente emitidos.

Características principales:
- **Sin ítems de venta** — Los ítems son registros de retención, no productos/servicios
- **Referencia documentos existentes** — Cada ítem contiene tipo, número y fecha del documento retenido
- **Códigos de retención** — Cada retención tiene un código que determina el porcentaje aplicado
- **Validación de fechas** — La fecha del documento retenido debe estar dentro del período permitido
- **Conversión a letras** — El total de IVA retenido se convierte automáticamente a palabras
- **Sin validación genérica** — `Create` ejecuta solo `RetentionRulesValidator`; no invoca `ValidateDTEDocument`

---

## Estructura del Servicio

```go
type retentionService struct {
    validator        *validator.RetentionRulesValidator
    dteManager       dte_documents.DTEManager
    seqNumberManager dte_documents.SequentialNumberManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.RetentionRulesValidator` | Se validan reglas de negocio de retención |
| `dteManager` | `DTEManager` | Se accede a documentos existentes |
| `seqNumberManager` | `SequentialNumberManager` | Se gestiona la reserva de números de control |

### Constructor

```go
func NewRetentionService(
    seqNumberManager dte_documents.SequentialNumberManager,
    dteManager dte_documents.DTEManager,
) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *retentionService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → *InputRetentionData
2. Convertir TotalIVARetention a letras (TotalIVARetentionLetters, utils.InLetters)
3. Crear documento base (DTEDocument) SIN ítems
4. Ensamblar RetentionModel con ítems y resumen de retención
5. Validar reglas de negocio (RetentionRulesValidator)
6. Generar número de control (SequentialNumberManager) y código de generación (UUID)
7. Retornar RetentionModel validado
```

El cast no está protegido y `RetentionSummary` se desreferencia sin verificación de nulo en el paso 2: un resumen ausente produce un pánico.

### Diferencia clave en la creación del documento base

```go
func (s *retentionService) createBaseDocument(data *retention_models.InputRetentionData) *models.DTEDocument
```

A diferencia de otros DTEs, la retención:
- **No mapea ítems** al documento base (los ítems de retención son una estructura diferente)
- **No asigna resumen estándar** al documento base — El resumen se maneja con `RetentionSummary`
- Si no se envía receptor, usa un receptor vacío con dirección vacía

---

## Modelos Específicos de Retención

### RetentionModel

```go
type RetentionModel struct {
    *models.DTEDocument
    RetentionItems   []RetentionItem
    RetentionSummary *RetentionSummary
}
```

**Método especial:**

```go
func (r *RetentionModel) GetTotalByItems() (decimal.Decimal, decimal.Decimal)
```

Calcula `TotalSubjectRetention` y `TotalIVARetention` sumando los valores de todos los ítems.

### InputRetentionData

```go
type InputRetentionData struct {
    *models.InputDataCommon
    RetentionItems   []RetentionItem
    RetentionSummary *RetentionSummary
}
```

**Método especial:**

```go
func (d *InputRetentionData) IsAllPhysical() bool
```

Retorna `true` si todos los ítems referencian documentos físicos (no electrónicos). El método existe en el modelo, pero no se invoca desde el código de dominio ni de aplicación actual.

### RetentionItem

```go
type RetentionItem struct {
    Number          item.ItemNumber         // Número secuencial del ítem
    DTEType         document.DTEType        // Tipo de DTE del documento retenido
    DocumentType    document.OperationType  // Tipo de documento (físico=1, electrónico=2)
    DocumentNumber  document.DocumentNumber // Número del documento retenido
    EmissionDate    temporal.EmissionDate   // Fecha de emisión del documento retenido
    RetentionAmount financial.Amount        // Monto sujeto a retención
    ReceptionCodeMH document.RetentionCode  // Código de retención MH
    RetentionIVA    financial.Amount        // Monto de IVA retenido
    Description     string                  // Descripción del ítem
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `Number` | `ItemNumber` | Número secuencial (1, 2, 3...) |
| `DTEType` | `DTEType` | Tipo del DTE retenido (`01`, `03`, `14`) |
| `DocumentType` | `OperationType` | `1` = Físico, `2` = Electrónico |
| `DocumentNumber` | `DocumentNumber` | Número de control del documento retenido |
| `EmissionDate` | `EmissionDate` | Fecha de emisión del documento original |
| `RetentionAmount` | `Amount` | Monto base sujeto a retención |
| `ReceptionCodeMH` | `RetentionCode` | Código que determina el porcentaje de retención |
| `RetentionIVA` | `Amount` | Monto calculado del IVA retenido |
| `Description` | `string` | Descripción libre del ítem |

### RetentionSummary

```go
type RetentionSummary struct {
    TotalSubjectRetention    financial.Amount  // Total sujeto a retención
    TotalIVA                 financial.Amount  // Total IVA 13% (campo del resumen del CRE)
    TotalIVARetention        financial.Amount  // Total IVA retenido
    TotalIVARetentionLetters string            // Total en letras
    Observations             *string           // Observaciones
}
```

---

## Códigos de Retención

Los códigos de retención (`ReceptionCodeMH`) determinan el porcentaje a aplicar sobre el `RetentionAmount`:

| Código | Descripción | Porcentaje |
|---|---|---|
| `22` | Retención IVA 1% | 1% |
| `C4` | Retención IVA 13% | 13% |
| `C9` | Retención IVA otros | Sin porcentaje en `GetRetentionAmount` |

El value object `RetentionCode` acepta los tres códigos (`AllowedRetentionCodes`), pero `GetRetentionAmount` solo define `22` y `C4`. Con `C9` el valor esperado de IVA retenido por ítem resulta `0`, por lo que solo se acepta un `RetentionIVA` igual a `0`.

### Fórmula de Cálculo del IVA Retenido

```
RetentionIVA = RetentionAmount * GetRetentionAmount(ReceptionCodeMH)
```

La tolerancia (`±0.01`) y las demás fórmulas se documentan en [calculations.md](../../../api/calculations.md).

---

## Tipos de DTE Válidos para Retención

Solo se pueden retener los siguientes tipos de DTE:

| Código | Tipo |
|---|---|
| `01` | Factura Electrónica |
| `03` | CCF Electrónico |
| `14` | Factura de Sujeto Excluido Electrónica |

Definidos en `ValidRetentionDTETypes` y expuestos por el value object `DTEType` mediante `NewDTETypeForRetention` (error `InvalidDTETypeForRetention`); las estrategias del validador de retención no revisan este tipo.

---

## Validación de Fechas

La fecha de emisión del documento retenido debe estar dentro del período permitido:

### Reglas de Período

Se aplica a cada ítem (físico o electrónico), comparando la fecha de emisión del documento retenido con la fecha de emisión del comprobante de retención:

1. **Mismo mes**: Se acepta cualquier fecha del mismo mes que la retención
2. **Mes anterior**: Se acepta si el comprobante de retención se emite dentro de los primeros **10 días hábiles** del mes siguiente al del documento
3. **Días hábiles**: Solo se excluyen sábados y domingos (no se consideran asuetos ni el calendario `BusinessCalendar`)

```
Si RetentionDate = 2024-03-15
Documentos válidos:
  - Cualquier fecha de marzo 2024
  - Fechas de febrero 2024 solo si RetentionDate cae dentro de los 10 días hábiles
    posteriores al 1 de marzo (2024-03-15 es el día hábil 11: no sería válido)
```

---

## Totales

### Total Sujeto a Retención

```
TotalSubjectRetention = Sum(item.RetentionAmount)    para cada ítem
```

### Total IVA Retenido

```
TotalIVARetention = Sum(item.RetentionIVA)    para cada ítem
```

Ambos totales del resumen se comparan con la suma de los ítems (`GetTotalByItems`), con tolerancia `±0.01`.

### Total IVA 13% del resumen (`TotalIVA`)

`retention_models.ExpectedTotalIVA` define el valor esperado: `TotalSubjectRetention * 0.13` solo cuando el código de retención del último ítem es `C4` y todos los documentos son electrónicos; en cualquier otro caso se espera `0.00`. Un valor distinto produce `InvalidTotalIVA`.

### Conversión a Letras

El `TotalIVARetention` se convierte automáticamente a palabras y se almacena en `TotalIVARetentionLetters`.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Pánico por cast (cast sin comprobación) | El `interface{}` no es `*InputRetentionData` |
| `RequiredField` | Sin ítems de retención |
| `InvalidRetentionIVA` | IVA retenido del ítem no coincide con la fórmula (parámetros: ítem, valor declarado, valor esperado) |
| `DateOutOfAllowedRange` | Fecha del documento fuera del período permitido (parámetros: ítem, número de documento) |
| `InvalidTotalSubjectRetention` | Total sujeto a retención distinto de la suma de ítems |
| `InvalidTotalIVARetention` | Total de IVA retenido distinto de la suma de ítems |
| `InvalidTotalIVA` | `TotalIVA` distinto del esperado |
| `ValidationFailed` | Envoltorio de los errores de validación anteriores |

---

## Diagrama de Flujo

```
InputRetentionData
    │
    ▼
createBaseDocument() ──→ DTEDocument (sin ítems estándar)
    │
    ▼
RetentionModel {
    DTEDocument
    RetentionItems[]     ← Estructura diferente a ítems normales
    RetentionSummary     ← Solo totales de retención
}
    │
    ├── Convertir total a letras (antes de ensamblar el modelo)
    │
    ▼
validate() ──→ RetentionRulesValidator
    │
    ├── RetentionItemStrategy
    │     ├── Calcular IVA por código de retención
    │     └── Validar fechas en período permitido
    │
    └── RetentionTotalStrategy
          ├── Validar totales vs suma de ítems
          └── Validar TotalIVA (ExpectedTotalIVA)
    │
    ▼
generateCodeAndIdentifiers()
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/retention/retention_service.go` | Servicio principal |
| `internal/domain/dte/retention/retention_models/` | Modelos de retención |
| `internal/domain/dte/retention/validator/` | Estrategias de validación |

---

## Notas

1. **Estructura diferente**: Los ítems de retención NO heredan de `models.Item`. Tienen su propia estructura completamente diferente.
2. **Sin resumen financiero estándar**: El `RetentionSummary` solo contiene totales de retención, `TotalIVA` y observaciones, no el resumen completo con descuentos, condiciones de pago, etc.
3. **IsAllPhysical()**: Indica si todos los documentos retenidos son físicos; actualmente no se usa. La regla equivalente de `TotalIVA` está en `RetentionTotalStrategy.validateTotalIVA`.
4. **Conversión a letras**: Se ejecuta automáticamente como parte del flujo de creación, antes de la validación.
5. **Códigos de retención**: Son específicos del Ministerio de Hacienda de El Salvador y determinan el porcentaje a aplicar.
