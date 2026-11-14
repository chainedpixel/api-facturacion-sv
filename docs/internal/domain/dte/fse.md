# Factura de Sujeto Excluido Electrónica (DTE 14) - FSE Service

> **Código DTE:** `14`
> **Paquete:** `internal/domain/dte/fse`
> **Servicio:** `fseService`
> **Interfaz implementada:** `ports.DTEService`

## Descripción General

La Factura de Sujeto Excluido (FSE) se emite cuando el emisor (contribuyente) adquiere bienes o servicios de un **sujeto excluido del IVA** — personas naturales que no son contribuyentes registrados. En este tipo de documento, se registra una **compra**, no una venta.

Características principales:
- **No tiene IVA** — El sujeto excluido no cobra IVA
- **Registra compras** — El campo principal del ítem es `Purchase`, no tipos de venta
- **Retención de renta** — El emisor puede retener renta (`IncomeRetention`); la retención de IVA no aplica y debe ser 0 (eliminada en la Normativa de Cumplimiento DTE v2.0, Anexo VI)
- **Receptor especial** — Usa `FSEReceiver` con estructura diferente al receptor estándar

---

## Estructura del Servicio

```go
type fseService struct {
    validator        *validator.FSERulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
}
```

### Dependencias

| Dependencia | Tipo | Propósito |
|---|---|---|
| `validator` | `*validator.FSERulesValidator` | Se validan reglas específicas de FSE |
| `seqNumberManager` | `dte_documents.SequentialNumberManager` | Se gestiona la reserva de números de control |

### Constructor

```go
func NewFSEService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService
```

---

## Método Principal: `Create`

```go
func (s *fseService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error)
```

### Flujo de Ejecución

```
1. Cast del input → *FSEData
2. Verificar que exista FSEReceiver (si no, ReceiverIsRequired)
3. Crear documento base FSE (con receptor especial)
4. Ensamblar FSEModel con ítems, resumen y receptor específicos
5. Validar reglas específicas de FSE (FSERulesValidator)
6. Validar reglas genéricas de DTE (ValidateDTEDocument)
7. Generar número de control (SequentialNumberManager) y código de generación (UUID)
8. Retornar FSEModel validado
```

El `FSERulesValidator` compone cuatro estrategias: `FSEItemStrategy`, `FSETaxStrategy`, `FSEReceiverStrategy` y `FSEDiscountStrategy`, y agrupa los errores en un error compuesto.

### Diferencia en Creación del Documento Base

A diferencia de otros DTEs, el FSE usa `createBaseFSEDocument()` en lugar de `createBaseDocument()` porque:
- Los ítems se convierten con `convertFSEItemsToInterface()` para mapear `FSEItem` → `interfaces.Item`
- El receptor usa el modelo `FSEReceiver` en lugar de `Receiver` estándar

---

## Modelos Específicos del FSE

### FSEModel

```go
type FSEModel struct {
    *models.DTEDocument
    FSEItems    []FSEItem
    FSESummary  FSESummary
    FSEReceiver FSEReceiver
}
```

### FSEData (Input)

```go
type FSEData struct {
    *models.InputDataCommon
    Items       []FSEItem
    FSESummary  *FSESummary
    FSEReceiver *FSEReceiver
}
```

### FSEItem

```go
type FSEItem struct {
    *models.Item
    Purchase financial.Amount  // Monto de compra
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `Purchase` | `Amount` | Monto de la compra del ítem. Es el campo principal del FSE. |

**Fórmula del Purchase:**
```
Purchase = (Quantity * UnitPrice) - Discount
```

> Los campos `TaxedSale`, `ExemptSale`, `NonSubjectSale` del ítem base no se utilizan en FSE.

### FSESummary

```go
type FSESummary struct {
    *models.Summary
    TotalPurchase   financial.Amount  // Total de compras
    IVARetention    financial.Amount  // Retención de IVA
    IncomeRetention financial.Amount  // Retención de renta
    Observations    *string           // Observaciones
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `TotalPurchase` | `Amount` | Suma total de todas las compras (`Sum(item.Purchase)`) |
| `IVARetention` | `Amount` | Retención de IVA aplicada al sujeto excluido |
| `IncomeRetention` | `Amount` | Retención de Impuesto sobre la Renta |
| `Observations` | `*string` | Observaciones opcionales del documento |

### FSEReceiver

```go
type FSEReceiver struct {
    *models.Receiver
    DocumentType        document.DTEType
    DocumentNumber      identification.DocumentNumber
    ActivityCode        *identification.ActivityCode
    ActivityDescription *string
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `DocumentType` | `DTEType` | Tipo de documento de identidad del receptor |
| `DocumentNumber` | `DocumentNumber` | Número de documento del receptor |
| `ActivityCode` | `*ActivityCode` | Código de actividad económica (opcional; si se envía debe ser válido) |
| `ActivityDescription` | `*string` | Descripción de la actividad (opcional; máximo 150 caracteres) |

El nombre y la dirección del receptor (del `Receiver` base) son obligatorios.

#### Tipos de Documento Válidos para FSEReceiver

| Código | Tipo | Validación |
|---|---|---|
| `36` | NIT | 9 o 14 dígitos |
| `13` | DUI | 9 dígitos |
| `02` | Carnet de Residente | Máximo 20 caracteres |
| `03` | Pasaporte | Máximo 20 caracteres |
| `37` | Otro Documento | Máximo 20 caracteres |

---

## Cálculos

Las fórmulas del FSE (total de compras, subtotal y total a pagar) y la tolerancia aplicada se documentan en [calculations.md](../../../api/calculations.md). Reglas que valida el dominio:

- `TotalPurchase` debe ser la suma de `Purchase` de los ítems.
- `SubTotal` debe ser `TotalPurchase` menos el descuento sobre no sujeto (`NonSubjectDiscount`); `TotalDiscount` debe ser la suma de descuentos por ítem más `NonSubjectDiscount`.
- `TotalToPay` debe ser `SubTotal` menos `IncomeRetention`.
- `IVARetention` debe ser 0 (`IVARetentionNotAllowedInFSE`) e `IncomeRetention` no puede ser negativa.
- `TotalTaxed` debe ser 0 (el FSE no tiene ventas gravadas).

---

## Validaciones de Descuentos

- Por ítem: el descuento no puede exceder el bruto `Quantity * UnitPrice` (`ExcessiveItemDiscount`) y `Purchase` debe ser `(Quantity * UnitPrice) - Discount` (`InvalidItemSaleAmount`).
- En el resumen: `InvalidTotalPurchase`, `InvalidSubTotal` e `InvalidTotalDiscount` cuando los totales declarados no coinciden con los calculados.

---

## Condiciones de Error

| Error | Causa |
|---|---|
| `FSEItemInvalidEmpty` | Lista de ítems vacía |
| `FSEItemInvalidPurchase` | Monto de compra <= 0 |
| `FSEItemInvalidDescription` | Descripción vacía |
| `FSEItemInvalidQuantity` | Cantidad <= 0 |
| `FSEItemInvalidUnitPrice` | Precio unitario <= 0 |
| `FSETaxInvalidIVARetention` | Retención de IVA inválida |
| `IVARetentionNotAllowedInFSE` | Retención de IVA distinta de 0 |
| `FSETaxInvalidIncomeRetention` | Retención de renta negativa |
| `FSETaxInvalidTotalToPay` | Cálculo de total a pagar incorrecto |
| `FSETaxInvalidTaxedSale` | Venta gravada diferente de 0 |
| `ReceiverIsRequired` | No se envió el receptor FSE |
| `FSEReceiverRequiredName` | Nombre del receptor faltante |
| `FSEReceiverRequiredAddress` | Dirección del receptor faltante |
| `FSEReceiverInvalidNIT` | NIT con formato inválido |
| `FSEReceiverInvalidDUI` | DUI con formato inválido |
| `InvalidItemSaleAmount` | Purchase no coincide con `(Qty * UnitPrice) - Discount` |
| `ExcessiveItemDiscount` | Descuento excede el valor bruto |
| `InvalidTotalPurchase` | Total de compras distinto de la suma de las compras |
| `InvalidSubTotal` | Subtotal distinto de total de compras menos descuento |
| `InvalidTotalDiscount` | Total de descuentos inconsistente |
| `FSEReceiverRequiredDocumentType`, `FSEReceiverRequiredDocumentNumber`, `FSEReceiverInvalidDocumentType`, `FSEReceiverInvalidDocumentLength` | Tipo o número de documento del receptor ausente o inválido |
| `FSEReceiverInvalidActivityCode`, `FSEReceiverInvalidActivityDescription` | Actividad económica inválida |

---

## Diagrama Comparativo: FSE vs Factura/CCF

```
Factura/CCF (Venta)              FSE (Compra)
─────────────────               ──────────────
Emisor vende                    Emisor compra
Receptor compra                 Receptor vende (sujeto excluido)

TaxedSale > 0                   TaxedSale = 0
ExemptSale posible              Purchase > 0
IVA calculado                   No hay IVA
                                IncomeRetention sobre compra
                                IVARetention no aplica (0)
```

---

## Archivos Relacionados

| Archivo | Propósito |
|---|---|
| `internal/domain/dte/fse/fse_service.go` | Servicio principal |
| `internal/domain/dte/fse/fse_models/` | Modelos del FSE |
| `internal/domain/dte/fse/validator/` | Estrategias de validación |

---

## Notas

1. **Concepto invertido**: El FSE registra una **compra** del emisor, no una venta. El emisor es quien paga.
2. **Sin IVA**: El campo `TaxedSale` debe ser siempre 0. El IVA no aplica en operaciones con sujetos excluidos.
3. **Retenciones**: El emisor puede retener renta; la retención de IVA no aplica.
4. **Receptor especial**: Se usa `FSEReceiver` con validaciones de documento de identidad diferentes al receptor estándar.
5. **Descuentos**: Se aplican a nivel de ítem (afectan `Purchase`) y a nivel de resumen (`NonSubjectDiscount`).
