# Factura de Sujeto Excluido (FSE)

## Descripción

La **Factura de Sujeto Excluido** (código DTE `14`) se utiliza para documentar compras realizadas a personas que **no son contribuyentes del IVA** (sujetos excluidos). A diferencia de otros DTE, este documento registra el **monto de compra** (no venta gravada) y **no aplica IVA**: no tiene ventas gravadas, tributos ni retención de IVA (Normativa de Cumplimiento DTE v2.0, Anexo IV, campo 95 para el monto de compras; Anexo VI para la eliminación de la retención de IVA en la FSE).

**Soporta contingencia:** Sí (ver [Contingencia](#contingencia))

---

## Endpoint

```
POST /api/v1/dte/fse
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

> Los datos que el servicio genera por sí mismo **no se envían**: número de control, código de generación, fecha y hora de emisión, ambiente, versión y datos del emisor. La fecha de emisión es siempre la del momento de la petición. Los campos desconocidos del request (por ejemplo `extension`, que la FSE no admite) se ignoran sin generar error.

> **Normativa:** los campos de la sección Identificación (versión, ambiente, tipo de documento, código de generación, número de control, modelo de facturación, tipo de transmisión, fecha, hora y moneda) corresponden a los campos 1 a 12 del Anexo IV de la Normativa de Cumplimiento DTE v2.0; la versión de la estructura de la FSE es 2 (campo 1). El código de generación y el número de control se rigen además por las reglas 7.1.1 y 7.1.2, y los datos del emisor corresponden a los campos 18 a 33 del mismo Anexo.

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `FSEItem[]` | Sí | Lista de ítems (1–2000) |
| `excluded_subject` | `FSEReceiver` | Sí | Datos del sujeto excluido (receptor). **Nótese que el nombre del campo es `excluded_subject`, no `receiver`** |
| `summary` | `FSESummary` | Sí | Resumen financiero |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |

La FSE no admite `third_party_sale`, `related_docs`, `other_docs` ni `extension`.

---

### `items[]` — Ítems

El número de ítem lo asigna el servicio según la posición en el arreglo (`numItem` = 1, 2, 3...); el campo `number` del request se ignora.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto (el tipo `4` se rechaza en la FSE con `InvalidFSEItemType`) |
| `description` | string | Sí | Descripción (1–1500 caracteres) |
| `quantity` | float | Sí | Cantidad (> 0, hasta `99999999999.99`) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99) |
| `unit_price` | float | Sí | Precio unitario (> 0, hasta 8 decimales) |
| `discount` | float | No | **Monto** de descuento del ítem (no porcentaje). El servicio acepta cualquier monto de `0` a `99999999999.99`; un valor negativo se rechaza con `InvalidAmount` |
| `purchase` | float | Sí | Monto de compra del ítem (> 0). Ver regla |
| `code` | string | No | Código del ítem (1–24 caracteres efectivos; el mensaje de error indica "1 a 25") |

`taxes`, `tax_code` y `related_doc` no forman parte de la FSE: si se envían se ignoran.

> **Normativa:** los campos del ítem corresponden a la sección Cuerpo del Anexo IV de la Normativa de Cumplimiento DTE v2.0: número de ítem (campo 74; máximo 2000 en la regla 2), `type` (campo 75), `quantity` (campo 82), `code` (campo 83), `unit_measure` (campo 85, CAT-014), `description` (campo 86, hasta 1,500 caracteres), `unit_price` (campo 87; sin IVA, regla 2), `discount` (campo 89; no puede ser mayor al monto del ítem, regla 3) y `purchase` (campo 95: `(precio × cantidad) - descuento`, regla 2, con la holgura de la regla 3). Los campos del cuerpo se redondean según la regla 7.2 literal a).

> **Regla:** `purchase = (quantity × unit_price) - discount` (tolerancia ±0.01, `InvalidItemSaleAmount`) y `discount ≤ quantity × unit_price` (`ExcessiveItemDiscount`). Ver [calculations.md](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95) (sección 2.1; Anexo IV, campo 95, reglas 2 y 3).

---

### `excluded_subject` — Sujeto excluido

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | `"36"` NIT, `"13"` DUI, `"02"` Carnet de residente, `"03"` Pasaporte, `"37"` Otro |
| `document_number` | string | Sí | Número de documento. NIT y DUI se aceptan **con o sin guiones** (se eliminan). NIT: 9 o 14 dígitos. DUI: exactamente 9 dígitos. Otros tipos: hasta 20 caracteres |
| `name` | string | Sí | Nombre del sujeto excluido |
| `address` | `Address` | Sí | Dirección (la FSE no admite `null`) |
| `activity_code` | string | No | Código de actividad económica (2–6 dígitos) |
| `activity_description` | string | No | Descripción de actividad económica (máx. 150 caracteres) |
| `phone` | string | No | Teléfono (8–30 caracteres) |
| `email` | string | No | Correo electrónico válido, de 3 a 100 caracteres. Se verifica que el dominio tenga registro MX |

Los campos `nit`, `nrc` y `commercial_name` no forman parte del sujeto excluido y no se emiten en el documento.

> **Normativa:** en la FSE el receptor es el Sujeto Excluido (Anexo IV de la Normativa de Cumplimiento DTE v2.0, sección 4): `document_type` (campo 38, CAT-022), `document_number` (campo 39; hasta 20 dígitos, regla 1, y el DUI con 9 dígitos sin guiones, regla 10), `name` (campo 42), `address` (campos 47 a 50, que en la FSE no admiten `null`), `phone` (campo 56) y `email` (campo 57, que puede ser `null`, regla 2). El NIT (campo 40), el NRC (campo 41) y el nombre comercial (campo 45) no aplican a la FSE.

#### `address`

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí | Código de departamento (`01`–`14`) |
| `municipality` | string | Sí | Código de municipio válido para el departamento (CAT-013) |
| `district` | string | No | Código de distrito (`01`–`99`, dos dígitos). Si se omite o se envía vacío se usa `"01"` |
| `complement` | string | Sí | Dirección complementaria (1–200 caracteres) |

---

### `summary` — Resumen

Los montos del resumen se envían con **máximo 2 decimales**. Los campos `total_purchase`, `sub_total` y `total_to_pay` no pueden ser `0` (`RequiredField`).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_purchase` | float | Sí | Total de compra = suma de `purchase` de los ítems |
| `non_subject_discount` | float | Sí | Descuento global (se emite como `descu`) |
| `total_discount` | float | Sí | Total de descuentos = suma de `discount` de los ítems + `non_subject_discount` |
| `sub_total` | float | Sí | Subtotal = `total_purchase - non_subject_discount` |
| `income_retention` | float | Sí | Retención de renta (≥ 0) |
| `iva_retention` | float | Sí | **Debe ser `0`**: la retención de IVA no aplica a FSE (`IVARetentionNotAllowedInFSE`) |
| `total_to_pay` | float | Sí | Total a pagar = `sub_total - income_retention` |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito, `3`=Otro |
| `payment_types` | `Payment[]` | Sí | Formas de pago |
| `total_in_words` | string | No | Total en letras. Si se omite, el servicio lo genera a partir de `total_to_pay` |
| `observations` | string | No | Observaciones |

"Requerido" en esta tabla indica que el campo forma parte del contrato; si un monto no aplica, se envía `0`.

> **Normativa:** `total_purchase` (campo 127: suma de los campos de compras, mayor a `0`), `non_subject_discount` (campo 133: descuento global al total de operaciones, que no puede ser mayor al total, regla 2), `total_discount` (campo 135, regla 5), `sub_total` (campo 139, regla 2), `income_retention` (campo 147), `total_to_pay` (campo 153, regla 6), `total_in_words` (campo 156), `operation_condition` (campo 159) y `observations` (campo 168). El Anexo VI (versión 2.0, Anexos II y IV) elimina el campo 'IVA Retenido' para la FSE. Los montos del resumen siguen las reglas 7.2 literal b) y 7.3.

El resumen de la FSE **no usa** los campos `total_non_subject`, `total_exempt`, `total_taxed`, `sub_total_sales`, `exempt_discount`, `discount_percentage` ni `total_non_taxed`; si se envían se ignoran (en particular, `total_taxed` no se lee, por lo que la FSE nunca tiene venta gravada).

---

### `summary.payment_types[]` — Formas de pago

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `code` | string | Condicional | Código de forma de pago: `"01"` a `"14"` o `"99"` (CAT-017). Obligatorio con condición `1` o `3`; con condición `2` debe ir `null` |
| `amount` | float | Sí | Monto. Con condición `2` debe enviarse `0.00` |
| `period` | integer | Condicional | Número de días, meses o años del plazo, mayor que cero. Obligatorio con condición `2` |
| `term` | string | Condicional | Plazo (`"01"`, `"02"` o `"03"`, CAT-018). Obligatorio con condición `2` |
| `reference` | string | No | Referencia del pago. Con condición `2` debe ir `null` |

**Reglas con condición `2` (crédito)**, por cada forma de pago:
- `code` debe ir `null` (`CreditPaymentCodeNotAllowed`).
- `amount` debe ser exactamente `0.00` (`InvalidCreditPaymentAmount`). No se compara la suma con `total_to_pay`.
- `reference` debe ir `null` (`CreditPaymentReferenceNotAllowed`).
- `term` es obligatorio (`CreditPaymentTermRequired`).
- `period` es obligatorio y mayor que cero (`CreditPaymentPeriodRequired`).
- Se requiere al menos una forma de pago: una lista vacía con `total_to_pay` mayor que 0 se rechaza con `InvalidPaymentTotal`.

Ejemplo de forma de pago a crédito a 30 días:

```json
"payment_types": [
  { "code": null, "amount": 0.00, "reference": null, "term": "01", "period": 30 }
]
```

En el JSON transmitido a Hacienda la forma de pago queda con `codigo` y `referencia` en `null`, `montoPago` en `0.00`, y `plazo` y `periodo` informados.

**Reglas con condición `1` (contado) o `3` (otro):**
- `code` es obligatorio en cada forma de pago (`PaymentCodeRequired`).
- La suma de `amount` debe coincidir con `total_to_pay` (tolerancia ±0.01, `InvalidPaymentTotal`).
- Con condición `3` se admiten varias formas de pago con monto mayor que 0; la parte no cobrada se informa con el código `13` (cuentas por pagar del receptor). Ejemplo: `[{"code": "01", "amount": 40.00}, {"code": "13", "amount": 60.00}]` para un `total_to_pay` de `100.00`.

**Total a pagar `0.00`:** la condición debe ser `1` (`NoPaymentDueRequiresCashCondition`). En la FSE no se exige el código `99` (esa exigencia aplica a FE, CCF y FEXE).
- Condición `1`: si ningún pago usa el código `"01"`, los pagos no deben informar `period` ni `term` (`InvalidPaymentTermsOF`).

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

Las fórmulas completas y sus tolerancias están en [calculations.md](./calculations.md): venta por ítem ([2.1](./calculations.md#21-venta-por-ítem-anexo-iv-campos-90-a-92-y-95)), totales de operaciones ([3.2](./calculations.md#32-totales-de-operaciones-campos-123-a-125-y-127)), descuentos globales ([3.4](./calculations.md#34-descuentos-globales-campos-130-a-135)), subtotal ([3.5](./calculations.md#35-subtotal-campo-139)), total a pagar ([3.10](./calculations.md#310-total-a-pagar-campo-153)) y pagos ([3.11](./calculations.md#311-pagos)). El redondeo (regla 7.2) y la holgura de ±0.01 (regla 7.3, Cuadros 1, 2 y 3) de la Normativa de Cumplimiento DTE v2.0 se describen en las secciones [1.1](./calculations.md#11-decimales-y-redondeo) y [1.2](./calculations.md#12-holgura). Resumen de las reglas que se validan (todas con tolerancia ±0.01):

- `purchase` de cada ítem = `(quantity × unit_price) - discount` (Anexo IV, campo 95).
- `total_purchase` = suma de `purchase` de los ítems (campo 127).
- `sub_total` = `total_purchase - non_subject_discount` (campo 139, regla 2).
- `total_discount` = suma de `discount` de los ítems + `non_subject_discount` (campo 135, regla 5).
- `total_to_pay` = `sub_total - income_retention` (campo 153, regla 6).
- La suma de pagos debe coincidir con `total_to_pay` (campo 161, regla 4).

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Ítems | Mínimo 1, máximo 2000 (Anexo IV, campo 74, regla 2) |
| `purchase`, `quantity`, `unit_price` | Deben ser > 0 por ítem |
| Descripción del ítem | 1–1500 caracteres (Anexo IV, campo 86) |
| Unidad de medida | 1–99 |
| Sujeto excluido | `document_type`, `document_number`, `name` y `address` obligatorios |
| `iva_retention` | Debe ser `0` (también se rechaza si es negativo; eliminado por el Anexo VI) |
| `income_retention` | Debe ser ≥ 0 |
| Montos | Entre `0` y `99999999999.99` |
| Textos | `activity_description` hasta 150 caracteres |

---

## Respuesta exitosa

**HTTP 201 Created**

La emisión normal devuelve en `data` el **Archivo DTE**: la estructura del documento tal como fue firmada y transmitida, más los atributos `firmaElectronica` (JWS firmado) y `selloRecibido` (sello de recepción de Hacienda). El sello de recepción se repite en `reception_stamp` en la raíz de la respuesta.

> **Normativa:** el Archivo DTE se rige por la regla 4 (Entrega) del Anexo I y el Cuadro 11 de la Normativa de Cumplimiento DTE v2.0: contiene la estructura de datos íntegra, el documento firmado y el Sello de Recepción (`selloRecibido`, Anexo I, regla 3.4). La regla 11.1 exige entregar al receptor el DTE íntegro, con su Representación Gráfica y el Sello de Recepción.

```json
{
  "success": true,
  "reception_stamp": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=9D2F0B55-3C4E-4A6B-8F9C-2B3C4D5E6F70&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 2,
      "ambiente": "01",
      "tipoDte": "14",
      "numeroControl": "DTE-14-N0010001-000000000000001",
      "codigoGeneracion": "9D2F0B55-3C4E-4A6B-8F9C-2B3C4D5E6F70",
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
      "codEstable": "N001",
      "codPuntoVenta": "0001"
    },
    "receptor": {
      "tipoDocumento": "13",
      "numDocumento": "012345678",
      "nombre": "Juan Carlos Pérez",
      "codActividad": "46900",
      "descActividad": "Servicios profesionales",
      "direccion": { "departamento": "09", "municipio": "10", "distrito": "01", "complemento": "Calle Principal, Local 1" },
      "telefono": null,
      "correo": "juan@example.com"
    },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoItem": 1,
        "cantidad": 2,
        "codigo": "CODIGO-PRODUCTO",
        "uniMedida": 59,
        "descripcion": "Zapatilla AF1 Blanco Unisex",
        "precioUni": 65,
        "montoDescu": 10,
        "compra": 120
      }
    ],
    "resumen": {
      "totalCompra": 120,
      "descu": 0,
      "totalDescu": 10,
      "subTotal": 120,
      "reteRenta": 0,
      "totalPagar": 120,
      "totalLetras": "CIENTO VEINTE 00/100",
      "condicionOperacion": 1,
      "pagos": [
        { "codigo": "01", "montoPago": 120, "referencia": null, "plazo": null, "periodo": null }
      ],
      "observaciones": null
    },
    "apendice": null,
    "firmaElectronica": "eyJhbGciOiJSUzUxMiJ9.eyJpZGVudGlmaWNhY2lvbiI6...<firma>",
    "selloRecibido": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C"
  }
}
```

Notas sobre la estructura:

- El receptor se emite en `receptor` (no en `sujetoExcluido`). `numDocumento` se emite sin guiones.
- `resumen` usa los nombres propios de la FSE: `totalCompra`, `descu`, `totalDescu`, `subTotal`, `reteRenta`, `totalPagar`. No incluye `ivaRete` ni `ivaRete1`, tributos ni campos de venta gravada.
- Los ítems usan `compra` en lugar de las ventas por tipo, y no incluyen `tributos`, `psv` ni `noGravado`.
- `emisor` no incluye `nombreComercial` ni `tipoEstablecimiento`; solo `codEstable` y `codPuntoVenta` como códigos de establecimiento y punto de venta.
- Los campos opcionales sin valor se emiten como `null`. `apendice` y `pagos` son `null` cuando no se enviaron.
- El orden de las llaves dentro de `data` no es significativo. Para entregar el documento al receptor se debe conservar `data` íntegro, sin alterarlo.
- Si el Archivo DTE no pudiera armarse (firma no disponible), `data` contiene únicamente la estructura del documento, sin `firmaElectronica` ni `selloRecibido`.

---

## Contingencia

La FSE soporta contingencia. Si la transmisión a Hacienda falla por causas no atribuibles al documento (red, indisponibilidad del servicio, timeout, etc.), el documento se almacena en contingencia a la espera de retransmisión y se responde igualmente **HTTP 201**, pero:

- `reception_stamp` es `null`.
- `data` es la estructura del documento **sin** `firmaElectronica` ni `selloRecibido` (no es un Archivo DTE).
- `data.identificacion` cambia: `tipoModelo` pasa a `2` (diferido), `tipoOperacion` a `2` (contingencia), y se informan `tipoContingencia` (1–5, según el catálogo de contingencia) y `motivoContin`.

No se activa contingencia cuando el documento falla validación o Hacienda lo rechaza (`RECHAZADO`): en esos casos se devuelve el error correspondiente y el número de control se libera. Las condiciones de activación y los tipos de contingencia se describen en el [README](./README.md#contingencia).

> **Normativa:** la contingencia se rige por la regla 13.2 y el Cuadro 7 de la Normativa de Cumplimiento DTE v2.0, que comprende a la FSE entre los documentos admitidos. El Anexo IV (campos 6 a 9) define el modelo de facturación, el tipo de transmisión, el tipo de contingencia (CAT-005, códigos 1 a 5) y el motivo de contingencia, requerido cuando el tipo es `5` (campo 9, regla 1). Un documento en contingencia no cuenta con Sello de Recepción hasta su transmisión (regla 11.2).

---

## Respuestas de error

Los errores de validación devuelven `HTTP 400` con la estructura `error.code`, `error.message` y `error.details`. El valor de `code` depende de la etapa en la que se detecta el problema:

| Etapa | `code` | `message` | `details` |
|-------|--------|-----------|-----------|
| Lectura del request (JSON inválido) | `BAD_REQUEST` | `Invalid request format` | vacío |
| Estructura mínima (falta `items`, `summary` o `excluded_subject`, `document_type`, `document_number`) | Tipo del error en mayúsculas (`REQUIREDFIELD`, ...) | Mensaje del campo | `No hay más detalles disponibles` |
| Mapeo de una sección (sujeto excluido, ítems, resumen) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDFIELD`, ...) o `ERRORMAPPING` | `Error al mapear la sección FSE->...` | Mensaje del campo |
| Reglas específicas de la FSE | `VALIDATIONFAILED` | `Validaciones no superadas, revise los detalles a continuación` | Lista de mensajes de cada regla incumplida |
| Reglas comunes a los DTE | `MANY_ERRORS` (o el tipo del error si es uno solo) | `No se pudo crear el DTE, revise los detalles a continuación` | Lista de mensajes |
| Rechazo de Hacienda | `HACIENDA_<codigo>` | Descripción de Hacienda | Estado, fecha de proceso y observaciones |
| Error interno | `SYSTEM_ERROR` (`HTTP 500`) | `Ha ocurrido un error inesperado` | vacío |

Con la depuración activa del servidor, los mensajes de validación se prefijan con la clave `[Clave]`. Las reglas específicas de la FSE se evalúan primero; si pasan, se evalúan las reglas comunes.

### Reglas específicas de la FSE (`VALIDATIONFAILED`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `FSEItemInvalidEmpty`, `FSEItemInvalidPurchase`, `FSEItemInvalidDescription`, `FSEItemInvalidQuantity`, `FSEItemInvalidUnitPrice` | Sin ítems, o un ítem con `purchase`, `quantity` o `unit_price` ≤ 0, o descripción vacía |
| `InvalidFSEItemType` | Ítem de tipo `4` (impuesto), no permitido en la FSE |
| `ExcessiveItemDiscount` | `discount` mayor que `quantity × unit_price` |
| `InvalidItemSaleAmount` | `purchase` no coincide con `(quantity × unit_price) - discount` |
| `InvalidTotalPurchase` | `total_purchase` no coincide con la suma de `purchase` |
| `InvalidSubTotal` | `sub_total` no coincide con `total_purchase - non_subject_discount` |
| `InvalidTotalDiscount` | `total_discount` no coincide con la suma de descuentos |
| `FSETaxInvalidIVARetention`, `FSETaxInvalidIncomeRetention` | Una retención es negativa |
| `IVARetentionNotAllowedInFSE` | `iva_retention ≠ 0` |
| `FSETaxInvalidTotalToPay` | `total_to_pay` no coincide con `sub_total - income_retention` |
| `FSEReceiverRequiredDocumentType`, `FSEReceiverRequiredDocumentNumber`, `FSEReceiverRequiredName`, `FSEReceiverRequiredAddress` | Faltan datos obligatorios del sujeto excluido |
| `FSEReceiverInvalidDocumentType`, `FSEReceiverInvalidNIT`, `FSEReceiverInvalidDUI`, `FSEReceiverInvalidDocumentLength` | Tipo o número de documento inválido |
| `FSEReceiverInvalidActivityCode`, `FSEReceiverInvalidActivityDescription` | Código de actividad inválido o descripción mayor a 150 caracteres |

### Reglas comunes (`MANY_ERRORS`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `InvalidNITFormat`, `InvalidDUIFormat` | `document_number` con formato inválido para NIT (9 o 14 dígitos) o DUI (9 dígitos) |
| `ExceededItemsLimit`, `InvalidItemNumber`, `InvalidItemType` | Más de 2000 ítems, número o tipo de ítem inválido |
| `InvalidLength` | Descripción de ítem fuera de 1–1500 o código de ítem excedido |
| `InvalidNumberRange` | `unit_measure` fuera de 1–99 |
| `InvalidPaymentTermsOF` | Condición `1` sin pago en efectivo y con `period` o `term` informados |
| `InvalidPaymentTotal` | La suma de pagos no coincide con `total_to_pay` |
| `InvalidCreditPaymentAmount` | Con condición `2` algún `amount` de pago es distinto de `0.00` |
| `CreditPaymentCodeNotAllowed` | Con condición `2` una forma de pago trae `code` |
| `CreditPaymentReferenceNotAllowed` | Con condición `2` una forma de pago trae `reference` |
| `CreditPaymentTermRequired` | Con condición `2` una forma de pago no trae `term` |
| `CreditPaymentPeriodRequired` | Con condición `2` una forma de pago no trae `period` o es menor o igual a cero |
| `PaymentCodeRequired` | Con condición `1` o `3` una forma de pago no trae `code` |
| `NoPaymentDueRequiresCashCondition` | Total a pagar `0.00` con condición distinta de `1` |

---

## Ejemplo completo

### Request

```json
{
  "excluded_subject": {
    "document_type": "13",
    "document_number": "01234567-8",
    "name": "Juan Carlos Pérez",
    "activity_code": "46900",
    "activity_description": "Servicios profesionales",
    "address": {
      "department": "09",
      "municipality": "10",
      "complement": "Calle Principal, Local 1"
    },
    "phone": null,
    "email": "juan@example.com"
  },
  "items": [
    {
      "type": 1,
      "description": "Zapatilla AF1 Blanco Unisex",
      "quantity": 2,
      "unit_measure": 59,
      "unit_price": 65,
      "discount": 10,
      "purchase": 120,
      "code": "CODIGO-PRODUCTO"
    }
  ],
  "summary": {
    "total_purchase": 120,
    "non_subject_discount": 0,
    "total_discount": 10,
    "sub_total": 120,
    "iva_retention": 0,
    "income_retention": 0,
    "total_to_pay": 120,
    "operation_condition": 1,
    "payment_types": [
      {
        "code": "01",
        "amount": 120
      }
    ]
  }
}
```

**Verificación de cálculos:**
- `purchase = (quantity × unit_price) - discount = 2 × 65 - 10 = 120`
- `total_purchase = Σ item.purchase = 120`
- `sub_total = total_purchase - non_subject_discount = 120 - 0 = 120`
- `total_discount = Σ item.discount + non_subject_discount = 10 + 0 = 10`
- `total_to_pay = sub_total - income_retention = 120 - 0 = 120`
