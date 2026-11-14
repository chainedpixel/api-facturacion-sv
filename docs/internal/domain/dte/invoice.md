# Factura Electrónica (DTE 01) - Invoice Service

> **Código DTE:** `01`
> **Paquete:** `internal/domain/dte/invoice`
> **Servicio:** `invoiceService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Factura Electrónica es el documento tributario electrónico base del sistema. Se utiliza para registrar operaciones de venta de bienes y servicios gravados, exentos o no sujetos al IVA, dirigidas a consumidores finales.

---

## Estructura del Servicio

```go
type invoiceService struct {
    validator        *validator.InvoiceRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.InvoiceRulesValidator` | Se validan las reglas de negocio específicas de factura |
| `seqNumberManager` | `dte_documents.SequentialNumberManager` | Se gestiona la reserva y generación de números de control |

### Constructor

```go
func NewInvoiceService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService
```

Se crea una instancia del servicio de factura electrónica. El `InvoiceRulesValidator` se instancia internamente.

---

## Método Principal: `Create`

```go
func (s *invoiceService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input genérico → *InvoiceData
2. Crear documento base (DTEDocument) desde InvoiceData
3. Ensamblar ElectronicInvoice con ítems y resumen específicos
4. Validar reglas específicas de factura (InvoiceRulesValidator)
5. Validar reglas genéricas de DTE (ValidateDTEDocument)
6. Generar número de control (SequentialNumberManager) y código de generación (UUID)
7. Retornar ElectronicInvoice validada y con identificadores asignados
```

### Paso 1: Cast del Input

Se convierte el `interface{}` recibido al tipo concreto `*invoice_models.InvoiceData`. El cast no está protegido: si el tipo no coincide se produce un pánico.

### Paso 2: Crear Documento Base

```go
func createBaseDocument(data *invoice_models.InvoiceData) *models.DTEDocument
```

Se transforma `InvoiceData` en un `DTEDocument` base, mapeando:
- Identificación, Emisor, Receptor (si no se envía receptor se usa un receptor vacío con dirección vacía)
- Documentos Relacionados (la `Extension` de `InputDataCommon` es un remanente sin uso: no se recibe en el request, siempre es nil y no se envía a Hacienda)
- Otros Documentos, Venta a Terceros, Apéndices

### Paso 3: Ensamblar ElectronicInvoice

Se construye el modelo `ElectronicInvoice` embebiendo el documento base y agregando:
- `InvoiceItems` - Ítems con campos específicos de factura
- `InvoiceSummary` - Resumen financiero con campos específicos de factura

### Paso 4: Validar Reglas de Negocio

```go
func (s *invoiceService) validate(invoice *invoice_models.ElectronicInvoice) error
```

Se crea `InvoiceRulesValidator` con el documento y se ejecuta `Validate()`, que aplica tres estrategias (`InvoiceItemsStrategy`, `InvoiceTaxStrategy` e `InvoiceTotalsStrategy`) y agrupa los errores en un error compuesto (ver [validación de factura](../validation/invoice-validation.md)). Luego se ejecuta la validación genérica `ValidateDTEDocument`.

`InvoiceRelatedDocStrategy` (documentos relacionados permitidos `04`, `08` y `09`) existe en el paquete de estrategias, pero no forma parte de la lista de estrategias de `InvoiceRulesValidator`.

### Paso 5: Generar Identificadores

```go
func (s *invoiceService) generateCodeAndIdentifiers(ctx context.Context, invoice *invoice_models.ElectronicInvoice, branchID uint) error
```

1. Se genera y asigna el número de control (`generateControlNumber`)
2. Se genera un UUID v4 como código de generación (`GenerateCode()`)

```go
func (s *invoiceService) generateControlNumber(ctx context.Context, invoice *invoice_models.ElectronicInvoice, branchID uint) error
```

Se utiliza `seqNumberManager.GetNextControlNumber(ctx, dteType, branchID, posCode, establishmentCode)` con:
- `dteType`: `"01"`
- `branchID`: ID de la sucursal
- `posCode`: Código del punto de venta del emisor
- `establishmentCode`: Código del establecimiento del emisor

El número obtenido se asigna con `Identification.SetControlNumber` (error `FailedToSetControlNumber` si no es válido). Ver [sequential-numbers.md](../sequential-numbers.md).

---

## Modelos Específicos de Factura

### ElectronicInvoice

```go
type ElectronicInvoice struct {
    *models.DTEDocument
    InvoiceItems   []InvoiceItem
    InvoiceSummary InvoiceSummary
}
```

El modelo principal que extiende `DTEDocument` con ítems y resumen propios de factura.

### InvoiceData (Input)

```go
type InvoiceData struct {
    *models.InputDataCommon
    Items          []InvoiceItem
    InvoiceSummary *InvoiceSummary
}
```

Modelo de entrada que encapsula los datos comunes del DTE más los específicos de factura.

### InvoiceItem

```go
type InvoiceItem struct {
    *models.Item
    NonSubjectSale financial.Amount  // Venta no sujeta
    ExemptSale     financial.Amount  // Venta exenta
    TaxedSale      financial.Amount  // Venta gravada
    SuggestedPrice financial.Amount  // Precio sugerido
    NonTaxed       financial.Amount  // Monto no gravado
    IVAItem        financial.Amount  // IVA del ítem
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `NonSubjectSale` | `Amount` | Monto de venta no sujeta al IVA |
| `ExemptSale` | `Amount` | Monto de venta exenta de IVA |
| `TaxedSale` | `Amount` | Monto de venta gravada con IVA |
| `SuggestedPrice` | `Amount` | Precio sugerido del artículo |
| `NonTaxed` | `Amount` | Monto no gravado (operaciones especiales) |
| `IVAItem` | `Amount` | IVA calculado a nivel de ítem |

#### Regla de Exclusividad de Tipo de Venta

Cada ítem puede tener solo **un tipo de venta**. No se permite mezclar `NonSubjectSale`, `ExemptSale` y `TaxedSale` en el mismo ítem.

#### Cálculo del IVA a Nivel de Ítem

En factura, el IVA va **incluido en el precio** y se informa en `IVAItem` por ítem y en `TotalIva` en el resumen; no se informa como tributo (los códigos `20` y `C3` no se permiten en los tributos del ítem ni del resumen, `TaxNotAllowedInInvoice` y `TaxNotAllowedInInvoiceSummary`). La fórmula y la tolerancia se documentan en [calculations.md](../../../api/calculations.md).

### InvoiceSummary

```go
type InvoiceSummary struct {
    *models.Summary
    TaxedDiscount           financial.Amount  // Descuento sobre gravado
    IVARetention            financial.Amount  // Retención de IVA
    IncomeRetention         financial.Amount  // Retención de renta
    TotalIva                financial.Amount  // Total IVA
    BalanceInFavor          financial.Amount  // Saldo a favor
    ElectronicPaymentNumber *string           // Número de pago electrónico
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `TaxedDiscount` | `Amount` | Descuento aplicado sobre el monto gravado |
| `IVARetention` | `Amount` | Retención de IVA (solo con venta gravada, `InvalidIVARetentionWithoutTaxedAmount`) |
| `IncomeRetention` | `Amount` | Retención de renta; debe ser 0 (`IncomeRetentionNotAllowed`, eliminada en la Normativa de Cumplimiento DTE v2.0) |
| `TotalIva` | `Amount` | Total del IVA calculado |
| `BalanceInFavor` | `Amount` | Saldo a favor del contribuyente |
| `ElectronicPaymentNumber` | `*string` | Referencia de pago electrónico (opcional) |

---

## Cálculos

Las fórmulas de la factura (IVA por ítem, subtotales, total de la operación y total a pagar) y la tolerancia se documentan en [calculations.md](../../../api/calculations.md). Reglas de coherencia que valida el dominio:

- Los totales gravado, exento y no sujeto del resumen deben coincidir con la suma de los ítems.
- `SubTotalSales` es la suma de no sujeto, exento y gravado; `SubTotal` descuenta además los descuentos del resumen.
- Los descuentos no pueden ser negativos ni exceder su base (`NegativeDiscount`, `DiscountExceedsBase`).
- El total de la operación es el subtotal más los tributos del resumen.
- La venta de cada ítem debe ser `(precio unitario x cantidad) - descuento` (`InvalidItemSaleAmount`).
- Máximo 2000 ítems (`ExceededItemsLimit`).

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Pánico por cast | El `interface{}` no es `*InvoiceData` |
| `ValidationFailed` | Las reglas de negocio específicas no se cumplen (envuelve el error compuesto; ver validación) |
| Error de generación de código | Falla al generar UUID |
| Error de número de control | Falla al obtener el siguiente número (propagado desde `SequentialNumberManager`) o `FailedToSetControlNumber` |

---

## Diagrama de Flujo

```
InvoiceData
    │
    ▼
createBaseDocument() ──→ DTEDocument
    │
    ▼
ElectronicInvoice { DTEDocument + InvoiceItems + InvoiceSummary }
    │
    ▼
validate() ──→ InvoiceRulesValidator
    │                 │
    │    ┌────────────┼────────────┐
    │    ▼            ▼            ▼
    │  Items         Tax         Totals
    │  Strategy    Strategy     Strategy
    │
    ▼
ValidateDTEDocument() (reglas genéricas)
    │
    ▼
generateCodeAndIdentifiers()
    │
    ├──→ GetNextControlNumber() (Número de Control)
    │
    └──→ GenerateCode() (UUID v4)
    │
    ▼
ElectronicInvoice (validada y con identificadores)
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/invoice/invoice_service.go` | Servicio principal |
| `internal/domain/dte/invoice/invoice_models/` | Modelos de factura |
| `internal/domain/dte/invoice/validator/` | Estrategias de validación |
| `internal/domain/dte/common/models/` | Modelos base compartidos |
| `internal/domain/dte/common/constants/` | Constantes del sistema |

---

## Notas

1. **IVA incluido**: A diferencia del CCF, en factura el IVA está incluido en el precio y se informa en `IVAItem` y `TotalIva`.
2. **Receptor opcional**: Aunque la factura requiere receptor, algunos campos del receptor son opcionales (NRC, ActivityCode).
3. **Precio sugerido**: El campo `SuggestedPrice` es informativo y no afecta los cálculos.
4. **No gravado**: Los ítems con `NonTaxed > 0` no pueden tener otros tipos de venta y deben tener `UnitPrice = 0` (`InvalidUnitPriceForNonTaxed`).
