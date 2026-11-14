# Comprobante de Crédito Fiscal (CCF)

## Descripción

El **Comprobante de Crédito Fiscal** (código DTE `03`) se utiliza para documentar ventas de bienes o servicios entre **contribuyentes registrados** (emisor y receptor con NRC). A diferencia de la factura electrónica, el CCF detalla el IVA por separado como tributo en el resumen (`taxes`; Normativa de Cumplimiento DTE v2.0, Anexo IV, campos 136 a 138, y precios sin IVA en el campo 87, regla 2) y calcula el IVA sobre la base gravada **después de aplicar el descuento global**.

**Soporta contingencia:** Sí (ver [Contingencia](#contingencia))

---

## Endpoint

```
POST /api/v1/dte/ccf
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

> Los datos que el servicio genera por sí mismo **no se envían**: número de control, código de generación, fecha y hora de emisión, ambiente, versión y datos del emisor. La fecha de emisión es siempre la del momento de la petición. Los campos desconocidos del request (por ejemplo `extension`, que el CCF no admite) se ignoran sin generar error.

> **Normativa:** los campos de la sección Identificación (versión, ambiente, tipo de documento, código de generación, número de control, modelo de facturación, tipo de transmisión, fecha, hora y moneda) corresponden a los campos 1 a 12 del Anexo IV de la Normativa de Cumplimiento DTE v2.0; la versión de la estructura del CCF es 4 (campo 1). El código de generación y el número de control se rigen además por las reglas 7.1.1 y 7.1.2, y los datos del emisor corresponden a los campos 18 a 33 del mismo Anexo.

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `CreditItem[]` | Sí | Lista de ítems (1–2000) |
| `receiver` | `Receiver` | Sí | Datos del receptor. **Nombre, dirección, NIT, NRC y actividad económica son obligatorios** |
| `summary` | `CreditSummary` | Sí | Resumen financiero |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `payments` | `Payment[]` | No | Formas de pago. Si se envía, **reemplaza** a `summary.payment_types` |
| `other_docs` | `OtherDocument[]` | No | Otros documentos asociados (máx. 10) |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados (máx. 50) |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |
| `model_type` | integer | No | Se acepta pero **no tiene efecto**: la emisión normal siempre es modelo previo (`1`) y la contingencia la determina el servicio |

> **Normativa:** la emisión normal usa el modelo de facturación previo (código `1`) con transmisión normal, y la contingencia el código `2` (Anexo IV, campo 6, reglas 2 y 3; campo 7, reglas 2 y 3).

---

### `items[]` — Ítems

El número de ítem lo asigna el servicio según la posición en el arreglo (`numItem` = 1, 2, 3...); el campo `number` del request se ignora.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1500 caracteres) |
| `quantity` | float | Sí | Cantidad (> 0, hasta `99999999999.99`) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99; `99` obligatorio para tipo `4`) |
| `unit_price` | float | Sí | Precio unitario (≥ 0, hasta 8 decimales). **Debe ser `0` cuando `non_taxed > 0`** y **no puede ser `0` cuando `taxed_sale > 0`** |
| `discount` | float | No | **Monto** de descuento del ítem (no porcentaje). El servicio acepta cualquier monto de `0` a `99999999999.99`; un valor negativo se rechaza con `InvalidAmount` |
| `non_subject_sale` | float | No | Venta no sujeta (≥ 0) |
| `exempt_sale` | float | No | Venta exenta (≥ 0) |
| `taxed_sale` | float | No | Venta gravada (≥ 0) |
| `non_taxed` | float | No | No afecto (≥ 0) |
| `suggested_price` | float | No | Precio sugerido de venta (PSV) |
| `code` | string | No | Código del producto (1–24 caracteres efectivos; el mensaje de error indica "1 a 25") |
| `tax_code` | string | No | Código de tributo (CAT-015). Solo se emite como `codTributo` en ítems de tipo `4` |
| `related_doc` | string | Condicional | `document_number` del documento relacionado al que pertenece el ítem. Obligatorio en todos los ítems si hay `related_docs` |
| `taxes` | string[] | Condicional | Códigos de tributo del ítem (CAT-015). **Obligatorio cuando `taxed_sale > 0`** (debe incluir `"20"`) |

El CCF **no usa** `iva_item` por ítem.

> **Normativa:** los campos del ítem corresponden a la sección Cuerpo del Anexo IV de la Normativa de Cumplimiento DTE v2.0: número de ítem (campo 74; máximo 2000 en la regla 2), `type` (campo 75), `related_doc` (campo 80), `quantity` (campo 82), `code` (campo 83), `tax_code` (campo 84), `unit_measure` (campo 85, CAT-014; código `99` para los ítems tipo `4`, regla 4), `description` (campo 86, hasta 1,500 caracteres), `unit_price` (campo 87; sin IVA, regla 2, y `0` cuando se usa `non_taxed`, regla 4), `discount` (campo 89; no puede ser mayor al monto del ítem, regla 3), `non_subject_sale`, `exempt_sale` y `taxed_sale` (campos 90, 91 y 92), `taxes` (campo 96; requerido siempre que existan ventas gravadas, regla 4, y con el código `20` para los ítems tipo `4` con ventas gravadas, regla 15), `suggested_price` (campo 97) y `non_taxed` (campo 98). El IVA por ítem (campo 102) no aplica al CCF. Los campos del cuerpo se redondean según la regla 7.2 literal a) y se comparan con la holgura de la regla 7.3.

> **Regla:** Solo se puede usar **un tipo de venta por ítem**: `taxed_sale`, `exempt_sale`, `non_subject_sale` o `non_taxed` (`InvalidMixedSalesWithNonTaxed`, `InvalidMixedSalesWithExempt`, `InvalidMixedSalesWithNonSubject`).

> **Regla:** Ítems con `non_taxed`, `exempt_sale` o `non_subject_sale` mayores a `0` no pueden llevar `taxes`. Con `non_taxed > 0`, `unit_price` debe ser `0`.

> **Regla:** Ítems de tipo `1` (Producto) solo admiten el tributo `20` (IVA) en `taxes` (`InvalidTaxCodeOnly20`).

> **Regla:** Ítems de tipo `4` (Impuesto): `unit_measure = 99` y `taxes` debe contener exactamente `["20"]` (Anexo IV, campo 85, regla 4; campo 96, regla 15).

> **Regla:** La venta del ítem (no sujeta + exenta + gravada) debe cumplir `(unit_price × quantity) - discount` (Anexo IV, campos 90 a 92, regla 2; ver [calculations.md](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95), sección 2.1). Cuando `unit_price = 0` esta fórmula no se aplica.

---

### `receiver` — Receptor

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `name` | string | Sí | Nombre o razón social |
| `nit` | string | Sí | NIT del receptor (9 o 14 dígitos, con o sin guiones; se valida al mapear el request, `INVALIDPATTERN`) |
| `nrc` | string | Sí | NRC del receptor (1–8 dígitos) |
| `activity_code` | string | Sí | Código de actividad económica (2–6 dígitos) |
| `activity_description` | string | Sí | Descripción de actividad económica |
| `address` | `Address` | Sí | Dirección |
| `commercial_name` | string | No | Nombre comercial |
| `phone` | string | No | Teléfono (8–30 caracteres) |
| `email` | string | No | Correo electrónico válido, de 3 a 100 caracteres. Se verifica que el dominio tenga registro MX |
| `document_type` | string | **No admitido** | El CCF identifica al receptor por `nit`; enviar `document_type` se rechaza (`InvalidField`) |
| `document_number` | string | **No admitido** | Enviar `document_number` se rechaza (`InvalidField`) |

> **Normativa:** la sección Receptor corresponde a los campos 38 a 57 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `nit` (campo 40: el NIT debe poseer NRC activo, regla 1, y tener de 9 a 14 números sin guiones, regla 3), `nrc` (campo 41), `name` (campo 42), `activity_code` (campo 43, CAT-019), `activity_description` (campo 44, hasta 150 caracteres), `commercial_name` (campo 45), `address.department` (campo 47, CAT-012), `address.municipality` (campo 48, CAT-013), `address.district` (campo 49, CAT-008), `address.complement` (campo 50, hasta 200 caracteres), `phone` (campo 56) y `email` (campo 57). El CCF identifica al receptor por NIT y no usa el tipo ni el número de documento de identificación (campos 38 y 39), por lo que `document_type` y `document_number` no se admiten. Cuando se relaciona un documento, el NIT debe ser el mismo del receptor del documento relacionado (campo 40, regla 2).

#### `address` (receptor)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí | Código de departamento (`01`–`14`) |
| `municipality` | string | Sí | Código de municipio válido para el departamento (CAT-013) |
| `district` | string | No | Código de distrito (`01`–`99`, dos dígitos). Si se omite o se envía vacío se usa `"01"` |
| `complement` | string | Sí | Dirección complementaria (1–200 caracteres) |

---

### `summary` — Resumen financiero

Los montos del resumen se envían con **máximo 2 decimales** (`InvalidDecimals`). Los campos `sub_total`, `sub_total_sales`, `total_operation` y `total_to_pay` no pueden ser `0` (`RequiredField`).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total ventas no sujetas |
| `total_exempt` | float | Sí | Total ventas exentas |
| `total_taxed` | float | Sí | Total ventas gravadas |
| `sub_total_sales` | float | Sí | Suma de operaciones (`total_non_subject + total_exempt + total_taxed`) |
| `non_subject_discount` | float | Sí | Descuento global sobre no sujeto |
| `exempt_discount` | float | Sí | Descuento global sobre exento |
| `taxed_discount` | float | Sí | Descuento global sobre gravado |
| `discount_percentage` | float | Sí | Porcentaje de descuento general (0–100) |
| `total_discount` | float | Sí | Total de descuentos |
| `sub_total` | float | Sí | Subtotal después de descuentos globales |
| `total_operation` | float | Sí | Monto total de la operación (`sub_total` + tributos) |
| `total_non_taxed` | float | Sí | Total de montos no afectos (debe coincidir con la suma de `non_taxed` de los ítems) |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito, `3`=Otro |
| `taxes` | `Tax[]` | Condicional | Tributos del resumen. Obligatorio si `total_taxed > 0` (debe incluir el `20` con valor > 0); no se admite si `total_taxed = 0` |
| `payment_types` | `Payment[]` | Sí* | Formas de pago. *Se puede omitir si se envía `payments` en la raíz |
| `iva_perception` | float | Sí | Percepción de IVA (1 %) |
| `iva_retention` | float | Sí | Retención de IVA |
| `income_retention` | float | Sí | **Debe ser `0`**: la retención de renta no aplica a CCF (`IncomeRetentionNotAllowed`) |
| `balance_in_favor` | float | Sí | Saldo a favor |
| `total_iva` | float | No | **No debe enviarse o debe ser `0`**: el CCF no tiene `totalIva`; un valor distinto de `0` se rechaza (`InvalidField`) |
| `total_in_words` | string | No | Total en letras. Si se omite, el servicio lo genera a partir de `total_to_pay` |
| `observations` | string | No | Observaciones (máx. 3000 caracteres) |

"Requerido" en esta tabla indica que el campo forma parte del contrato; si un monto no aplica, se envía `0`.

> **Normativa:** los campos del resumen corresponden a la sección Resumen del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `total_non_subject`, `total_exempt` y `total_taxed` (campos 123 a 125), `sub_total_sales` (campo 129), `non_subject_discount`, `exempt_discount` y `taxed_discount` (campos 130 a 132), `discount_percentage` (campo 134), `total_discount` (campo 135, regla 3), `taxes` (campos 136 a 138; el valor del tributo es requerido siempre que existan ventas gravadas, campo 138, regla 2), `sub_total` (campo 139), `iva_perception` (campo 141; 1 % sobre ventas gravadas, regla 1), `iva_retention` (campo 142), `total_operation` (campo 150, regla 2), `total_non_taxed` (campo 151), `total_to_pay` (campo 153, regla 2), `total_in_words` (campo 156), `balance_in_favor` (campo 158), `operation_condition` (campo 159) y `observations` (campo 168). El campo 157 (`total_iva`) aplica solo a FE y CRE. Los montos del resumen siguen las reglas 7.2 literal b) y 7.3, y la eliminación de `income_retention` en el CCF consta en el Anexo VI (versión 2.0, Anexos II y IV).

---

### `summary.taxes[]` — Tributos

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `code` | string | Código de tributo: `"20"` IVA, `"C3"` IVA exportación, `"59"` Turismo, `"71"` Turismo aeropuerto, `"D5"` Otro especial. `"D1"` (FOVIAL) y `"C8"` (COTRANS) se rechazan con `UnsupportedTaxCode` |
| `description` | string | Descripción del tributo (requerida) |
| `value` | float | Valor del tributo con 2 decimales. Debe ser distinto de `0` (salvo `C3`) |

La fórmula de cada tributo está en [calculations.md](./calculations.md#36-tributos-del-resumen-campos-136-a-138) (sección 3.6). `code`, `description` y `value` corresponden a los campos 136, 137 y 138 del Anexo IV de la Normativa de Cumplimiento DTE v2.0. El IVA (`20`) se calcula sobre `total_taxed - taxed_discount`.

---

### Formas de pago (`payments` o `summary.payment_types`)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `code` | string | Sí | Código de forma de pago: `"01"` a `"14"` o `"99"` (CAT-017) |
| `amount` | float | Sí | Monto (2 decimales). En ventas a crédito (condición `2`) debe enviarse `0.00` (ver reglas) |
| `period` | integer | Condicional | Período. Ver reglas |
| `term` | string | Condicional | Plazo (`"01"`, `"02"` o `"03"`). Ver reglas |
| `reference` | string | No | Referencia del pago |

**Reglas:**
- La suma de `amount` debe coincidir con `total_to_pay` (tolerancia ±0.01, `InvalidPaymentTotal`).
- En ventas a crédito (condición `2`) el valor normativo de `amount` es `0.00` (Anexo IV: el monto por forma de pago "deberá completarse con $0.00") y es el que deben enviar los consumidores. Con al menos un pago y todos los `amount` en `0.00` no se compara la suma con `total_to_pay`.
- Por compatibilidad, la API acepta de forma transitoria que con condición `2` el `amount` sea igual al total a pagar (la suma se compara con `total_to_pay` como siempre). Esta tolerancia se eliminará en una versión futura, después de la cual con condición `2` solo se aceptará `0.00`.
- Una lista de pagos vacía con condición `2` y `total_to_pay` mayor que 0 se rechaza con `InvalidPaymentTotal`.
- `code` es requerido por la API en todos los pagos (`InvalidPaymentTypeInfo`), aunque el Anexo IV solo lo exige con condición `1` o `3`. `reference` puede omitirse (con condición `2` puede ser `null`).
- Condición `2` (Crédito): si ningún pago usa el código `"01"`, todos los pagos deben informar `period` y `term` (`InvalidPaymentTerms`).
- Condición `1` (Contado): si ningún pago usa el código `"01"`, los pagos no deben informar `period` ni `term` (`InvalidPaymentTermsOF`).

> **Normativa:** las formas de pago corresponden a los campos 159 a 164 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `operation_condition` (campo 159, CAT-016), `code` (campo 160, CAT-017), `amount` (campo 161), `reference` (campo 162), `term` (campo 163, CAT-018) y `period` (campo 164). La suma de los montos por forma de pago debe ser igual al total a pagar (campo 161, regla 4), dentro de la holgura general de la regla 7.3.

---

### `related_docs[]` — Documentos relacionados

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de DTE relacionado: NR `04`, CL `08` o DCL `09` (otro tipo se rechaza con `InvalidRelatedDocDTEType`) |
| `generation_type` | integer | Sí | `1`=Físico (alfanumérico, 1–20 caracteres), `2`=Electrónico (UUID en mayúsculas) |
| `document_number` | string | Sí | Número del documento relacionado |
| `emission_date` | string | No | Fecha de generación `YYYY-MM-DD`. Si se omite se usa la fecha actual. No puede ser futura |

**Reglas:**
- Máximo 50 documentos (`ExceededRelatedDocsLimit`) y todos del mismo tipo (`MixedDocumentTypesNotAllowed`).
- Si se envían `related_docs`, **todos los ítems** deben incluir `related_doc` igual al `document_number` de alguno de los documentos (`MissingItemRelatedDoc`, `InvalidItemRelatedDoc`).
- Con NR (`04`) el documento debe generarse en el mismo periodo tributario de la NR o, a más tardar, dentro de los 3 días siguientes al fin de ese periodo; con DCL (`09`), dentro del mismo periodo (`RelatedDocOutOfPeriod`).

> **Normativa:** la sección Documentos relacionados corresponde a los campos 14 a 17 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `document_type` (campo 14), `generation_type` (campo 15, CAT-007), `document_number` (campo 16) y `emission_date` (campo 17). Para el CCF se pueden relacionar los códigos `04` (NR), `08` (CL) y `09` (DCL) (campo 14, regla 4), hasta 50 documentos del mismo tipo (campo 14, regla 7; campo 16, regla 2), y no pueden relacionarse documentos electrónicos invalidados o rechazados (campo 16, regla 3). Los plazos respecto de la NR (mismo período tributario o hasta 3 días siguientes a ese período) y del DCL (mismo período tributario) corresponden al campo 17, regla 2, literales a) y b). El ítem referencia el documento mediante `related_doc` (campo 80, regla 2).

---

### `other_docs[]` — Otros documentos

Máximo 10 (`InvalidOtherDocsCount`).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_code` | integer | Sí | Código: `1` Emisor, `2` Receptor, `3` Médico, `4` Transporte. El código `4` solo se permite en la Factura de Exportación (`11`); en los demás tipos se rechaza con `InvalidAssociatedDocumentCodeForType`. Como la FEXE no está disponible como endpoint, en la práctica el código `4` se rechaza |
| `description` | string | Condicional | Requerido para códigos 1, 2 y 4 (máx. 100 caracteres). No debe enviarse con código `3` |
| `detail` | string | Condicional | Requerido para códigos 1, 2 y 4 (máx. 300 caracteres). No debe enviarse con código `3` |
| `doctor` | `Doctor` | Condicional | Requerido para código `3`; no debe enviarse con los demás códigos |

#### `other_docs[].doctor` (solo código `3`)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `name` | string | Sí | Nombre del médico (1–100 caracteres) |
| `nit` | string | Condicional | NIT del médico (9 o 14 dígitos). Debe enviarse `nit` **o** `identification`, no ambos |
| `identification` | string | Condicional | Documento de identificación del médico |
| `service_type` | integer | Sí | Tipo de servicio médico (1–6) |

> **Normativa:** la sección Otros documentos corresponde a la sección 5 (Documentos asociados) del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `document_code` (campo 58, CAT-021), `description` y `detail` (campos 59 y 60) y los datos del médico (campos 61 a 64: nombre, NIT, documento de identificación del médico no domiciliado y código del servicio, CAT-010). La sección admite de 1 a 10 documentos asociados (campos 59 y 60, regla 4) y el código `3` (médico) está previsto para FE y CCF (campo 58, regla 4).

---

### `third_party_sale` — Venta a nombre de tercero

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `nit` | string | Sí | NIT del tercero (9 o 14 dígitos, con o sin guiones) |
| `name` | string | Sí | Nombre del tercero |
| `fiscal_domicile` | integer | Sí | Domicilio fiscal (CAT-032): `1`=Domiciliado, `2`=No domiciliado. Cualquier otro valor se rechaza (`InvalidField`) |

> **Normativa:** la sección Venta por cuenta de terceros corresponde a los campos 69 a 71 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `nit` (campo 69; sin guiones, regla 6), `name` (campo 70) y `fiscal_domicile` (campo 71, CAT-032). El Anexo VI (versión 2.0, Anexos II y IV) incorpora el campo 71 'Domicilio Fiscal'.

---

### `appendixes[]` — Apéndices

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `field` | string | Sí | Identificador del campo (2–25 caracteres) |
| `label` | string | Sí | Etiqueta visible (3–50 caracteres) |
| `value` | string | Sí | Valor del campo (1–150 caracteres) |

El sello de recepción **no** se agrega al apéndice: se entrega en el Archivo DTE (ver [Respuesta exitosa](#respuesta-exitosa)). `field`, `label` y `value` corresponden a los campos 172, 173 y 174 del Anexo IV de la Normativa de Cumplimiento DTE v2.0 (sección 11, Apéndice). El Sello de Recepción se incorpora al Archivo DTE que se entrega al receptor (Anexo IV, sección 12, campo 175).

---

## Reglas de cálculo

Las fórmulas completas y sus tolerancias están en [calculations.md](./calculations.md): venta por ítem ([2.1](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95)), totales de operaciones ([3.2](./calculations.md#32-totales-de-operaciones-campos-123-a-125-y-127)), suma de operaciones ([3.3](./calculations.md#33-suma-de-operaciones-campo-129)), descuentos globales ([3.4](./calculations.md#34-descuentos-globales-campos-130-a-135)), subtotal ([3.5](./calculations.md#35-subtotal-campo-139)), tributos del resumen ([3.6](./calculations.md#36-tributos-del-resumen-campos-136-a-138)), IVA percibido ([3.8](./calculations.md#38-iva-percibido-iva-retenido-y-retención-de-renta)), monto total de la operación ([3.9](./calculations.md#39-monto-total-de-la-operación-campo-150)), total a pagar ([3.10](./calculations.md#310-total-a-pagar-campo-153)) y pagos ([3.11](./calculations.md#311-pagos)). El redondeo (regla 7.2) y la holgura de ±0.01 (regla 7.3, Cuadros 1, 2 y 3) de la Normativa de Cumplimiento DTE v2.0 se describen en las secciones [1.1](./calculations.md#11-decimales-y-redondeo) y [1.2](./calculations.md#12-holgura). Resumen de las reglas que se validan:

- **Diferencia clave respecto a la FE:** el IVA (`20`) del resumen se calcula sobre la base gravada **después** del descuento global: `(total_taxed - taxed_discount) × 0.13` (±0.01). Los demás tributos se calculan sobre `total_taxed`.
- Venta del ítem = `(unit_price × quantity) - discount` (±0.01; Anexo IV, campos 90 a 92).
- `total_taxed` debe coincidir con la suma de `taxed_sale` (±0.01); `total_exempt`, `total_non_subject` y `total_non_taxed` deben coincidir exactamente con la suma de los ítems (Anexo IV, campos 123 a 125 y 151).
- `sub_total_sales`, `sub_total` y `total_operation` se validan con la fórmula de CCF (±0.01; campos 129, 139 y 150).
- `iva_perception`, cuando es distinto de `0`, debe ser `total_taxed × 0.01` (±0.01; campo 141).
- `total_to_pay` = `total_operation + iva_perception - iva_retention + total_non_taxed` (±0.01; si resulta negativo se ingresa `0.00`; campo 153, reglas 2 y 7).
- La suma de pagos debe coincidir con `total_to_pay` (campo 161, regla 4).

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Receptor | `name`, `nit`, `nrc`, `activity_code`, `activity_description` y `address` obligatorios; `document_type` y `document_number` no se admiten |
| Ítems | Mínimo 1, máximo 2000 (`ExceededItemsLimit`; Anexo IV, campo 74, regla 2) |
| Descripción del ítem | 1–1500 caracteres (Anexo IV, campo 86) |
| Unidad de medida | 1–99 |
| `taxed_sale > 0` | El ítem requiere `taxes` (`MissingTaxesItem`) y el resumen requiere el tributo `20` con valor > 0 (`MissingTaxes`, `MissingIVAForTaxedAmount`) |
| `total_taxed = 0` | El resumen no puede traer tributos (`InvalidTaxes`) |
| Tipo de ítem `4` | `unit_measure = 99`, solo impuesto `20` |
| `income_retention` | Debe ser `0` (eliminado por el Anexo VI) |
| `total_iva` | Debe omitirse o ser `0` |
| Descuentos | No pueden superar su base (`DiscountExceedsBase`) |
| Montos de resumen | Máximo 2 decimales (`InvalidDecimals`, `InvalidMonetaryAmount`) |
| Montos | Entre `0` y `99999999999.99` |
| `other_docs` | Máximo 10 (Anexo IV, sección 5) |
| `related_docs` | Máximo 50 (Anexo IV, campo 14, regla 7); si están presentes, todos los ítems necesitan `related_doc` (campo 80, regla 2) |

---

## Respuesta exitosa

**HTTP 201 Created**

La emisión normal devuelve en `data` el **Archivo DTE**: la estructura del documento tal como fue firmada y transmitida, más los atributos `firmaElectronica` (JWS firmado) y `selloRecibido` (sello de recepción de Hacienda). El sello de recepción se repite en `reception_stamp` en la raíz de la respuesta.

> **Normativa:** el Archivo DTE se rige por la regla 4 (Entrega) del Anexo I y el Cuadro 11 de la Normativa de Cumplimiento DTE v2.0: contiene la estructura de datos íntegra, el documento firmado y el Sello de Recepción (`selloRecibido`, Anexo I, regla 3.4). La regla 11.1 exige entregar al receptor el DTE íntegro, con su Representación Gráfica y el Sello de Recepción.

```json
{
  "success": true,
  "reception_stamp": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=7C1E9A44-2B3D-4F5A-9E8B-1A2B3C4D5E6F&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 4,
      "ambiente": "01",
      "tipoDte": "03",
      "numeroControl": "DTE-03-N0010001-000000000000001",
      "codigoGeneracion": "7C1E9A44-2B3D-4F5A-9E8B-1A2B3C4D5E6F",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2026-05-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "emisor": {
      "nit": "06141101690011",
      "nrc": "1937",
      "nombre": "Emisor de Ejemplo S.A. de C.V.",
      "codActividad": "47190",
      "descActividad": "Venta al por menor de otros productos",
      "direccion": { "departamento": "06", "municipio": "23", "distrito": "01", "complemento": "Dirección de ejemplo" },
      "telefono": "22220000",
      "correo": "emisor@example.com",
      "nombreComercial": null,
      "codEstable": "N001",
      "codPuntoVenta": "0001"
    },
    "receptor": {
      "nit": "06141101690011",
      "nombre": "Receptor de Ejemplo S.A. de C.V.",
      "nrc": "1937",
      "codActividad": "46900",
      "descActividad": "Venta al por mayor de otros productos",
      "nombreComercial": "Receptor Ejemplo",
      "direccion": { "departamento": "06", "municipio": "23", "distrito": "01", "complemento": "Dirección de ejemplo, San Salvador" },
      "telefono": "22220000",
      "correo": "receptor@example.com"
    },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoItem": 1,
        "numeroDocumento": null,
        "codigo": "MAT-002",
        "codTributo": null,
        "descripcion": "CODO PVC 1",
        "cantidad": 10,
        "uniMedida": 59,
        "precioUni": 10,
        "montoDescu": 0,
        "ventaNoSuj": 0,
        "ventaExenta": 0,
        "ventaGravada": 100,
        "tributos": ["20"],
        "psv": 0,
        "noGravado": 0
      }
    ],
    "resumen": {
      "totalNoSuj": 0,
      "totalExenta": 0,
      "totalGravada": 100,
      "subTotalVentas": 100,
      "descuNoSuj": 0,
      "descuExenta": 0,
      "descuGravada": 10,
      "porcentajeDescuento": 10,
      "totalDescu": 10,
      "tributos": [
        { "codigo": "20", "descripcion": "Impuesto al Valor Agregado 13%", "valor": 11.7 }
      ],
      "subTotal": 90,
      "ivaRete": 0,
      "ivaPerci": 0,
      "montoTotalOperacion": 101.7,
      "totalNoGravado": 0,
      "totalPagar": 101.7,
      "totalLetras": "CIENTO UNO 70/100",
      "saldoFavor": 0,
      "condicionOperacion": 1,
      "pagos": [
        { "codigo": "01", "montoPago": 101.7, "referencia": null, "plazo": null, "periodo": null }
      ],
      "numPagoElectronico": null,
      "observaciones": null
    },
    "documentoRelacionado": null,
    "otrosDocumentos": null,
    "ventaTercero": null,
    "apendice": null,
    "firmaElectronica": "eyJhbGciOiJSUzUxMiJ9.eyJpZGVudGlmaWNhY2lvbiI6...<firma>",
    "selloRecibido": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C"
  }
}
```

Notas sobre la estructura:

- Los campos opcionales sin valor se emiten como `null` (no se omiten). `documentoRelacionado`, `otrosDocumentos`, `ventaTercero` y `apendice` solo traen contenido si se enviaron en el request.
- `receptor` se identifica por `nit`; no incluye `tipoDocumento` ni `numDocumento`.
- `resumen` no incluye `totalIva` ni retención de renta; sí incluye `ivaRete` e `ivaPerci`.
- `ventaTercero` incluye `nit`, `nombre` y `codDomiciliado` (el valor de `fiscal_domicile`).
- `emisor` solo incluye `codEstable` y `codPuntoVenta` como códigos de establecimiento y punto de venta; no existen `tipoEstablecimiento`, `codEstableMH` ni `codPuntoVentaMH`.
- El orden de las llaves dentro de `data` no es significativo. Para entregar el documento al receptor se debe conservar `data` íntegro, sin alterarlo.
- Si el Archivo DTE no pudiera armarse (firma no disponible), `data` contiene únicamente la estructura del documento, sin `firmaElectronica` ni `selloRecibido`.

---

## Contingencia

El CCF soporta contingencia. Si la transmisión a Hacienda falla por causas no atribuibles al documento (red, indisponibilidad del servicio, timeout, etc.), el documento se almacena en contingencia a la espera de retransmisión y se responde igualmente **HTTP 201**, pero:

- `reception_stamp` es `null`.
- `data` es la estructura del documento **sin** `firmaElectronica` ni `selloRecibido` (no es un Archivo DTE).
- `data.identificacion` cambia: `tipoModelo` pasa a `2` (diferido), `tipoOperacion` a `2` (contingencia), y se informan `tipoContingencia` (1–5, según el catálogo de contingencia) y `motivoContin`.

No se activa contingencia cuando el documento falla validación o Hacienda lo rechaza (`RECHAZADO`): en esos casos se devuelve el error correspondiente y el número de control se libera. Las condiciones de activación y los tipos de contingencia se describen en el [README](./README.md#contingencia).

> **Normativa:** la contingencia se rige por la regla 13.2 y el Cuadro 7 de la Normativa de Cumplimiento DTE v2.0, que comprende al CCF entre los documentos admitidos. El Anexo IV (campos 6 a 9) define el modelo de facturación, el tipo de transmisión, el tipo de contingencia (CAT-005, códigos 1 a 5) y el motivo de contingencia, requerido cuando el tipo es `5` (campo 9, regla 1). Un documento en contingencia no cuenta con Sello de Recepción hasta su transmisión (regla 11.2).

---

## Respuestas de error

Los errores de validación devuelven `HTTP 400` con la estructura `error.code`, `error.message` y `error.details`. El valor de `code` depende de la etapa en la que se detecta el problema:

| Etapa | `code` | `message` | `details` |
|-------|--------|-----------|-----------|
| Lectura del request (JSON inválido) | `BAD_REQUEST` | `Invalid request format` | vacío |
| Estructura mínima (falta `items`, `summary` o `receiver`, `total_iva ≠ 0`, `receiver.document_type`, etc.) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDFIELD`, ...) | Mensaje del campo | `No hay más detalles disponibles` |
| Mapeo de una sección (receptor, formatos, rangos, decimales) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDDECIMALS`, ...) o `ERRORMAPPING` | `Error al mapear la sección CCF->...` | Mensaje del campo |
| Reglas específicas del CCF | `VALIDATIONFAILED` | `Validaciones no superadas, revise los detalles a continuación` | Lista de mensajes de cada regla incumplida |
| Reglas comunes a los DTE | `MANY_ERRORS` (o el tipo del error si es uno solo) | `No se pudo crear el DTE, revise los detalles a continuación` | Lista de mensajes |
| Rechazo de Hacienda | `HACIENDA_<codigo>` | Descripción de Hacienda | Estado, fecha de proceso y observaciones |
| Error interno | `SYSTEM_ERROR` (`HTTP 500`) | `Ha ocurrido un error inesperado` | vacío |

Con la depuración activa del servidor, los mensajes de validación se prefijan con la clave `[Clave]`. Las reglas específicas del CCF se evalúan primero; si pasan, se evalúan las reglas comunes.

### Reglas específicas del CCF (`VALIDATIONFAILED`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `InvalidMixedSalesWithNonTaxed`, `InvalidMixedSalesWithExempt`, `InvalidMixedSalesWithNonSubject` | Un ítem mezcla tipos de venta |
| `InvalidTaxesWithNonTaxed`, `InvalidTaxesWithExempt`, `InvalidTaxesWithNonSubject` | Un ítem no gravado, exento o no sujeto tiene `taxes` |
| `InvalidUnitPriceWithNonTaxed` | Un ítem con `non_taxed` tiene `unit_price ≠ 0` |
| `MissingTaxesItem` | Un ítem con `taxed_sale > 0` no tiene `taxes` |
| `InvalidTaxCodeOnly20` | Un ítem de tipo `1` o `4` tiene un tributo distinto de `20` |
| `InvalidUnitMeasure`, `InvalidTaxRulesCCF` | Ítem tipo `4` con `unit_measure ≠ 99` o `taxes` distinto de `["20"]` |
| `InvalidItemSaleAmount` | La venta del ítem no coincide con `(unit_price × quantity) - discount` |
| `InvalidUnitPriceZero`, `InvalidUnitPrice` | `taxed_sale > 0` con `unit_price = 0` |
| `MissingItemRelatedDoc`, `InvalidItemRelatedDoc` | Hay `related_docs` y un ítem no referencia un documento válido |
| `InvalidTotalTaxed`, `InvalidTotalExempt`, `InvalidTotalNonSubject`, `InvalidTotalNonTaxed`, `InvalidNonTaxedAmount` | Un total del resumen no coincide con la suma de los ítems |
| `DiscountExceedsBase` | Un descuento global supera su base |
| `InvalidSubTotalSales` | `sub_total_sales` no coincide con la suma de operaciones |
| `InvalidSubTotalCalculation` | `sub_total` no coincide con la fórmula esperada |
| `MissingTaxes`, `MissingIVAForTaxedAmount` | `total_taxed > 0` sin tributos, o sin el tributo `20` con valor > 0 |
| `InvalidTaxes` | Hay tributos en el resumen con `total_taxed = 0` |
| `InvalidTaxCalculation` | Un tributo no coincide con su cálculo esperado |
| `InvalidPerceptionAmount` | `iva_perception` no coincide con `total_taxed × 0.01` |
| `InvalidTotalOperation` | `total_operation` no coincide con `sub_total` más tributos |
| `IncomeRetentionNotAllowed` | `income_retention ≠ 0` |
| `InvalidTotalToPayCalculation` | `total_to_pay` no coincide con la fórmula esperada |
| `InvalidMonetaryAmount` | Un monto del resumen o un pago tiene más de 2 decimales |
| `MissingNRC`, `RequiredField` | El receptor no tiene NRC, código o descripción de actividad |
| `ExceededRelatedDocsLimit`, `InvalidRelatedDocDTEType` | Más de 50 documentos relacionados, o tipo distinto de `04`, `08`, `09` |

### Reglas comunes (`MANY_ERRORS`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `UnsupportedTaxCode` | Código de tributo fuera del catálogo |
| `InvalidItemNumber`, `InvalidItemType` | Número o tipo de ítem inválido |
| `InvalidLength` | Descripción de ítem fuera de 1–1500, código de ítem excedido, textos de `other_docs` excedidos |
| `InvalidNumberRange` | `unit_measure` fuera de 1–99 |
| `InvalidPaymentTypeOP2`, `InvalidPaymentTerms`, `InvalidPaymentTermsOF` | Reglas de `period` y `term` según la condición de operación |
| `InvalidPaymentTotal` | La suma de pagos no coincide con `total_to_pay` |
| `MixedDocumentTypesNotAllowed` | Documentos relacionados de tipos distintos |
| `InvalidRelatedDocDate` | Fecha del documento relacionado futura |
| `InvalidRelatedDocNumberNormal`, `InvalidRelatedDocNumberContingency` | Número físico (>20 caracteres) o UUID inválido |
| `RelatedDocOutOfPeriod` | Documento fuera del periodo permitido respecto a la NR o el DCL |
| `RequiredField` | Falta el nombre del tercero (`ThirdPartyName`) o campos de `other_docs` |
| `InvalidOtherDocsCount`, `InvalidAssociatedDocumentCode`, `InvalidAssociatedDocumentCodeForType`, `InvalidMedicalDocFields`, `MutuallyExclusiveFields`, `InvalidServiceType` | Reglas de `other_docs` y `doctor` |

---

## Ejemplo completo

### Request

CCF con un descuento global gravado de `10.00` (diferencia clave en el cálculo del IVA):

```json
{
  "items": [
    {
      "type": 1,
      "description": "CODO PVC 1",
      "quantity": 10,
      "unit_measure": 59,
      "unit_price": 10,
      "discount": 0,
      "code": "MAT-002",
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 100,
      "suggested_price": 0,
      "non_taxed": 0,
      "taxes": ["20"]
    }
  ],
  "receiver": {
    "nit": "06141101690011",
    "nrc": "1937",
    "name": "Receptor de Ejemplo S.A. de C.V.",
    "commercial_name": "Receptor Ejemplo",
    "activity_code": "46900",
    "activity_description": "Venta al por mayor de otros productos",
    "address": {
      "department": "06",
      "municipality": "23",
      "complement": "Dirección de ejemplo, San Salvador"
    },
    "phone": "22220000",
    "email": "receptor@example.com"
  },
  "summary": {
    "total_non_subject": 0,
    "total_exempt": 0,
    "total_taxed": 100,
    "sub_total_sales": 100,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "taxed_discount": 10,
    "discount_percentage": 10,
    "total_discount": 10,
    "sub_total": 90,
    "total_operation": 101.7,
    "total_non_taxed": 0,
    "total_to_pay": 101.7,
    "operation_condition": 1,
    "iva_perception": 0,
    "iva_retention": 0,
    "income_retention": 0,
    "balance_in_favor": 0,
    "taxes": [
      {
        "code": "20",
        "description": "Impuesto al Valor Agregado 13%",
        "value": 11.7
      }
    ],
    "payment_types": [
      {
        "code": "01",
        "amount": 101.7
      }
    ]
  }
}
```

**Verificación de cálculos:**
- `taxed_sale = 10 × 10 - 0 = 100`
- `IVA = (100.00 - 10.00) × 0.13 = 11.70`
- `sub_total = 100.00 - 10.00 = 90.00`
- `total_operation = 90.00 + 11.70 = 101.70`
- `total_to_pay = 101.70 + 0 - 0 + 0 = 101.70` = suma de pagos
