# Sistema de Validación — Strategy Pattern

> **Paquete base:** `internal/domain/dte/common/validator/strategy`
> **Patrón:** Strategy Pattern
> **Interfaz:** `DTEValidationStrategy`

## Descripción General

El sistema de validación utiliza el **patrón Strategy** para componer cadenas de validación específicas por tipo de DTE. Cada estrategia es una unidad autónoma que valida un aspecto del documento. Los validadores por DTE orquestan múltiples estrategias para cubrir todas las reglas de negocio.

---

## Arquitectura

```
DTEValidationStrategy (interfaz)
│
├── Estrategias Comunes (reutilizadas por todos los DTEs)
│   ├── BasicRulesStrategy
│   ├── ModelTypeStrategy
│   ├── ItemValidationStrategy
│   ├── TaxCalculationStrategy
│   ├── PaymentTotalStrategy
│   ├── TemporalValidationStrategy
│   ├── ContingencyStrategy
│   ├── ExtensionStrategy
│   ├── OtherDocumentsStrategy
│   ├── RelatedDocsStrategy
│   └── ThirdPartyStrategy
│
└── Validadores Específicos por DTE (orquestan estrategias)
    ├── InvoiceRulesValidator         → InvoiceItemsStrategy + InvoiceTaxStrategy + InvoiceTotalsStrategy
    ├── CCFRulesValidator             → CCFItemStrategy + CCFTaxStrategy + CCFReceiverStrategy + CCFRelatedDocStrategy
    ├── FSERulesValidator             → FSEItemStrategy + FSETaxStrategy + FSEReceiverStrategy + FSEDiscountStrategy
    ├── CreditNoteRulesValidator      → CreditNoteItemStrategy + CreditNoteTaxStrategy + CreditNoteRelatedDocStrategy
    ├── DebitNoteRulesValidator        → DebitNoteItemStrategy + DebitNoteTaxStrategy + DebitNoteRelatedDocStrategy
    ├── RetentionRulesValidator       → RetentionItemStrategy + RetentionTotalStrategy
    ├── RemissionNoteRulesValidator   → RemissionNoteItemStrategy + RemissionNoteSummaryStrategy + RemissionNoteReceiverStrategy
    └── InvalidationRulesValidator    → InvalidationBasicStrategy + InvalidationDocumentStrategy + InvalidationReasonStrategy + InvalidationDateStrategy
```

---

## Interfaz DTEValidationStrategy

```go
type DTEValidationStrategy interface {
    Validate() *dte_errors.DTEError
}
```

Todas las estrategias implementan este método. Retorna `nil` si la validación es exitosa, o un `*DTEError` con los errores encontrados.

---

## Validación en Dos Niveles

Cada DTE se valida en dos niveles:

### Nivel 1: Validación General (DTEDocument.Validate())

Se ejecuta `ValidateModel[T]()` que usa **reflexión** para validar automáticamente todos los Value Objects dentro del modelo:

```go
func ValidateModel[T any](model T) []error
```

- Se recorre cada campo del struct
- Si el campo implementa `ValueObject[T].IsValid()`, se valida
- Se recorren recursivamente structs y slices anidados
- Retorna lista de errores de VOs inválidos

### Nivel 2: Validación de Reglas de Negocio (ValidateDTERules())

Se ejecutan las estrategias específicas del DTE:

```go
func (d *DTEDocument) ValidateDTERules() *dte_errors.DTEError {
    // Ejecutar estrategias comunes
    // Ejecutar estrategias específicas del DTE
}
```

---

## Estrategias Comunes

Estas estrategias se aplican a la mayoría de los DTEs.

### BasicRulesStrategy

**Campos validados:** Estructura básica del documento

| Validación | Regla |
|---|---|
| Documento | No debe ser nil |
| Identificación | No debe ser nil |
| Emisor | Debe existir |
| Ítems | Lista no vacía |
| Receptor | Requerido para: Factura (01), CCF (03), Remisión (04), Nota Crédito (05), Nota Débito (06), Exportación (11) |

**Validación de NIT/DUI del receptor:**

| Tipo | Patrón | Ejemplo |
|---|---|---|
| NIT | `^([0-9]{14}\|[0-9]{9})$` | `06142803101234` o `061428031` |
| DUI | `^[0-9]{8}-[0-9]{1}$` | `01234567-8` |

---

### ModelTypeStrategy

**Regla:** El modelo de facturación debe ser consistente con el tipo de transmisión.

| Transmisión | Modelo Requerido |
|---|---|
| Normal | `ModeloFacturacionPrevio` (1) |
| Contingencia | `ModeloFacturacionDiferido` (2) |

---

### ItemValidationStrategy

**Campos validados:** Estructura básica de cada ítem

| Validación | Límite |
|---|---|
| Total de ítems | Máximo 2000 |
| Número de ítem | 1 - 2000 |
| Tipo de ítem | Debe estar en `AllowedItemTypes` |
| Descripción | 1 - 1000 caracteres |
| Unidad de medida | 1 - 99 |
| Código de ítem | Máximo 25 caracteres |

---

### TaxCalculationStrategy

**Regla:** Todos los códigos de impuesto de los ítems deben estar en `MapAllowedTaxTypes`.

**Códigos permitidos:**

| Código | Nombre | Tasa |
|---|---|---|
| `20` | IVA | 13% |
| `C3` | IVA Exportación | 0% |
| `59` | Turismo | 1% |
| `71` | Turismo Aeropuerto | Monto fijo |
| `D1` | FOVIAL | 0.5% |
| `C8` | COTRANS | Monto fijo |
| `D5` | Especial/Otro | Sin validación |

---

### PaymentTotalStrategy

**No aplica a:** Nota de Crédito (05), Nota de Débito (06), Nota de Remisión (04)

| Condición | Regla |
|---|---|
| Operación a crédito (2) | No se permite pago en efectivo (`BilletesMonedas`). Debe tener plazo y período |
| Operación al contado (1) | No debe tener plazo ni período de pago |
| Total de pagos | `Sum(payment.Amount) == TotalToPay` (tolerancia ±0.01) |

---

### TemporalValidationStrategy

| Validación | Regla |
|---|---|
| Fecha de emisión | No puede ser futura |
| Hora de emisión | Si la fecha es hoy, la hora no puede ser futura |

---

### ContingencyStrategy

**Solo aplica cuando:** `OperationType == TransmisionContingencia`

| Validación | Regla |
|---|---|
| Tipo de contingencia | No debe ser nil |
| Razón de contingencia | Requerida si tipo es `OtroMotivo` (5) |
| Tipos permitidos | 1-5 (MH no disponible, falla sistema, falla internet, falla energía, otro) |

---

### ExtensionStrategy

**Regla:** Si el monto total de la operación es `>= $1,095.00`, la extensión es obligatoria.

La extensión incluye datos del responsable de la entrega y recepción del documento.

---

### OtherDocumentsStrategy

| Validación | Límite |
|---|---|
| Total documentos | Máximo 10 |
| Código asociado | 1-4 |

**Por código asociado:**

| Código | Tipo | Reglas |
|---|---|---|
| 1, 2, 4 | Documentos normales | Descripción (1-100 chars) y detalle (1-300 chars) requeridos. Doctor debe ser vacío |
| 3 | Documentos médicos | Descripción y detalle vacíos. Doctor requerido con nombre (1-100 chars), tipo servicio (1-6) |

**Regla de exclusividad médica:** El doctor debe tener NIT **o** Identificación, pero no ambos.

---

### RelatedDocsStrategy

| Validación | Límite |
|---|---|
| Total docs relacionados | Máximo 50 |
| Tipos mezclados | No permitido (todos del mismo tipo) |
| Fecha de emisión | No puede ser futura |
| Número (contingencia) | UUID válido: `^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$` |
| Número (normal) | 0-20 caracteres |

---

### ThirdPartyStrategy

**Cuando hay venta a terceros:**
- Todos los ítems deben tener documento relacionado
- El tercero debe tener nombre
- No se permiten ventas mixtas (propias + terceros)

---

## Tolerancias Numéricas

| Contexto | Tolerancia |
|---|---|
| Cálculos monetarios | ±0.01 |
| Subtotales financieros | ±0.0001 |
| Decimales máximos | 2 decimales para montos |

---

## Navegación por DTE

Para la documentación detallada de validación por cada DTE:

- [Validación de Factura (01)](./invoice-validation.md)
- [Validación de CCF (03)](./ccf-validation.md)
- [Validación de FSE (14)](./fse-validation.md)
- [Validación de Nota de Crédito (05)](./credit-note-validation.md)
- [Validación de Nota de Débito (06)](./debit-note-validation.md)
- [Validación de Retención (07)](./retention-validation.md)
- [Validación de Nota de Remisión (04)](./remission-note-validation.md)
- [Validación de Invalidación](./invalidation-validation.md)

---

## Archivos del Sistema de Validación

| Directorio | Contenido |
|---|---|
| `internal/domain/dte/common/validator/strategy/` | Estrategias comunes |
| `internal/domain/dte/common/validator/base_validator.go` | Validador base por reflexión |
| `internal/domain/dte/invoice/validator/` | Estrategias de factura |
| `internal/domain/dte/ccf/validator/` | Estrategias de CCF |
| `internal/domain/dte/fse/validator/` | Estrategias de FSE |
| `internal/domain/dte/credit_note/validator/` | Estrategias de nota de crédito |
| `internal/domain/dte/debit_note/validator/` | Estrategias de nota de débito |
| `internal/domain/dte/retention/validator/` | Estrategias de retención |
| `internal/domain/dte/remission_note/validator/` | Estrategias de nota de remisión |
| `internal/domain/dte/invalidation/validator/` | Estrategias de invalidación |
| `internal/domain/dte/common/dte_errors/` | Tipos de error |
