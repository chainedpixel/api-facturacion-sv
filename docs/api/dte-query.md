# Consulta de DTEs

## Descripción

La API expone cuatro endpoints de consulta, todos protegidos con `Authorization: Bearer <token>` y limitados a los DTEs de la sucursal autenticada (salvo el parámetro `all` del listado):

- **Obtener un DTE específico** por su `codigoGeneracion` (`GET /api/v1/dte/{id}`)
- **Obtener el Archivo DTE** de un documento (`GET /api/v1/dte/{id}/archivo`)
- **Obtener los eventos** de un documento, como su invalidación (`GET /api/v1/dte/{id}/eventos`)
- **Listar todos los DTEs** con filtros opcionales y paginación (`GET /api/v1/dte`)

---

## Endpoints

### 1. Obtener DTE por código de generación

```
GET /api/v1/dte/{codigoGeneracion}
Authorization: Bearer <token>
```

**Respuesta exitosa:** `HTTP 200 OK`

#### Parámetro de ruta

| Parámetro | Tipo | Descripción |
|-----------|------|-------------|
| `{codigoGeneracion}` | string (UUID) | El `codigoGeneracion` del DTE a consultar |

#### Respuesta exitosa

```json
{
  "success": true,
  "data": {
    "control_number": "DTE-01-N0010001-000000000000001",
    "generation_code": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
    "reception_stamp": "20240115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
    "transmission": "NORMAL",
    "status": "RECEIVED",
    "created_at": "2024-01-15T14:30:00Z",
    "updated_at": "2024-01-15T14:30:05Z",
    "json_data": { /* Estructura completa del DTE según tipo */ }
  }
}
```

#### Campos de respuesta

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `control_number` | string | Número de control del DTE |
| `generation_code` | string | UUID único del DTE |
| `reception_stamp` | string \| null | Sello de recibido de Hacienda. `null` si está en contingencia pendiente |
| `transmission` | string | Tipo de transmisión: `"NORMAL"` o `"CONTINGENCY"` |
| `status` | string | Estado actual del DTE (ver tabla de estados) |
| `created_at` | string (ISO 8601) | Fecha/hora de creación |
| `updated_at` | string (ISO 8601) | Fecha/hora de última actualización |
| `json_data` | object | Estructura completa del DTE en formato MH (varía según tipo) |

#### Estados posibles (`status`)

| Valor | Descripción |
|-------|-------------|
| `RECEIVED` | Procesado y aceptado por Hacienda |
| `PENDING` | En contingencia pendiente de transmisión a Hacienda |
| `REJECTED` | Rechazado por Hacienda |
| `INVALIDATED` | Anulado exitosamente |
| `EXPIRED` | Generado en contingencia y no transmitido dentro de las 72 horas desde el sello del evento de contingencia; se considera no emitido y no tiene sello de recepción |

> Esta respuesta no incluye la fecha de procesamiento (`processing_date`) ni la firma del documento; para obtener el documento firmado y sellado se usa `GET /api/v1/dte/{id}/archivo`.

#### Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `FAILEDTOGETDTE` | `codigoGeneracion` no corresponde a un DTE de la sucursal (`error.details` indica que no existe registro) |
| `401` | `UNAUTHORIZED` | Token ausente o inválido |
| `500` | `SYSTEM_ERROR` | Error interno |

---

### 2. Obtener el Archivo DTE

```
GET /api/v1/dte/{codigoGeneracion}/archivo
Authorization: Bearer <token>
```

**Respuesta exitosa:** `HTTP 200 OK`

Devuelve el **Archivo DTE** listo para entregar al receptor: la estructura firmada del documento, íntegra y sin alteraciones, más los atributos `firmaElectronica` (JWS) y `selloRecibido`. A diferencia del resto de endpoints, la respuesta **no** se envuelve en `success`/`data`: el cuerpo es el JSON del archivo tal cual, con `Content-Type: application/json` y `Content-Disposition: attachment; filename="<codigoGeneracion>.json"`.

```json
{
  "identificacion": { },
  "emisor": { },
  "receptor": { },
  "cuerpoDocumento": [ ],
  "resumen": { },
  "firmaElectronica": "<JWS>",
  "selloRecibido": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"
}
```

#### Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `FAILEDTOGETDTE` | El DTE no existe en la sucursal |
| `400` | `ARCHIVENOTAVAILABLE` | El documento se emitió antes de guardarse la firma electrónica, o la firma guardada no es válida |

---

### 3. Obtener los eventos de un DTE

```
GET /api/v1/dte/{codigoGeneracion}/eventos
Authorization: Bearer <token>
```

**Respuesta exitosa:** `HTTP 200 OK`

Devuelve la lista de eventos transmitidos sobre el documento (hoy, su invalidación). Si el documento existe pero no tiene eventos, `data` es una lista vacía. Cada elemento tiene la misma estructura que la respuesta de la [anulación](./invalidation.md#respuesta-exitosa): datos del evento, `hacienda_response` y `archivo_dte`.

```json
{
  "success": true,
  "data": [
    {
      "generation_code": "A1B2C3D4-0000-4000-8000-1234567890AB",
      "branch_id": 1,
      "event_type": "INVALIDATION",
      "related_dte_id": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
      "replacement_dte_id": null,
      "reception_stamp": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
      "processing_date": "15/01/2026 14:30:05",
      "status": "RECEIVED",
      "created_at": "2026-01-15T14:30:06-06:00",
      "hacienda_response": { },
      "archivo_dte": { }
    }
  ]
}
```

#### Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `FAILEDTOGETDTE` | El DTE no existe en la sucursal |

---

### 4. Listar DTEs

```
GET /api/v1/dte
Authorization: Bearer <token>
```

**Respuesta exitosa:** `HTTP 200 OK`

Retorna todos los DTEs de la sucursal autenticada, con soporte para filtros y paginación.

#### Query Parameters

| Parámetro | Tipo | Requerido | Descripción |
|-----------|------|-----------|-------------|
| `page` | integer | No | Número de página. Por defecto: `1` |
| `page_size` | integer | No | Cantidad de registros por página. Por defecto: `5`. Un valor inválido o menor a 1 se reemplaza por el valor por defecto (igual que `page`, que se reemplaza por `1`) |
| `status` | string | No | Filtrar por estado (ver valores válidos) |
| `type` | string | No | Filtrar por tipo de DTE. Soporta múltiples separados por coma |
| `transmission` | string | No | Filtrar por tipo de transmisión |
| `startDate` | string (RFC3339) | No | Inicio del rango; se compara con la fecha de registro del documento (`created_at`) |
| `endDate` | string (RFC3339) | No | Fecha de fin del rango de consulta |
| `all` | boolean | No | Si es `true`, no se filtra por sucursal y se listan los DTEs de todas las sucursales. El servicio no aplica una verificación de permisos adicional para este parámetro |

#### Valores válidos por parámetro

**`status`** (case-insensitive):

| Valor | Descripción |
|-------|-------------|
| `RECEIVED` | Aceptados por Hacienda |
| `PENDING` | En contingencia, pendientes |
| `REJECTED` | Rechazados por Hacienda |
| `INVALIDATED` | Anulados |
| `EXPIRED` | Contingencia vencida (72 horas) |

**`transmission`** (case-insensitive):

| Valor | Descripción |
|-------|-------------|
| `NORMAL` | Transmisión directa a Hacienda |
| `CONTINGENCY` | Enviados vía contingencia |

**`type`** — Códigos de tipo de DTE:

| Código | Tipo de DTE |
|--------|-------------|
| `01` | Factura Electrónica |
| `03` | Comprobante de Crédito Fiscal |
| `04` | Nota de Remisión Electrónica |
| `05` | Nota de Crédito Electrónica |
| `06` | Nota de Débito Electrónica |
| `07` | Comprobante de Retención Electrónico |
| `08` | Comprobante de Liquidación Electrónico |
| `09` | Doc. Contable de Liquidación Electrónico |
| `11` | Factura de Exportación Electrónica |
| `14` | Factura de Sujeto Excluido |
| `15` | Comprobante de Donación Electrónico |

Para múltiples tipos: `?type=01,03,06`

**`startDate` / `endDate`** — Formato RFC3339:

```
?startDate=2024-01-01T00:00:00Z&endDate=2024-01-31T23:59:59Z
```

#### Respuesta exitosa

```json
{
  "success": true,
  "data": {
    "documents": [
      {
        "status": "RECEIVED",
        "transmission_type": "NORMAL",
        "document": { /* Estructura completa del DTE */ }
      }
    ],
    "summary": {
      "total": 150,
      "received": 140,
      "invalid": 5,
      "rejected": 3,
      "pending": 2,
      "by_contingency": 10,
      "by_normal": 140
    },
    "pagination": {
      "total_pages": 30,
      "page": 1,
      "page_size": 5
    }
  }
}
```

#### Campos de respuesta

**`documents[]`**

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `status` | string | Estado del DTE (`RECEIVED`, `PENDING`, `REJECTED`, `INVALIDATED`, `EXPIRED`) |
| `transmission_type` | string | Tipo de transmisión (`NORMAL`, `CONTINGENCY`) |
| `document` | object | Estructura completa del DTE en formato MH (sin firma ni sello; para el Archivo DTE usar `/archivo`) |

Los documentos se ordenan por fecha de registro descendente.

**`summary`**

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `total` | integer | Total de DTEs en el resultado (aplicando filtros) |
| `received` | integer | Cantidad con estado `RECEIVED` |
| `invalid` | integer | Cantidad con estado `INVALIDATED` |
| `rejected` | integer | Cantidad con estado `REJECTED` |
| `pending` | integer | Cantidad con estado `PENDING` |

> `summary` no tiene un contador para `EXPIRED`: esos documentos cuentan en `total` y `by_contingency`, pero no en `received`, `invalid`, `rejected` ni `pending`.

| `by_contingency` | integer | Con transmisión `CONTINGENCY` |
| `by_normal` | integer | Con transmisión `NORMAL` |

**`pagination`**

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `total_pages` | integer | Total de páginas disponibles |
| `page` | integer | Página actual |
| `page_size` | integer | Registros por página |

---

## Respuestas de error

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `INVALIDQUERYPARAM` | `status`, `transmission` o `type` con valor inválido |
| `400` | `INVALIDQUERYPARAM` | `startDate` o `endDate` no tiene formato RFC3339 |
| `401` | `UNAUTHORIZED` | Token ausente o inválido |
| `500` | `SYSTEM_ERROR` | Error interno |

---

## Ejemplos

### Obtener un DTE específico

```
GET /api/v1/dte/3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C
Authorization: Bearer eyJhbGci...
```

### Obtener el Archivo DTE de un documento

```
GET /api/v1/dte/3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C/archivo
Authorization: Bearer eyJhbGci...
```

### Obtener los eventos de un documento

```
GET /api/v1/dte/3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C/eventos
Authorization: Bearer eyJhbGci...
```

### Listar DTEs con paginación

```
GET /api/v1/dte?page=2&page_size=10
Authorization: Bearer eyJhbGci...
```

### Filtrar por estado

```
GET /api/v1/dte?status=RECEIVED&page=1&page_size=20
Authorization: Bearer eyJhbGci...
```

### Filtrar por tipo de DTE

```
GET /api/v1/dte?type=01
Authorization: Bearer eyJhbGci...
```

### Filtrar por múltiples tipos

```
GET /api/v1/dte?type=01,03,06&status=RECEIVED
Authorization: Bearer eyJhbGci...
```

### Filtrar por rango de fechas

```
GET /api/v1/dte?startDate=2024-01-01T00:00:00Z&endDate=2024-01-31T23:59:59Z&type=01
Authorization: Bearer eyJhbGci...
```

### Filtrar DTEs en contingencia

```
GET /api/v1/dte?transmission=CONTINGENCY&status=PENDING
Authorization: Bearer eyJhbGci...
```
