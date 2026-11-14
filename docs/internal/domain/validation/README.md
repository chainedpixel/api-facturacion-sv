# Sistema de Validación — Strategy Pattern

> **Paquete base:** `internal/domain/dte/common/validator/strategy`
> **Patrón:** Strategy Pattern
> **Interfaz:** `DTEValidationStrategy`

## Descripción General

El sistema de validación utiliza el **patrón Strategy** para componer cadenas de validación específicas por tipo de DTE. Cada estrategia es una unidad autónoma que valida un aspecto del documento. Los validadores por DTE orquestan múltiples estrategias para cubrir todas las reglas de negocio.

Las fórmulas de cálculo (venta por ítem, IVA, tributos, subtotales, totales, retenciones), los decimales permitidos por campo, la holgura y los códigos de error de cada cálculo se documentan una sola vez en [Cálculos que valida la API](../../../api/calculations.md). Esta documentación las resume y enlaza.

---

## Arquitectura

```
DTEValidationStrategy (interfaz)
│
├── Estrategias Comunes (reutilizadas por todos los DTEs)
│   ├── BasicRulesStrategy
│   ├── TemporalValidationStrategy
│   ├── ContingencyStrategy
│   ├── ModelTypeStrategy
│   ├── TaxCalculationStrategy
│   ├── PaymentTotalStrategy
│   ├── ItemValidationStrategy
│   ├── DocumentTypeStrategy
│   ├── RelatedDocsStrategy
│   ├── ThirdPartyStrategy
│   └── OtherDocumentsStrategy
│
├── Reglas compartidas (funciones reutilizadas por los validadores de cada DTE)
│   ├── calculation_rules.go      → holgura, venta por ítem, suma de operaciones, monto total, descuentos globales en NC/ND, totales de NC/ND
│   └── related_docs_rules.go     → tipos permitidos y plazos respecto del documento relacionado
│
└── Validadores Específicos por DTE (orquestan estrategias)
    ├── InvoiceRulesValidator         → InvoiceItemsStrategy + InvoiceTaxStrategy + InvoiceTotalsStrategy
    │                                   (InvoiceRelatedDocStrategy existe pero no está registrada en el validador)
    ├── CCFRulesValidator             → CCFItemStrategy + CCFTaxStrategy + CCFReceiverStrategy + CCFRelatedDocStrategy
    ├── FSERulesValidator             → FSEItemStrategy + FSETaxStrategy + FSEReceiverStrategy + FSEDiscountStrategy
    ├── CreditNoteRulesValidator      → CreditNoteItemStrategy + CreditNoteTaxStrategy + CreditNoteRelatedDocStrategy
    ├── DebitNoteRulesValidator       → DebitNoteItemStrategy + DebitNoteTaxStrategy + DebitNoteRelatedDocStrategy
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

## Orden de la Validación

Para cada DTE que emite el microservicio (FE, CCF, NR, NC, ND y FSE) el servicio de dominio ejecuta, en este orden y deteniéndose en el primer paso que falle:

1. **Validador propio del documento** (`<Documento>RulesValidator`): ejecuta todas sus estrategias y agrupa los errores de las distintas estrategias en un `CompositeError`.
2. **Validación general** (`ValidateDTEDocument`):
   - `ValidateModel[T]()` recorre por **reflexión** el modelo (campos exportados, structs y slices anidados) y ejecuta `IsValid()` de cada Value Object; retorna la lista de Value Objects inválidos.
   - `ValidateDTERules()` ejecuta las estrategias comunes en este orden: `BasicRulesStrategy`, `TemporalValidationStrategy`, `ContingencyStrategy`, `ModelTypeStrategy`, `TaxCalculationStrategy`, `PaymentTotalStrategy`, `ItemValidationStrategy`, `DocumentTypeStrategy`, `RelatedDocsStrategy`, `ThirdPartyStrategy` y `OtherDocumentsStrategy`. Los errores de las estrategias y de los Value Objects se agrupan en una sola respuesta.
3. **Validaciones de servicio** posteriores (documentos relacionados electrónicos, saldo en NC y ND, numeración de control).

El **Comprobante de Retención (07)** solo ejecuta el paso 1 (`RetentionRulesValidator`); no pasa por la validación general.

Cada estrategia devuelve **el primer error** que encuentra; los validadores agrupan los errores de estrategias distintas.

---

## Estrategias Comunes

Estas estrategias se aplican a los DTE emitidos por el microservicio, salvo que se indique lo contrario.

### BasicRulesStrategy

**Campos validados:** Estructura básica del documento

| Validación | Regla | Error |
|---|---|---|
| Documento, identificación y emisor | No deben ser nil | `RequiredFieldMissing` |
| Ítems | Lista no vacía | `RequiredFieldMissing` |
| Receptor | Requerido para: Factura (01), CCF (03), Remisión (04), Nota Crédito (05), Nota Débito (06) y Exportación (11) | `RequiredFieldMissing` |

**Validación de NIT/DUI del receptor** (solo si se informan tipo y número):

| Tipo | Patrón | Ejemplo | Error |
|---|---|---|---|
| NIT (`36`) | `^([0-9]{14}\|[0-9]{9})$` | `06142803101234` o `061428031` | `InvalidNITFormat` |
| DUI (`13`) | `^[0-9]{9}$` (el DUI se almacena y se envía sin guion: el Value Object quita el guion del request) | `012345678` | `InvalidDUIFormat` |

---

### ModelTypeStrategy

**Regla:** El modelo de facturación debe ser consistente con el tipo de transmisión. Error: `InvalidModelType`.

| Transmisión | Modelo Requerido |
|---|---|
| Normal | `ModeloFacturacionPrevio` (1) |
| Contingencia | `ModeloFacturacionDiferido` (2) |

---

### ItemValidationStrategy

**Campos validados:** Estructura básica de cada ítem

| Validación | Límite | Error |
|---|---|---|
| Total de ítems | Máximo 2000 | `ExceededItemsLimit` |
| Número de ítem | 1 - 2000 | `InvalidItemNumber` |
| Tipo de ítem | Debe estar en `AllowedItemTypes` (1 producto, 2 servicio, 3 ambos, 4 impuesto) | `InvalidItemType` |
| Descripción | 1 - 1500 caracteres (Anexo IV, campo 86) | `InvalidLength` |
| Unidad de medida | 1 - 99 | `InvalidNumberRange` |
| Código de ítem | Máximo 25 caracteres | `InvalidLength` |

Los errores de los ítems se agrupan en un error compuesto. Los cálculos por ítem están en [Cálculos que valida la API, sección 2](../../../api/calculations.md#2-cuerpo-del-documento-ítems).

---

### TaxCalculationStrategy

**Regla:** Todos los códigos de impuesto de los ítems deben estar en `MapAllowedTaxTypes` (`20`, `C3`, `59`, `71`, `D5`). Error: `UnsupportedTaxCode` (incluye `D1` y `C8`, que no se soportan).

Las bases y tasas con que se recalcula el valor de cada tributo del resumen se documentan en [Cálculos que valida la API, sección 3.6](../../../api/calculations.md#36-tributos-del-resumen-campos-136-a-138).

---

### PaymentTotalStrategy

**No aplica a:** Nota de Remisión (04). En Nota de Crédito (05) y Nota de Débito (06) solo se valida la condición 1 cuando el total a pagar es 0.00

| Condición | Regla | Error |
|---|---|---|
| Operación a crédito (2) | Cada pago lleva el código en `null` | `CreditPaymentCodeNotAllowed` |
| Operación a crédito (2) | Cada pago lleva el monto en `0.00`; la suma no se compara con el total | `InvalidCreditPaymentAmount` |
| Operación a crédito (2) | Cada pago lleva la referencia en `null` | `CreditPaymentReferenceNotAllowed` |
| Operación a crédito (2) | Cada pago lleva plazo (`term`) | `CreditPaymentTermRequired` |
| Operación a crédito (2) | Cada pago lleva período (`period`) mayor que cero | `CreditPaymentPeriodRequired` |
| Operación al contado (1) u otra (3) | Cada pago lleva código | `PaymentCodeRequired` |
| Total a pagar 0.00 | FE, CCF y FEXE: condición 1 y al menos un pago, todos con código `99` (`ValidateNoPaymentDue`). FSE, NC y ND: solo condición 1; con saldo a favor la misma regla se aplica desde `InvoiceTotalsStrategy` y `CCFTaxStrategy` (`ValidateBalanceInFavorPayments`) | `NoPaymentDueRequiresCashCondition`, `NoPaymentDueRequiresCode99` |
| Operación al contado (1) sin pago en efectivo | Los pagos no deben tener plazo ni período | `InvalidPaymentTermsOF` |
| Total de pagos | Con condición 1 o 3, `Sum(payment.Amount) == TotalToPay` (holgura ±0.01). Una lista de pagos vacía con crédito y total mayor que 0 también se rechaza | `InvalidPaymentTotal` |

Ver también [Cálculos, sección 3.11](../../../api/calculations.md#311-pagos).

---

### TemporalValidationStrategy

| Validación | Regla | Error |
|---|---|---|
| Fecha de emisión | No puede ser posterior a la fecha actual | `InvalidDateTime` |
| Hora de emisión | Si la fecha de emisión es la de hoy, la hora no puede ser posterior a la actual | `InvalidEmissionTime` |

---

### ContingencyStrategy

**Solo aplica cuando:** `OperationType == TransmisionContingencia`

| Validación | Regla | Error |
|---|---|---|
| Tipo de contingencia | No debe ser nil | `MissingContingencyType` |
| Razón de contingencia | No puede ser vacía si el tipo es `OtroMotivo` (5) | `MissingContingencyReason` |
| Tipos permitidos | 1-5 (MH no disponible, falla de conexión del sistema, falla de internet, falla de energía, otro) | `InvalidContingencyType` |

---

### DocumentTypeStrategy

**Solo aplica a:** CCF (03). El receptor debe tener NRC (`MissingNRC`).

---

### OtherDocumentsStrategy

| Validación | Límite | Error |
|---|---|---|
| Total documentos | Máximo 10 | `InvalidOtherDocsCount` |
| Código asociado | 1-4 | `InvalidAssociatedDocumentCode` |
| Código 4 (transporte) | Solo para FEXE (`11`); otros tipos lo rechazan. La FEXE no tiene endpoint, por lo que en la práctica se rechaza | `InvalidAssociatedDocumentCodeForType` |

**Por código asociado:**

| Código | Tipo | Reglas |
|---|---|---|
| 1, 2, 4 | Documentos normales | Descripción (1-100 chars) y detalle (1-300 chars) requeridos. Doctor debe ser vacío |
| 3 | Documentos médicos | Descripción y detalle vacíos (`InvalidMedicalDocFields`). Doctor requerido con nombre (1-100 chars) y tipo servicio (1-6, `InvalidServiceType`) |

**Regla de exclusividad médica:** El doctor debe tener NIT **o** Identificación, pero no ambos (`MutuallyExclusiveFields`).

---

### RelatedDocsStrategy

**Archivos:** `related_docs_strategy.go` y `related_docs_rules.go` (`internal/domain/dte/common/validator/strategy/`). Se aplica solo si el documento trae documentos relacionados.

| Validación | Regla | Error |
|---|---|---|
| Total docs relacionados | Máximo 50 | `ExceededRelatedDocsLimit` |
| Tipos mezclados | No permitido (todos del mismo tipo) | `MixedDocumentTypesNotAllowed` |
| Fecha de emisión del relacionado | No puede ser posterior al día de hoy (se compara el día, no la hora) | `InvalidRelatedDocDate` |
| Número (generación electrónica, tipo 2) | UUID válido: `^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$` | `InvalidRelatedDocNumberContingency` |
| Número (otra generación) | Máximo 20 caracteres | `InvalidRelatedDocNumberNormal` |
| Tipo permitido según el documento emitido | Solo la NR tiene restricción en esta estrategia: puede relacionar `01`, `03` o `11` | `InvalidRelatedDocDTEType` |
| Plazo de generación respecto del relacionado | Ver tabla siguiente | `RelatedDocOutOfPeriod` |

**Plazos de generación (`ValidateRelatedDocPeriod`)**, contados desde la fecha de emisión del documento relacionado (si se desconoce, no se valida). La comparación es por día:

| Documento relacionado | Plazo máximo para generar el documento | Referencia en la Normativa |
|---|---|---|
| Nota de remisión (`04`) | Fin del mes de la NR más 3 días | Anexo IV, "Fecha de emisión" del documento relacionado, regla 2.a (FE, CCFE o FEXE que relacionan una NR) |
| Documento contable de liquidación (`09`) | Fin del mes del documento relacionado | Mismo campo, regla 2.b (FE o CCFE que relacionan un DCL) |
| CCF (`03`) o CR (`07`) ajustado por una NC o ND | 3 meses desde la fecha de generación del relacionado (`calendar.AddMonthsClamped`: suma de calendario con tope en el último día del mes, p. ej. 30 de noviembre + 3 meses = 28 de febrero) | Mismo campo, regla 2.c (NCE o NDE que relacionan un CCF o CR) |

Para los demás casos no se evalúa plazo. Cada documento aplica además reglas propias sobre los relacionados en su estrategia específica (tipos permitidos y referencia obligatoria en los ítems); ver la sección "Documentos relacionados" de cada guía.

---

### ThirdPartyStrategy

**Cuando hay venta a terceros:** el tercero debe tener nombre (`RequiredField`). Las ventas por cuenta de terceros van en documentos separados de las propias (Anexo IV, sección 6): al informar el tercero, todo el documento corresponde a él. El validador no compara ítems individuales contra el tercero.

---

## Tolerancias Numéricas

| Contexto | Regla |
|---|---|
| Campos calculados | Holgura de ±0.01 (regla 7.3 de la Normativa), implementada en `WithinTolerance` (`calculation_rules.go`) |
| Resumen | Múltiplos de 0.01 (`InvalidDecimals`) |
| Cuerpo | Hasta 8 decimales (se redondean a 8) |

Las fórmulas, los decimales por campo, los ejemplos y los códigos de error se encuentran en [Cálculos que valida la API](../../../api/calculations.md).

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
| `internal/domain/dte/common/validator/strategy/` | Estrategias comunes, reglas compartidas de cálculo (`calculation_rules.go`) y de documentos relacionados (`related_docs_rules.go`) |
| `internal/domain/dte/common/validator/base_validator.go` | Validador base por reflexión |
| `internal/domain/dte/invoice/validator/` | Estrategias de factura |
| `internal/domain/dte/ccf/validator/` | Estrategias de CCF |
| `internal/domain/dte/fse/validator/` | Estrategias de FSE |
| `internal/domain/dte/credit_note/validator/` | Estrategias de nota de crédito |
| `internal/domain/dte/debit_note/validator/` | Estrategias de nota de débito |
| `internal/domain/dte/retention/validator/` | Estrategias de retención |
| `internal/domain/dte/remission_note/validator/` | Estrategias de nota de remisión |
| `internal/domain/dte/invalidation/validator/` | Estrategias de invalidación |
| `internal/domain/dte/invalidation/policy/` | Plazos y fecha del evento de invalidación |
| `internal/domain/dte/common/calendar/` | Calendario de días hábiles |
| `internal/domain/dte/common/dte_errors/` | Tipos de error |
