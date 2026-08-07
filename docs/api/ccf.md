# Comprobante de Crédito Fiscal (CCF)

## Descripción

El **Comprobante de Crédito Fiscal** (código DTE `03`) se utiliza para documentar ventas de bienes o servicios entre **contribuyentes registrados** (emisor y receptor con NRC). A diferencia de la factura electrónica, el CCF detalla el IVA por separado y calcula el IVA sobre la base gravada **después de aplicar descuentos**.

**Soporta contingencia:** Sí

---

## Endpoint

```
POST /api/v1/dte/ccf
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `CreditItem[]` | Sí | Lista de ítems (1–2000) |
| `receiver` | `Receiver` | Sí | Datos del receptor (**NRC obligatorio**) |
| `model_type` | integer | Sí | `1` = previo (normal), `2` = diferido (contingencia) |
| `summary` | `CreditSummary` | Sí | Resumen financiero |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `payments` | `Payment[]` | No | Formas de pago |
| `other_docs` | `OtherDocument[]` | No | Otros documentos (máx. 10) |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados (máx. 50) |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |

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
| `discount` | float | Sí | Porcentaje de descuento (0–100) |
| `non_subject_sale` | float | Sí | Venta no sujeta |
| `exempt_sale` | float | Sí | Venta exenta |
| `taxed_sale` | float | Sí | Venta gravada |
| `non_taxed` | float | Sí | No afecto |
| `suggested_price` | float | Sí | Precio sugerido de venta |
| `code` | string | No | Código del producto (máx. 25 chars) |
| `tax_code` | string | No | Código de tributo especial |
| `related_doc` | string | No | Referencia a documento relacionado |
| `taxes` | string[] | No | Códigos de impuestos |

> **Regla:** Solo un tipo de venta por ítem (`taxed_sale`, `exempt_sale`, `non_subject_sale` o `non_taxed`). No se permiten mezclas.

> **Regla:** Ítems con `non_taxed > 0` no pueden tener impuestos ni `unit_price != 0`.

> **Regla:** Ítems de tipo `1` (Producto) con `taxed_sale > 0` solo pueden tener el impuesto `20` (IVA).

> **Regla:** Ítems de tipo `4` (Impuesto): `unit_measure = 99`, solo impuesto `20`.

---

### `receiver` — Receptor

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `nrc` | string | Sí | NRC del receptor (**obligatorio para CCF**) |
| `nit` | string | No | NIT del receptor |
| `name` | string | No | Nombre o razón social |
| `document_type` | string | No | Tipo de documento |
| `document_number` | string | No | Número de documento |
| `address` | `Address` | No | Dirección |
| `phone` | string | No | Teléfono |
| `email` | string | No | Correo electrónico |
| `activity_code` | string | No | Código de actividad económica |
| `activity_description` | string | No | Descripción de actividad económica |
| `commercial_name` | string | No | Nombre comercial |

---

### `summary` — Resumen financiero

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total ventas no sujetas |
| `total_exempt` | float | Sí | Total ventas exentas |
| `total_taxed` | float | Sí | Total ventas gravadas |
| `sub_total_sales` | float | Sí | Subtotal de ventas |
| `non_subject_discount` | float | Sí | Descuento no sujeto |
| `exempt_discount` | float | Sí | Descuento exento |
| `taxed_discount` | float | Sí | Descuento gravado |
| `discount_percentage` | float | Sí | Porcentaje de descuento |
| `total_discount` | float | Sí | Total descuentos |
| `sub_total` | float | Sí | Subtotal neto |
| `total_operation` | float | Sí | Total de la operación |
| `total_non_taxed` | float | Sí | Total no afecto |
| `total_to_pay` | float | Sí | Total a pagar |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito |
| `taxes` | `Tax[]` | No | Impuestos |
| `payment_types` | `Payment[]` | Sí | Formas de pago |
| `total_iva` | float | Sí | Total de IVA |
| `iva_perception` | float | Sí | Percepción de IVA (1%) |
| `iva_retention` | float | Sí | Retención de IVA |
| `income_retention` | float | Sí | Retención de renta |
| `balance_in_favor` | float | Sí | Saldo a favor |
| `total_in_words` | string | No | Total en letras |

---

## Reglas de cálculo

### Diferencia clave respecto a Factura: el IVA en CCF se calcula DESPUES del descuento

```
// IVA:
IVA = (total_taxed - taxed_discount) × 0.13
// tolerancia: ±0.01
```

### Por ítem

```
// Venta gravada por ítem (cuando taxed_sale > 0):
taxed_sale = unit_price × quantity - discount_amount
// donde discount_amount = (discount% / 100) × (unit_price × quantity)
// tolerancia: ±0.01

// El CCF NO usa el campo iva_item por ítem
```

### Totales de ítems

```
Σ item.taxed_sale       = summary.total_taxed        // tolerancia ±0.01
Σ item.exempt_sale      = summary.total_exempt        // exacto, sin tolerancia
Σ item.non_subject_sale = summary.total_non_subject   // exacto, sin tolerancia
Σ item.non_taxed        = summary.total_non_taxed     // exacto, sin tolerancia

// Subtotal de ventas:
sub_total_sales = total_taxed + total_exempt + total_non_subject  // tolerancia ±0.01
```

### Descuentos

```
taxed_discount      ≤ total_taxed
exempt_discount     ≤ total_exempt
non_subject_discount ≤ total_non_subject
// Todos ≥ 0
```

### Subtotal

```
sub_total = total_taxed - taxed_discount
          + total_exempt - exempt_discount
          + total_non_subject - non_subject_discount
// tolerancia: ±0.01
```

### Impuestos sobre `total_taxed` (base ANTES del descuento, excepto IVA)

| Código | Nombre | Fórmula |
|--------|--------|---------|
| `20` | IVA | `(total_taxed - taxed_discount) × 0.13` |
| `C3` | IVA Exportación | `total_taxed × 0.00` |
| `59` | Turismo | `total_taxed × 0.05` |
| `71` | Turismo Aeropuerto | `$7.00` (fijo) |
| `D1` | FOVIAL | `total_taxed × 0.20` |
| `C8` | COTRANS | `$0.10` (fijo) |
| `D5` | Otro especial | Libre — no se valida |

### Percepción

```
iva_perception = total_taxed × 0.01
// tolerancia: ±0.01
// Solo cuando total_taxed > 0
```

### Total de la operación

```
// No se valida directamente; se valida total_to_pay:
total_to_pay = total_operation + iva_perception - iva_retention - income_retention
// Si total_non_taxed > 0: + total_non_taxed
// tolerancia: ±0.01
```

### Pagos

```
Σ payment.amount = total_to_pay  // tolerancia ±0.01
```

> **Condición Crédito sin efectivo:** cada pago necesita `period` y `term`.
> **Condición Contado:** los pagos no pueden tener `period` ni `term`.

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| NRC del receptor | **Obligatorio** para CCF |
| Items | 1–2000 |
| Fecha de emisión | No futura |
| `extension` | Obligatoria si `total_operation ≥ 1095.00` |
| `total_taxed > 0` | Requiere impuesto `20` (IVA) en `taxes` |
| Tipo de ítem `4` | `unit_measure = 99`, solo impuesto `20` |
| Montos monetarios | Máximo 2 decimales |
| `related_docs` | Máx. 50; si presentes, todos los ítems requieren `related_doc` |

---

## Respuesta exitosa

**HTTP 201 Created**

```json
{
  "success": true,
  "reception_stamp": "20240115101530ABCD1234...",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX&fechaEmi=2024-01-15",
  "data": {
    "identificacion": {
      "tipoDte": "03",
      "numeroControl": "DTE-03-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "fecEmi": "2024-01-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "emisor": { },
    "receptor": { },
    "cuerpoDocumento": [ ],
    "resumen": { },
    "documentoRelacionado": [],
    "otrosDocumentos": [],
    "apendice": []
  }
}
```

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `MissingTaxes` | `total_taxed > 0` pero no hay impuestos en `taxes` |
| `400` | `MissingIVAForTaxedAmount` | `total_taxed > 0` pero no hay IVA (`20`) en `taxes` |
| `400` | `InvalidTaxCalculation` | Un impuesto no coincide con su tasa esperada |
| `400` | `InvalidSubTotalCalculation` | `sub_total` calculado no coincide |
| `400` | `InvalidSubTotalSales` | `sub_total_sales` no coincide con suma de ventas |
| `400` | `InvalidTotalToPayCalculation` | `total_to_pay` no coincide |
| `400` | `InvalidPerceptionAmount` | `iva_perception ≠ total_taxed × 0.01` |
| `400` | `InvalidMonetaryAmount` | Monto con más de 2 decimales |
| `400` | `DiscountExceedsBase` | Descuento supera su base |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó el documento |

---

## Ejemplo completo

### Request

```json
{
    "items": [
        {
            "type": 1,
            "description": "CODO PVC 1",
            "quantity": 50,
            "unit_measure": 59,
            "unit_price": 0.66371681,
            "discount": 0,
            "code": "MAT-002",
            "non_subject_sale": 0,
            "exempt_sale": 0,
            "taxed_sale": 33.1858405,
            "suggested_price": 0,
            "non_taxed": 0,
            "taxes": ["20"]
        }
    ],
    "receiver": {
        "nrc": "1937",
        "nit": "06141101690011",
        "name": "Super Selectos S.A. de C.V.",
        "commercial_name": "Super Selectos",
        "activity_code": "01271",
        "activity_description": "Cultivo de café",
        "address": {
            "department": "06",
            "municipality": "23",
            "complement": "Dirección de ejemplo, San Salvador"
        },
        "phone": "22220000",
        "email": "receptor@example.com"
    },
    "summary": {
        "operation_condition": 1,
        "total_taxed": 33.19,
        "total_exempt": 0,
        "total_non_taxed": 0,
        "total_non_subject": 0,
        "sub_total_sales": 33.19,
        "sub_total": 33.19,
        "total_operation": 37.5,
        "total_to_pay": 37.5,
        "iva_retention": 0,
        "taxes": [
            {
                "value": 4.31,
                "description": "IVA 13%",
                "code": "20"
            }
        ],
        "payment_types": [
            {
                "code": "01",
                "amount": 37.5
            }
        ]
    },
    "extension": {
        "delivery_name": "Nombre del Entregador",
        "delivery_document": "06141101690011",
        "receiver_name": "Super Selectos S.A. de C.V.",
        "receiver_document": "06141101690011"
    }
}
```
### Ejemplo con descuento (diferencia clave en IVA)

Si `taxed_sale = 500.00` y `taxed_discount = 50.00`:
- `IVA = (500.00 - 50.00) × 0.13 = 450.00 × 0.13 = 58.50`
- `sub_total = 500.00 - 50.00 = 450.00`
- `total_operation = 450.00 + 58.50 = 508.50`
- `total_to_pay = 508.50`
