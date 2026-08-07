# Nota de Débito Electrónica

## Descripción

La **Nota de Débito Electrónica** (código DTE `06`) se utiliza para aumentar el valor de una Factura Electrónica o CCF ya emitido (por ejemplo, por cargos adicionales, corrección de precio hacia arriba, etc.). Al igual que la Nota de Crédito, requiere documentos relacionados y cada ítem debe referenciarlos.

**Soporta contingencia:** Sí

---

## Endpoint

```
POST /api/v1/dte/debitnote
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `DebitNoteItem[]` | Sí | Lista de ítems (1–2000) |
| `receiver` | `Receiver` | Sí | Datos del receptor |
| `model_type` | integer | Sí | `1`=previo, `2`=diferido |
| `summary` | `DebitNoteSummary` | Sí | Resumen financiero |
| `related_docs` | `RelatedDocument[]` | Sí | Documentos que se están ajustando (**obligatorio**) |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `payments` | `Payment[]` | No | Formas de pago (no se validan contra total) |
| `other_docs` | `OtherDocument[]` | No | Otros documentos (máx. 10) |
| `appendixes` | `Appendix[]` | No | Apéndices |

---

### `items[]` — Ítems

Idénticos a los de [Nota de Crédito](./credit-note.md#items--ítems). Campo `related_doc` **obligatorio**.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (1–2000) |
| `type` | integer | Sí | `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción (1–1000 chars) |
| `quantity` | float | Sí | Cantidad |
| `unit_measure` | integer | Sí | Unidad (1–99; `99` para tipo `4`) |
| `unit_price` | float | Sí | Precio unitario |
| `discount` | float | Sí | Descuento porcentual |
| `non_subject_sale` | float | Sí | Venta no sujeta |
| `exempt_sale` | float | Sí | Venta exenta |
| `taxed_sale` | float | Sí | Venta gravada |
| `related_doc` | string | Sí | **Obligatorio.** Referencia al documento relacionado |
| `code` | string | No | Código (máx. 25 chars) |
| `tax_code` | string | No | Tributo especial |
| `taxes` | string[] | No | Códigos de impuestos |

---

### `receiver` — Receptor

Ver estructura en [Factura Electrónica](./invoice.md#receiver--receptor).

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

Idéntico a [Nota de Crédito](./credit-note.md#related_docs--documentos-relacionados-obligatorio).

---

## Reglas de cálculo

> **Nota:** Las fórmulas de la Nota de Débito son idénticas a las de la Nota de Crédito. La diferencia es semántica: la Nota de Débito **incrementa** el valor del documento original.

> **Importante:** La Nota de Débito NO valida pagos contra `total_to_pay`.

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
| `20` | IVA | `total_taxed × 0.13` |
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
| `related_docs` | **Obligatorio**; solo tipos `01` (FE) y `03` (CCF) |
| `related_doc` en ítems | **Obligatorio** en cada ítem |
| Items | 1–2000 |
| `total_taxed > 0` | Requiere al menos un impuesto |
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
      "tipoDte": "06",
      "numeroControl": "DTE-06-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
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
| `400` | `MissingItemRelatedDoc` | Ítem sin `related_doc` |
| `400` | `InvalidItemRelatedDoc` | `related_doc` no existe en `related_docs` |
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
      "description": "Cargo adicional - 315122-111/Zapatilla AF1 Blanco Unisex",
      "quantity": 1,
      "unit_price": 57.5221239,
      "unit_measure": 99,
      "discount": 0,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 57.5221239,
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
    "total_taxed": 57.52,
    "sub_total_sales": 57.52,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "total_discount": 0,
    "taxes": [
      { "code": "20", "description": "Impuesto al Valor Agregado 13%", "value": 7.48 }
    ],
    "sub_total": 57.52,
    "iva_perception": 0,
    "iva_retention": 0,
    "income_retention": 0,
    "total_operation": 65,
    "total_to_pay": 65,
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
- `taxed_sale = unit_price × quantity = 57.5221239 × 1 = 57.5221239`
- `total_taxed (redondeado) = 57.52`
- `IVA = 57.52 × 0.13 = 7.4776 ≈ 7.48`
- `sub_total = total_taxed = 57.52`
- `total_to_pay = sub_total + IVA = 57.52 + 7.48 = 65.00`