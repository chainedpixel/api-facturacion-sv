# Validación de FSE (DTE 14)

> **Validador:** `FSERulesValidator`
> **Paquete:** `internal/domain/dte/fse/validator`

## Estrategias Compuestas

1. **FSEItemStrategy** — Validación de ítems de compra
2. **FSETaxStrategy** — Validación de retenciones y totales
3. **FSEReceiverStrategy** — Validación del receptor sujeto excluido
4. **FSEDiscountStrategy** — Validación de descuentos por ítem y en resumen

---

## 1. FSEItemStrategy

**Archivo:** `internal/domain/dte/fse/validator/fse_item_strategy.go`

| Validación | Regla |
|---|---|
| Lista de ítems | Al menos un ítem requerido |
| Purchase | Debe ser `> 0` |
| Description | No puede ser vacía |
| Quantity | Debe ser `> 0` |
| UnitPrice | Debe ser `> 0` |

> **Nota:** A diferencia de otros DTEs, el FSE valida `Purchase` como campo principal, no tipos de venta.

---

## 2. FSETaxStrategy

**Archivo:** `internal/domain/dte/fse/validator/fse_tax_strategy.go`

### Retenciones

| Validación | Regla |
|---|---|
| IVARetention | Debe ser `>= 0` |
| IncomeRetention | Debe ser `>= 0` |

### Totales

| Fórmula | Tolerancia |
|---|---|
| `TotalPurchase = Sum(item.Purchase)` | ±0.01 |
| `TotalToPay = TotalPurchase - IVARetention - IncomeRetention` | ±0.01 |

### Restricción de IVA

```
TaxedSale DEBE ser 0
```

El FSE es un documento de compra a sujeto excluido — no hay venta gravada.

---

## 3. FSEReceiverStrategy

**Archivo:** `internal/domain/dte/fse/validator/fse_receiver_strategy.go`

### Campos Requeridos

| Campo | Requerido | Descripción |
|---|---|---|
| DocumentType | Sí | Tipo de documento de identidad |
| DocumentNumber | Sí | Número de documento |
| Name | Sí | Nombre del sujeto excluido |
| Address | Sí | Dirección |

### Validación por Tipo de Documento

| Tipo | Código | Validación del Número |
|---|---|---|
| NIT | `36` | 9 o 14 dígitos exactos |
| DUI | `13` | 9 dígitos exactos |
| Carnet Residente | `02` | Máximo 20 caracteres |
| Pasaporte | `03` | Máximo 20 caracteres |
| Otro | `37` | Máximo 20 caracteres |

### Campos Opcionales

| Campo | Validación (si presente) |
|---|---|
| ActivityCode | Formato válido |
| ActivityDescription | Máximo 150 caracteres |

---

## 4. FSEDiscountStrategy

**Archivo:** `internal/domain/dte/fse/validator/fse_discount_strategy.go`

### Descuento por Ítem

| Fórmula | Tolerancia |
|---|---|
| `Purchase = (Quantity * UnitPrice) - Discount` | ±0.01 |
| `Discount <= (Quantity * UnitPrice)` | — |

### Descuentos en Resumen

| Fórmula | Tolerancia |
|---|---|
| `TotalPurchase = Sum(item.Purchase)` | ±0.01 |
| `SubTotal = TotalPurchase - NonSubjectDiscount` | ±0.01 |

### Total Descuento

```
TotalDiscount = Sum(item.Discount) + NonSubjectDiscount    (±0.01)
```

---

## Errores Específicos de FSE

| Error | Causa |
|---|---|
| `FSEItemInvalidEmpty` | Lista de ítems vacía |
| `FSEItemInvalidPurchase` | Purchase <= 0 |
| `FSEItemInvalidDescription` | Descripción vacía |
| `FSEItemInvalidQuantity` | Quantity <= 0 |
| `FSEItemInvalidUnitPrice` | UnitPrice <= 0 |
| `FSETaxInvalidIVARetention` | Retención IVA negativa |
| `FSETaxInvalidIncomeRetention` | Retención renta negativa |
| `FSETaxInvalidTotalToPay` | Total a pagar incorrecto |
| `FSETaxInvalidTaxedSale` | TaxedSale != 0 |
| `FSEReceiverRequiredName` | Nombre faltante |
| `FSEReceiverRequiredAddress` | Dirección faltante |
| `FSEReceiverRequiredDocumentType` | Tipo de documento faltante |
| `FSEReceiverRequiredDocumentNumber` | Número de documento faltante |
| `FSEReceiverInvalidNIT` | NIT con formato inválido |
| `FSEReceiverInvalidDUI` | DUI con formato inválido |
| `FSEReceiverInvalidDocumentLength` | Número excede 20 caracteres |
| `InvalidItemPurchase` | Purchase no coincide con fórmula |
| `ExcessiveItemDiscount` | Descuento excede valor bruto |
| `InvalidTotalPurchase` | Total no coincide con suma |
| `InvalidSubTotal` | Subtotal incorrecto |
| `InvalidTotalDiscount` | Total descuento incorrecto |
