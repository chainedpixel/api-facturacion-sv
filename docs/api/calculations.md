# Cálculos que valida la API

Este documento describe, por tipo de documento, los cálculos que la API recalcula y compara contra los valores enviados. La API **no calcula ni corrige** montos: el cliente debe enviar cada campo ya calculado y redondeado; si algún valor no coincide con la fórmula (dentro de la holgura), la solicitud se rechaza con el código de error indicado.

Fuente de verdad: el código de los validadores (`internal/domain/dte/common/validator/strategy/calculation_rules.go` y las estrategias de cada documento en `internal/domain/dte/*/validator/strategy/`). Los mensajes de error están en `assets/i18n/es.yaml`. Las reglas de redondeo y holgura corresponden a las reglas 7.2 y 7.3 de la Normativa de Cumplimiento DTE v2.0 (Cuadros 1, 2 y 3), y los números de campo citados corresponden al Anexo IV de la misma Normativa.

Las reglas estructurales (campos obligatorios, catálogos, documentos relacionados) se documentan en [Validación de DTE](../internal/domain/validation/README.md) y en la guía de cada documento ([FE](./invoice.md), [CCF](./ccf.md), [NC](./credit-note.md), [ND](./debit-note.md), [NR](./remission-note.md), [FSE](./fse.md), [CR](./retention.md)).

---

## 1. Reglas generales

### 1.1 Decimales y redondeo

| Sección | Decimales | Regla de la Normativa (7.2) |
|---|---|---|
| Cuerpo (por ítem) | Hasta 8 | Si el noveno decimal es 5 o mayor, el octavo sube 1 |
| Resumen | 2 | Si el tercer decimal es 5 o mayor, el segundo sube 1 |

Ejemplos de la Normativa: `1.123456784` a `1.12345678`; `1.123456785` a `1.12345679`; `1.124` a `1.12`; `1.125` a `1.13`.

Comportamiento de la API:

- **Cuerpo**: los montos del ítem (`unit_price`, `non_subject_sale`, `exempt_sale`, `taxed_sale`, `suggested_price`, `non_taxed`, `iva_item`, `purchase`) se redondean internamente a 8 decimales (mitad hacia arriba) al leer la solicitud. No se rechazan valores con más decimales.
- **Resumen**: los montos del resumen deben ser **múltiplos exactos de 0.01**. La API no redondea: si el valor tiene más de dos decimales se rechaza con `InvalidDecimals`.
- **Resumen calculado desde los ítems**: la Normativa (regla 7.3, último párrafo) exige calcular el resumen con los decimales de los campos por ítem (los 8 decimales ya redondeados) y redondear al final a 2 decimales. El orden es: calcular cada ítem y redondearlo a 8 decimales, sumar, y recién entonces redondear el resultado a 2 decimales.
- Excepciones a los 2 decimales del resumen: en NR los montos del resumen admiten hasta 8 decimales (solo `tributes[].value` exige múltiplos de `0.01`), y en FSE `income_retention` e `iva_retention` admiten hasta 8 decimales (el total a pagar sí debe ser múltiplo de `0.01`).
- Rango de los montos: de `0` a `99999999999.99`. Fuera de ese rango se devuelve `InvalidAmount`. Los únicos campos de la Normativa que admiten valores negativos (cargos/abonos que no afectan la base imponible, campos 98 y 151) no los admiten en la API (ver sección 2.5).

### 1.2 Holgura

Todo campo calculado acepta una diferencia de **±0.01** entre el valor enviado y el recalculado (regla 7.3, Cuadro 1). La holgura existe para absorber diferencias de redondeo; no debe usarse para ajustar valores a propósito.

Implementación común: `WithinTolerance(esperado, enviado)` es verdadera cuando `|esperado - enviado| <= 0.01`.

Excepciones donde la comparación es exacta (sin holgura):

| Comparación | Documentos |
|---|---|
| `total_non_subject` contra la suma de `non_subject_sale` de los ítems | CCF, NC, ND |
| `total_exempt` contra la suma de `exempt_sale` de los ítems | CCF, NC, ND |
| `total_non_taxed` contra la suma de `non_taxed` de los ítems | CCF |

Como el resumen se envía con 2 decimales y los ítems pueden tener hasta 8, estas comparaciones exactas solo coinciden cuando la suma de los ítems ya tiene 2 decimales.

### 1.3 Uso de la holgura de ±0.01 en el cuerpo

La venta de cada ítem se compara con `(unit_price x quantity) - discount` con la misma holgura (ver sección 2.1). Cada ítem se calcula con sus propios decimales; no se arrastra el redondeo del resumen.

### 1.4 Orden de las validaciones y errores

- Cada estrategia de validación devuelve **el primer error** que encuentra y no sigue evaluando.
- El validador de cada documento ejecuta todas sus estrategias y agrupa los errores de las distintas estrategias en una respuesta compuesta (`CompositeError`). Una misma causa puede reportarse por dos estrategias distintas (por ejemplo, el subtotal de la FE se verifica en dos estrategias con códigos distintos).
- Primero se ejecuta el validador propio del documento y, si no hay errores, el validador común (formato de receptor, pagos, documentos relacionados, etc.).

---

## 2. Cuerpo del documento (ítems)

### 2.1 Venta por ítem (Anexo IV, campos 90 a 92 y 95)

| Campo de la API | Documentos | Fórmula | Holgura | Error |
|---|---|---|---|---|
| `non_subject_sale` + `exempt_sale` + `taxed_sale` (suma de las tres) | FE, CCF | `(unit_price x quantity) - discount` | ±0.01 | `InvalidItemSaleAmount` |
| `purchase` | FSE | `(unit_price x quantity) - discount` | ±0.01 | `InvalidItemSaleAmount` |
| `non_subject_sale`, `exempt_sale`, `taxed_sale` | NR, NC, ND | No se recalcula contra precio por cantidad; solo se valida el resto de reglas de la sección 2.2 | | |

Reglas de la fórmula:

- Solo se evalúa cuando `unit_price` es distinto de `0`. Los ítems de cargos/abonos que no afectan la base imponible usan `unit_price = 0` y no aplican esta fórmula.
- `discount` es un **monto** (no un porcentaje), mayor o igual a `0` (campo 89 de la Normativa: "no podrá ser mayor al monto del ítem"). En la API el campo `discount` por ítem acepta valores de `0` a `100`; fuera de ese rango se rechaza con `InvalidDiscount` (aplica a FE, CCF, NC, ND, NR y FSE).
- Un ítem va en **una sola** categoría de venta (no sujeta, exenta o gravada); las otras van en `0`.
- `suggested_price` (precio sugerido de venta) es informativo y no interviene en ningún cálculo.
- `quantity` debe ser mayor que 0 (se evalúa redondeada a 2 decimales) y no mayor a `99999999999.99`.
- Precio unitario según el documento (campo 87): en **FE** se envía con IVA incluido (precio consumidor final); en CCF, NR, NC, ND y FSE, sin IVA.

### 2.2 Reglas por ítem relacionadas con los cálculos

| Regla | FE | CCF | NC | ND | NR | FSE | Error |
|---|---|---|---|---|---|---|---|
| Más de una categoría de venta en el ítem | Rechaza | Rechaza (con mensajes por categoría) | Rechaza | Rechaza | No evalúa | No aplica | `MixedSalesTypesNotAllowed`, `InvalidMixedSalesWithNonTaxed`, `InvalidMixedSalesWithExempt`, `InvalidMixedSalesWithNonSubject` |
| `taxed_sale > 0` con `unit_price = 0` | Rechaza | Rechaza | Rechaza | Rechaza | No evalúa | No aplica | `MissingItemUnitPrice` (FE), `InvalidUnitPriceZero` (FE, CCF, NC, ND) |
| Suma de ventas del ítem mayor que `unit_price x quantity` | Rechaza | No evalúa (lo cubre la fórmula 2.1) | No evalúa | No evalúa | No evalúa | No aplica | `ExcessiveItemTotal` |
| Venta gravada sin impuestos en el ítem | No aplica | Rechaza | Rechaza | Rechaza | No evalúa | No aplica | `MissingTaxesItem` (CCF), `MissingItemTaxes` (NC, ND) |
| Venta exenta, no sujeta o `non_taxed` con impuestos en el ítem | No aplica | Rechaza | No evalúa | No evalúa | No evalúa | No aplica | `InvalidTaxesWithExempt`, `InvalidTaxesWithNonSubject`, `InvalidTaxesWithNonTaxed` |
| Descuento del ítem mayor que `unit_price x quantity` | No evalúa | No evalúa | No evalúa | No evalúa | No evalúa | Rechaza | `ExcessiveItemDiscount` |
| Venta del ítem en `0` (todas las categorías) | No evalúa | No evalúa | No evalúa | No evalúa | Rechaza | No aplica | `AtLeastOneSaleTypeRequired` |

### 2.3 IVA por ítem en la FE (campo 102)

Solo la **FE** envía `iva_item`:

```
iva_item = (taxed_sale / 1.13) x 0.13        holgura ±0.01
```

- Es obligatorio cuando `taxed_sale > 0` (campo `iva_item` requerido; si falta la solicitud se rechaza con `RequiredField`).
- Si `iva_item > 0` y `taxed_sale = 0`: `InvalidIVAItemWithoutTaxedSale`.
- Si no coincide con la fórmula: `InvalidIVAItemCalculation`.
- Según la Normativa es informativo: no se suma al total de la operación; el IVA ya va incluido en el precio.

Los documentos CCF, NR, NC, ND y FSE no envían `iva_item`. En CCF, NC y ND el IVA aparece solo como tributo `20` en el resumen.

### 2.4 Tributos por ítem (campo `taxes`)

| Documento | Regla aplicada por la API | Error |
|---|---|---|
| FE | No admite los códigos `20` ni `C3` en el ítem (el IVA va incluido en el precio y se informa en `iva_item`) | `TaxNotAllowedInInvoice` |
| CCF | Con venta gravada, `taxes` es obligatorio. Los ítems tipo 1 (producto) solo admiten `20`. Los ítems tipo 4 (impuesto) requieren `unit_measure = 99` y `taxes = ["20"]` | `MissingTaxesItem`, `InvalidTaxCodeOnly20`, `InvalidUnitMeasure`, `InvalidTaxRulesCCF` |
| NC, ND | Con venta gravada, `taxes` es obligatorio. Los ítems que no son tipo 4 solo admiten códigos del catálogo permitido. Los ítems tipo 4 requieren `unit_measure = 99` y `taxes = ["20"]` | `MissingItemTaxes`, `InvalidTaxType`, `InvalidUnitMeasure`, `InvalidTaxRulesCCF` |
| NR, FSE | No se aplican reglas propias de tributos por ítem | |

Todos los documentos validan que cada código del ítem pertenezca al conjunto soportado: `20`, `C3`, `59`, `71`, `D1`, `C8`, `D5` (`UnsupportedTaxCode`).

### 2.5 Cargos y abonos que no afectan la base imponible (`non_taxed`, campo 98)

- Aplica a FE y CCF (el campo `non_taxed` existe en los ítems de esos dos documentos).
- El ítem que lleva `non_taxed` debe tener `unit_price = 0` y no puede tener otra categoría de venta (`InvalidMixedSalesWithNonTaxed`, `InvalidUnitPriceForNonTaxed`, `InvalidUnitPriceWithNonTaxed`). En CCF tampoco puede tener impuestos.
- Resumen: `total_non_taxed = Σ non_taxed` de los ítems. Si el resumen tiene valor y los ítems no (o al revés): `InvalidNonTaxedAmount`. Si ambos tienen valor pero difieren: `InvalidNonTaxedAmountCalculation` (FE, holgura ±0.01) o `InvalidTotalNonTaxed` (CCF, comparación exacta).
- La API **solo admite valores mayores o iguales a 0**: el cargo suma al total a pagar; no es posible enviar un abono (valor negativo) porque el valor se rechaza con `InvalidAmount`.
- En NC y ND el resumen admite `total_non_taxed` (entra en el total a pagar) pero sus ítems no tienen el campo `non_taxed`, por lo que la API no lo contrasta contra ningún ítem.

---

## 3. Resumen

### 3.1 Campos del resumen por documento

| Campo de la API | FE | CCF | NC | ND | NR | FSE |
|---|---|---|---|---|---|---|
| `total_non_subject`, `total_exempt`, `total_taxed` | si | si | si | si | si (`non_subject_total`, `exempt_total`, `taxed_total`) | no |
| `sub_total_sales` | si | si | si | si | si (`subtotal_sales`) | no |
| `total_purchase` | no | no | no | no | no | si |
| `non_subject_discount`, `exempt_discount`, `taxed_discount`, `discount_percentage` | si | si | no admite (deben ser 0) | no admite (deben ser 0) | si | solo `non_subject_discount` (descuento global, campo 133) |
| `total_discount` | si | si | si | si | si | si |
| `taxes` (tributos del resumen) | si | si | si | si | si (`tributes`) | no |
| `sub_total` | si | si | si | si | si (`subtotal`) | si |
| `iva_perception` | no | si | si | si | no | no |
| `total_iva` | si | no | no | no | no | no |
| `iva_retention` | si | si | si | si | no | debe ser 0 |
| `income_retention` | debe ser 0 | debe ser 0 | debe ser 0 | debe ser 0 | no | si |
| `total_operation` | si | si | si | si | si (`total_amount`) | no (se asigna `total_purchase`) |
| `total_non_taxed` | si | si | si | si | no | no |
| `total_to_pay` | si | si | si | si | no (se asigna `total_amount`) | si |
| `payment_types` | si | si | no se contrasta con el total (NC solo valida los decimales) | no se valida | no | si |

Todos los campos de resumen `sub_total`, `sub_total_sales`, `total_operation` y `total_to_pay` son obligatorios y distintos de `0` en FE, CCF, NC, ND y FSE (`RequiredField`). En FSE, `total_purchase` también se asigna a `total_operation` y `sub_total_sales`, por lo que no puede ser `0`.

### 3.2 Totales de operaciones (campos 123 a 125 y 127)

| Campo | Fórmula | Holgura | Error |
|---|---|---|---|
| `total_taxed` | `Σ taxed_sale` de los ítems | ±0.01 | `InvalidTotalTaxed` |
| `total_exempt` | `Σ exempt_sale` de los ítems | ±0.01 en FE y NR; exacta en CCF, NC, ND | `InvalidTotalExempt` |
| `total_non_subject` | `Σ non_subject_sale` de los ítems | ±0.01 en FE y NR; exacta en CCF, NC, ND | `InvalidTotalNonSubject` |
| `total_purchase` (FSE) | `Σ purchase` de los ítems | ±0.01 | `InvalidTotalPurchase` |

En NR los totales solo se verifican si el campo fue enviado.

### 3.3 Suma de operaciones (campo 129)

```
sub_total_sales = total_non_subject + total_exempt + total_taxed        holgura ±0.01
```

Aplica a FE, CCF, NC y ND. Error: `InvalidSubTotalSales`. En NR el campo `subtotal_sales` es obligatorio pero la API no lo contrasta contra los totales.

### 3.4 Descuentos globales (campos 130 a 135)

| Campo | Regla | Documentos | Error |
|---|---|---|---|
| `non_subject_discount` | `0 <= valor <= total_non_subject` | FE, CCF | `DiscountExceedsBase` (FE: también `DiscountExceedsSubtotal`) |
| `exempt_discount` | `0 <= valor <= total_exempt` | FE, CCF | `DiscountExceedsBase` |
| `taxed_discount` | `0 <= valor <= total_taxed` | FE, CCF | `DiscountExceedsBase` |
| `non_subject_discount` (FSE: descuento global al total, campo 133) | Se resta de `total_purchase` en el subtotal (sección 3.5) | FSE | `InvalidSubTotal` |
| `discount_percentage` | Entre 0 y 100, sin símbolo de porcentaje; la API no lo recalcula | FE, CCF, NR | `InvalidDiscount` |
| Descuentos negativos | Rechazados | FE, ND | `NegativeDiscount` |
| Descuento global o `discount_percentage` distinto de 0 | **Rechazado**: las notas de crédito y débito no admiten descuentos globales (el descuento se aplica por ítem) | NC, ND | `GlobalDiscountNotAllowedInAdjustment` (parámetro: nombre del campo) |
| `total_discount` | Descuentos por ítem + descuentos globales: `Σ discount + non_subject_discount + exempt_discount + taxed_discount` | ND: `Σ discount` (los globales son 0); FSE: `Σ discount + non_subject_discount` (campo 135.5) | ND: `InvalidTotalDiscountCalculation`; FSE: `InvalidTotalDiscount` |

Notas:

- `total_discount` es informativo (campo 135.1 de la Normativa: "no tendrá efecto en los valores totales del documento"). Los descuentos por ítem ya están descontados dentro de la venta del ítem, por lo que no se restan otra vez del subtotal.
- La API valida la igualdad de `total_discount` únicamente en ND y FSE. En FE, CCF, NC y NR solo se valida que no sea negativo (FE).
- Los descuentos de NC/ND se rechazan antes que cualquier otro cálculo del resumen.

### 3.5 Subtotal (campo 139)

| Documento | Fórmula | Holgura | Error |
|---|---|---|---|
| FE, CCF, ND | `total_taxed - taxed_discount + total_exempt - exempt_discount + total_non_subject - non_subject_discount` | ±0.01 | FE: `InvalidSubTotalCalculation` e `InvalidSubTotal`; CCF y ND: `InvalidSubTotalCalculation` |
| NC | `sub_total_sales - taxed_discount - exempt_discount - non_subject_discount` (los descuentos son 0, por lo que `sub_total = sub_total_sales`) | ±0.01 | `InvalidSubTotalCalculation` |
| FSE | `total_purchase - non_subject_discount` | ±0.01 | `InvalidSubTotal` |
| NR | No se recalcula; solo interviene en el monto total de la operación (sección 3.9) | | |

En FE los dos códigos (`InvalidSubTotalCalculation` e `InvalidSubTotal`) provienen de dos estrategias que evalúan la misma fórmula; ambos aparecen en la respuesta compuesta.

### 3.6 Tributos del resumen (campos 136 a 138)

Reglas de estructura:

- `taxes[].value` debe ser múltiplo de `0.01`; solo el código `C3` admite valor `0` (`RequiredField` en otro caso).
- Con `total_taxed = 0` el arreglo `taxes` debe estar vacío (`InvalidTaxes`).
- CCF, NC y ND: con `total_taxed > 0`, `taxes` es obligatorio (`MissingTaxes`) y debe incluir el tributo `20` con valor mayor que 0 (`MissingIVAForTaxedAmount`).
- FE: `taxes` es opcional. No admite los tributos `20` ni `C3` (`TaxNotAllowedInInvoiceSummary`): el IVA se informa en `total_iva`.
- NR: los tributos se informan en `tributes`; solo se suman al monto total (sección 3.9). La API no recalcula su valor contra la tasa.

Fórmula del valor de cada tributo (holgura ±0.01; error `InvalidTaxCalculation`, con parámetros: código, esperado, enviado):

| Código | Tributo | Base y tasa que aplica la API |
|---|---|---|
| `20` | IVA | CCF: `(total_taxed - taxed_discount) x 0.13`. NC y ND: `total_taxed x 0.13` (no admiten descuento global) |
| `C3` | IVA exportación | `total_taxed x 0` (valor `0`) |
| `59` | Turismo | `total_taxed x 0.05` |
| `71` | Turismo aeropuerto | Valor fijo `7.00` |
| `D1` | FOVIAL | `total_taxed x 0.20` |
| `C8` | COTRANS | Valor fijo `0.10` |
| `D5` | Especial u otro | Sin validación del valor |

Notas:

- La base de los tributos distintos de `20` es `total_taxed` **sin** restar `taxed_discount`, también en CCF.
- En FE el tributo se compara contra `total_taxed` (que incluye IVA, porque el precio de la FE lo incluye).
- En ND el desvío del valor del tributo contra la fórmula solo se registra en el log y **no se rechaza** (la validación del código de tributo y de su presencia sí se aplica).
- En NR no se aplica ninguna de estas fórmulas.

### 3.7 IVA de la FE en el resumen (campo 157, `total_iva`)

```
total_iva = Σ iva_item - (taxed_discount / 1.13) x 0.13        holgura ±0.01
```

- Con `total_taxed > 0`, `total_iva` debe ser mayor que 0 (`MissingIVAForTaxedAmount`); si no coincide con la fórmula: `InvalidTotalIVA`.
- Es informativo: no se suma al monto total de la operación.

### 3.8 IVA percibido, IVA retenido y retención de renta

| Campo | Documentos | Regla | Error |
|---|---|---|---|
| `iva_perception` | CCF, NC, ND | Si es distinto de `0`: `total_taxed x 0.01`, holgura ±0.01. Con `0` no se valida | `InvalidPerceptionAmount` |
| `iva_retention` | FE, CCF, NC, ND | La API **no valida el monto ni la tasa**; el valor enviado solo interviene en el total a pagar. En FE debe ser `0` si `total_taxed = 0` | FE: `InvalidIVARetentionWithoutTaxedAmount` |
| `income_retention` | FE, CCF, NC, ND | **Rechazado** si es distinto de `0` (eliminado por la Normativa v2.0, Anexo VI) | `IncomeRetentionNotAllowed` |
| `iva_retention` (FSE) | FSE | **Rechazado** si es distinto de `0` (eliminado por la Normativa v2.0, Anexo VI) | `IVARetentionNotAllowedInFSE` |
| `income_retention` (FSE) | FSE | Mayor o igual a 0; la API **no valida la tasa**, solo que el total a pagar la reste (sección 3.10) | `FSETaxInvalidIncomeRetention` si es negativa |

- La base de la percepción es `total_taxed` (sin restar `taxed_discount`); la API no valida que se calcule sobre otra base.
- Los montos de percepción y retención deben ser múltiplos de `0.01` (`InvalidDecimals` en el mapeo; `InvalidMonetaryAmount` en la validación de FE, CCF y NC).

### 3.9 Monto total de la operación (campo 150)

| Documento | Fórmula | Holgura | Error |
|---|---|---|---|
| FE, CCF, NR | `sub_total + Σ taxes[].value` | ±0.01 | `InvalidTotalOperation` |
| NC, ND | `sub_total_sales + Σ taxes[].value` | ±0.01 | `InvalidTotalOperation` |

Notas:

- En NC y ND (Normativa v2.0) el monto total parte de `sub_total_sales`, no del `sub_total`.
- En FE sin tributos adicionales, `total_operation = sub_total`.
- En NR se evalúa solo si `subtotal` y `total_amount` están presentes (son obligatorios en la solicitud).
- Si el documento relacionado es un CR (07), la Normativa indica que el monto total es el `iva_retention` de la nota; la API no implementa esa variante: aplica siempre la fórmula de la tabla.

### 3.10 Total a pagar (campo 153)

| Documento | Fórmula | Holgura | Error |
|---|---|---|---|
| FE | `total_operation - iva_retention + total_non_taxed` | ±0.01 | `InvalidTotalToPayCalculation` |
| CCF | `total_operation + iva_perception - iva_retention + total_non_taxed` | ±0.01 | `InvalidTotalToPayCalculation` |
| NC, ND | `total_operation + iva_perception - iva_retention + total_non_taxed` | ±0.01 | `InvalidTotalToPayCalculation` |
| FSE | `sub_total - income_retention` | ±0.01 | `FSETaxInvalidTotalToPay` |
| NR | No existe: `total_to_pay` se asigna con `total_amount` | | |

Notas:

- En FE y CCF, si el resultado de la fórmula es negativo se toma `0.00` como valor esperado. Sin embargo, el campo `total_to_pay` es obligatorio y distinto de `0` en la solicitud, por lo que un total a pagar de `0.00` se rechaza en el mapeo.
- Para la ND, después de validar, el servicio recalcula `total_to_pay` y reemplaza el valor enviado. El valor guardado es: `sub_total + (si total_taxed > 0: iva_perception - iva_retention - income_retention) + Σ taxes + total_non_taxed`. Dado que `income_retention` es 0, coincide con la fórmula de validación salvo cuando `total_taxed = 0`, caso en que la percepción y la retención no se incluyen en el valor guardado.
- `total_in_words`, si no se envía, se genera a partir de `total_to_pay` (FE, CCF, FSE), de `total_operation` (NC, ND) o de `total_amount` (NR).
- `balance_in_favor` (saldo a favor, campo 158) se mapea pero la API no lo valida contra ningún cálculo y no admite valores negativos (`InvalidAmount`); la Normativa permite negativos en ese campo.

### 3.11 Pagos

Aplica a FE, CCF y FSE (no a NC, ND ni NR).

| Regla | Error |
|---|---|
| `Σ payment_types[].amount` coincide con `total_to_pay`, holgura ±0.01 | `InvalidPaymentTotal` |
| Condición de operación `2` (crédito) sin pago en efectivo (`01`): cada pago debe llevar `term` y `period`, y el código de pago no puede ser `01` | `InvalidPaymentTerms`, `InvalidPaymentTypeOP2` |
| Condición de operación `1` (contado) sin pago en efectivo: los pagos no deben llevar `term` ni `period` | `InvalidPaymentTermsOF` |
| `payment_types[].amount` debe ser múltiplo de `0.01` (FE, CCF, NC) | `InvalidMonetaryAmount` |
| Cada pago requiere `code` y `amount` distinto de `0` | `InvalidPaymentTypeInfo` |

### 3.12 Saldo del documento relacionado en NC y ND

La Normativa (Anexo IV, campos 90 a 92) indica que la suma de las ventas de los ítems de las notas relacionadas con un mismo CCF no puede ser mayor que el total de operaciones menos el descuento global de ese documento. La API lo controla con un saldo por documento:

- El saldo inicial de un CCF o CR emitido es su `total_taxed`, `total_exempt` y `total_non_subject`.
- Al emitir una **NC** cuyo documento relacionado es electrónico (`generation_type = 2`), cada total de la NC (`total_taxed`, `total_exempt`, `total_non_subject`) no puede superar el saldo restante correspondiente; si lo supera se rechaza con `InvalidCreditNoteTransaction`. La NC descuenta del saldo; si la NC se invalida, el saldo se restituye.
- Para la **ND** solo se verifica que el documento relacionado tenga control de saldo; la ND suma al saldo del documento.
- El saldo no resta el descuento global del documento original (`taxed_discount`, `exempt_discount`, `non_subject_discount`).

---

## 4. Comprobante de Retención (CR)

| Campo | Fórmula | Holgura | Error |
|---|---|---|---|
| `items[].iva_amount` (IVA retenido por ítem, campo 105) | `taxed_amount x tasa` según `retention_code`: `22` = `0.01`, `C4` = `0.13` | ±0.01 | `InvalidRetentionIVA` |
| `summary.total_retention_amount` (total monto sujeto a retención) | `Σ items[].taxed_amount` | ±0.01 | `InvalidTotalSubjectRetention` |
| `summary.total_retention_iva` (total IVA retenido) | `Σ items[].iva_amount` | ±0.01 | `InvalidTotalIVARetention` |
| `summary.total_iva` (campo 157, total IVA 13%) | Con código `C4` y todos los documentos electrónicos (`type = 2`): `total_retention_amount x 0.13`. En los demás casos: `0.00` | ±0.01 | `InvalidTotalIVA` |

Notas:

- Todos los ítems deben tener el mismo `retention_code` (`MixedRetentionCodes`); el código debe ser `22`, `C4` o `C9` (`InvalidRetentionCode`).
- `summary.total_iva` es opcional: si se omite (o se envía `0`), la API lo calcula con la regla anterior. Si se envía, debe coincidir con ella.
- `total_retention_amount` y `total_retention_iva` son obligatorios y distintos de `0`; deben ser múltiplos de `0.01` (`InvalidDecimals`). Los montos por ítem admiten hasta 8 decimales.
- Para el código `C9` la API no tiene tasa definida, por lo que espera un IVA retenido por ítem igual a `0`.
- Los documentos retenidos se validan por fecha (ver [validación de retención](../internal/domain/validation/retention-validation.md)).

---

## 5. Ejemplos numéricos

Los ejemplos están verificados contra las fórmulas de la API (sección 3). Las ventas por ítem se calculan con 8 decimales y el resumen se redondea a 2.

### 5.1 FE con precio con IVA incluido

| Ítem | `unit_price` | `quantity` | `discount` | `taxed_sale` | `iva_item` |
|---|---|---|---|---|---|
| 1 | 25.00 | 2 | 0.00 | 50.00 | 5.75221239 |
| 2 | 30.00 | 1 | 3.00 | 27.00 | 3.10619469 |

Resumen: `total_taxed` 77.00; `sub_total_sales` 77.00; `taxed_discount` 0.00; `total_discount` 3.00; `sub_total` 77.00; `taxes` vacío; `iva_retention` 0.00; `total_operation` 77.00; `total_non_taxed` 0.00; `total_to_pay` 77.00; `total_iva` 8.86 (`5.75221239 + 3.10619469 = 8.85840708`, redondeado a 2 decimales).

Con un descuento global a ventas gravadas de 10.00: `taxed_discount` 10.00; `sub_total` 67.00; `total_iva` = 8.85840708 - (10.00 / 1.13 x 0.13 = 1.15044248) = 7.70796460, es decir `7.71`; `total_operation` 67.00.

Con el tributo `59` (turismo): valor `77.00 x 0.05 = 3.85` (base `total_taxed`); `total_operation` = 77.00 + 3.85 = 80.85.

### 5.2 CCF con precio sin IVA

| Ítem | `unit_price` | `quantity` | `discount` | `taxed_sale` |
|---|---|---|---|---|
| 1 | 20.00 | 3 | 0.00 | 60.00 |
| 2 | 40.00 | 1 | 5.00 | 35.00 |

Sin descuento global: `total_taxed` 95.00; `sub_total_sales` 95.00; `total_discount` 5.00; `sub_total` 95.00; `taxes` = [`20`: 12.35]; `total_operation` 107.35; `iva_perception` 0.00; `iva_retention` 0.00; `total_to_pay` 107.35.

Con un descuento global a ventas gravadas de 5.00: `taxed_discount` 5.00; `sub_total` 90.00; tributo `20` = `(95.00 - 5.00) x 0.13 = 11.70`; `total_operation` 101.70.

Con percepción de IVA (base `total_taxed`): `iva_perception` = `95.00 x 0.01 = 0.95`; `total_to_pay` = 107.35 + 0.95 = 108.30.

### 5.3 NC sobre el CCF anterior (devolución de 1 unidad del ítem 1)

Ítem: `taxed_sale` 20.00, `taxes` = [`20`], `related_doc` = código de generación del CCF.

Resumen: `total_taxed` 20.00; `sub_total_sales` 20.00; `sub_total` 20.00; descuentos en 0.00; `taxes` = [`20`: 2.60]; `total_operation` = 20.00 + 2.60 = 22.60; `iva_perception` 0.00; `total_to_pay` 22.60.

Si el CCF original tuvo percepción: `iva_perception` = `20.00 x 0.01 = 0.20` y `total_to_pay` = 22.60 + 0.20 = 22.80.

### 5.4 FSE

Un ítem de 4 unidades a 10.00 con descuento de 2.00: `purchase` = `4 x 10.00 - 2.00 = 38.00`.

Resumen: `total_purchase` 38.00; `non_subject_discount` 0.00; `total_discount` 2.00; `sub_total` 38.00; `income_retention` 3.80 (por ejemplo, el 10 % del subtotal; la API no valida la tasa); `total_to_pay` = 38.00 - 3.80 = 34.20.

### 5.5 NR

Ítem: `unit_price` 18.50, `quantity` 10, `discount` 5.00; `taxed_sale` = `10 x 18.50 - 5.00 = 180.00`.

Resumen: `taxed_total` 180.00; `subtotal` 180.00; `tributes` = [`20`: 23.40]; `total_amount` = 180.00 + 23.40 = 203.40.

### 5.6 CR con documentos físicos y código `C4`

Dos documentos con `taxed_amount` 100.00 y 50.00 (código `C4`): `iva_amount` 13.00 y 6.50; `total_retention_amount` 150.00; `total_retention_iva` 19.50; `total_iva` 0.00 (documentos físicos).

Con el código `22`: `iva_amount` 1.00 y 0.50; `total_retention_iva` 1.50; `total_iva` 0.00.

---

## 6. Códigos de error de cálculo

Todos los mensajes están en `assets/i18n/es.yaml`. Los parámetros se listan en el orden en que los recibe el mensaje.

| Código | Cuándo se devuelve |
|---|---|
| `InvalidDecimals` | Un campo del resumen no es múltiplo de `0.01` |
| `InvalidMonetaryAmount` | Un monto validado (`total_operation`, `total_to_pay`, `iva_retention`, `iva_perception`, pagos) no es múltiplo de `0.01` |
| `InvalidAmount` | Monto fuera del rango `0` a `99999999999.99` |
| `InvalidDiscount` | `discount` por ítem o `discount_percentage` fuera del rango `0` a `100` |
| `InvalidItemSaleAmount` | La venta del ítem no coincide con `(unit_price x quantity) - discount` |
| `InvalidIVAItemCalculation` | `iva_item` de FE incorrecto |
| `InvalidTotalTaxed`, `InvalidTotalExempt`, `InvalidTotalNonSubject` | El total del resumen no coincide con la suma de los ítems |
| `InvalidSubTotalSales` | `sub_total_sales` incorrecto |
| `InvalidSubTotalCalculation`, `InvalidSubTotal` | `sub_total` incorrecto |
| `InvalidTaxCalculation` | Valor de un tributo del resumen incorrecto |
| `InvalidTotalIVA` | `total_iva` de FE (o `total_iva` de CR) incorrecto |
| `InvalidPerceptionAmount` | `iva_perception` distinto de 1 % de `total_taxed` |
| `InvalidTotalOperation` | `total_operation` incorrecto |
| `InvalidTotalToPayCalculation` | `total_to_pay` incorrecto (FE, CCF, NC, ND) |
| `FSETaxInvalidTotalToPay` | `total_to_pay` incorrecto (FSE) |
| `InvalidPaymentTotal` | La suma de pagos no coincide con `total_to_pay` |
| `InvalidTotalDiscountCalculation`, `InvalidTotalDiscount` | `total_discount` incorrecto (ND y FSE) |
| `DiscountExceedsBase`, `DiscountExceedsSubtotal`, `ExcessiveItemDiscount`, `NegativeDiscount` | Descuento mayor que su base o negativo |
| `GlobalDiscountNotAllowedInAdjustment` | Descuento global o porcentaje de descuento distinto de 0 en NC o ND |
| `IncomeRetentionNotAllowed` | `income_retention` distinto de 0 en FE, CCF, NC o ND |
| `IVARetentionNotAllowedInFSE` | `iva_retention` distinto de 0 en FSE |
| `InvalidIVARetentionWithoutTaxedAmount` | `iva_retention` mayor que 0 en FE sin venta gravada |
| `InvalidNonTaxedAmount`, `InvalidNonTaxedAmountCalculation`, `InvalidTotalNonTaxed` | `total_non_taxed` no coincide con los ítems |
| `MissingIVAForTaxedAmount`, `MissingTaxes`, `InvalidTaxes` | Presencia del tributo `20` o de `taxes` en el resumen |
| `TaxNotAllowedInInvoice`, `TaxNotAllowedInInvoiceSummary` | Tributo `20` o `C3` en una FE |
| `InvalidRetentionIVA`, `InvalidTotalSubjectRetention`, `InvalidTotalIVARetention` | Cálculos del CR |
| `InvalidCreditNoteTransaction` | Los totales de la NC superan el saldo restante del documento relacionado |
