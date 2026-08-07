# API de Facturación Electrónica — Documentación para consumidores

Esta documentación describe cómo interactuar con la API REST de facturación electrónica para El Salvador (Ministerio de Hacienda). Está orientada a **consumidores de la API** — equipos de desarrollo que integran sus sistemas con este servicio.

---

## Base URL

```
https://<host>/api/v1
```

Todos los endpoints descritos en esta documentación son relativos a esta base URL.

---

## Autenticación

La API usa **JWT Bearer Token**. Para obtenerlo:

### `POST /api/v1/auth/login`

**Body (JSON):**

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `api_key` | string | Sí | Llave de API asignada al sistema |
| `api_secret` | string | Sí | Secreto de API |
| `nit` | string | Sí | NIT del contribuyente |
| `nrc` | string | Sí | NRC del contribuyente |
| `password` | string | Sí | Contraseña de Hacienda |

**Respuesta exitosa (`200 OK`):**

```json
{
  "success": true,
  "data": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Uso del token:** Incluir en el header `Authorization` de todas las peticiones protegidas:

```
Authorization: Bearer <token>
```

---

## Formato de respuestas

### Respuesta exitosa — DTE creado

Cuando se crea un DTE exitosamente, la respuesta tiene la siguiente estructura:

```json
{
  "success": true,
  "reception_stamp": "2024ABC123...",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX&fechaEmi=2024-01-15",
  "data": { }
}
```

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `success` | boolean | Siempre `true` en respuestas exitosas |
| `reception_stamp` | string \| null | Sello de recepción emitido por Hacienda. `null` si el documento fue a contingencia |
| `qr_link` | string \| null | URL pública para consultar el DTE en el portal de Hacienda |
| `data` | object | Estructura completa del DTE según el esquema de Hacienda |

### Respuesta exitosa — otras operaciones

```json
{
  "success": true,
  "data": { }
}
```

### Respuesta de error

```json
{
  "success": false,
  "error": {
    "message": "Descripción del error",
    "code": "CODIGO_ERROR",
    "details": ["Detalle 1", "Detalle 2"]
  }
}
```

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `error.message` | string | Descripción legible del error |
| `error.code` | string | Código identificador del error (ver tabla abajo) |
| `error.details` | string[] | Detalles adicionales; puede ser vacío |

---

## Tipos de error

| Categoría | HTTP Status | Código | Descripción |
|-----------|-------------|--------|-------------|
| Validación | `400` | Específico por campo | El request contiene campos inválidos o viola reglas de negocio |
| Hacienda rechazó | `400` | `HACIENDA_<codigoMsg>` | Hacienda procesó el documento pero lo rechazó |
| Error HTTP de Hacienda | Varía | `HACIENDA_<httpCode>` | Hacienda retornó un error HTTP (5xx, 429, etc.) |
| Error de servicio | `400` | `<CODIGO_SERVICIO>` | Error de negocio interno (número secuencial, mapeo, etc.) |
| Error del sistema | `500` | `SYSTEM_ERROR` | Error interno inesperado del servidor |
| No autorizado | `401` | `UNAUTHORIZED` | Token JWT ausente, expirado o inválido |
| No encontrado | `404` | `NOT_FOUND` | Recurso no encontrado |
| Timeout | `408` | `REQUEST_TIMEOUT` | La petición excedió el tiempo límite |

---

## Endpoints disponibles

### Autenticación (públicos)

| Método | Path | Descripción |
|--------|------|-------------|
| `POST` | `/api/v1/auth/login` | Obtener token JWT |
| `POST` | `/api/v1/auth/register` | Registrar nuevo sistema |

### DTE — Creación (requieren autenticación)

| Método | Path | DTE | Contingencia |
|--------|------|-----|:---:|
| `POST` | `/api/v1/dte/invoices` | Factura Electrónica | Sí |
| `POST` | `/api/v1/dte/ccf` | Comprobante de Crédito Fiscal | Sí |
| `POST` | `/api/v1/dte/creditnote` | Nota de Crédito Electrónica | No |
| `POST` | `/api/v1/dte/debitnote` | Nota de Débito Electrónica | Sí |
| `POST` | `/api/v1/dte/fse` | Factura de Sujeto Excluido | Sí |
| `POST` | `/api/v1/dte/retention` | Comprobante de Retención | No |
| `POST` | `/api/v1/dte/remissionnote` | Nota de Remisión Electrónica | Sí |
| `POST` | `/api/v1/dte/invalidation` | Anulación de DTE | No |

### DTE — Consulta (requieren autenticación)

| Método | Path | Descripción |
|--------|------|-------------|
| `GET` | `/api/v1/dte` | Listar todos los DTEs del sistema |
| `GET` | `/api/v1/dte/{id}` | Obtener DTE por código de generación (UUID) |

### Otros

| Método | Path | Descripción |
|--------|------|-------------|
| `GET` | `/api/v1/health` | Estado del servicio |
| `GET` | `/api/v1/metrics` | Métricas de uso de endpoints |

---

## Contingencia

Cuando un DTE que soporta contingencia no puede transmitirse a Hacienda (por problemas de red, indisponibilidad del servicio, timeout, etc.), el sistema lo almacena automáticamente y retorna **HTTP 201** con la estructura del DTE — pero sin `reception_stamp`.

### Cuándo se activa

| Condición | Tipo de contingencia |
|-----------|---------------------|
| Error de red (connection refused, broken pipe) | `1` — Falla de conexión al sistema |
| Red no disponible (no route to host, timeout de red) | `2` — Falla de servicio de Internet |
| HTTP 502/503/504 de Hacienda | `3` — No disponibilidad de MH |
| HTTP 408/504 (timeout de petición) | `3` — No disponibilidad de MH |
| HTTP 429 (rate limiting) | `3` — No disponibilidad de MH |
| Mantenimiento / sobrecarga (según descripción) | `3` — No disponibilidad de MH |
| Timeout de contexto (deadline exceeded) | `3` — No disponibilidad de MH |
| HTTP 401/403 con token inválido | `1` — Falla de conexión al sistema |
| Cualquier otro error no clasificado | `5` — Otro motivo |

### Cuándo NO se activa

- El documento tiene errores de validación (campos incorrectos)
- Hacienda devuelve estado `RECHAZADO`
- Errores de negocio internos (mapeo, número secuencial, etc.)

### Respuesta durante contingencia

La respuesta es `HTTP 201` con el DTE completo pero `reception_stamp: null`:

```json
{
  "success": true,
  "reception_stamp": null,
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?...",
  "data": {
    "identificacion": {
      "tipoContingencia": 3,
      "motivoContin": "No disponibilidad del servicio de MH",
      ...
    },
    ...
  }
}
```

---

## Validaciones generales

La API implementa múltiples capas de validación antes de enviar cualquier documento a Hacienda.

### Tipos de errores de validación

| Categoría | Descripción |
|-----------|-------------|
| `VALIDATION` | Campos con formato incorrecto, longitud excedida o campos obligatorios faltantes |
| `BUSINESS` | Datos estructuralmente correctos pero que violan reglas de negocio o fiscales (ej. totales no coinciden, documento relacionado no encontrado) |
| `SYSTEM` | Errores internos inesperados (base de datos, red, etc.) |

### Reglas comunes

- **NIT:** 14 dígitos, sin guiones.
- **DUI:** 9 dígitos, sin guiones.
- **NRC:** Máximo 8 caracteres.
- **Textos:** Evitar caracteres especiales no estándar o emojis; respetar los límites de longitud.
- **Cálculos:** La API valida que `Precio Unitario × Cantidad = Venta Gravada` (tolerancia ±$0.01). Enviar los cálculos ya verificados para evitar rechazos por redondeo.

---

## Números de control

El Ministerio de Hacienda exige numeración única, consecutiva y sin saltos por tipo de documento. La API gestiona esto internamente — **nunca debes enviar el número de control en el request**.

### Ciclo de vida de un correlativo

1. **Reserva:** Al recibir el `POST`, la API reserva el siguiente número disponible.
2. **Uso:** Genera y transmite el documento a Hacienda.
3. **Confirmación:** Si la transmisión es exitosa, el número queda `CONFIRMADO`.
4. **Liberación (Rollback):** Si ocurre un error de validación o Hacienda rechaza definitivamente (4xx), el número se libera y vuelve a estar disponible para la siguiente petición.

**Reintentos seguros:** Si recibes un error 500 (interno del servidor, no de Hacienda), es seguro reintentar. El sistema garantiza la integridad de la numeración.

---

## Documentación por tipo de DTE

- [Factura Electrónica (FE)](./invoice.md)
- [Comprobante de Crédito Fiscal (CCF)](./ccf.md)
- [Nota de Crédito Electrónica](./credit-note.md)
- [Nota de Débito Electrónica](./debit-note.md)
- [Factura de Sujeto Excluido (FSE)](./fse.md)
- [Comprobante de Retención Electrónico](./retention.md)
- [Nota de Remisión Electrónica](./remission-note.md)
- [Anulación de DTE](./invalidation.md)
- [Consulta de DTEs](./dte-query.md)
