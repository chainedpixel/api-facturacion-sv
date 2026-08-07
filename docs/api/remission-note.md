# Nota de Remisión Electrónica

## Descripción

La **Nota de Remisión Electrónica** (código DTE `04`) se utiliza para documentar el **traslado de bienes** sin que implique una transacción de compraventa inmediata. Es utilizada principalmente en envíos de mercancía entre establecimientos, entregas a consignación, o traslados internos.

**Soporta contingencia:** Sí

---

## Endpoint

```
POST /api/v1/dte/remissionnote
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `receiver` | `RemissionReceiver` | Sí | Destinatario de los bienes |
| `items` | `RemissionItem[]` | Sí | Ítems / bienes trasladados |
| `summary` | `RemissionSummary` | Sí | Resumen del traslado |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `third_party_sale` | `ThirdPartySale` | No | Venta a nombre de tercero |
| `related_docs` | `RelatedDocument[]` | No | Documentos relacionados |
| `appendixes` | `Appendix[]` | No | Apéndices |

---

### `receiver` — Receptor / Destinatario

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `real_state` | string | Sí | Código del título del bien (exactamente 2 chars, ver tabla abajo) |
| `name` | string | Sí | Nombre o razón social del destinatario |
| `document_type` | string | Sí | Tipo de documento del destinatario |
| `document_number` | string | Sí | Número de documento del destinatario |
| `address` | `Address` | Sí | Dirección de destino |
| `email` | string | Sí | Correo electrónico del destinatario |
| `nit` | string | No | NIT del destinatario (requerido si `document_type = "36"`) |
| `nrc` | string | No | NRC del destinatario |
| `phone` | string | No | Teléfono |
| `activity_code` | string | No | Código de actividad económica |
| `activity_description` | string | No | Descripción de actividad |

#### Valores válidos para `real_state`

| Código | Descripción |
|--------|-------------|
| `"01"` | Venta |
| `"02"` | Consignación |
| `"03"` | Exhibición |
| `"04"` | Traslado interno |
| `"05"` | Préstamo |
| `"99"` | Otros |

---

### `items[]` — Ítems / Bienes

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (mínimo 1) |
| `type` | integer | Sí | Tipo: `1`=Producto, `2`=Servicio, `3`=Ambos, `4`=Impuesto |
| `description` | string | Sí | Descripción del bien (1–1000 chars) |
| `quantity` | float | Sí | Cantidad (> 0) |
| `unit_measure` | integer | Sí | Unidad de medida (1–99) |
| `unit_price` | float | Sí | Precio unitario (≥ 0) |
| `discount` | float | No | Monto de descuento (≥ 0) |
| `non_subject_sale` | float | No | Monto no sujeto (≥ 0) |
| `exempt_sale` | float | No | Monto exento (≥ 0) |
| `taxed_sale` | float | No | Monto gravado (≥ 0) |
| `document_number` | string | No | Número de documento relacionado al ítem |
| `code` | string | No | Código del bien |
| `tax_code` | string | No | Código de tributo |
| `tributes` | string[] | No | Códigos de tributos aplicados |

---

### `summary` — Resumen

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `non_subject_total` | float | No | Total no sujeto (≥ 0) |
| `exempt_total` | float | No | Total exento (≥ 0) |
| `taxed_total` | float | No | Total gravado (≥ 0) |
| `subtotal_sales` | float | Sí | Subtotal de ventas (≥ 0) |
| `non_subject_discount` | float | No | Descuento no sujeto (≥ 0) |
| `exempt_discount` | float | No | Descuento exento (≥ 0) |
| `taxed_discount` | float | No | Descuento gravado (≥ 0) |
| `discount_percent` | float | No | Porcentaje de descuento (0–100) |
| `total_discount` | float | No | Total descuentos (≥ 0) |
| `tributes` | `Tax[]` | No | Impuestos |
| `subtotal` | float | Sí | Subtotal neto (≥ 0) |
| `total_amount` | float | Sí | Total del documento (≥ 0) |
| `amount_in_words` | string | Sí | Total en letras (máx. 200 chars) |
| `payments` | `Payment[]` | No | Formas de pago (no se validan) |

---

## Reglas de cálculo

> **Característica especial:** En la Nota de Remisión, `subtotal` y `total_amount` **deben ser iguales** (no hay impuestos adicionales).

### Totales de ítems

```
Σ item.non_subject_sale = summary.non_subject_total  // tolerancia ±0.01
Σ item.exempt_sale      = summary.exempt_total        // tolerancia ±0.01
Σ item.taxed_sale       = summary.taxed_total         // tolerancia ±0.01
```

### Subtotal = Total

```
subtotal = total_amount
// tolerancia: ±0.01

// Esto significa: no hay impuestos adicionales al subtotal en notas de remisión
```

### Restricciones

```
// Todos los montos deben ser ≥ 0:
non_subject_total ≥ 0
exempt_total ≥ 0
taxed_total ≥ 0
total_amount ≥ 0

// Pagos: NO se valida que la suma de pagos = total_amount
```

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| `real_state` | Obligatorio; valores válidos: `"01"`–`"05"` o `"99"` (exactamente 2 chars) |
| `receiver.name` | Obligatorio |
| `receiver.document_type` | Obligatorio |
| `receiver.document_number` | Obligatorio |
| `receiver.address` | Obligatorio |
| `receiver.email` | Obligatorio |
| `receiver.nit` | Requerido si `document_type = "36"` |
| Items | Mínimo 1 |
| `quantity` | Debe ser > 0 |
| `type` | Debe ser `1`, `2`, `3` o `4` |
| `unit_measure` | Entre 1 y 99 |
| `subtotal = total_amount` | Ambos deben ser iguales |
| Montos negativos | No permitidos en ningún campo del resumen |
| `amount_in_words` | Obligatorio, máx. 200 chars |
| Pagos | No se validan contra `total_amount` |
| `extension` | Obligatoria si `total_operation ≥ 1095.00` |

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
      "tipoDte": "04",
      "numeroControl": "DTE-04-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "fecEmi": "2024-01-15"
    },
    "emisor": { },
    "receptor": {
      "bienTitulo": "04",
      "nit": "06141804941035",
      "nombre": "Sucursal Norte",
      "correo": "sucursal@empresa.com",
      "direccion": { }
    },
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
| `400` | `TotalMismatch` | Totales de ítems no coinciden con resumen |
| `400` | `InvalidCalculation` | `subtotal ≠ total_amount` |
| `400` | `InvalidValue` | Algún monto del resumen es negativo |
| `400` | `RequiredField` | Falta `real_state`, `subtotal`, `total_amount` o `amount_in_words` |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó |

---

## Ejemplo completo

### Request

```json
{
  "receiver": {
    "document_type": "13",
    "document_number": "XXXXXXXX-X",
    "name": "Empresa ABC",
    "nit": "XXXX-XXXXXX-XXX-X",
    "commercial_name": "Comercial XYZ",
    "activity_code": "14108",
    "activity_description": "Maquilado de prendas de vestir, accesorios y otros",
    "real_state": "01",
    "address": {
      "department": "09",
      "municipality": "10",
      "complement": "Calle Principal, Local 1"
    },
    "phone": "XXXXXXXX",
    "email": "correo@ejemplo.com"
  },
  "items": [
    {
      "number": 1,
      "type": 1,
      "description": "Traslado - 315122-111/Zapatilla AF1 Blanco Unisex",
      "quantity": 1,
      "unit_price": 65,
      "unit_measure": 59,
      "discount": 0,
      "non_subject_sale": 0,
      "exempt_sale": 0,
      "taxed_sale": 65,
      "code": "CODIGO-PRODUCTO",
      "taxes": ["20"]
    }
  ],
  "summary": {
    "non_subject_total": 0,
    "exempt_total": 0,
    "taxed_total": 65,
    "subtotal_sales": 65,
    "non_subject_discount": 0,
    "exempt_discount": 0,
    "discount_percent": 0,
    "total_discount": 0,
    "subtotal": 65,
    "total_amount": 65,
    "total_to_pay": 65,
    "amount_in_words": "SESENTA Y CINCO DÓLARES EXACTOS"
  }
}
```

**Verificación de cálculos:**
- `taxed_sale = unit_price × quantity = 65 × 1 = 65 = summary.taxed_total`
- `subtotal = 65 = total_amount` (regla especial de remisión)
- Todos los montos ≥ 0