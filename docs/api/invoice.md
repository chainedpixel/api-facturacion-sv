# Factura Electrónica (FE)

## Descripción

La **Factura Electrónica** (código DTE `01`) se utiliza para documentar transacciones de venta de bienes o servicios a **consumidores finales**. El receptor es opcional: si no se envía, el documento se emite sin datos del receptor. El precio de la FE **incluye el IVA**, que se informa por ítem en `iva_item` y en el resumen en `total_iva` (no como tributo; Normativa de Cumplimiento DTE v2.0, Anexo IV, campo 87 regla 3, campo 102 y campo 157).

**Soporta contingencia:** Sí (ver [Contingencia](#contingencia))

---

## Endpoint

```
POST /api/v1/dte/invoices
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

> Los datos que el servicio genera por sí mismo **no se envían**: número de control, código de generación, fecha y hora de emisión, ambiente, versión y datos del emisor. La fecha de emisión es siempre la del momento de la petición. Los campos desconocidos del request (por ejemplo `extension`, que la FE no admite) se ignoran sin generar error.

> **Normativa:** los campos de la sección Identificación (versión, ambiente, tipo de documento, código de generación, número de control, modelo de facturación, tipo de transmisión, fecha, hora y moneda) corresponden a los campos 1 a 12 del Anexo IV de la Normativa de Cumplimiento DTE v2.0; la versión de la estructura de la FE es 2 (campo 1). El código de generación y el número de control se rigen además por las reglas 7.1.1 y 7.1.2, y los datos del emisor corresponden a los campos 18 a 33 del mismo Anexo.

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `InvoiceItem[]` | Sí | Lista de ítems (1–2000) |
| `summary` | `InvoiceSummary` | Sí | Resumen financiero del documento |
| `receiver` | `Receiver` | No | Datos del receptor. Se puede enviar `null` u omitir (consumidor final) |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `payments` | `Payment[]` | No | Formas de pago. Si se envía, **reemplaza** a `summary.payment_types` |
| `other_docs` | `OtherDocument[]` | No | Otros documentos asociados (máx. 10) |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados (máx. 50) |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |
| `model_type` | integer | No | Se acepta pero **no tiene efecto**: la emisión normal siempre es modelo previo (`1`) y la contingencia la determina el servicio |

> **Normativa:** la emisión normal usa el modelo de facturación previo (código `1`) con transmisión normal, y la contingencia el código `2` (Anexo IV, campo 6, reglas 2 y 3; campo 7, reglas 2 y 3).

---

### `items[]` — Ítems de la factura

El número de ítem lo asigna el servicio según la posición en el arreglo (`numItem` = 1, 2, 3...); el campo `number` del request se ignora.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1500 caracteres) |
| `quantity` | float | Sí | Cantidad (> 0, hasta `99999999999.99`) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99) |
| `unit_price` | float | Sí | Precio unitario (≥ 0, hasta 8 decimales). **Debe ser `0` cuando `non_taxed > 0`** y **no puede ser `0` cuando `taxed_sale > 0`** |
| `discount` | float | No | **Monto** de descuento del ítem (no porcentaje). El servicio acepta cualquier monto de `0` a `99999999999.99`; un valor negativo se rechaza con `InvalidAmount` |
| `non_subject_sale` | float | No | Venta no sujeta (≥ 0) |
| `exempt_sale` | float | No | Venta exenta (≥ 0) |
| `taxed_sale` | float | No | Venta gravada (≥ 0). Si es `> 0`, `iva_item` es obligatorio |
| `non_taxed` | float | No | No afecto (≥ 0). No se puede mezclar con otros tipos de venta |
| `suggested_price` | float | No | Precio sugerido de venta (PSV) |
| `iva_item` | float | Condicional | IVA incluido en el precio del ítem. Obligatorio (≠ 0) si `taxed_sale > 0`; no puede ser `> 0` si `taxed_sale = 0` |
| `code` | string | No | Código del producto (1–24 caracteres efectivos; el mensaje de error indica "1 a 25") |
| `tax_code` | string | No | Código de tributo (CAT-015). Solo se emite como `codTributo` en ítems de tipo `4` |
| `related_doc` | string | Condicional | `document_number` del documento relacionado al que pertenece el ítem. Obligatorio en todos los ítems si hay `related_docs` |
| `taxes` | string[] | No | Códigos de tributo aplicados al ítem (CAT-015). **No admite `20` ni `C3`** en FE |

Todos los campos numéricos se envían como número JSON. Los montos son siempre ≥ 0.

> **Normativa:** los campos del ítem corresponden a la sección Cuerpo del Anexo IV de la Normativa de Cumplimiento DTE v2.0: número de ítem (campo 74; máximo 2000 en la regla 2), `type` (campo 75), `related_doc` (campo 80), `quantity` (campo 82), `code` (campo 83), `tax_code` (campo 84), `unit_measure` (campo 85, CAT-014), `description` (campo 86, hasta 1,500 caracteres), `unit_price` (campo 87; en la FE con IVA incluido, regla 3, y `0` cuando se usa `non_taxed`, regla 4), `discount` (campo 89; no puede ser mayor al monto del ítem, regla 3), `non_subject_sale`, `exempt_sale` y `taxed_sale` (campos 90, 91 y 92), `taxes` (campo 96; en la FE no admite `20` ni `C3`, regla 2), `suggested_price` (campo 97), `non_taxed` (campo 98) e `iva_item` (campo 102). Los campos del cuerpo se redondean según la regla 7.2 literal a) y se comparan con la holgura de la regla 7.3.

> **Regla:** Solo se puede usar **un tipo de venta por ítem**: `taxed_sale`, `exempt_sale`, `non_subject_sale` o `non_taxed` (`MixedSalesTypesNotAllowed`, `InvalidMixedSalesWithNonTaxed`).

> **Regla:** En FE el IVA va incluido en el precio: un ítem con el tributo `20` o `C3` en `taxes` se rechaza (`TaxNotAllowedInInvoice`; Anexo IV, campo 96, regla 2).

> **Regla:** La venta del ítem debe cumplir `(unit_price × quantity) - discount` (Anexo IV, campos 90 a 92, regla 2; ver [calculations.md](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95), sección 2.1). Cuando `unit_price = 0` esta fórmula no se aplica.

---

### `receiver` — Receptor (opcional)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Condicional | `"36"` NIT, `"13"` DUI, `"02"` Carnet de residente, `"03"` Pasaporte, `"37"` Otro. Si se envía, `document_number` es obligatorio y viceversa |
| `document_number` | string | Condicional | Número de documento. NIT y DUI se aceptan **con o sin guiones** (se eliminan). NIT: 9 o 14 dígitos. DUI: 9 dígitos. Otros tipos: 3–20 caracteres |
| `name` | string | No | Nombre o razón social |
| `nrc` | string | No | NRC (1–8 dígitos). No se admite junto con `document_type = "13"` (DUI) |
| `activity_code` | string | No | Código de actividad económica (2–6 dígitos) |
| `activity_description` | string | No | Descripción de actividad económica |
| `address` | `Address` | No | Dirección |
| `phone` | string | No | Teléfono (8–30 caracteres) |
| `email` | string | No | Correo electrónico válido, de 3 a 100 caracteres. Se verifica que el dominio tenga registro MX |
| `nit` | string | **No admitido** | Enviar `receiver.nit` se rechaza (`InvalidFieldValue`). Para identificar al receptor por NIT usar `document_type = "36"` y `document_number` |
| `commercial_name` | string | No | Se acepta pero no se emite en la FE |

#### `address` (receptor)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí* | Código de departamento (`01`–`14`) |
| `municipality` | string | Sí* | Código de municipio válido para el departamento (CAT-013) |
| `district` | string | No | Código de distrito (`01`–`99`, dos dígitos). Si se omite o se envía vacío se usa `"01"` |
| `complement` | string | Sí* | Dirección complementaria (1–200 caracteres) |

\* Si se envía `address`, `department`, `municipality` y `complement` son obligatorios (`AddressWithReceiver`).

> **Normativa:** la sección Receptor corresponde a los campos 38 a 57 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `document_type` (campo 38, CAT-022), `document_number` (campo 39), `nrc` (campo 41), `name` (campo 42), `address.department` (campo 47, CAT-012), `address.municipality` (campo 48, CAT-013), `address.district` (campo 49, CAT-008), `address.complement` (campo 50, hasta 200 caracteres), `phone` (campo 56) y `email` (campo 57). En la FE el tipo y número de documento y el nombre pueden ser `null` cuando el monto total de la operación es menor al monto legalmente establecido, y deben informarse cuando lo supera (campo 38, reglas 5 y 6; campo 39, reglas 2 y 3; campo 42, reglas 2 y 3). El código `13` (DUI) corresponde a personas naturales sin NRC (campo 38, regla 7) y exige 9 dígitos sin guiones (campo 39, regla 10); el número de documento admite hasta 20 caracteres (campo 39, regla 1). El NIT del receptor (campo 40) y el nombre comercial (campo 45) no aplican a la FE, y el distrito no es obligatorio en la factura (campo 49).

---

### `summary` — Resumen financiero

Los montos del resumen se envían con **máximo 2 decimales** (`InvalidDecimals`). Los campos `sub_total`, `sub_total_sales`, `total_operation` y `total_to_pay` no pueden ser `0` (`RequiredField`).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total de ventas no sujetas |
| `total_exempt` | float | Sí | Total de ventas exentas |
| `total_taxed` | float | Sí | Total de ventas gravadas |
| `sub_total_sales` | float | Sí | Suma de operaciones (`total_non_subject + total_exempt + total_taxed`) |
| `non_subject_discount` | float | Sí | Descuento global sobre no sujeto |
| `exempt_discount` | float | Sí | Descuento global sobre exento |
| `taxed_discount` | float | Sí | Descuento global sobre gravado |
| `discount_percentage` | float | Sí | Porcentaje de descuento general (0–100) |
| `total_discount` | float | Sí | Total de descuentos |
| `sub_total` | float | Sí | Subtotal después de descuentos globales |
| `total_operation` | float | Sí | Monto total de la operación |
| `total_non_taxed` | float | Sí | Total de montos no afectos (debe coincidir con la suma de `non_taxed` de los ítems) |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito, `3`=Otro |
| `payment_types` | `Payment[]` | Sí* | Formas de pago. *Se puede omitir si se envía `payments` en la raíz |
| `taxes` | `Tax[]` | No | Tributos adicionales del resumen. **No admite `20` ni `C3`** |
| `iva_retention` | float | Sí | Retención de IVA (solo válida si `total_taxed > 0`) |
| `income_retention` | float | Sí | **Debe ser `0`**: la retención de renta no aplica a FE (`IncomeRetentionNotAllowed`) |
| `total_iva` | float | Sí | IVA total contenido en el documento (ver cálculos) |
| `balance_in_favor` | float | Sí | Saldo a favor |
| `total_in_words` | string | No | Total en letras. Si se omite, el servicio lo genera a partir de `total_to_pay` |
| `observations` | string | No | Observaciones (máx. 3000 caracteres) |

"Requerido" en esta tabla indica que el campo forma parte del contrato; si un monto no aplica, se envía `0`.

> **Normativa:** los campos del resumen corresponden a la sección Resumen del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `total_non_subject`, `total_exempt` y `total_taxed` (campos 123 a 125), `sub_total_sales` (campo 129), `non_subject_discount`, `exempt_discount` y `taxed_discount` (campos 130 a 132), `discount_percentage` (campo 134), `total_discount` (campo 135), `taxes` (campos 136 a 138), `sub_total` (campo 139), `iva_retention` (campo 142), `total_operation` (campo 150), `total_non_taxed` (campo 151), `total_to_pay` (campo 153), `total_in_words` (campo 156), `total_iva` (campo 157), `balance_in_favor` (campo 158), `operation_condition` (campo 159) y `observations` (campo 168). Los montos del resumen siguen las reglas 7.2 literal b) y 7.3, y la eliminación de `income_retention` en la FE consta en el Anexo VI (versión 2.0, Anexos II y IV).

---

### `summary.taxes[]` — Tributos adicionales

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `code` | string | Código de tributo: `"59"` Turismo, `"71"` Turismo aeropuerto, `"D5"` Otro especial; `"D1"` (FOVIAL) y `"C8"` (COTRANS) se rechazan con `UnsupportedTaxCode` (también existen `"20"` y `"C3"` en el catálogo, pero se rechazan en FE) |
| `description` | string | Descripción del tributo (requerida) |
| `value` | float | Valor del tributo con 2 decimales. Debe ser distinto de `0` |

Si `total_taxed = 0` no se admiten tributos en el resumen (`InvalidTaxes`). La fórmula de cada tributo está en [calculations.md](./calculations.md#36-tributos-del-resumen-campos-136-a-138) (sección 3.6). `code`, `description` y `value` corresponden a los campos 136, 137 y 138 del Anexo IV de la Normativa de Cumplimiento DTE v2.0.

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
| `document_type` | string | Sí | Tipo de DTE relacionado. Los tipos previstos para FE son NR `04`, CL `08` y DCL `09` |
| `generation_type` | integer | Sí | `1`=Físico (alfanumérico, 1–20 caracteres), `2`=Electrónico (UUID en mayúsculas) |
| `document_number` | string | Sí | Número del documento relacionado |
| `emission_date` | string | No | Fecha de generación `YYYY-MM-DD`. Si se omite se usa la fecha actual. No puede ser futura |

**Reglas (comunes a FE y CCF):**
- Máximo 50 documentos (`ExceededRelatedDocsLimit`) y todos del mismo tipo (`MixedDocumentTypesNotAllowed`).
- Si se envían `related_docs`, **todos los ítems** deben incluir `related_doc` igual al `document_number` de alguno de los documentos (`MissingItemRelatedDoc`, `InvalidItemRelatedDoc`).
- Con NR (`04`) el documento debe generarse en el mismo periodo tributario de la NR o, a más tardar, dentro de los 3 días siguientes al fin de ese periodo; con DCL (`09`), dentro del mismo periodo (`RelatedDocOutOfPeriod`).
- En el estado actual del código, la FE **no verifica** que `document_type` sea uno de `04`, `08` o `09` (solo se exige un tipo de DTE válido); el CCF sí lo verifica.

> **Normativa:** la sección Documentos relacionados corresponde a los campos 14 a 17 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `document_type` (campo 14), `generation_type` (campo 15, CAT-007), `document_number` (campo 16) y `emission_date` (campo 17). Para la FE se pueden relacionar los códigos `04` (NR), `08` (CL) y `09` (DCL) (campo 14, regla 4), hasta 50 documentos del mismo tipo (campo 14, regla 7; campo 16, regla 2), y no pueden relacionarse documentos electrónicos invalidados o rechazados (campo 16, regla 3). Los plazos respecto de la NR (mismo período tributario o hasta 3 días siguientes a ese período) y del DCL (mismo período tributario) corresponden al campo 17, regla 2, literales a) y b). El ítem referencia el documento mediante `related_doc` (campo 80, regla 2).

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

Las fórmulas completas y sus tolerancias están en [calculations.md](./calculations.md): venta por ítem ([2.1](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95)), `iva_item` ([2.3](./calculations.md#23-iva-por-ítem-en-la-fe-campo-102)), totales de operaciones ([3.2](./calculations.md#32-totales-de-operaciones-campos-123-a-125-y-127)), suma de operaciones ([3.3](./calculations.md#33-suma-de-operaciones-campo-129)), subtotal ([3.5](./calculations.md#35-subtotal-campo-139)), tributos del resumen ([3.6](./calculations.md#36-tributos-del-resumen-campos-136-a-138)), `total_iva` ([3.7](./calculations.md#37-iva-de-la-fe-en-el-resumen-campo-157-total_iva)), monto total de la operación ([3.9](./calculations.md#39-monto-total-de-la-operación-campo-150)), total a pagar ([3.10](./calculations.md#310-total-a-pagar-campo-153)) y pagos ([3.11](./calculations.md#311-pagos)). El redondeo (regla 7.2) y la holgura de ±0.01 (regla 7.3, Cuadros 1, 2 y 3) de la Normativa de Cumplimiento DTE v2.0 se describen en las secciones [1.1](./calculations.md#11-decimales-y-redondeo) y [1.2](./calculations.md#12-holgura). Resumen de las reglas que se validan:

- Venta del ítem = `(unit_price × quantity) - discount` y `iva_item = (taxed_sale / 1.13) × 0.13`, ambos con tolerancia ±0.01 (Anexo IV, campos 90 a 92 y 102).
- Los totales del resumen (`total_taxed`, `total_exempt`, `total_non_subject`, `total_non_taxed`) deben coincidir con la suma de los ítems (±0.01; Anexo IV, campos 123 a 125 y 151).
- `sub_total_sales`, `sub_total`, `total_operation` y `total_to_pay` se validan con la fórmula de FE (±0.01; campos 129, 139, 150 y 153); `total_iva` se valida contra la suma de `iva_item` menos el IVA del descuento global gravado (campo 157, regla 2).
- La suma de pagos debe coincidir con `total_to_pay` (campo 161, regla 4).

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Ítems | Mínimo 1, máximo 2000 (`ExceededItemsLimit`; Anexo IV, campo 74, regla 2) |
| Descripción del ítem | 1–1500 caracteres (Anexo IV, campo 86) |
| Unidad de medida | 1–99 |
| Tipo tributario por ítem | Un solo tipo de venta por ítem |
| Tributos `20` y `C3` | No se admiten en ítems ni en el resumen |
| `income_retention` | Debe ser `0` (eliminado por el Anexo VI) |
| `iva_retention` | Solo válida si `total_taxed > 0` (`InvalidIVARetentionWithoutTaxedAmount`) |
| `total_taxed > 0` | Requiere `total_iva > 0` (`MissingIVAForTaxedAmount`) |
| Descuentos | No pueden ser negativos ni superar su base (`NegativeDiscount`, `DiscountExceedsBase`, `DiscountExceedsSubtotal`) |
| Montos de resumen | Máximo 2 decimales (`InvalidDecimals`, `InvalidMonetaryAmount`) |
| Montos | Entre `0` y `99999999999.99` |
| Receptor | `document_type` y `document_number` van juntos; NIT 9 o 14 dígitos; DUI 9 dígitos |
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
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 2,
      "ambiente": "01",
      "tipoDte": "01",
      "numeroControl": "DTE-01-N0010001-000000000000001",
      "codigoGeneracion": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2026-05-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "resumen": {
      "totalNoSuj": 0,
      "totalExenta": 0,
      "totalGravada": 225,
      "subTotalVentas": 225,
      "descuNoSuj": 0,
      "descuExenta": 0,
      "descuGravada": 10,
      "porcentajeDescuento": 4.44,
      "totalDescu": 10,
      "tributos": null,
      "subTotal": 215,
      "ivaRete": 0,
      "montoTotalOperacion": 215,
      "totalNoGravado": 0,
      "totalPagar": 215,
      "totalLetras": "DOSCIENTOS QUINCE 00/100",
      "totalIva": 24.73,
      "saldoFavor": 0,
      "condicionOperacion": 1,
      "pagos": [
        { "codigo": "01", "montoPago": 215, "referencia": null, "plazo": null, "periodo": null }
      ],
      "numPagoElectronico": null,
      "observaciones": null
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
      "nombre": null,
      "tipoDocumento": null,
      "numDocumento": null,
      "nrc": null,
      "codActividad": null,
      "descActividad": null,
      "direccion": null,
      "telefono": null,
      "correo": null
    },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoItem": 1,
        "numeroDocumento": null,
        "codigo": "EJEMPLO-CODIGO",
        "codTributo": null,
        "descripcion": "Producto de ejemplo",
        "cantidad": 3,
        "uniMedida": 59,
        "precioUni": 75,
        "montoDescu": 0,
        "ventaNoSuj": 0,
        "ventaExenta": 0,
        "ventaGravada": 225,
        "tributos": null,
        "psv": 0,
        "noGravado": 0,
        "ivaItem": 25.88
      }
    ],
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
- `receptor.direccion` es `null` si no se envió `address`; cuando se envía incluye `distrito`.
- `ventaTercero` incluye `nit`, `nombre` y `codDomiciliado` (el valor de `fiscal_domicile`).
- `emisor` solo incluye `codEstable` y `codPuntoVenta` como códigos de establecimiento y punto de venta; no existen `tipoEstablecimiento`, `codEstableMH` ni `codPuntoVentaMH`.
- La FE no incluye `extension` ni retención de renta en la estructura emitida.
- El orden de las llaves dentro de `data` no es significativo. Para entregar el documento al receptor se debe conservar `data` íntegro, sin alterarlo.
- Si el Archivo DTE no pudiera armarse (firma no disponible), `data` contiene únicamente la estructura del documento, sin `firmaElectronica` ni `selloRecibido`.

---

## Contingencia

La FE soporta contingencia. Si la transmisión a Hacienda falla por causas no atribuibles al documento (red, indisponibilidad del servicio, timeout, etc.), el documento se almacena en contingencia a la espera de retransmisión y se responde igualmente **HTTP 201**, pero:

- `reception_stamp` es `null`.
- `data` es la estructura del documento **sin** `firmaElectronica` ni `selloRecibido` (no es un Archivo DTE).
- `data.identificacion` cambia: `tipoModelo` pasa a `2` (diferido), `tipoOperacion` a `2` (contingencia), y se informan `tipoContingencia` (1–5, según el catálogo de contingencia) y `motivoContin`.

```json
{
  "success": true,
  "reception_stamp": null,
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 2,
      "ambiente": "01",
      "tipoDte": "01",
      "numeroControl": "DTE-01-N0010001-000000000000001",
      "codigoGeneracion": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
      "tipoModelo": 2,
      "tipoOperacion": 2,
      "tipoContingencia": 1,
      "motivoContin": "Servicio del Ministerio de Hacienda no disponible",
      "fecEmi": "2026-05-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "emisor": { },
    "receptor": { },
    "cuerpoDocumento": [ ],
    "resumen": { }
  }
}
```

No se activa contingencia cuando el documento falla validación o Hacienda lo rechaza (`RECHAZADO`): en esos casos se devuelve el error correspondiente y el número de control se libera. Las condiciones de activación y los tipos de contingencia se describen en el [README](./README.md#contingencia).

> **Normativa:** la contingencia se rige por la regla 13.2 y el Cuadro 7 de la Normativa de Cumplimiento DTE v2.0, que comprende a la FE entre los documentos admitidos. El Anexo IV (campos 6 a 9) define el modelo de facturación, el tipo de transmisión, el tipo de contingencia (CAT-005, códigos 1 a 5) y el motivo de contingencia, requerido cuando el tipo es `5` (campo 9, regla 1). Un documento en contingencia no cuenta con Sello de Recepción hasta su transmisión (regla 11.2).

---

## Respuestas de error

Los errores de validación devuelven `HTTP 400` con la estructura `error.code`, `error.message` y `error.details`. El valor de `code` depende de la etapa en la que se detecta el problema:

| Etapa | `code` | `message` | `details` |
|-------|--------|-----------|-----------|
| Lectura del request (JSON inválido) | `BAD_REQUEST` | `Invalid request format` | vacío |
| Estructura mínima (falta `items` o `summary`, `receiver.nit`, etc.) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDFIELDVALUE`, ...) | Mensaje del campo | `No hay más detalles disponibles` |
| Mapeo de una sección (formatos, rangos, decimales) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDDECIMALS`, ...) o `ERRORMAPPING` | `Error al mapear la sección Invoice->...` | Mensaje del campo |
| Reglas específicas de la FE | `VALIDATIONFAILED` | `Validaciones no superadas, revise los detalles a continuación` | Lista de mensajes de cada regla incumplida |
| Reglas comunes a los DTE | `MANY_ERRORS` (o el tipo del error si es uno solo) | `No se pudo crear el DTE, revise los detalles a continuación` | Lista de mensajes |
| Rechazo de Hacienda | `HACIENDA_<codigo>` | Descripción de Hacienda | Estado, fecha de proceso y observaciones |
| Error interno | `SYSTEM_ERROR` (`HTTP 500`) | `Ha ocurrido un error inesperado` | vacío |

Con la depuración activa del servidor, los mensajes de validación se prefijan con la clave `[Clave]`. Las reglas específicas de la FE se evalúan primero; si pasan, se evalúan las reglas comunes.

### Reglas específicas de la FE (`VALIDATIONFAILED`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `MixedSalesTypesNotAllowed` | Un ítem tiene más de un tipo de venta |
| `InvalidMixedSalesWithNonTaxed` | Un ítem con `non_taxed` tiene otro tipo de venta |
| `InvalidUnitPriceForNonTaxed` | Un ítem con `non_taxed` tiene `unit_price ≠ 0` |
| `MissingItemUnitPrice` | `taxed_sale > 0` con `unit_price = 0` |
| `InvalidIVAItemWithoutTaxedSale` | `iva_item > 0` sin `taxed_sale` |
| `InvalidIVAItemCalculation` | `iva_item` no coincide con `(taxed_sale / 1.13) × 0.13` |
| `ExcessiveItemTotal` | La suma de ventas del ítem excede `unit_price × quantity` |
| `InvalidItemSaleAmount` | La venta del ítem no coincide con `(unit_price × quantity) - discount` |
| `TaxNotAllowedInInvoice` | Un ítem incluye el tributo `20` o `C3` |
| `TaxNotAllowedInInvoiceSummary` | El resumen incluye el tributo `20` o `C3` |
| `InvalidTotalTaxed`, `InvalidTotalExempt`, `InvalidTotalNonSubject` | El total del resumen no coincide con la suma de los ítems |
| `InvalidSubTotalSales` | `sub_total_sales` no coincide con la suma de operaciones |
| `InvalidSubTotal`, `InvalidSubTotalCalculation` | `sub_total` no coincide con la fórmula esperada |
| `NegativeDiscount` | Un descuento del resumen es negativo |
| `DiscountExceedsBase`, `DiscountExceedsSubtotal` | Un descuento supera su base o el subtotal |
| `InvalidTotalOperation` | `total_operation` no coincide con `sub_total` más tributos |
| `InvalidTaxes` | Hay tributos en el resumen con `total_taxed = 0` |
| `InvalidTaxCalculation` | Un tributo no coincide con su cálculo esperado |
| `MissingIVAForTaxedAmount` | `total_taxed > 0` con `total_iva ≤ 0` |
| `InvalidTotalIVA` | `total_iva` no coincide con el valor esperado |
| `IncomeRetentionNotAllowed` | `income_retention ≠ 0` |
| `InvalidIVARetentionWithoutTaxedAmount` | `iva_retention > 0` sin venta gravada |
| `InvalidNonTaxedAmount`, `InvalidNonTaxedAmountCalculation` | `total_non_taxed` no coincide con los ítems |
| `InvalidTotalToPayCalculation` | `total_to_pay` no coincide con la fórmula esperada |
| `InvalidMonetaryAmount` | Un monto del resumen o un pago tiene más de 2 decimales |
| `MissingItemRelatedDoc`, `InvalidItemRelatedDoc` | Hay `related_docs` y un ítem no referencia un documento válido |
| `ExceededItemsLimit` | Más de 2000 ítems |

### Reglas comunes (`MANY_ERRORS`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `InvalidNITFormat`, `InvalidDUIFormat` | `document_number` de NIT (9 o 14 dígitos) o DUI (9 dígitos) con formato inválido |
| `UnsupportedTaxCode` | Código de tributo fuera del catálogo |
| `InvalidItemNumber`, `InvalidItemType` | Número o tipo de ítem inválido |
| `InvalidLength` | Descripción de ítem fuera de 1–1500, código de ítem excedido, textos de `other_docs` excedidos |
| `InvalidNumberRange` | `unit_measure` fuera de 1–99 |
| `InvalidPaymentTypeOP2`, `InvalidPaymentTerms`, `InvalidPaymentTermsOF` | Reglas de `period` y `term` según la condición de operación |
| `InvalidPaymentTotal` | La suma de pagos no coincide con `total_to_pay` |
| `ExceededRelatedDocsLimit`, `MixedDocumentTypesNotAllowed` | Más de 50 documentos relacionados o tipos mezclados |
| `InvalidRelatedDocDate` | Fecha del documento relacionado futura |
| `InvalidRelatedDocNumberNormal`, `InvalidRelatedDocNumberContingency` | Número físico (>20 caracteres) o UUID inválido |
| `RelatedDocOutOfPeriod` | Documento fuera del periodo permitido respecto a la NR o el DCL |
| `RequiredField` | Falta el nombre del tercero (`ThirdPartyName`) o campos de `other_docs` |
| `InvalidOtherDocsCount`, `InvalidAssociatedDocumentCode`, `InvalidAssociatedDocumentCodeForType`, `InvalidMedicalDocFields`, `MutuallyExclusiveFields`, `InvalidServiceType` | Reglas de `other_docs` y `doctor` |

---

## Ejemplo completo

### Request

Factura a consumidor final con un descuento global gravado de `10.00`:

```json
{
  "items": [
    {
      "type": 1,
      "description": "Producto de ejemplo",
      "quantity": 3,
      "unit_measure": 59,
      "code": "EJEMPLO-CODIGO",
      "unit_price": 75,
      "discount": 0,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 225,
      "non_taxed": 0,
      "suggested_price": 0,
      "iva_item": 25.88
    }
  ],
  "receiver": null,
  "summary": {
    "total_non_subject": 0,
    "total_exempt": 0,
    "total_taxed": 225,
    "sub_total_sales": 225,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "taxed_discount": 10,
    "discount_percentage": 4.44,
    "total_discount": 10,
    "sub_total": 215,
    "total_operation": 215,
    "total_non_taxed": 0,
    "total_to_pay": 215,
    "operation_condition": 1,
    "iva_retention": 0,
    "income_retention": 0,
    "total_iva": 24.73,
    "balance_in_favor": 0,
    "payment_types": [
      {
        "code": "01",
        "amount": 215
      }
    ]
  }
}
```

**Verificación de cálculos:**
- `taxed_sale = 75 × 3 - 0 = 225`; `iva_item = 225 / 1.13 × 0.13 = 25.88`
- `sub_total = 225 - 10 = 215`; `total_operation = 215` (sin tributos adicionales)
- `total_iva = 25.88 - (10 / 1.13 × 0.13) = 25.88 - 1.15 = 24.73`
- `total_to_pay = 215 - 0 = 215` = suma de pagos
