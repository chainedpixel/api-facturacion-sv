# Factura Electrónica (FE)

## Descripción

La **Factura Electrónica** (código DTE `01`) se utiliza para documentar transacciones de venta de bienes o servicios a **consumidores finales** (personas naturales o jurídicas sin número de NRC). Es el documento más común para ventas al público en general.

**Soporta contingencia:** Sí

---

## Endpoint

```
POST /api/v1/dte/invoices
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `InvoiceItem[]` | Sí | Lista de ítems (1–2000) |
| `receiver` | `Receiver` | Sí | Datos del receptor |
| `model_type` | integer | Sí | Tipo de modelo: `1` = previo (transmisión normal), `2` = diferido (contingencia) |
| `summary` | `InvoiceSummary` | Sí | Resumen financiero del documento |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `extension` | `Extension` | No* | Datos de entrega. **Obligatorio si `total_operation ≥ 1095.00`** |
| `payments` | `Payment[]` | No | Formas de pago (reemplaza los de `summary`) |
| `other_docs` | `OtherDocument[]` | No | Otros documentos asociados (máx. 10) |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados (máx. 50) |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |

---

### `items[]` — Ítems de la factura

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (1–2000, secuencial) |
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1000 caracteres) |
| `quantity` | float | Sí | Cantidad (> 0) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99; usar `99` para tipo `4`) |
| `unit_price` | float | Sí | Precio unitario. **Debe ser `0` cuando `non_taxed > 0`** |
| `discount` | float | Sí | Descuento en porcentaje (0–100) |
| `non_subject_sale` | float | Sí | Venta no sujeta |
| `exempt_sale` | float | Sí | Venta exenta |
| `taxed_sale` | float | Sí | Venta gravada |
| `non_taxed` | float | Sí | No afecto (PSV/donaciones). No mezclar con otros tipos de venta |
| `suggested_price` | float | Sí | Precio sugerido de venta (PSV) |
| `iva_item` | float | Sí | IVA incluido en el precio del ítem (ver cálculo) |
| `code` | string | No | Código del producto (máx. 25 caracteres) |
| `tax_code` | string | No | Código de tributo especial |
| `related_doc` | string | No | Número del documento relacionado (debe coincidir con `related_docs`) |
| `taxes` | string[] | No | Códigos de impuestos aplicados al ítem |

> **Regla:** Solo se puede usar **un tipo de venta por ítem**: `taxed_sale`, `exempt_sale`, `non_subject_sale`, o `non_taxed`. No se permiten mezclas.

> **Regla:** Ítems de tipo `4` (Impuesto): `unit_measure` debe ser `99` y solo se permite el impuesto `20` (IVA).

> **Regla:** Ítems de tipo `1` (Producto) no pueden tener el impuesto `20` (IVA) en el array `taxes` ni en el resumen.

---

### `receiver` — Receptor

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | No | Tipo de documento (ej. `"13"` = DUI, `"36"` = NIT) |
| `document_number` | string | No | Número de documento |
| `name` | string | No | Nombre o razón social |
| `nrc` | string | No | NRC del receptor |
| `nit` | string | No | NIT del receptor |
| `address` | `Address` | No | Dirección |
| `phone` | string | No | Teléfono |
| `email` | string | No | Correo electrónico |
| `activity_code` | string | No | Código de actividad económica |
| `activity_description` | string | No | Descripción de actividad económica |
| `commercial_name` | string | No | Nombre comercial |

#### `receiver.address`

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí | Código de departamento |
| `municipality` | string | Sí | Código de municipio |
| `complement` | string | Sí | Dirección complementaria |

---

### `summary` — Resumen financiero

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total de ventas no sujetas |
| `total_exempt` | float | Sí | Total de ventas exentas |
| `total_taxed` | float | Sí | Total de ventas gravadas |
| `sub_total_sales` | float | Sí | Subtotal de ventas (antes de descuentos) |
| `non_subject_discount` | float | Sí | Descuento sobre no sujeto |
| `exempt_discount` | float | Sí | Descuento sobre exento |
| `discount_percentage` | float | Sí | Porcentaje de descuento general |
| `total_discount` | float | Sí | Total de descuentos |
| `sub_total` | float | Sí | Subtotal después de descuentos |
| `total_operation` | float | Sí | Total de la operación (subtotal + impuestos) |
| `total_non_taxed` | float | Sí | Total de montos no afectos |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | Condición: `1`=Contado, `2`=Crédito |
| `taxes` | `Tax[]` | No | Impuestos del documento |
| `payment_types` | `Payment[]` | Sí | Formas de pago |
| `total_in_words` | string | No | Total en letras |
| `taxed_discount` | float | Sí | Descuento sobre gravado |
| `iva_retention` | float | Sí | Retención de IVA |
| `income_retention` | float | Sí | Retención de renta |
| `total_iva` | float | Sí | Total de IVA |
| `balance_in_favor` | float | Sí | Saldo a favor |

---

### `summary.taxes[]` — Impuestos

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `code` | string | Código: `"20"`=IVA, `"C3"`=IVAExport, `"59"`=Turismo, `"71"`=TurismoAeropuerto, `"D1"`=FOVIAL, `"C8"`=COTRANS, `"D5"`=OtroEspecial |
| `description` | string | Descripción del tributo |
| `value` | float | Valor calculado (ver fórmulas en sección de cálculos) |

---

### `summary.payment_types[]` — Formas de pago

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `code` | string | Sí | Código de forma de pago (ej. `"01"`=Efectivo, `"02"`=Cheque, `"03"`=Transferencia) |
| `amount` | float | Sí | Monto (exactamente 2 decimales) |
| `period` | integer | No* | Período en días. **Requerido si condición = Crédito y no hay pago en efectivo** |
| `term` | string | No* | Plazo. **Requerido si condición = Crédito y no hay pago en efectivo** |
| `reference` | string | No | Referencia del pago |

---

### `extension` — Datos de entrega

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `delivery_name` | string | No | Nombre de quien entrega (máx. 100 caracteres) |
| `delivery_document` | string | No | Documento de quien entrega (máx. 25 caracteres) |
| `receiver_name` | string | No | Nombre de quien recibe (máx. 100 caracteres) |
| `receiver_document` | string | No | Documento de quien recibe (máx. 25 caracteres) |
| `observation` | string | No | Observaciones |
| `vehicule_plate` | string | No | Placa del vehículo (1-10 caracteres) |

---

### `related_docs[]` — Documentos relacionados

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de DTE relacionado |
| `generation_type` | integer | Sí | `1`=Normal (número ≤ 20 chars), `2`=Contingencia (UUID) |
| `document_number` | string | Sí | Número del documento relacionado |
| `emission_date` | string | Sí | Fecha de emisión (`YYYY-MM-DD`, no puede ser futura) |

> **Regla:** Si se envían `related_docs`, **todos los ítems** deben tener el campo `related_doc` apuntando a uno de los documentos listados.

---

### `other_docs[]` — Otros documentos

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_code` | integer | Sí | Código: `1`, `2`, `3` (médico) o `4` |
| `description` | string | No* | Descripción. **Requerido para códigos 1, 2 y 4** |
| `detail` | string | No* | Detalle. **Requerido para códigos 1, 2 y 4** |
| `doctor` | `Doctor` | No* | Datos del médico. **Requerido solo para código `3`** |

#### `other_docs[].doctor` (solo código `3`)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `name` | string | Sí | Nombre del médico (1–100 chars) |
| `nit` | string | No* | NIT del médico. Solo uno de `nit` o `identification` |
| `identification` | string | No* | Documento de identidad. Solo uno de `nit` o `identification` |
| `service_type` | integer | Sí | Tipo de servicio médico (1–6) |

---

### `third_party_sale` — Venta a nombre de tercero

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `nit` | string | Sí | NIT del tercero |
| `name` | string | Sí | Nombre del tercero |

> **Regla:** Si se incluye `third_party_sale`, **todos los ítems** deben tener `related_doc`.

---

### `appendixes[]` — Apéndices

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `field` | string | Identificador del campo |
| `label` | string | Etiqueta visible |
| `value` | string | Valor del campo |

---

## Reglas de cálculo

### Por ítem

```
// Venta gravada:
taxed_sale = unit_price × quantity - discount_amount
// donde: discount_amount = (discount% / 100) × (unit_price × quantity)
// tolerancia: ±0.01

// IVA por ítem (cuando iva_item > 0):
iva_item = (taxed_sale / 1.13) × 0.13
// tolerancia: ±0.01

// Total ventas por ítem no puede exceder:
taxed_sale + exempt_sale + non_subject_sale ≤ unit_price × quantity
```

### Totales de ítems vs. resumen

```
Σ item.taxed_sale       = summary.total_taxed        // tolerancia ±0.01
Σ item.exempt_sale      = summary.total_exempt        // tolerancia ±0.01
Σ item.non_subject_sale = summary.total_non_subject   // tolerancia ±0.01
Σ item.non_taxed        = summary.total_non_taxed     // tolerancia ±0.01
```

### Subtotal

```
sub_total = total_taxed - taxed_discount
          + total_exempt - exempt_discount
          + total_non_subject - non_subject_discount
// tolerancia: ±0.0001
```

### Total de la operación

```
total_operation = sub_total + Σ taxes.value
// tolerancia: ±0.0001
```

### Impuestos sobre `total_taxed`

| Código | Nombre | Fórmula |
|--------|--------|---------|
| `20` | IVA | `total_taxed × 0.13` |
| `C3` | IVA Exportación | `total_taxed × 0.00` |
| `59` | Turismo | `total_taxed × 0.05` |
| `71` | Turismo Aeropuerto | `$7.00` (fijo) |
| `D1` | FOVIAL | `total_taxed × 0.20` |
| `C8` | COTRANS | `$0.10` (fijo) |
| `D5` | Otro especial | Libre — no se valida |

> **Tolerancia de impuestos:** ±0.01

### Total a pagar

```
// Cuando total_taxed > 0:
total_to_pay = total_operation - iva_retention - income_retention
// Si total_non_taxed > 0: + total_non_taxed

// Cuando total_taxed = 0:
total_to_pay = total_operation
// iva_retention e income_retention DEBEN ser 0
```

### Pagos

```
Σ payment.amount = total_to_pay  // tolerancia ±0.01
```

**Condición `2` (Crédito) sin pago en efectivo:**
- Cada pago debe incluir `period` y `term`
- El código `"01"` (Billetes y Monedas) no puede usarse como único método

**Condición `1` (Contado):**
- Los pagos no deben incluir `period` ni `term`

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Items | Mínimo 1, máximo 2000 |
| Fecha de emisión | No puede ser futura |
| `extension` | Obligatoria si `total_operation ≥ 1095.00` |
| `model_type` | `1` para transmisión normal; `2` para contingencia |
| Tipo de ítem `4` | `unit_measure = 99`, solo impuesto `20` permitido |
| Descuentos | Ningún descuento puede ser negativo ni superar su base |
| `iva_retention` | Solo válida si `total_taxed > 0` |
| `income_retention` | Solo válida si `total_taxed > 0` |
| `total_taxed > 0` | Requiere impuesto `20` (IVA) en `taxes` |
| Montos monetarios | Máximo 2 decimales (sin fracciones de centavo) |
| `related_docs` | Máximo 50; si están presentes todos los ítems necesitan `related_doc` |
| `other_docs` | Máximo 10 |
| Código de ítem | Máximo 25 caracteres |

---

## Respuesta exitosa

**HTTP 201 Created**

```json
{
  "success": true,
  "reception_stamp": "20240115101530ABCD1234...",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C&fechaEmi=2024-01-15",
  "data": {
    "identificacion": {
      "version": 1,
      "ambiente": "01",
      "tipoDte": "01",
      "numeroControl": "DTE-01-N0010001-000000000000001",
      "codigoGeneracion": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2024-01-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "emisor": { },
    "receptor": { },
    "cuerpoDocumento": [ ],
    "resumen": { },
    "apendice": []
  }
}
```

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `InvalidTaxedAmount` | `taxed_sale` del ítem no coincide con `unit_price × quantity - discount` |
| `400` | `InvalidIVAItemCalculation` | `iva_item` no coincide con `(taxed_sale / 1.13) × 0.13` |
| `400` | `InvalidSubTotal` | `sub_total` no coincide con la fórmula esperada |
| `400` | `InvalidTotalOperation` | `total_operation` no coincide con `sub_total + Σ impuestos` |
| `400` | `InvalidTotalToPayCalculation` | `total_to_pay` no coincide con la fórmula esperada |
| `400` | `InvalidTaxCalculation` | Un impuesto no coincide con su tasa esperada |
| `400` | `MissingIVAForTaxedAmount` | `total_taxed > 0` pero no hay IVA en `taxes` |
| `400` | `InvalidPaymentTotal` | Suma de pagos ≠ `total_to_pay` |
| `400` | `MixedSalesTypesNotAllowed` | Un ítem tiene más de un tipo de venta |
| `400` | `DiscountExceedsBase` | Un descuento supera su monto base |
| `400` | `InvalidMonetaryAmount` | Un monto tiene más de 2 decimales |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó el documento |
| `500` | `SYSTEM_ERROR` | Error interno del servidor |

---

## Ejemplo completo

### Request

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
      "discount": 30,
      "taxed_sale": 195,
      "iva_item": 22.43
    }
  ],
  "receiver": null,
  "summary": {
    "total_taxed": 195,
    "sub_total": 185,
    "sub_total_sales": 195,
    "taxed_discount": 10,
    "discount_percentage": 13.33,
    "total_discount": 30,
    "total_operation": 185,
    "total_to_pay": 185,
    "operation_condition": 1,
    "total_iva": 21.28,
    "payment_types": [
      {
        "code": "01",
        "amount": 185
      }
    ]
  }
}
```