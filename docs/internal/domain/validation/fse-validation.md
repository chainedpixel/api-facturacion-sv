# Validación de FSE (DTE 14)

> **Validador:** `FSERulesValidator`
> **Paquete:** `internal/domain/dte/fse/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

Todas se ejecutan; cada estrategia agrupa sus propios errores y el validador agrupa los de las cuatro:

1. **FSEItemStrategy** — Validación de ítems de compra
2. **FSETaxStrategy** — Validación de retenciones y total a pagar
3. **FSEReceiverStrategy** — Validación del receptor sujeto excluido
4. **FSEDiscountStrategy** — Validación de compras, descuentos y subtotal

Luego se aplican las [estrategias comunes](./README.md) (incluida `PaymentTotalStrategy`: la suma de pagos debe coincidir con `TotalToPay`).

---

## 1. FSEItemStrategy

**Archivo:** `internal/domain/dte/fse/validator/strategy/fse_item_strategy.go`

| Validación | Regla | Error |
|---|---|---|
| Lista de ítems | Al menos un ítem requerido | `FSEItemInvalidEmpty` |
| Purchase | Debe ser `> 0` | `FSEItemInvalidPurchase` |
| Description | No puede ser vacía | `FSEItemInvalidDescription` |
| Quantity | Debe ser `> 0` | `FSEItemInvalidQuantity` |
| UnitPrice | Debe ser `> 0` | `FSEItemInvalidUnitPrice` |

> **Nota:** A diferencia de otros DTEs, el FSE valida `Purchase` como campo principal, no tipos de venta.

---

## 2. FSETaxStrategy

**Archivo:** `internal/domain/dte/fse/validator/strategy/fse_tax_strategy.go`

### Retenciones

| Validación | Regla | Error |
|---|---|---|
| IVARetention | Debe ser `>= 0` y además **debe ser `0`** (el IVA retenido fue eliminado en FSE por la Normativa v2.0, Anexo VI) | `FSETaxInvalidIVARetention`, `IVARetentionNotAllowedInFSE` |
| IncomeRetention | Debe ser `>= 0` (campo 147; la API no valida la tasa ni la base) | `FSETaxInvalidIncomeRetention` |

### Total a Pagar (campo 153)

| Fórmula | Tolerancia | Error |
|---|---|---|
| `TotalToPay = SubTotal - IncomeRetention` | ±0.01 | `FSETaxInvalidTotalToPay` |

El total de compras y el subtotal se validan en `FSEDiscountStrategy`.

### Restricción de IVA

Si `TotalTaxed > 0` se devuelve `FSETaxInvalidTaxedSale`. El mapeo de FSE no recibe `total_taxed`, por lo que en la práctica el valor es 0.

---

## 3. FSEReceiverStrategy

**Archivo:** `internal/domain/dte/fse/validator/strategy/fse_receiver_strategy.go`

### Campos Requeridos

| Campo | Requerido | Descripción | Error |
|---|---|---|---|
| DocumentType | Sí | Tipo de documento de identidad (36, 13, 02, 03 o 37) | `FSEReceiverRequiredDocumentType`, `FSEReceiverInvalidDocumentType` |
| DocumentNumber | Sí | Número de documento | `FSEReceiverRequiredDocumentNumber` |
| Name | Sí | Nombre del sujeto excluido | `FSEReceiverRequiredName` |
| Address | Sí | Dirección | `FSEReceiverRequiredAddress` |

### Validación por Tipo de Documento

| Tipo | Código | Validación del Número | Error |
|---|---|---|---|
| NIT | `36` | 9 o 14 dígitos exactos | `FSEReceiverInvalidNIT` |
| DUI | `13` | 9 dígitos exactos | `FSEReceiverInvalidDUI` |
| Carnet Residente | `02` | Máximo 20 caracteres | `FSEReceiverInvalidDocumentLength` |
| Pasaporte | `03` | Máximo 20 caracteres | `FSEReceiverInvalidDocumentLength` |
| Otro | `37` | Máximo 20 caracteres | `FSEReceiverInvalidDocumentLength` |

### Campos Opcionales

| Campo | Validación (si presente) | Error |
|---|---|---|
| ActivityCode | Formato válido | `FSEReceiverInvalidActivityCode` |
| ActivityDescription | Máximo 150 caracteres | `FSEReceiverInvalidActivityDescription` |

---

## 4. FSEDiscountStrategy

**Archivo:** `internal/domain/dte/fse/validator/strategy/fse_discount_strategy.go`

Los errores de las tres verificaciones se agrupan.

### Compra y Descuento por Ítem

| Fórmula | Tolerancia | Error |
|---|---|---|
| `Discount <= (Quantity * UnitPrice)` | — | `ExcessiveItemDiscount` |
| `Purchase = (Quantity * UnitPrice) - Discount` (campo 95) | ±0.01 | `InvalidItemSaleAmount` |

### Resumen

| Fórmula | Tolerancia | Error |
|---|---|---|
| `TotalPurchase = Sum(item.Purchase)` | ±0.01 | `InvalidTotalPurchase` |
| `SubTotal = TotalPurchase - NonSubjectDiscount` (en FSE `non_subject_discount` es el descuento global al total, campo 133) | ±0.01 | `InvalidSubTotal` |
| `TotalDiscount = Sum(item.Discount) + NonSubjectDiscount` (informativo, campo 135) | ±0.01 | `InvalidTotalDiscount` |

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
| `IVARetentionNotAllowedInFSE` | Retención IVA distinta de 0 |
| `FSETaxInvalidIncomeRetention` | Retención renta negativa |
| `FSETaxInvalidTotalToPay` | Total a pagar incorrecto |
| `FSETaxInvalidTaxedSale` | TotalTaxed distinto de 0 |
| `FSEReceiverRequiredName` | Nombre faltante |
| `FSEReceiverRequiredAddress` | Dirección faltante |
| `FSEReceiverRequiredDocumentType` | Tipo de documento faltante |
| `FSEReceiverRequiredDocumentNumber` | Número de documento faltante |
| `FSEReceiverInvalidDocumentType` | Tipo de documento fuera del catálogo |
| `FSEReceiverInvalidNIT` | NIT con formato inválido |
| `FSEReceiverInvalidDUI` | DUI con formato inválido |
| `FSEReceiverInvalidDocumentLength` | Número excede 20 caracteres |
| `FSEReceiverInvalidActivityCode` | Código de actividad inválido |
| `FSEReceiverInvalidActivityDescription` | Descripción de actividad > 150 caracteres |
| `ExcessiveItemDiscount` | Descuento excede el valor bruto del ítem |
| `InvalidItemSaleAmount` | `Purchase` no coincide con la fórmula |
| `InvalidTotalPurchase` | Total de compras no coincide con la suma |
| `InvalidSubTotal` | Subtotal incorrecto |
| `InvalidTotalDiscount` | Total de descuentos incorrecto |
