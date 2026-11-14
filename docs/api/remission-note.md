# Nota de Remisión Electrónica

## Descripción

La **Nota de Remisión Electrónica** (código DTE `04`) se utiliza para documentar el **traslado de bienes** sin que implique una transacción de compraventa inmediata. Es utilizada principalmente en envíos de mercancía entre establecimientos, entregas a consignación, o traslados internos. El título a que se remiten los bienes corresponde al campo 54 del Anexo IV de la Normativa de Cumplimiento DTE v2.0 (CAT-025).

**Soporta contingencia:** Sí (ver [Contingencia](#contingencia))

---

## Endpoint

```
POST /api/v1/dte/remissionnote
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

> Los datos que el servicio genera por sí mismo **no se envían**: número de control, código de generación, fecha y hora de emisión, ambiente, versión y datos del emisor. La fecha de emisión es siempre la del momento de la petición. Los campos desconocidos del request (por ejemplo `extension`, que la NRE no admite) se ignoran sin generar error.

> **Normativa:** los campos de la sección Identificación (versión, ambiente, tipo de documento, código de generación, número de control, modelo de facturación, tipo de transmisión, fecha, hora y moneda) corresponden a los campos 1 a 12 del Anexo IV de la Normativa de Cumplimiento DTE v2.0; la versión de la estructura de la NRE es 4 (campo 1). El código de generación y el número de control se rigen además por las reglas 7.1.1 y 7.1.2, y los datos del emisor corresponden a los campos 18 a 33 del mismo Anexo.

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `receiver` | `RemissionReceiver` | Sí | Destinatario de los bienes |
| `items` | `RemissionItem[]` | Sí | Ítems / bienes trasladados (1–2000) |
| `summary` | `RemissionSummary` | Sí | Resumen del traslado |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados (máx. 50); solo FE `01`, CCF `03` o FEXE `11` |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |

---

### `receiver` — Receptor / Destinatario

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `real_state` | string | Sí | Código del título del bien (CAT-025, exactamente 2 caracteres; ver tabla abajo) |
| `name` | string | Sí | Nombre o razón social del destinatario |
| `document_type` | string | Sí | `"36"` NIT, `"13"` DUI, `"02"` Carnet de residente, `"03"` Pasaporte, `"37"` Otro |
| `document_number` | string | Sí | Número de documento. NIT y DUI se aceptan **con o sin guiones** (se eliminan). NIT: 9 o 14 dígitos. DUI: 9 dígitos. Otros tipos: 3–20 caracteres |
| `address` | `Address` | Sí | Dirección de destino |
| `email` | string | Sí | Correo electrónico válido, de 3 a 100 caracteres. Se verifica que el dominio tenga registro MX |
| `nit` | string | Condicional | NIT del destinatario. **Obligatorio si `document_type = "36"`** (9 o 14 dígitos). No se emite en el documento: el receptor se identifica con `document_type` y `document_number` |
| `activity_code` | string | Condicional | Código de actividad económica (2–6 dígitos). **Obligatorio si `document_type = "36"`** |
| `activity_description` | string | Condicional | Descripción de actividad económica. **Obligatoria si `document_type = "36"`** |
| `nrc` | string | No | NRC (1–8 dígitos). No se emite si `document_type = "13"` (DUI) |
| `commercial_name` | string | No | Nombre comercial |
| `phone` | string | No | Teléfono (8–30 caracteres) |

#### `address` (destinatario)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí | Código de departamento (`01`–`14`) |
| `municipality` | string | Sí | Código de municipio válido para el departamento (CAT-013) |
| `district` | string | No | Código de distrito (`01`–`99`, dos dígitos). Si se omite o se envía vacío se usa `"01"` |
| `complement` | string | Sí | Dirección complementaria (1–200 caracteres) |

#### Valores válidos para `real_state`

| Código | Descripción |
|--------|-------------|
| `"01"` | Venta |
| `"02"` | Consignación |
| `"03"` | Exhibición |
| `"04"` | Traslado interno |
| `"05"` | Préstamo |
| `"99"` | Otros |

> **Normativa:** la sección Receptor corresponde a los campos 38 a 57 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `real_state` (campo 54, CAT-025), `document_type` (campo 38, CAT-022), `document_number` (campo 39), `nrc` (campo 41), `name` (campo 42), `activity_code` (campo 43), `activity_description` (campo 44), `commercial_name` (campo 45), `address` (campos 47 a 50), `phone` (campo 56) y `email` (campo 57, que en la NRE no puede ser `null`, regla 2). Cuando el documento tiene un documento relacionado, el número de documento del receptor debe corresponder al del documento relacionado (campo 39, regla 9). El NIT (campo 40) no aplica a la NRE.

---

### `items[]` — Ítems / Bienes

A diferencia de FE, CCF y FSE, en la NRE el número de ítem **sí se envía** y debe ser secuencial a partir de `1` en el orden del arreglo (`InvalidSequence`).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (1–2000, secuencial desde 1) |
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción del bien (1–1500 caracteres) |
| `quantity` | float | Sí | Cantidad (> 0) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99) |
| `unit_price` | float | Sí | Precio unitario (≥ 0) |
| `discount` | float | No | **Monto** de descuento (≥ 0). El servicio acepta cualquier monto de `0` a `99999999999.99`; un valor negativo se rechaza con `InvalidAmount` |
| `non_subject_sale` | float | Condicional | Monto no sujeto (≥ 0) |
| `exempt_sale` | float | Condicional | Monto exento (≥ 0) |
| `taxed_sale` | float | Condicional | Monto gravado (≥ 0) |
| `code` | string | No | Código del bien (1–24 caracteres efectivos; el mensaje de error indica "1 a 25") |
| `tributes` | string[] | No | Códigos de tributo aplicados al ítem (se emiten como `tributos`) |
| `document_number` | string | Condicional | `document_number` del documento relacionado al que pertenece el ítem (se emite como `numeroDocumento`). Obligatorio en todos los ítems si hay `related_docs`; debe omitirse si no los hay |

Al menos uno de `non_subject_sale`, `exempt_sale` o `taxed_sale` debe ser mayor a `0` (`AtLeastOneSaleTypeRequired`).

El campo `tax_code` de un ítem se acepta pero **no se emite**: `codTributo` sale siempre `null`. El `document_number` del ítem se emite como `numeroDocumento`. El servicio no valida que la venta del ítem coincida con `unit_price × quantity`.

> **Normativa:** los campos del ítem corresponden a la sección Cuerpo del Anexo IV de la Normativa de Cumplimiento DTE v2.0: número de ítem (campo 74; enteros consecutivos, regla 1, y máximo 2000, regla 2), `type` (campo 75), `quantity` (campo 82), `code` (campo 83), `unit_measure` (campo 85, CAT-014), `description` (campo 86), `unit_price` (campo 87; sin IVA, regla 2), `discount` (campo 89), `non_subject_sale`, `exempt_sale` y `taxed_sale` (campos 90, 91 y 92) y `tributes` (campo 96, que en la NRE depende del tipo de documento relacionado: reglas 20 y 21). Los campos del cuerpo se redondean según la regla 7.2 literal a) y se comparan con la holgura de la regla 7.3.

---

### `summary` — Resumen

Los montos de `tributes` se envían con **2 decimales**. Los demás montos son ≥ 0.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `non_subject_total` | float | No | Total no sujeto. Si se envía, debe coincidir con la suma de `non_subject_sale` (±0.01) |
| `exempt_total` | float | No | Total exento. Si se envía, debe coincidir con la suma de `exempt_sale` (±0.01) |
| `taxed_total` | float | No | Total gravado. Si se envía, debe coincidir con la suma de `taxed_sale` (±0.01) |
| `subtotal_sales` | float | Sí | Suma de operaciones |
| `non_subject_discount` | float | No | Descuento global sobre no sujeto |
| `exempt_discount` | float | No | Descuento global sobre exento |
| `taxed_discount` | float | No | Descuento global sobre gravado |
| `discount_percent` | float | No | Porcentaje de descuento (0–100) |
| `total_discount` | float | No | Total de descuentos |
| `tributes` | `Tax[]` | No | Tributos del resumen (`code`, `description`, `value`) |
| `subtotal` | float | Sí | Subtotal |
| `total_amount` | float | Sí | Monto total de la operación (`subtotal` + tributos) |
| `amount_in_words` | string | No | Total en letras. Si se omite, el servicio lo genera a partir de `total_amount` |
| `observations` | string | No | Observaciones (máx. 3000 caracteres) |
| `payments` | `Payment[]` | No | Formas de pago (`code` requerido; `amount` puede ser `0`). Se validan en formato pero **no se emiten** en el documento |

El servicio no exige explícitamente `subtotal_sales`, `subtotal` ni `total_amount`: si se omiten se toman como `0` (y la validación de `subtotal`/`total_amount` no se ejecuta). Para un documento válido ante Hacienda deben enviarse con su valor.

> **Normativa:** los campos del resumen corresponden a la sección Resumen del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `non_subject_total`, `exempt_total` y `taxed_total` (campos 123 a 125), `subtotal_sales` (campo 129), `non_subject_discount`, `exempt_discount` y `taxed_discount` (campos 130 a 132), `discount_percent` (campo 134), `total_discount` (campo 135, regla 3), `tributes` (campos 136 a 138; el tributo `C3` lleva valor `0`, campo 138, regla 8), `subtotal` (campo 139), `total_amount` (campo 150, regla 2), `amount_in_words` (campo 156) y `observations` (campo 168). Las formas de pago (campos 160 a 164) no aplican a la NRE según la columna DTE del Anexo IV. Los montos del resumen siguen las reglas 7.2 literal b) y 7.3.

#### `summary.tributes[]`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `code` | string | Código de tributo (CAT-015): `"20"`, `"C3"`, `"59"`, `"71"`, `"D5"`. `"D1"` y `"C8"` se rechazan con `UnsupportedTaxCode` |
| `description` | string | Descripción del tributo (requerida) |
| `value` | float | Valor con 2 decimales. Debe ser distinto de `0` (salvo `C3`) |

---

### `related_docs[]` — Documentos relacionados

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de DTE relacionado: FE `01`, CCF `03` o FEXE `11` (otro tipo se rechaza con `InvalidRelatedDocDTEType`) |
| `generation_type` | integer | Sí | `1`=Físico (alfanumérico, 1–20 caracteres), `2`=Electrónico (UUID en mayúsculas) |
| `document_number` | string | Sí | Número del documento relacionado |
| `emission_date` | string | Condicional | Fecha de generación `YYYY-MM-DD`. **Obligatoria para documentos físicos** (`generation_type = 1`). Si se omite en uno electrónico se usa la fecha actual. No puede ser futura |

Máximo 50 documentos (`ExceededRelatedDocsLimit`) y todos del mismo tipo (`MixedDocumentTypesNotAllowed`).

**Vínculo con los ítems** (misma regla que FE y CCF; en la NRE el campo del ítem es `document_number`):
- Si hay `related_docs`, todos los ítems deben incluir `document_number` (`MissingItemRelatedDoc`) y este debe ser el `document_number` de alguno de los documentos; la comparación es exacta (`InvalidItemRelatedDoc`).
- Cada documento de `related_docs` debe aparecer en el `document_number` de al menos un ítem (`UnreferencedRelatedDoc`).
- Si no se envían `related_docs`, ningún ítem puede llevar `document_number` (`InvalidItemRelatedDoc`).

> **Normativa:** la sección Documentos relacionados corresponde a los campos 14 a 17 del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `document_type` (campo 14), `generation_type` (campo 15, CAT-007), `document_number` (campo 16) y `emission_date` (campo 17). Para la NRE solo se pueden relacionar los códigos `01` (FE), `03` (CCF) y `11` (FEXE) (campo 14, regla 6), hasta 50 documentos del mismo tipo (campo 14, regla 7; campo 16, regla 2), y no pueden relacionarse documentos electrónicos invalidados o rechazados (campo 16, regla 3). La fecha se informa en formato `YYYY-MM-DD` (campo 17, regla 1) y, si el documento relacionado es electrónico, se valida contra la fecha de generación de ese documento (campo 17, regla 3).

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

Las fórmulas completas y sus tolerancias están en [calculations.md](./calculations.md): totales de operaciones ([3.2](./calculations.md#32-totales-de-operaciones-campos-123-a-125-y-127)), suma de operaciones ([3.3](./calculations.md#33-suma-de-operaciones-campo-129)), descuentos globales ([3.4](./calculations.md#34-descuentos-globales-campos-130-a-135)), tributos del resumen ([3.6](./calculations.md#36-tributos-del-resumen-campos-136-a-138)) y monto total de la operación ([3.9](./calculations.md#39-monto-total-de-la-operación-campo-150)). El redondeo (regla 7.2) y la holgura de ±0.01 (regla 7.3, Cuadros 1, 2 y 3) de la Normativa de Cumplimiento DTE v2.0 se describen en las secciones [1.1](./calculations.md#11-decimales-y-redondeo) y [1.2](./calculations.md#12-holgura). Resumen de las reglas que se validan:

- `total_amount` = `subtotal` + suma de `tributes.value` (±0.01). Sin tributos, `subtotal` y `total_amount` deben ser iguales (Anexo IV, campo 150, regla 2).
- Si se envían `non_subject_total`, `exempt_total` o `taxed_total`, cada uno debe coincidir con la suma de los ítems (±0.01; Anexo IV, campos 123 a 125).
- `non_subject_total`, `exempt_total`, `taxed_total` y `total_amount` no pueden ser negativos.
- Las formas de pago **no** se validan contra `total_amount` (los campos 160 a 164 del Anexo IV no aplican a la NRE).

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| `real_state` | Obligatorio; `"01"`–`"05"` o `"99"` (exactamente 2 caracteres) |
| `receiver.name`, `document_type`, `document_number`, `email`, `address` | Obligatorios |
| `receiver.nit`, `activity_code`, `activity_description` | Obligatorios si `document_type = "36"` |
| Ítems | Mínimo 1, máximo 2000; numeración secuencial desde 1 (Anexo IV, campo 74, reglas 1 y 2) |
| Tipos de venta | Cada ítem debe tener al menos un tipo de venta > 0 |
| `quantity` | Debe ser > 0 |
| `type` | `1`, `2`, `3` o `4` |
| `unit_measure` | Entre 1 y 99 |
| `total_amount` | `subtotal` + tributos del resumen (±0.01) |
| Pagos | No se validan contra `total_amount` |
| `related_docs` | Máximo 50; solo tipos `01`, `03`, `11`; fecha obligatoria en documentos físicos; cada documento referenciado por al menos un ítem |

---

## Respuesta exitosa

**HTTP 201 Created**

La emisión normal devuelve en `data` el **Archivo DTE**: la estructura del documento tal como fue firmada y transmitida, más los atributos `firmaElectronica` (JWS firmado) y `selloRecibido` (sello de recepción de Hacienda). El sello de recepción se repite en `reception_stamp` en la raíz de la respuesta.

> **Normativa:** el Archivo DTE se rige por la regla 4 (Entrega) del Anexo I y el Cuadro 11 de la Normativa de Cumplimiento DTE v2.0: contiene la estructura de datos íntegra, el documento firmado y el Sello de Recepción (`selloRecibido`, Anexo I, regla 3.4). La regla 11.1 exige entregar al receptor el DTE íntegro, con su Representación Gráfica y el Sello de Recepción.

```json
{
  "success": true,
  "reception_stamp": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=5A6B7C8D-9E0F-4A1B-8C2D-3E4F5A6B7C8D&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 4,
      "ambiente": "01",
      "tipoDte": "04",
      "numeroControl": "DTE-04-N0010001-000000000000001",
      "codigoGeneracion": "5A6B7C8D-9E0F-4A1B-8C2D-3E4F5A6B7C8D",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2026-05-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "documentoRelacionado": null,
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
      "tipoDocumento": "36",
      "numDocumento": "06141804941035",
      "nrc": null,
      "nombre": "Sucursal Norte",
      "codActividad": "14108",
      "descActividad": "Maquilado de prendas de vestir",
      "nombreComercial": null,
      "direccion": { "departamento": "09", "municipio": "10", "distrito": "01", "complemento": "Calle Principal, Local 1" },
      "telefono": null,
      "correo": "sucursal@example.com",
      "bienTitulo": "04"
    },
    "ventaTercero": null,
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoItem": 1,
        "numeroDocumento": null,
        "codigo": "CODIGO-PRODUCTO",
        "codTributo": null,
        "descripcion": "Traslado - Zapatilla AF1 Blanco Unisex",
        "cantidad": 1,
        "uniMedida": 59,
        "precioUni": 65,
        "montoDescu": 0,
        "ventaNoSuj": 0,
        "ventaExenta": 0,
        "ventaGravada": 65,
        "tributos": null
      }
    ],
    "resumen": {
      "totalNoSuj": 0,
      "totalExenta": 0,
      "totalGravada": 65,
      "subTotalVentas": 65,
      "descuNoSuj": 0,
      "descuExenta": 0,
      "descuGravada": 0,
      "porcentajeDescuento": 0,
      "totalDescu": 0,
      "tributos": null,
      "subTotal": 65,
      "montoTotalOperacion": 65,
      "totalLetras": "SESENTA Y CINCO 00/100",
      "observaciones": null
    },
    "apendice": null,
    "firmaElectronica": "eyJhbGciOiJSUzUxMiJ9.eyJpZGVudGlmaWNhY2lvbiI6...<firma>",
    "selloRecibido": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C"
  }
}
```

Notas sobre la estructura:

- `receptor` se identifica con `tipoDocumento` y `numDocumento`; no incluye `nit` y agrega `bienTitulo`. `numDocumento` se emite sin guiones para NIT y DUI.
- `resumen` no incluye `condicionOperacion`, `totalPagar` ni `pagos`.
- Los campos opcionales sin valor se emiten como `null`. `documentoRelacionado`, `ventaTercero` y `apendice` solo traen contenido si se enviaron en el request.
- `ventaTercero` incluye `nit`, `nombre` y `codDomiciliado` (el valor de `fiscal_domicile`).
- `emisor` solo incluye `codEstable` y `codPuntoVenta` como códigos de establecimiento y punto de venta; no existen `tipoEstablecimiento`, `codEstableMH` ni `codPuntoVentaMH`.
- El orden de las llaves dentro de `data` no es significativo. Para entregar el documento al receptor se debe conservar `data` íntegro, sin alterarlo.
- Si el Archivo DTE no pudiera armarse (firma no disponible), `data` contiene únicamente la estructura del documento, sin `firmaElectronica` ni `selloRecibido`.

---

## Contingencia

La NRE soporta contingencia. Si la transmisión a Hacienda falla por causas no atribuibles al documento (red, indisponibilidad del servicio, timeout, etc.), el documento se almacena en contingencia a la espera de retransmisión y se responde igualmente **HTTP 201**, pero:

- `reception_stamp` es `null`.
- `data` es la estructura del documento **sin** `firmaElectronica` ni `selloRecibido` (no es un Archivo DTE).
- `data.identificacion` cambia: `tipoModelo` pasa a `2` (diferido), `tipoOperacion` a `2` (contingencia), y se informan `tipoContingencia` (1–5, según el catálogo de contingencia) y `motivoContin`.

No se activa contingencia cuando el documento falla validación o Hacienda lo rechaza (`RECHAZADO`): en esos casos se devuelve el error correspondiente y el número de control se libera. Las condiciones de activación y los tipos de contingencia se describen en el [README](./README.md#contingencia).

> **Normativa:** la contingencia se rige por la regla 13.2 y el Cuadro 7 de la Normativa de Cumplimiento DTE v2.0, que comprende a la NRE entre los documentos admitidos. El Anexo IV (campos 6 a 9) define el modelo de facturación, el tipo de transmisión, el tipo de contingencia (CAT-005, códigos 1 a 5) y el motivo de contingencia, requerido cuando el tipo es `5` (campo 9, regla 1). Un documento en contingencia no cuenta con Sello de Recepción hasta su transmisión (regla 11.2).

---

## Respuestas de error

Los errores de validación devuelven `HTTP 400` con la estructura `error.code`, `error.message` y `error.details`. El valor de `code` depende de la etapa en la que se detecta el problema:

| Etapa | `code` | `message` | `details` |
|-------|--------|-----------|-----------|
| Lectura del request (JSON inválido) | `BAD_REQUEST` | `Invalid request format` | vacío |
| Estructura mínima (falta `items`, `summary`, `receiver` o `real_state`; `related_docs` sin fecha o sin tipo) | Tipo del error en mayúsculas (`REQUIREDFIELD`, ...) | Mensaje del campo | `No hay más detalles disponibles` |
| Mapeo de una sección (receptor, ítems, resumen) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDVALUE`, ...) o `ERRORMAPPING` | `Error al mapear la sección RemissionNote->...` | Mensaje del campo |
| Reglas específicas de la NRE | `VALIDATIONFAILED` | `Validaciones no superadas, revise los detalles a continuación` | Lista de mensajes de cada regla incumplida |
| Reglas comunes a los DTE | `MANY_ERRORS` (o el tipo del error si es uno solo) | `No se pudo crear el DTE, revise los detalles a continuación` | Lista de mensajes |
| Rechazo de Hacienda | `HACIENDA_<codigo>` | Descripción de Hacienda | Estado, fecha de proceso y observaciones |
| Error interno | `SYSTEM_ERROR` (`HTTP 500`) | `Ha ocurrido un error inesperado` | vacío |

Con la depuración activa del servidor, los mensajes de validación se prefijan con la clave `[Clave]`. Las reglas específicas de la NRE se evalúan primero; si pasan, se evalúan las reglas comunes.

### Reglas específicas de la NRE (`VALIDATIONFAILED`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `AtLeastOneSaleTypeRequired` | Un ítem no tiene ningún tipo de venta > 0 |
| `InvalidSequence` | La numeración de ítems no es secuencial desde 1 |
| `InvalidTotalNonSubject`, `InvalidTotalExempt`, `InvalidTotalTaxed` | Un total del resumen no coincide con la suma de los ítems |
| `InvalidAmount` | Un total o el monto total es negativo |
| `InvalidTotalOperation` | `total_amount` no coincide con `subtotal` más tributos |
| `RequiredBienTitulo`, `InvalidBienTitulo` | `real_state` ausente o fuera de `01`–`05`, `99` |
| `RequiredField` | Falta `name`, `document_type`, `document_number`, `address`, `email` o `nit` (con tipo `36`) |
| `InvalidLength` | `real_state` con más de 2 caracteres |
| `MissingItemRelatedDoc` | Hay `related_docs` y un ítem no trae `document_number` |
| `UnreferencedRelatedDoc` | Un documento de `related_docs` no aparece en el `document_number` de ningún ítem |
| `InvalidItemRelatedDoc` | Un ítem referencia en `document_number` un documento que no está en `related_docs` |

### Reglas comunes (`MANY_ERRORS`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `InvalidNITFormat`, `InvalidDUIFormat` | `document_number` con formato inválido para NIT (9 o 14 dígitos) o DUI (9 dígitos) |
| `ExceededItemsLimit`, `InvalidItemNumber`, `InvalidItemType` | Más de 2000 ítems, número o tipo de ítem inválido |
| `InvalidLength` | Descripción de ítem fuera de 1–1500 o código de ítem excedido |
| `InvalidNumberRange` | `unit_measure` fuera de 1–99 |
| `UnsupportedTaxCode` | Código en `tributes` fuera del catálogo |
| `ExceededRelatedDocsLimit`, `MixedDocumentTypesNotAllowed` | Más de 50 documentos relacionados o tipos mezclados |
| `InvalidRelatedDocDTEType` | `document_type` relacionado distinto de `01`, `03`, `11` |
| `InvalidRelatedDocDate` | Fecha del documento relacionado futura |
| `InvalidRelatedDocNumberNormal`, `InvalidRelatedDocNumberContingency` | Número físico (>20 caracteres) o UUID inválido |
| `RequiredField` | Falta el nombre del tercero (`ThirdPartyName`) |

---

## Ejemplo completo

### Request

```json
{
  "receiver": {
    "document_type": "36",
    "document_number": "0614-180494-103-5",
    "nit": "0614-180494-103-5",
    "name": "Sucursal Norte",
    "commercial_name": "Sucursal Norte",
    "activity_code": "14108",
    "activity_description": "Maquilado de prendas de vestir",
    "real_state": "04",
    "address": {
      "department": "09",
      "municipality": "10",
      "complement": "Calle Principal, Local 1"
    },
    "phone": "22220000",
    "email": "sucursal@example.com"
  },
  "items": [
    {
      "number": 1,
      "type": 1,
      "description": "Traslado - Zapatilla AF1 Blanco Unisex",
      "quantity": 1,
      "unit_price": 65,
      "unit_measure": 59,
      "discount": 0,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 65,
      "code": "CODIGO-PRODUCTO"
    }
  ],
  "summary": {
    "non_subject_total": 0,
    "exempt_total": 0,
    "taxed_total": 65,
    "subtotal_sales": 65,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "taxed_discount": 0,
    "discount_percent": 0,
    "total_discount": 0,
    "subtotal": 65,
    "total_amount": 65,
    "amount_in_words": "SESENTA Y CINCO 00/100"
  }
}
```

**Verificación de cálculos:**
- `taxed_total = Σ taxed_sale = 65`
- `total_amount = subtotal + tributos = 65 + 0 = 65`
- Todos los montos ≥ 0
