# Nota de Crédito Electrónica

## Descripción

La **Nota de Crédito Electrónica** (código DTE `05`) se utiliza para anular o ajustar parcialmente un Comprobante de Crédito Fiscal (CCF) o un Comprobante de Retención (CR) ya emitido y recibido por Hacienda. Requiere obligatoriamente documentos relacionados y cada ítem debe referenciar uno de ellos.

**Soporta contingencia:** No

---

## Endpoint

```
POST /api/v1/dte/creditnote
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `CreditNoteItem[]` | Sí | Lista de ítems (1–2000) |
| `receiver` | `Receiver` | Sí | Datos del receptor (ver sección `receiver`) |
| `summary` | `CreditNoteSummary` | Sí | Resumen financiero |
| `related_docs` | `RelatedDocument[]` | Sí | Documentos que se están ajustando (1–50) |
| `model_type` | integer | No | Se ignora: la nota siempre se genera con modelo de facturación previo (`1`) y transmisión normal |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero (se transmite como `ventaTercero`) |
| `payments` | `Payment[]` | No | Se acepta pero no forma parte del DTE de la nota; las formas de pago no se validan |
| `other_docs` | `OtherDocument[]` | No | Se acepta pero no forma parte del DTE de la nota |
| `appendixes` | `Appendix[]` | No | Apéndices (se transmiten como `apendice`) |

> La Nota de Crédito no tiene sección `extension`. El número de control, el código de generación, la fecha y la hora de emisión los asigna el servicio.

---

### `items[]` — Ítems

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1500 caracteres) |
| `quantity` | float | Sí | Cantidad |
| `unit_measure` | integer | Sí | Unidad de medida (1–99; `99` para tipo `4`) |
| `unit_price` | float | Sí | Precio unitario |
| `discount` | float | Sí | Descuento del ítem, **monto** no negativo (`InvalidAmount` si es negativo; no tiene tope de 100). Se transmite sin transformación como `montoDescu` |
| `non_subject_sale` | float | Sí | Venta no sujeta |
| `exempt_sale` | float | Sí | Venta exenta |
| `taxed_sale` | float | Sí | Venta gravada |
| `related_doc` | string | Sí | Código de generación del documento relacionado al que pertenece el ítem |
| `code` | string | No | Código del producto (máx. 25 caracteres) |
| `tax_code` | string | No | Código de tributo especial |
| `taxes` | string[] | Condicional | Códigos de tributo. Obligatorio si `taxed_sale > 0` |

> **Numeración:** el número de ítem (`numItem`) lo asigna el servicio según la posición en `items` (1, 2, 3...); el campo `number` del request no se utiliza.

> **Regla:** `related_doc` es obligatorio en cada ítem y debe coincidir con el `document_number` de alguno de los documentos de `related_docs`.

> **Regla:** Solo un tipo de venta (gravada, exenta o no sujeta) por ítem.

> **Regla:** Ítems con `taxed_sale > 0` requieren al menos un tributo, y todos deben ser códigos soportados.

> **Regla:** Ítems de tipo `4`: `unit_measure = 99` y único tributo `20`.

> **Regla:** Si `taxed_sale > 0`, `unit_price` no puede ser `0`.

> **Nota:** a diferencia del CCF y la Factura, en la Nota de Crédito no se verifica que `taxed_sale` sea igual a `unit_price × quantity` menos el descuento; solo se verifican los totales del resumen contra la suma de las ventas de los ítems.

---

### `receiver` — Receptor

El receptor de una nota se identifica por NIT y debe coincidir con el receptor del documento relacionado.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `nit` | string | Sí | NIT del receptor (14 o 9 dígitos; los guiones se eliminan) |
| `nrc` | string | Sí | NRC del receptor |
| `name` | string | Sí | Nombre |
| `email` | string | Sí | Correo electrónico |
| `address` | `Address` | Sí | Dirección (ver [Factura Electrónica](./invoice.md#receiver--receptor-opcional)) |
| `activity_code` | string | Sí | Código de actividad económica |
| `activity_description` | string | Sí | Descripción de la actividad económica |
| `commercial_name` | string | No | Nombre comercial |
| `phone` | string | No | Teléfono |

> `document_type` y `document_number` del receptor no se utilizan: al DTE se envía siempre `tipoDocumento = "36"` (NIT) y `numDocumento` con el NIT informado. Cuando se ajusta un CCF solo se admite el tipo `36`.

> **Regla:** el NIT debe ser igual al del receptor del CCF o CR relacionado; de lo contrario se rechaza con `NOTMATCHINGRECEIVERNIT`.

---

### `summary` — Resumen

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total no sujeto |
| `total_exempt` | float | Sí | Total exento |
| `total_taxed` | float | Sí | Total gravado |
| `sub_total_sales` | float | Sí | Subtotal de ventas |
| `non_subject_discount` | float | Sí | Debe ser `0` (`GlobalDiscountNotAllowedInAdjustment`) |
| `exempt_discount` | float | Sí | Debe ser `0` (`GlobalDiscountNotAllowedInAdjustment`) |
| `taxed_discount` | float | Sí | Debe ser `0` (`GlobalDiscountNotAllowedInAdjustment`) |
| `discount_percentage` | float | Sí | Debe ser `0` (`GlobalDiscountNotAllowedInAdjustment`) |
| `total_discount` | float | Sí | Total de descuentos |
| `sub_total` | float | Sí | Subtotal |
| `total_operation` | float | Sí | Monto total de la operación |
| `total_non_taxed` | float | Sí | Total no gravado (cargos/abonos) |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito, `3`=Otro |
| `taxes` | `Tax[]` | Condicional | Tributos del resumen. Obligatorio si `total_taxed > 0` (debe incluir el `20`); debe omitirse si `total_taxed = 0` |
| `iva_perception` | float | Sí | IVA percibido (1 % de `total_taxed`) |
| `iva_retention` | float | Sí | IVA retenido |
| `income_retention` | float | Sí | Debe ser `0`: la retención de renta no aplica a la Nota de Crédito (`IncomeRetentionNotAllowed`) |
| `total_in_words` | string | No | Total en letras; si se omite se genera a partir de `total_operation` |
| `observations` | string | No | Observaciones |

> Las Notas de Crédito y Débito no admiten descuentos globales ni porcentaje de descuento: el descuento se aplica por ítem.

---

### `related_docs[]` — Documentos relacionados

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de DTE ajustado: CCF `03` o CR `07` |
| `generation_type` | integer | Sí | `2`=Electrónico (código de generación UUID en mayúsculas); `1`=Físico (ver nota) |
| `document_number` | string | Sí | Código de generación del documento original |
| `emission_date` | string | Sí | `YYYY-MM-DD`. Para documentos electrónicos el servicio la reemplaza por la fecha de generación guardada del documento |

> **Regla:** solo se permiten los tipos `03` (CCF) y `07` (CR), y todos los documentos relacionados deben ser del mismo tipo (`MixedDocumentTypesNotAllowed`). Máximo 50.

> **Regla de plazo:** la nota debe generarse a más tardar 3 meses después de la fecha de generación del CCF o CR relacionado; de lo contrario se rechaza con `RelatedDocOutOfPeriod`. El mensaje indica el documento, su fecha y la fecha límite. Esta regla corresponde al numeral de documentos relacionados de la Normativa de Cumplimiento DTE v2.0 (la nota debe generarse a más tardar dentro de 3 meses respecto de la fecha de generación del CCF o CR). Los 3 meses se suman con `calendar.AddMonthsClamped`, que limita el resultado al último día del mes cuando el día no existe (30 de noviembre + 3 meses = 28 de febrero); es la misma función que usa el plazo de invalidación.

> **Regla:** el documento relacionado debe existir en la sucursal autenticada, estar en estado `RECEIVED` (`RelatedDocumentNotReceived`) y tener el mismo NIT de receptor.

> **Regla:** la fecha del documento relacionado no puede ser posterior a la fecha actual.

> **Nota sobre `generation_type = 1`:** el servicio busca todos los documentos relacionados en su base de datos por `document_number`, por lo que en la práctica solo se aceptan documentos emitidos por este servicio.

---

## Reglas de cálculo

Las fórmulas completas, redondeos y tolerancias se detallan en [Reglas de cálculo](./calculations.md). Resumen de lo que valida la Nota de Crédito (tolerancia ±0.01 salvo indicación):

- `Σ taxed_sale = total_taxed` (±0.01); `Σ exempt_sale = total_exempt` y `Σ non_subject_sale = total_non_subject` (exactos).
- `sub_total_sales = total_taxed + total_exempt + total_non_subject`.
- `sub_total = sub_total_sales` (no hay descuentos globales).
- Cada tributo del resumen: `20` = `total_taxed × 0.13`; `59` = `total_taxed × 0.05`; `71` = `7.00` fijo; `C3` = `0`; `D5` no se recalcula. `D1` (FOVIAL) y `C8` (COTRANS) se rechazan con `UnsupportedTaxCode`.
- `iva_perception`, si es distinto de `0`, debe ser `total_taxed × 0.01`.
- `total_operation = sub_total_sales + Σ tributos del resumen`.
- `total_to_pay = total_operation + iva_perception - iva_retention + total_non_taxed`.
- Los montos monetarios (`iva_perception`, `total_operation` y montos de pago) admiten máximo 2 decimales.

---

## Control de saldo

Al emitirse un CCF o CR se registra su saldo (gravado, exento y no sujeto). La Nota de Crédito descuenta de ese saldo sus totales (`total_taxed`, `total_exempt`, `total_non_subject`). Si alguno excede el saldo restante del documento relacionado, se rechaza con `INVALIDCREDITNOTETRANSACTION`.

---

## Respuesta exitosa

**HTTP 201 Created**

`data` contiene el **Archivo DTE**: la estructura firmada del documento, sin alteraciones, más `firmaElectronica` y `selloRecibido`. Es el archivo listo para entregar al receptor y se puede recuperar luego con `GET /api/v1/dte/{id}/archivo` (ver [Consulta de DTEs](./dte-query.md)).

```json
{
  "success": true,
  "reception_stamp": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=00&codGen=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX&fechaEmi=2026-01-15",
  "data": {
    "identificacion": {
      "version": 4,
      "ambiente": "00",
      "tipoDte": "05",
      "numeroControl": "DTE-05-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2026-01-15",
      "horEmi": "14:30:00",
      "tipoMoneda": "USD",
      "fusion": null
    },
    "emisor": { },
    "receptor": { "tipoDocumento": "36", "numDocumento": "XXXXXXXXXXXXXX" },
    "cuerpoDocumento": [ ],
    "resumen": { },
    "documentoRelacionado": [ ],
    "ventaTercero": null,
    "apendice": [],
    "firmaElectronica": "<JWS>",
    "selloRecibido": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"
  }
}
```

> `totalIva` del resumen y de los ítems se informa únicamente cuando el documento ajustado es un CR (`07`); al ajustar un CCF el IVA se informa solo como tributo `20` del resumen y `totalIva` va en `0`.

---

## Respuestas de error

Los errores usan la estructura descrita en [Formato de respuestas](./README.md#respuesta-de-error). El campo `error.code` se devuelve en mayúsculas.

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `REQUIREDFIELD` y otros códigos de formato | Campo obligatorio ausente o con formato inválido (por ejemplo `related_docs` vacío, NIT inválido, descuento negativo) |
| `400` | `VALIDATIONFAILED` | Falló una regla propia de la Nota de Crédito; `error.details` lista cada regla: `MissingItemRelatedDoc`, `InvalidItemRelatedDoc`, `InvalidRelatedDocTypeForCreditNote`, `MixedSalesTypesNotAllowed`, `GlobalDiscountNotAllowedInAdjustment`, `IncomeRetentionNotAllowed`, `InvalidTaxCalculation`, `InvalidSubTotalCalculation`, `InvalidTotalOperation`, `InvalidTotalToPayCalculation`, `InvalidPerceptionAmount`, `MissingIVAForTaxedAmount`, etc. |
| `400` | `MANY_ERRORS` | Falló una regla general de DTE; `error.details` incluye, por ejemplo, `RelatedDocOutOfPeriod`, `MixedDocumentTypesNotAllowed`, `ExceededRelatedDocsLimit`, `InvalidRelatedDocDate` |
| `400` | `FAILEDTOGETDTE` | El documento relacionado no existe en la sucursal |
| `400` | `RELATEDDOCUMENTNOTRECEIVED` | El documento relacionado no está en estado `RECEIVED` |
| `400` | `NOTMATCHINGRECEIVERNIT` | El NIT del receptor no coincide con el del documento relacionado |
| `400` | `INVALIDCREDITNOTETRANSACTION` | Los totales exceden el saldo restante del documento relacionado |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó el documento |

---

## Ejemplo completo

### Request

```json
{
  "items": [
    {
      "type": 1,
      "description": "Devolución - 201-210-501-20",
      "quantity": 1,
      "unit_price": 13.2743363,
      "unit_measure": 59,
      "discount": 0,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 4.4247788,
      "taxes": ["20"],
      "code": "CODIGO-PRODUCTO",
      "related_doc": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
    }
  ],
  "receiver": {
    "nit": "XXXXXXXXXXXXXX",
    "nrc": "XXXXXXX",
    "name": "Empresa ABC",
    "commercial_name": "Comercial XYZ",
    "activity_code": "14108",
    "activity_description": "Maquilado de prendas de vestir, accesorios y otros",
    "address": {
      "department": "09",
      "municipality": "10",
      "complement": "Calle Principal, Local 1"
    },
    "phone": "XXXXXXXX",
    "email": "correo@ejemplo.com"
  },
  "summary": {
    "total_non_subject": 0,
    "total_exempt": 0,
    "total_taxed": 4.42,
    "sub_total_sales": 4.42,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "taxed_discount": 0,
    "discount_percentage": 0,
    "total_discount": 0,
    "taxes": [
      { "code": "20", "description": "Impuesto al Valor Agregado 13%", "value": 0.57 }
    ],
    "sub_total": 4.42,
    "iva_perception": 0,
    "iva_retention": 0,
    "income_retention": 0,
    "total_operation": 4.99,
    "total_non_taxed": 0,
    "total_to_pay": 4.99,
    "operation_condition": 1
  },
  "related_docs": [
    {
      "document_type": "03",
      "generation_type": 2,
      "document_number": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "emission_date": "2026-01-08"
    }
  ]
}
```

**Verificación de cálculos:**
- `total_taxed = Σ taxed_sale = 4.4247788 ≈ 4.42` (tolerancia ±0.01)
- `sub_total_sales = sub_total = 4.42`
- `IVA = 4.42 × 0.13 = 0.5746 ≈ 0.57` (tolerancia ±0.01)
- `total_operation = 4.42 + 0.57 = 4.99`
- `total_to_pay = 4.99 + 0 - 0 + 0 = 4.99`
