# Comprobante de Retención Electrónico (CRE)

## Descripción

El **Comprobante de Retención Electrónico** (código DTE `07`) documenta las retenciones de IVA que un contribuyente aplica a sus proveedores. Cada ítem referencia un documento (físico o electrónico) sobre el cual se está practicando la retención.

**Soporta contingencia:** No

---

## Endpoint

```
POST /api/v1/dte/retention
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `RetentionItem[]` | Sí | Documentos sobre los que se retiene |
| `receiver` | `Receiver` | No | Datos del receptor (proveedor al que se le retiene) |
| `summary` | `RetentionSummary` | No | Totales (se calcula automáticamente si se omite) |
| `extension` | `Extension` | No* | **Obligatoria si `total_operation ≥ 1095.00`** |
| `appendixes` | `Appendix[]` | No | Apéndices |

---

### `items[]` — Documentos retenidos

Cada ítem representa un documento sobre el que se practica la retención.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo: `1` = Físico, `2` = Electrónico |
| `document_number` | string | Sí | Número del documento. Físico: número correlativo (≤ 20 chars). Electrónico: UUID |
| `description` | string | Sí | Descripción de la operación |
| `retention_code` | string | Sí | Código de retención: `"22"` = 1%, `"C4"` = 13% |
| `iva_amount` | float | No* | IVA retenido calculado. **Requerido para documentos físicos** |
| `taxed_amount` | float | No* | Monto sujeto a retención. **Requerido para documentos físicos** |
| `emission_date` | string | No* | Fecha del documento (`YYYY-MM-DD`). **Requerida para documentos físicos** |
| `dte_type` | string | No* | Tipo de DTE del documento electrónico. Requerido para tipo `2` |

#### Tipo `1` (Documento Físico)

Se requieren: `document_number`, `description`, `retention_code`, `taxed_amount`, `iva_amount`, `emission_date`.

El `iva_amount` es calculado y validado por el sistema:
```
iva_amount = taxed_amount × GetRetentionRate[retention_code]
```

#### Tipo `2` (Documento Electrónico)

Solo requiere: `document_number` (UUID), `description`, `retention_code`, `dte_type`.
Los montos (`taxed_amount`, `iva_amount`) se obtienen automáticamente de la base de datos del DTE referenciado.

---

### `summary` — Totales de retención

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_retention_amount` | float | No | Suma de todos los `taxed_amount` |
| `total_retention_iva` | float | No | Suma de todos los `iva_amount` |

---

## Reglas de cálculo

### IVA retenido por ítem

```
// Tasas disponibles:
// Código "22" (Retención 1%):  iva_amount = taxed_amount × 0.01
// Código "C4" (Retención 13%): iva_amount = taxed_amount × 0.13
// Código "C9" (Otras):         sin validación de tasa

// tolerancia: ±0.01
```

### Totales del resumen

```
Σ item.taxed_amount = summary.total_retention_amount  // tolerancia ±0.01
Σ item.iva_amount   = summary.total_retention_iva     // tolerancia ±0.01
```

### Rango de fechas permitido para documentos físicos

El sistema valida que la fecha del documento retenido esté dentro de un período permitido respecto a la fecha de emisión de la retención:

```
Regla 1: La fecha del documento está en el MISMO mes que la retención
         → Siempre válido

Regla 2: La fecha del documento está en el MES ANTERIOR a la retención
         → Válido solo si la retención se emite dentro de los primeros
           10 días hábiles (lunes a viernes) del mes siguiente

Regla 3: Cualquier otro caso → Inválido
```

**Ejemplo:**
- Documento de enero 2024 → Se puede retener en enero 2024 (misma mes)
- Documento de enero 2024 → Se puede retener hasta los primeros 10 días hábiles de febrero 2024
- Documento de diciembre 2023 → Se puede retener en diciembre 2023 o primeros 10 días hábiles de enero 2024

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Items | Mínimo 1 |
| `retention_code` | Solo `"22"`, `"C4"` o `"C9"` |
| `iva_amount` | Debe coincidir con `taxed_amount × tasa` |
| Fecha del documento | Dentro del período permitido |
| Documentos físicos | `taxed_amount`, `iva_amount` y `emission_date` obligatorios |
| Documentos electrónicos | `document_number` en formato UUID |
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
      "tipoDte": "07",
      "numeroControl": "DTE-07-N0010001-000000000000001",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "fecEmi": "2024-01-15"
    },
    "emisor": { },
    "receptor": { },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoDte": "01",
        "tipoDoc": 1,
        "numDocumento": "00001",
        "fechaEmision": "2024-01-10",
        "montoSujetoGrav": 500.00,
        "codigoRetencionMH": "22",
        "ivaRetenido": 5.00,
        "descripcion": "Servicios de consultoría"
      }
    ],
    "resumen": {
      "totalSujetoRetencion": 500.00,
      "totalIVAretenido": 5.00,
      "totalIVAretenidoLetras": "CINCO DOLARES CON 00/100"
    },
    "apendice": []
  }
}
```

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `InvalidRetentionIVA` | `iva_amount ≠ taxed_amount × tasa` |
| `400` | `DateOutOfAllowedRange` | Fecha del documento fuera del período permitido |
| `400` | `RequiredField` | Faltan campos requeridos para documentos físicos |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó |

---

## Ejemplos

### Ejemplo 1: Retención sobre documento físico

```json
{
    "items": [
        {
            "type": 1,
            "document_number": "S221001345",
            "description": "Compra de equipos informáticos",
            "retention_code": "22",
            "taxed_amount": 226.50,
            "iva_amount": 2.26,
            "emission_date": "2026-05-05",
            "dte_type": "03"
        }
    ],
    "receiver": {
        "document_type": "36",
        "document_number": "08211311610013",
        "nrc": "1027018",
        "name": "Nombre de Ejemplo",
        "commercial_name": "Comercial de Ejemplo",
        "activity_code": "47522",
        "activity_description": "Venta al por menor de artículos de ferretería",
        "address": {
            "department": "08",
            "municipality": "23",
            "complement": "Dirección de ejemplo"
        },
        "phone": "22220000",
        "email": "ejemplo@example.com"
    },
    "summary": {
        "total_retention_amount": 226.50,
        "total_retention_iva": 2.26
    },
    "extension": {
        "delivery_name": "Nombre de Entrega",
        "delivery_document": "00000000-0",
        "receiver_name": "Nombre de Ejemplo",
        "receiver_document": "08211311610013"
    }
}
```

---

### Ejemplo 2: Retención sobre documento electrónico

```json
{
    "receiver": {
        "document_type": "36",
        "document_number": "06141101690011",
        "nrc": "1937",
        "name": "Empresa de Ejemplo S.A. de C.V.",
        "activity_code": "10005",
        "activity_description": "Servicios profesionales",
        "commercial_name": "Comercial Ejemplo",
        "address": {
            "department": "06",
            "municipality": "20",
            "complement": "Calle de Ejemplo #321"
        },
        "phone": "22229999",
        "email": "ejemplo@example.com"
    },
    "items": [
        {
            "type": 2,
            "document_number":"E1E31344-D7D8-466D-BF99-FE1BCFEAD10C",
            "description": "Retención IVA por compra de bienes",
            "retention_code": "C4",
            "iva_amount": 13.00,
            "taxed_amount": 100.00,
            "emission_date": "2026-05-06",
            "dte_type": "03"
        }
    ],
    "summary": {
        "total_retention_amount": 100.00,
        "total_retention_iva": 13.00
    }
}
```