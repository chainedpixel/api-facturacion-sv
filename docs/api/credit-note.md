# Nota de Crédito Electrónica

## Descripción

La **Nota de Crédito Electrónica** (código DTE `05`) se utiliza para anular o ajustar parcialmente una Factura Electrónica o CCF ya emitido. Requiere obligatoriamente documentos relacionados y cada ítem debe referenciar uno de ellos.

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
| `receiver` | `Receiver` | Sí | Datos del receptor |
| `model_type` | integer | Sí | `1`=previo, `2`=diferido |
| `summary` | `CreditNoteSummary` | Sí | Resumen financiero |
| `related_docs` | `RelatedDocument[]` | Sí | Documentos que se están ajustando (**obligatorio**) |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `payments` | `Payment[]` | No | Formas de pago (no se validan contra total) |
| `other_docs` | `OtherDocument[]` | No | Otros documentos (máx. 10) |
| `appendixes` | `Appendix[]` | No | Apéndices |

---

### `items[]` — Ítems

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (1–2000) |
| `type` | integer | Sí | `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1000 chars) |
| `quantity` | float | Sí | Cantidad |
| `unit_measure` | integer | Sí | Unidad de medida (1–99; `99` para tipo `4`) |
| `unit_price` | float | Sí | Precio unitario |
| `discount` | float | Sí | Descuento porcentual |
| `non_subject_sale` | float | Sí | Venta no sujeta |
| `exempt_sale` | float | Sí | Venta exenta |
| `taxed_sale` | float | Sí | Venta gravada |
| `related_doc` | string | Sí | **Obligatorio.** Número del documento relacionado |
| `code` | string | No | Código del producto (máx. 25 chars) |
| `tax_code` | string | No | Código de tributo especial |
| `taxes` | string[] | No | Códigos de impuestos |

> **Regla:** `related_doc` es **obligatorio** en cada ítem y debe coincidir con alguno de los documentos en `related_docs`.

> **Regla:** Solo un tipo de venta por ítem.

> **Regla:** Ítems con `taxed_sale > 0` requieren al menos un impuesto.

> **Regla:** Ítems de tipo `4`: `unit_measure = 99`, único impuesto `20`.

---

### `receiver` — Receptor

Ver estructura en [Factura Electrónica](./invoice.md#receiver--receptor). No requiere NRC.

---

### `summary` — Resumen

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total no sujeto |
| `total_exempt` | float | Sí | Total exento |
| `total_taxed` | float | Sí | Total gravado |
| `sub_total_sales` | float | Sí | Subtotal de ventas |
| `non_subject_discount` | float | Sí | Descuento no sujeto |
| `exempt_discount` | float | Sí | Descuento exento |
| `taxed_discount` | float | Sí | Descuento gravado |
| `discount_percentage` | float | Sí | Porcentaje de descuento |
| `total_discount` | float | Sí | Total descuentos |
| `sub_total` | float | Sí | Subtotal neto |
| `total_operation` | float | Sí | Total de la operación |
| `total_non_taxed` | float | Sí | No afecto |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito |
| `taxes` | `Tax[]` | No | Impuestos |
| `payment_types` | `Payment[]` | Sí | Formas de pago |
| `iva_perception` | float | Sí | Percepción IVA (1%) |
| `iva_retention` | float | Sí | Retención IVA |
| `income_retention` | float | Sí | Retención renta |

---

### `related_docs[]` — Documentos relacionados (**Obligatorio**)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de DTE ajustado (solo FE `01` o CCF `03`) |
| `generation_type` | integer | Sí | `1`=Normal, `2`=Contingencia |
| `document_number` | string | Sí | Número del DTE original |
| `emission_date` | string | Sí | Fecha de emisión original (`YYYY-MM-DD`) |

> **Regla:** Solo se permiten tipos de DTE `01` (Factura) y `03` (CCF) en los documentos relacionados.
> **Regla:** Máximo 50 documentos relacionados.
> **Regla:** No se puede mezclar tipos de documento (todos deben ser del mismo tipo).

---

## Reglas de cálculo

> **Importante:** La Nota de Crédito NO valida pagos contra `total_to_pay`.

### Totales de ítems

```
Σ item.taxed_sale       = summary.total_taxed        // tolerancia ±0.01
Σ item.exempt_sale      = summary.total_exempt        // exacto
Σ item.non_subject_sale = summary.total_non_subject   // exacto

// Subtotal de ventas:
sub_total_sales = total_taxed + total_exempt + total_non_subject  // tolerancia ±0.01
```

### Subtotal

```
sub_total = total_taxed - taxed_discount
          + total_exempt - exempt_discount
          + total_non_subject - non_subject_discount
// tolerancia: ±0.01
```

### Impuestos sobre `total_taxed`

| Código | Nombre | Fórmula |
|--------|--------|---------|
| `20` | IVA | `total_taxed × 0.13` (sin descontar descuento, diferente al CCF) |
| `59` | Turismo | `total_taxed × 0.05` |
| `71` | Turismo Aeropuerto | `$7.00` (fijo) |
| `D1` | FOVIAL | `total_taxed × 0.20` |
| `C8` | COTRANS | `$0.10` (fijo) |

> **Tolerancia:** ±0.01

### Percepción

```
iva_perception = total_taxed × 0.01  // tolerancia ±0.01
```

### Total a pagar

```
total_to_pay = sub_total + iva_perception - iva_retention - income_retention + Σ taxes
// tolerancia: ±0.01
```

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| `related_docs` | **Obligatorio**; tipo `01` o `03` únicamente |
| `related_doc` en ítems | **Obligatorio** en cada ítem |
| Items | 1–2000 |
| `total_taxed > 0` | Cada ítem gravado requiere al menos un impuesto |
| `extension` | Obligatoria si `total_operation ≥ 1095.00` |
| Montos | Máximo 2 decimales |
| Pagos | **No se validan** contra `total_to_pay` |

---

## Respuesta exitosa

**HTTP 201 Created**

```json
{
  "success": true,
  "reception_stamp": "20240115...",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?...",
  "data": {
    "identificacion": {
      "tipoDte": "05",
      "numeroControl": "DTE-05-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "fecEmi": "2024-01-15"
    },
    "emisor": { },
    "receptor": { },
    "cuerpoDocumento": [ ],
    "resumen": { },
    "documentoRelacionado": [ ],
    "apendice": []
  }
}
```

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `MissingItemRelatedDoc` | Un ítem no tiene `related_doc` |
| `400` | `InvalidItemRelatedDoc` | `related_doc` del ítem no existe en `related_docs` |
| `400` | `InvalidTaxCalculation` | Impuesto calculado incorrectamente |
| `400` | `InvalidSubTotalCalculation` | `sub_total` incorrecto |
| `400` | `InvalidTotalToPayCalculation` | `total_to_pay` incorrecto |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó |

---

## Ejemplo completo

### Request

```json
{
  "items": [
    {
      "number": 1,
      "type": 1,
      "description": "Devolución - 201-210-501-20",
      "quantity": 1,
      "unit_price": 13.2743363,
      "unit_measure": 59,
      "discount": 8.85,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 4.4247788,
      "taxes": ["20"],
      "code": "CODIGO-PRODUCTO",
      "related_doc": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
    }
  ],
  "receiver": {
    "nit": "XXXX-XXXXXX-XXX-X",
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
    "total_discount": 8.85,
    "taxes": [
      { "code": "20", "description": "Impuesto al Valor Agregado 13%", "value": 0.58 }
    ],
    "sub_total": 4.42,
    "iva_perception": 0,
    "iva_retention": 0,
    "income_retention": 0,
    "total_operation": 5,
    "total_to_pay": 5,
    "operation_condition": 1,
    "electronic_payment_number": null
  },
  "related_docs": [
    {
      "document_type": "03",
      "generation_type": 2,
      "document_number": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "emission_date": "2026-05-08"
    }
  ]
}
```

**Verificación de cálculos:**
- `taxed_sale ≈ unit_price - discount = 13.2743363 - 8.8495575 ≈ 4.4248`
- `total_taxed (redondeado) = 4.42`
- `IVA = 4.42 × 0.13 = 0.5746 ≈ 0.58`
- `sub_total = total_taxed = 4.42`
- `total_to_pay = sub_total + IVA = 4.42 + 0.58 = 5.00`