# Validación de CCF Electrónico (DTE 03)

> **Validador:** `CCFRulesValidator`
> **Paquete:** `internal/domain/dte/ccf/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API](../../../api/calculations.md); aquí se resumen por estrategia.

## Estrategias Compuestas

Todas se ejecutan y los errores de estrategias distintas se agrupan:

1. **CCFItemStrategy** — Validación de ítems, tipos de venta y reglas de impuestos
2. **CCFTaxStrategy** — Validación de cálculos fiscales, IVA y percepción
3. **CCFReceiverStrategy** — Validación del receptor (NRC obligatorio)
4. **CCFRelatedDocStrategy** — Validación de documentos relacionados

Luego se aplican las [estrategias comunes](./README.md) (incluida `DocumentTypeStrategy`, que también exige NRC en CCF).

---

## 1. CCFItemStrategy

**Archivo:** `internal/domain/dte/ccf/validator/strategy/ccf_item_strategy.go`

Por cada ítem se evalúan, en orden: exclusividad de ventas, reglas de impuestos, reglas del tipo 4, no gravado y fórmula de la venta. Se devuelve el primer error.

### Exclusividad de Tipos de Venta

| Tipo | Restricción con otros tipos | Restricción de impuestos | Error |
|---|---|---|---|
| `NonTaxed` | No mezclar con otros; `UnitPrice = 0` | No puede tener impuestos | `InvalidMixedSalesWithNonTaxed`, `InvalidTaxesWithNonTaxed`, `InvalidUnitPriceWithNonTaxed` |
| `Exempt` | No mezclar con otros | No puede tener impuestos | `InvalidMixedSalesWithExempt`, `InvalidTaxesWithExempt` |
| `NonSubject` | No mezclar con otros | No puede tener impuestos | `InvalidMixedSalesWithNonSubject`, `InvalidTaxesWithNonSubject` |
| `Taxed` | Puede estar solo | Debe tener impuestos (`MissingTaxesItem`) | — |

### Reglas de Impuestos por Tipo de Ítem

| Tipo de Ítem | Código | Restricción | Error |
|---|---|---|---|
| Producto | 1 | Solo IVA (código `20`) | `InvalidTaxCodeOnly20` |
| Servicio | 2 | Cualquier código del catálogo soportado | `UnsupportedTaxCode` (estrategia común) |
| Producto y Servicio | 3 | Cualquier código del catálogo soportado | `UnsupportedTaxCode` (estrategia común) |
| Impuesto | 4 | Solo IVA (`taxes = ["20"]`), unidad de medida = `99` | `InvalidUnitMeasure`, `InvalidTaxRulesCCF`, `InvalidTaxCodeOnly20` |

### Fórmula de la Venta y Precio Unitario

- `NonSubjectSale + ExemptSale + TaxedSale ≈ (UnitPrice * Quantity) - Discount` (±0.01, campos 90 a 92), omitida si `UnitPrice = 0`. Error: `InvalidItemSaleAmount`.
- Si `TaxedSale > 0` → `UnitPrice` no puede ser 0 (`InvalidUnitPriceZero`).

### Documentos Relacionados en Ítems

- Si existen docs relacionados → todos los ítems deben referenciar uno (`MissingItemRelatedDoc`)
- Cada referencia debe existir en la lista de documentos (`InvalidItemRelatedDoc`)

---

## 2. CCFTaxStrategy

**Archivo:** `internal/domain/dte/ccf/validator/strategy/ccf_tax_strategy.go`

Orden interno: totales base, IVA, percepción, montos monetarios, totales y consistencia de no gravado. Se devuelve el primer error.

### Validación de Totales Base

```
Sum(item.TaxedSale)      == Summary.TotalTaxed        (±0.01)   InvalidTotalTaxed
Sum(item.ExemptSale)     == Summary.TotalExempt       (exacto)  InvalidTotalExempt
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (exacto)  InvalidTotalNonSubject
SubTotalSales = TotalTaxed + TotalExempt + TotalNonSubject      (±0.01)   InvalidSubTotalSales
```

### Descuentos

```
TaxedDiscount      <= TotalTaxed         DiscountExceedsBase
ExemptDiscount     <= TotalExempt        DiscountExceedsBase
NonSubjectDiscount <= TotalNonSubject    DiscountExceedsBase
```

`TotalDiscount` no se contrasta contra la suma de descuentos.

### Tributos del Resumen

Si `TotalTaxed = 0` y existen tributos → `InvalidTaxes`. Si `TotalTaxed > 0` y no hay tributos → `MissingTaxes`.

El IVA del CCF se calcula **después del descuento** global (error `InvalidTaxCalculation`, ±0.01):

```
IVA (20) = (TotalTaxed - TaxedDiscount) * 0.13
```

> **Diferencia con la Factura:** En FE el IVA va incluido en el precio y se informa en `TotalIva`; el CCF informa el IVA como tributo `20` del resumen.

Los demás tributos usan `TotalTaxed` sin restar el descuento; las tasas están en [Cálculos, sección 3.6](../../../api/calculations.md#36-tributos-del-resumen-campos-136-a-138).

### Percepción de IVA

```
Si IVAPerception != 0:
    IVAPerception = TotalTaxed * 0.01    (±0.01)    InvalidPerceptionAmount
```

La retención de IVA (`IVARetention`) no se recalcula: su valor solo interviene en el total a pagar.

### Montos Monetarios

`IVAPerception`, `TotalOperation`, `TotalToPay` y cada monto de pago deben ser múltiplos de 0.01 (`InvalidMonetaryAmount`). Los montos del resumen también se leen con 2 decimales exactos (`InvalidDecimals`).

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount
         + TotalExempt - ExemptDiscount
         + TotalNonSubject - NonSubjectDiscount
```
Holgura ±0.01. Error: `InvalidSubTotalCalculation`.

### IVA Obligatorio

```
Si TotalTaxed > 0 → debe existir el tributo 20 con valor > 0    MissingIVAForTaxedAmount
```

### Monto Total de la Operación (campo 150)

```
TotalOperation = SubTotal + Sum(Taxes)    (±0.01)    InvalidTotalOperation
```

### Retención de Renta

`IncomeRetention` debe ser `0` (eliminada para el CCF por la Normativa v2.0): `IncomeRetentionNotAllowed`.

### Total a Pagar (campo 153)

```
TotalToPay = TotalOperation + IVAPerception - IVARetention + TotalNonTaxed
```
Holgura ±0.01; si el resultado es negativo se espera `0`. Error: `InvalidTotalToPayCalculation`.

### Consistencia de No Gravado

```
Si Summary.TotalNonTaxed > 0 y Sum(item.NonTaxed) == 0  → InvalidNonTaxedAmount
Sum(item.NonTaxed) == Summary.TotalNonTaxed (comparación exacta)  → InvalidTotalNonTaxed
```

---

## 3. CCFReceiverStrategy

**Archivo:** `internal/domain/dte/ccf/validator/strategy/ccf_receiver_strategy.go`

| Campo | Requerido | Descripción | Error |
|---|---|---|---|
| Receptor | Sí | Debe existir | — |
| NRC | **Sí** | Número de Registro de Contribuyente | `MissingNRC` |
| ActivityCode | **Sí** | Código de actividad económica | `RequiredField` |
| ActivityDescription | **Sí** | Descripción de la actividad | `RequiredField` |

---

## 4. CCFRelatedDocStrategy

**Archivo:** `internal/domain/dte/ccf/validator/strategy/ccf_related_doc_strategy.go`

| Validación | Regla | Error |
|---|---|---|
| Máximo docs | 50 | `ExceededRelatedDocsLimit` |
| Tipos válidos | `04` (Remisión), `08` (Liquidación), `09` (Doc. Contable Liquidación) | `InvalidRelatedDocDTEType` |
| Ítems | Todos deben referenciar un doc relacionado | `MissingItemRelatedDoc` |
| Referencias | Deben existir en la lista del documento | `InvalidItemRelatedDoc` |

La estrategia común `RelatedDocsStrategy` aplica además el plazo de generación respecto de la NR (fin del mes del relacionado más 3 días) y del documento contable de liquidación (fin del mes). Ver [README, RelatedDocsStrategy](./README.md#relateddocsstrategy).

---

## Errores Específicos de CCF

| Error | Causa |
|---|---|
| `MissingNRC` | Receptor sin NRC |
| `InvalidTaxCodeOnly20` | Producto con impuesto diferente a IVA |
| `InvalidMixedSalesWithNonTaxed` | No gravado mezclado con otros |
| `InvalidTaxesWithNonTaxed` | Impuestos en ítem no gravado |
| `InvalidUnitPriceWithNonTaxed` | Precio unitario distinto de 0 con no gravado |
| `InvalidMixedSalesWithExempt` | Exento mezclado con otros |
| `InvalidTaxesWithExempt` | Impuestos en ítem exento |
| `InvalidMixedSalesWithNonSubject` | No sujeto mezclado con otros |
| `InvalidTaxesWithNonSubject` | Impuestos en ítem no sujeto |
| `MissingTaxesItem` | Gravado sin impuestos |
| `InvalidItemSaleAmount` | Venta del ítem no coincide con `(UnitPrice * Quantity) - Discount` |
| `InvalidPerceptionAmount` | Percepción no coincide con 1% |
| `MissingTaxes`, `InvalidTaxes` | Gravado sin tributos en el resumen; tributos sin gravado |
| `MissingIVAForTaxedAmount` | Gravado > 0 sin tributo `20` |
| `InvalidTaxCalculation` | Valor de un tributo del resumen incorrecto |
| `InvalidSubTotalSales`, `InvalidSubTotalCalculation` | Suma de operaciones o subtotal incorrectos |
| `InvalidTotalOperation` | `TotalOperation` incorrecto |
| `InvalidTotalToPayCalculation` | `TotalToPay` incorrecto |
| `IncomeRetentionNotAllowed` | `IncomeRetention` distinto de 0 |
| `InvalidNonTaxedAmount`, `InvalidTotalNonTaxed` | `TotalNonTaxed` no coincide con los ítems |

---

## Diagrama: CCF vs Factura (Validación de IVA)

```
FACTURA:
  IVA por ítem = (TaxedSale / 1.13) * 0.13        (iva_item)
  IVA en resumen = total_iva = Sum(IVAItem) - (TaxedDiscount / 1.13) * 0.13
  Los tributos del resumen NO incluyen 20 ni C3
  Monto total = SubTotal + Sum(otros tributos)

CCF:
  No hay IVA por ítem
  IVA en resumen = tributo 20 = (TotalTaxed - TaxedDiscount) * 0.13
  Monto total = SubTotal + Sum(tributos, incluido el 20)
```
