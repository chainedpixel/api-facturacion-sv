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
- **Conversión a letras** — El total se convierte automáticamente a palabras

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
1. Cast del input → InputRetentionData
2. Crear documento base (DTEDocument) SIN ítems
3. Ensamblar RetentionModel con ítems y resumen de retención
4. Convertir total a letras (TotalInWords)
5. Validar reglas de negocio
6. Generar código de generación (UUID) y número de control
7. Retornar RetentionModel validado
```

### Diferencia clave en la creación del documento base

```go
func (s *retentionService) createBaseDocument(data *retention_models.InputRetentionData) *models.DTEDocument
```

A diferencia de otros DTEs, la retención:
- **No mapea ítems** al documento base (los ítems de retención son una estructura diferente)
- **No tiene resumen financiero estándar** — El resumen se maneja con `RetentionSummary`

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

Retorna `true` si todos los ítems referencian documentos físicos (no electrónicos). Se usa para determinar reglas de validación especiales.

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
| `DTEType` | `DTEType` | Tipo del DTE retenido (`01`, `03`, `11`) |
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
    TotalIVARetention        financial.Amount  // Total IVA retenido
    TotalIVARetentionLetters string            // Total en letras
}
```

---

## Códigos de Retención

Los códigos de retención (`ReceptionCodeMH`) determinan el porcentaje a aplicar sobre el `RetentionAmount`:

| Código | Descripción | Porcentaje |
|---|---|---|
| `22` | Retención IVA 1% | 1% |
| `C4` | Retención IVA 13% | 13% |
| `C9` | Retención IVA otros | Variable |

### Fórmula de Cálculo del IVA Retenido

```
RetentionIVA = RetentionAmount * GetRetentionAmount(ReceptionCodeMH)
```

Donde `GetRetentionAmount()` retorna el porcentaje asociado al código de retención.

Tolerancia: `±0.01`

---

## Tipos de DTE Válidos para Retención

Solo se pueden retener los siguientes tipos de DTE:

| Código | Tipo |
|---|---|
| `01` | Factura Electrónica |
| `03` | CCF Electrónico |
| `11` | Factura de Exportación Electrónica |

Definidos en `ValidRetentionDTETypes`.

---

## Validación de Fechas

La fecha de emisión del documento retenido debe estar dentro del período permitido:

### Reglas de Período

1. **Mismo mes**: La fecha del documento puede ser del mismo mes que la retención
2. **Mes anterior**: Se permite hasta **10 días hábiles** del mes siguiente
3. **Días hábiles**: Se excluyen fines de semana del cálculo

```
Si RetentionDate = 2024-03-15
Documentos válidos:
  - Cualquier fecha de marzo 2024
  - Fechas de febrero 2024 (si están dentro de 10 días hábiles del inicio de marzo)
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

### Conversión a Letras

El `TotalIVARetention` se convierte automáticamente a palabras y se almacena en `TotalIVARetentionLetters`.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido | El `interface{}` no es `*InputRetentionData` |
| `RequiredField` | Campos obligatorios faltantes en ítems |
| `InvalidRetentionIVA` | Cálculo de IVA retenido no coincide con fórmula |
| `DateOutOfAllowedRange` | Fecha del documento fuera del período permitido |
| Error de validación | Reglas de negocio no cumplidas |

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
    ├── Convertir total a letras
    │
    ▼
validate() ──→ RetentionRulesValidator
    │
    ├── RetentionItemStrategy
    │     ├── Calcular IVA por código de retención
    │     └── Validar fechas en período permitido
    │
    └── RetentionTotalStrategy
          └── Validar totales vs suma de ítems
    │
    ▼
generateCodeAndIdentifiers()
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/retention/retention_service.go` | Servicio principal |
| `internal/domain/dte/retention/models/` | Modelos de retención |
| `internal/domain/dte/retention/validator/` | Estrategias de validación |

---

## Notas

1. **Estructura diferente**: Los ítems de retención NO heredan de `models.Item`. Tienen su propia estructura completamente diferente.
2. **Sin resumen financiero estándar**: El `RetentionSummary` solo tiene 3 campos, no el resumen completo con descuentos, condiciones de pago, etc.
3. **IsAllPhysical()**: Se verifica si todos los documentos retenidos son físicos, lo que puede afectar validaciones específicas.
4. **Conversión a letras**: Se ejecuta automáticamente como parte del flujo de creación, antes de la validación.
5. **Códigos de retención**: Son específicos del Ministerio de Hacienda de El Salvador y determinan el porcentaje a aplicar.
