# Validación de CCF Electrónico (DTE 03)

> **Validador:** `CCFRulesValidator`
> **Paquete:** `internal/domain/dte/ccf/validator`

## Estrategias Compuestas

1. **CCFItemStrategy** — Validación de ítems, tipos de venta y reglas de impuestos
2. **CCFTaxStrategy** — Validación de cálculos fiscales, IVA y percepción
3. **CCFReceiverStrategy** — Validación del receptor (NRC obligatorio)
4. **CCFRelatedDocStrategy** — Validación de documentos relacionados

---

## 1. CCFItemStrategy

**Archivo:** `internal/domain/dte/ccf/validator/ccf_item_strategy.go`

### Exclusividad de Tipos de Venta

| Tipo | Restricción con otros tipos | Restricción de impuestos |
|---|---|---|
| `NonTaxed` | No mezclar con otros | No puede tener impuestos |
| `Exempt` | No mezclar con otros | No puede tener impuestos |
| `NonSubject` | No mezclar con otros | No puede tener impuestos |
| `Taxed` | Puede estar solo | Debe tener impuestos |

### Reglas de Impuestos por Tipo de Ítem

| Tipo de Ítem | Código | Restricción |
|---|---|---|
| Producto | 1 | Solo IVA (código `20`) |
| Servicio | 2 | Cualquier impuesto permitido |
| Producto y Servicio | 3 | Cualquier impuesto permitido |
| Impuesto | 4 | Solo IVA (código `20`), unidad de medida = `99` |

### Validación de Precio Unitario

- Si `TaxedSale > 0` → `UnitPrice` no puede ser 0
- Si `NonTaxed > 0` y `UnitPrice == 0` → `UnitPrice` debe ser igual a `TaxedSale`

### Documentos Relacionados en Ítems

- Si existen docs relacionados → todos los ítems deben referenciar uno
- Cada referencia debe existir en la lista de documentos

### Total a Pagar con No Gravado

```
TotalToPay = TotalOperation + TotalNonTaxed + IVAPerception - IVARetention - IncomeRetention
```
Tolerancia: ±0.01

---

## 2. CCFTaxStrategy

**Archivo:** `internal/domain/dte/ccf/validator/ccf_tax_strategy.go`

### Validación de Totales Base

```
Sum(item.TaxedSale)      == Summary.TotalTaxed       (±0.01)
Sum(item.ExemptSale)     == Summary.TotalExempt       (±0.01)
Sum(item.NonSubjectSale) == Summary.TotalNonSubject   (±0.01)
```

### Descuentos

```
TaxedDiscount    <= TotalTaxed
ExemptDiscount   <= TotalExempt
NonSubjectDiscount <= TotalNonSubject
```

### SubTotal Sales

```
SubTotalSales = TotalTaxed + TotalExempt + TotalNonSubject    (±0.01)
```

### Cálculo de IVA (diferente a Factura)

El IVA del CCF se calcula **después del descuento**:

```
IVA = (TotalTaxed - TaxedDiscount) * 0.13    (±0.01)
```

> **Diferencia clave con Factura:** En factura el IVA se calcula sobre `TotalTaxed`, en CCF sobre `TotalTaxed - TaxedDiscount`.

### Otros Impuestos

| Código | Fórmula |
|---|---|
| `C3` (IVA Export) | `TotalTaxed * 0.00` |
| `59` (Tourism) | `TotalTaxed * 0.01` |
| `71` (Tourism Airport) | Monto fijo |
| `D1` (FOVIAL) | `TotalTaxed * 0.005` |
| `C8` (COTRANS) | Monto fijo |

### Percepción de IVA

```
Si IVAPerception > 0:
    IVAPerception = TotalTaxed * 0.01    (±0.01)
```

### Validación de Montos Monetarios

Máximo 2 decimales para todos los montos.

### SubTotal

```
SubTotal = TotalTaxed - TaxedDiscount
         + TotalExempt - ExemptDiscount
         + TotalNonSubject - NonSubjectDiscount
```
Tolerancia: ±0.01

### IVA Obligatorio

```
Si TotalTaxed > 0 → debe existir IVA con valor > 0 en impuestos
```

### Total a Pagar

```
TotalToPay = SubTotal + IVAPerception - IVARetention - IncomeRetention + TotalNonTaxed
```
Tolerancia: ±0.01

### Consistencia de No Gravado

```
Si Summary.TotalNonTaxed > 0:
    Sum(item.NonTaxed) debe ser > 0
    Sum(item.NonTaxed) == Summary.TotalNonTaxed    (±0.01)
```

---

## 3. CCFReceiverStrategy

**Archivo:** `internal/domain/dte/ccf/validator/ccf_receiver_strategy.go`

| Campo | Requerido | Descripción |
|---|---|---|
| Receptor | Sí | Debe existir |
| NRC | **Sí** | Número de Registro de Contribuyente |
| ActivityCode | **Sí** | Código de actividad económica |
| ActivityDescription | **Sí** | Descripción de la actividad |

---

## 4. CCFRelatedDocStrategy

**Archivo:** `internal/domain/dte/ccf/validator/ccf_related_doc_strategy.go`

| Validación | Regla |
|---|---|
| Máximo docs | 50 |
| Tipos válidos | Solo `04` (Remisión), `08` (Liquidación), `09` (Doc. Contable Liquidación) |
| Ítems | Todos deben referenciar un doc relacionado |
| Referencias | Deben existir en la lista del documento |

---

## Errores Específicos de CCF

| Error | Causa |
|---|---|
| `MissingNRC` | Receptor sin NRC |
| `InvalidTaxCodeOnly20` | Producto con impuesto diferente a IVA |
| `InvalidMixedSalesWithNonTaxed` | No gravado mezclado con otros |
| `InvalidTaxesWithNonTaxed` | Impuestos en ítem no gravado |
| `InvalidUnitPriceWithNonTaxed` | Precio unitario incorrecto con no gravado |
| `InvalidMixedSalesWithExempt` | Exento mezclado con otros |
| `InvalidTaxesWithExempt` | Impuestos en ítem exento |
| `InvalidMixedSalesWithNonSubject` | No sujeto mezclado con otros |
| `InvalidTaxesWithNonSubject` | Impuestos en ítem no sujeto |
| `MissingTaxesItem` | Gravado sin impuestos |
| `InvalidPerceptionAmount` | Percepción no coincide con 1% |
| `MissingIVAForTaxedAmount` | Gravado > 0 sin IVA |
| `InvalidTotalToPayNonTaxedCCF` | Total con no gravado incorrecto |

---

## Diagrama: CCF vs Factura (Validación de IVA)

```
FACTURA:
  IVA por ítem = (TaxedSale / 1.13) * 0.13
  IVA en resumen = baseTaxed * 0.13
  (baseTaxed = TotalTaxed, SIN descontar)

CCF:
  No hay IVA por ítem
  IVA en resumen = (TotalTaxed - TaxedDiscount) * 0.13
  (baseTaxed = TotalTaxed - TaxedDiscount, CON descuento)
```
