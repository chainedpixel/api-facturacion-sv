# Anulación de DTE

## Descripción

La **Anulación** permite invalidar un DTE previamente emitido y transmitido a Hacienda. Dependiendo del tipo de anulación, puede ser obligatorio indicar un documento de reemplazo.

**Soporta contingencia:** No

---

## Endpoint

```
POST /api/v1/dte/invalidation
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 200 OK`

---

## Request Body

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `generation_code` | string | Sí | UUID del DTE a anular (`codigoGeneracion`) |
| `reason` | `Reason` | Sí | Motivo y responsables de la anulación |
| `replacement_generation_code` | string | No* | UUID del DTE de reemplazo. **Obligatorio para tipos `1` y `3`**. Prohibido para tipo `2` |

---

### `reason` — Motivo de anulación

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo de anulación (ver tabla abajo) |
| `responsible_name` | string | Sí | Nombre del responsable de la anulación |
| `responsible_doc_type` | string | Sí | Tipo de documento del responsable (ej. `"13"` = DUI) |
| `responsible_num_doc` | string | Sí | Número de documento del responsable |
| `requestor_name` | string | Sí | Nombre de quien solicita la anulación |
| `requestor_doc_type` | string | Sí | Tipo de documento del solicitante |
| `requestor_num_doc` | string | Sí | Número de documento del solicitante |
| `reason_field` | string | No* | Descripción del motivo (texto libre). **Obligatorio para tipo `3`** |

#### Tipos de anulación

| `type` | Descripción | `replacement_generation_code` | `reason_field` |
|--------|-------------|-------------------------------|----------------|
| `1` | Error en datos — se emitió un DTE de reemplazo | **Obligatorio** | Opcional |
| `2` | Anulación sin reemplazo — el DTE se elimina sin sustituir | **Prohibido** | Opcional |
| `3` | Otro motivo — anulación con reemplazo y justificación | **Obligatorio** | **Obligatorio** |

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| `generation_code` | Debe ser un UUID válido de un DTE previamente emitido |
| `reason.type` | Valores válidos: `1`, `2` o `3` |
| `reason.responsible_name` | No vacío |
| `reason.responsible_doc_type` | Tipo de documento válido |
| `reason.responsible_num_doc` | Número de documento válido según tipo |
| `reason.requestor_name` | No vacío |
| `replacement_generation_code` | **Obligatorio** para tipos `1` y `3`; **prohibido** para tipo `2` |
| `reason.reason_field` | **Obligatorio** para tipo `3`; opcional para `1` y `2` |
| El DTE referenciado | Debe existir y estar en estado válido (no ya anulado ni rechazado) |
| Fecha límite — Factura Electrónica (`01`) | Se puede anular hasta **90 días** después de la fecha de emisión |
| Fecha límite — Resto de DTEs | Se puede anular hasta **24 horas** después de la fecha de emisión |

---

## Respuesta exitosa

**HTTP 200 OK**

```json
{
  "success": true,
  "data": {
    "identificacion": {
      "version": 2,
      "ambiente": "01",
      "codigoGeneracion": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "fecAnula": "2024-01-15",
      "horAnula": "14:30:00"
    },
    "emisor": {
      "nit": "...",
      "nombre": "...",
      "tipoEstablecimiento": "...",
      "correo": "..."
    },
    "documento": {
      "tipoDte": "01",
      "codigoGeneracion": "UUID-DEL-DTE-ORIGINAL",
      "codigoGeneracionR": null,
      "tipoAnulacion": 1,
      "fechaEmision": "2024-01-10",
      "montoIva": 13.00,
      "codigoGeneracionR": null
    },
    "motivo": {
      "tipoAnulacion": 1,
      "motivoAnulacion": "Error en datos del receptor",
      "nombreResponsable": "Carlos García",
      "tipDocResponsable": "13",
      "numDocResponsable": "01234567-8",
      "nombreSolicita": "María López",
      "tipDocSolicita": "13",
      "numDocSolicita": "08765432-1"
    }
  }
}
```

> **Nota:** La anulación no incluye `reception_stamp` ni `qr_link` ya que no genera un nuevo DTE sino que invalida uno existente.

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | Código específico | DTE no encontrado para el `generation_code` |
| `400` | Código específico | Fecha de anulación anterior a la emisión del DTE |
| `400` | Código específico | Tipo de anulación inválido |
| `400` | `HACIENDA_<codigo>` | Hacienda rechazó la anulación |
| `500` | `SYSTEM_ERROR` | Error interno |

---

## Ejemplo completo

### Request — Tipo 1: Error en datos (con reemplazo)

> `replacement_generation_code` es **obligatorio** para tipo `1`.

```json
{
  "generation_code": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
  "replacement_generation_code": "9A7B6C5D-3E2F-1A0B-CDEF-1234567890AB",
  "reason": {
    "type": 1,
    "responsible_name": "Carlos García",
    "responsible_doc_type": "13",
    "responsible_num_doc": "01234567-8",
    "requestor_name": "María López",
    "requestor_doc_type": "13",
    "requestor_num_doc": "08765432-1"
  }
}
```

---

### Request — Tipo 2: Anulación sin reemplazo

> `replacement_generation_code` debe **omitirse** para tipo `2`.

```json
{
  "generation_code": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
  "reason": {
    "type": 2,
    "responsible_name": "Carlos García",
    "responsible_doc_type": "13",
    "responsible_num_doc": "01234567-8",
    "requestor_name": "María López",
    "requestor_doc_type": "13",
    "requestor_num_doc": "08765432-1"
  }
}
```

---

### Request — Tipo 3: Otro motivo (con reemplazo y justificación)

> `replacement_generation_code` y `reason_field` son **ambos obligatorios** para tipo `3`.

```json
{
  "generation_code": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
  "replacement_generation_code": "9A7B6C5D-3E2F-1A0B-CDEF-1234567890AB",
  "reason": {
    "type": 3,
    "responsible_name": "Carlos García",
    "responsible_doc_type": "13",
    "responsible_num_doc": "01234567-8",
    "requestor_name": "María López",
    "requestor_doc_type": "13",
    "requestor_num_doc": "08765432-1",
    "reason_field": "Se emitió documento corregido por cambio de condiciones pactadas"
  }
}
```

---

## Consultar DTE

Para obtener el UUID de un DTE antes de anularlo:

```
GET /api/v1/dte/{codigoGeneracion}
Authorization: Bearer <token>
```

O listar todos los DTEs:

```
GET /api/v1/dte
Authorization: Bearer <token>
```
