# Factura de Sujeto Excluido (FSE)

## Descripción

La **Factura de Sujeto Excluido** (código DTE `14`) se utiliza para documentar compras realizadas a personas naturales que **no son contribuyentes del IVA** (sujetos excluidos). A diferencia de otros DTEs, este documento registra el **monto de compra** (no venta gravada), y **no aplica IVA**.

**Soporta contingencia:** Sí

---

## Endpoint

```
POST /api/v1/dte/fse
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `FSEItem[]` | Sí | Lista de ítems |
| `excluded_subject` | `FSEReceiver` | Sí | Datos del sujeto excluido |
| `summary` | `FSESummary` | Sí | Resumen financiero |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `appendixes` | `Appendix[]` | No | Apéndices |

---

### `items[]` — Ítems

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `number` | integer | Sí | Número de ítem (1–2000) |
| `type` | integer | Sí | `1`=Producto, `2`=Servicio, `3`=Ambos |
| `description` | string | Sí | Descripción (1–1000 chars) |
| `quantity` | float | Sí | Cantidad |
| `unit_measure` | integer | Sí | Unidad de medida (1–99) |
| `unit_price` | float | Sí | Precio unitario |
| `discount` | float | Sí | Descuento en monto (no porcentaje) |
| `purchase` | float | Sí | Monto de compra neto (ver fórmula) |
| `code` | string | No | Código del ítem (máx. 25 chars) |

> **Regla:** `purchase = quantity × unit_price - discount` (tolerancia ±0.01)
> **Regla:** `discount ≤ quantity × unit_price`

---

### `excluded_subject` — Sujeto excluido

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | Tipo de documento (ej. `"13"` = DUI) |
| `document_number` | string | Sí | Número de documento |
| `name` | string | No | Nombre del sujeto excluido |
| `activity_code` | string | No | Código de actividad económica |
| `activity_description` | string | No | Descripción de actividad |
| `address` | `Address` | No | Dirección |
| `phone` | string | No | Teléfono |
| `email` | string | No | Correo electrónico |

---

### `summary` — Resumen

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_non_subject` | float | Sí | Total no sujeto (generalmente `0`) |
| `total_exempt` | float | Sí | Total exento (generalmente `0`) |
| `total_taxed` | float | Sí | Total gravado — **DEBE ser `0`** en FSE |
| `sub_total_sales` | float | Sí | Subtotal de ventas |
| `non_subject_discount` | float | Sí | Descuento a nivel de resumen |
| `exempt_discount` | float | Sí | Descuento exento |
| `discount_percentage` | float | Sí | Porcentaje de descuento |
| `total_discount` | float | Sí | Total de descuentos |
| `sub_total` | float | Sí | Subtotal neto (ver fórmula) |
| `total_operation` | float | Sí | Total de la operación |
| `total_non_taxed` | float | Sí | Total no afecto |
| `total_to_pay` | float | Sí | Total a pagar (ver fórmula) |
| `operation_condition` | integer | Sí | `1`=Contado, `2`=Crédito |
| `payment_types` | `Payment[]` | Sí | Formas de pago |
| `total_in_words` | string | No | Total en letras |
| `total_purchase` | float | Sí | Total de compra = Σ item.purchase |
| `iva_retention` | float | Sí | Retención de IVA (≥ 0) |
| `income_retention` | float | Sí | Retención de renta (≥ 0) |
| `observations` | string | No | Observaciones |

---

## Reglas de cálculo

### Por ítem

```
// Monto de compra por ítem:
purchase = quantity × unit_price - discount
// tolerancia: ±0.01

// El descuento no puede exceder el valor bruto:
discount ≤ quantity × unit_price
```

### Total de compra

```
Σ item.purchase = summary.total_purchase  // tolerancia ±0.01
```

### Descuento total

```
total_discount = Σ item.discount + summary.non_subject_discount
// tolerancia: ±0.01
```

### Subtotal

```
sub_total = total_purchase - non_subject_discount
// tolerancia: ±0.01
```

### Total a pagar

```
total_to_pay = total_purchase - iva_retention - income_retention
// Nota: FSE NO aplica IVA; total_taxed DEBE ser 0
// iva_retention ≥ 0
// income_retention ≥ 0
```

### Regla exclusiva: sin IVA

```
// PROHIBIDO: total_taxed > 0 en FSE
// FSE documenta compras a no contribuyentes — no hay IVA de por medio
```

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| `total_taxed` | **Debe ser `0`** — FSE no aplica IVA |
| `iva_retention` | Debe ser ≥ 0 |
| `income_retention` | Debe ser ≥ 0 |
| Items | Al menos 1 |
| `purchase` | Debe ser > 0 por ítem |
| `quantity` | Debe ser > 0 |
| `unit_price` | Debe ser > 0 |
| Fecha | No futura |
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
      "tipoDte": "14",
      "numeroControl": "DTE-14-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "fecEmi": "2024-01-15"
    },
    "emisor": { },
    "sujetoExcluido": {
      "tipoDocumento": "13",
      "numDocumento": "01234567-8",
      "nombre": "Juan Carlos Pérez",
      "codActividad": "46900",
      "descActividad": "Servicios profesionales",
      "direccion": { },
      "telefono": "22345678",
      "correo": "juan@example.com"
    },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoItem": 2,
        "cantidad": 1,
        "codigo": null,
        "uniMedida": 59,
        "descripcion": "Servicio de consultoría",
        "precioUni": 100.00,
        "montoDescu": 0,
        "compra": 100.00
      }
    ],
    "resumen": {
      "totalCompra": 100.00,
      "descu": 0,
      "totalDescu": 0,
      "subTotal": 100.00,
      "ivaRete1": 0,
      "reteRenta": 0,
      "totalPagar": 100.00,
      "totalLetras": "CIEN DOLARES CON 00/100",
      "condicionOperacion": 1,
      "pagos": [{ "codigo": "01", "montoPago": 100.00 }],
      "observaciones": null
    },
    "apendice": null
  }
}
```

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `FSETaxInvalidTaxedSale` | `total_taxed > 0` — FSE no permite IVA |
| `400` | `FSETaxInvalidTotalToPay` | `total_to_pay ≠ total_purchase - retenciones` |
| `400` | `InvalidItemPurchase` | `purchase ≠ quantity × unit_price - discount` |
| `400` | `ExcessiveItemDiscount` | `discount > quantity × unit_price` |
| `400` | `InvalidTotalPurchase` | `total_purchase ≠ Σ item.purchase` |
| `400` | `InvalidSubTotal` | `sub_total ≠ total_purchase - non_subject_discount` |
| `400` | `InvalidTotalDiscount` | `total_discount` incorrecto |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó |

---

## Ejemplo completo

### Request

```json
{
  "excluded_subject": {
    "document_type": "13",
    "document_number": "XXXXXXXX-X",
    "name": "Empresa ABC",
    "commercial_name": "Empresa ABC",
    "activity_code": null,
    "activity_description": "Actividad comercial",
    "address": {
      "department": "09",
      "municipality": "10",
      "complement": "Calle Principal, Local 1"
    },
    "phone": null,
    "email": "correo@ejemplo.com"
  },
  "items": [
    {
      "quantity": 2,
      "description": "315122-111/Zapatilla AF1 Blanco Unisex",
      "unit_price": 65,
      "type": 1,
      "unit_measure": 59,
      "discount": 10,
      "purchase": 120,
      "code": "CODIGO-PRODUCTO"
    }
  ],
  "summary": {
    "total_purchase": 120,
    "total_discount": 10,
    "sub_total": 120,
    "iva_retention": 0,
    "income_retention": 0,
    "total_to_pay": 120,
    "operation_condition": 1,
    "payment_types": [
      {
        "code": "01",
        "amount": 120,
        "reference": null,
        "term": null,
        "period": null
      }
    ],
    "non_subject_discount": 0
  }
}
```

**Verificación de cálculos:**
- `purchase = quantity × unit_price - discount = 2 × 65 - 10 = 120`
- `total_purchase = Σ item.purchase = 120`
- `sub_total = total_purchase - non_subject_discount = 120 - 0 = 120`
- `total_to_pay = total_purchase - iva_retention - income_retention = 120 - 0 - 0 = 120`
- `total_taxed = 0` (FSE no aplica IVA)
