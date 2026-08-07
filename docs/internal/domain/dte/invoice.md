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
1. Cast del input genérico → InvoiceData
2. Crear documento base (DTEDocument) desde InvoiceData
3. Ensamblar ElectronicInvoice con ítems y resumen específicos
4. Validar reglas de negocio de factura
5. Generar código de generación (UUID) y número de control
6. Retornar ElectronicInvoice validada y con identificadores asignados
```

### Paso 1: Cast del Input

Se convierte el `interface{}` recibido al tipo concreto `*invoice_models.InvoiceData`. Si el cast falla, se retorna un error de tipo.

### Paso 2: Crear Documento Base

```go
func (s *invoiceService) createBaseDocument(data *invoice_models.InvoiceData) *models.DTEDocument
```

Se transforma `InvoiceData` en un `DTEDocument` base, mapeando:
- Identificación, Emisor, Receptor
- Extensión, Documentos Relacionados
- Otros Documentos, Venta a Terceros, Apéndices

### Paso 3: Ensamblar ElectronicInvoice

Se construye el modelo `ElectronicInvoice` embebiendo el documento base y agregando:
- `InvoiceItems` - Ítems con campos específicos de factura
- `InvoiceSummary` - Resumen financiero con campos específicos de factura

### Paso 4: Validar Reglas de Negocio

```go
func (s *invoiceService) validate(invoice *invoice_models.ElectronicInvoice) error
```

Se ejecuta `s.validator.Validate(invoice)` que aplica las estrategias de validación específicas de factura (ver [validación de factura](../validation/invoice-validation.md)).

### Paso 5: Generar Identificadores

```go
func (s *invoiceService) generateCodeAndIdentifiers(ctx context.Context, invoice *invoice_models.ElectronicInvoice, branchID uint) error
```

1. Se genera un UUID v4 como código de generación (`GenerateCode()`)
2. Se reserva el siguiente número de control secuencial

```go
func (s *invoiceService) generateControlNumber(ctx context.Context, invoice *invoice_models.ElectronicInvoice, branchID uint) error
```

Se utiliza `seqNumberManager.ReserveNextNumber()` con los parámetros:
- `dteType`: `"01"`
- `branchID`: ID de la sucursal
- `posCode`: Código del punto de venta
- `establishmentCode`: Código del establecimiento
- `documentData`: Datos del documento para trazabilidad
- `isContingency`: `false` para transmisión normal

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

En factura, el IVA se calcula **incluido en el precio** (IVA incluido):

```
IVA = (TaxedSale / 1.13) * 0.13
```

Tolerancia: `±0.01`

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
| `IVARetention` | `Amount` | Retención de IVA (1% cuando aplica) |
| `IncomeRetention` | `Amount` | Retención de renta |
| `TotalIva` | `Amount` | Total del IVA calculado |
| `BalanceInFavor` | `Amount` | Saldo a favor del contribuyente |
| `ElectronicPaymentNumber` | `*string` | Referencia de pago electrónico (opcional) |

---

## Fórmulas de Cálculo

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount + TotalExempt - ExemptDiscount + TotalNonSubject - NonSubjectDiscount
```

### Total Operación

```
TotalOperation = SubTotal + Sum(Taxes)
```

### Total a Pagar

```
TotalToPay = SubTotal + Taxes - IVARetention - IncomeRetention + NonTaxed
```

### Validación de Totales de Ítems

```
Sum(item.TaxedSale)      == Summary.TotalTaxed       (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)
```

---

## Condiciones de Error

| Error | Causa |
|---|---|
| Cast fallido del input | El `interface{}` no es `*InvoiceData` |
| Error de validación | Las reglas de negocio no se cumplen (ver validación) |
| Error de generación de código | Falla al generar UUID |
| Error de número de control | Falla al reservar número secuencial |

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
    │  Items      Tax/Totals   Related Docs
    │  Strategy   Strategy     Strategy
    │
    ▼
generateCodeAndIdentifiers()
    │
    ├──→ GenerateCode() (UUID v4)
    │
    └──→ ReserveNextNumber() (Número de Control)
    │
    ▼
ElectronicInvoice (validada y con identificadores)
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/invoice/invoice_service.go` | Servicio principal |
| `internal/domain/dte/invoice/models/invoice_model.go` | Modelos de factura |
| `internal/domain/dte/invoice/validator/` | Estrategias de validación |
| `internal/domain/dte/common/models/` | Modelos base compartidos |
| `internal/domain/dte/common/constants/` | Constantes del sistema |

---

## Notas

1. **IVA incluido**: A diferencia del CCF, en factura el IVA está incluido en el precio. El cálculo es `TaxedSale / 1.13 * 0.13`.
2. **Receptor opcional**: Aunque la factura requiere receptor, algunos campos del receptor son opcionales (NRC, ActivityCode).
3. **Precio sugerido**: El campo `SuggestedPrice` es informativo y no afecta los cálculos.
4. **No gravado**: Los ítems con `NonTaxed > 0` no pueden tener otros tipos de venta y deben tener `UnitPrice = 0`.
