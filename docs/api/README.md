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
| `data` | object | **Archivo DTE**: la estructura firmada del documento, íntegra y sin alteraciones, más `firmaElectronica` (JWS) y `selloRecibido`. En contingencia (sin sello) es la estructura del DTE sin firma |

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
| `error.code` | string | Código identificador del error, siempre en mayúsculas (ver tabla abajo) |
| `error.details` | string[] | Detalles adicionales (por ejemplo, el mensaje de cada regla incumplida); puede ser vacío o nulo |

> Con `DEBUG=true` los mensajes de validación se prefijan con el nombre de la regla entre corchetes, por ejemplo `[RelatedDocOutOfPeriod] ...`.

### Códigos de error de validación

Las reglas de negocio no siempre se devuelven con su nombre como `error.code`; el código depende de dónde se detectó el error:

| `error.code` | Significado |
|--------------|-------------|
| Nombre de la regla en mayúsculas (`REQUIREDFIELD`, `INVALIDQUERYPARAM`, `NOTMATCHINGRECEIVERNIT`, `PENDINGADJUSTMENTNOTES`...) | Error de un campo, de una consulta o de una regla puntual |
| `VALIDATIONFAILED` | Falló el validador propio del tipo de documento (por ejemplo Nota de Crédito o Débito); `error.details` lista cada regla incumplida |
| `MANY_ERRORS` | Fallaron una o varias reglas generales de DTE; `error.details` lista cada una |

---

## Tipos de error

| Categoría | HTTP Status | Código | Descripción |
|-----------|-------------|--------|-------------|
| Validación | `400` | Específico por regla, `VALIDATIONFAILED` o `MANY_ERRORS` | El request contiene campos inválidos o viola reglas de negocio |
| Hacienda rechazó | `400` | `HACIENDA_<codigoMsg>` | Hacienda procesó el documento pero lo rechazó |
| Error HTTP de Hacienda | Varía | `HACIENDA_<httpCode>` | Hacienda retornó un error HTTP (5xx, 429, etc.) |
| Error de servicio | `400` | `<CODIGO_SERVICIO>` | Error de negocio interno (número secuencial, mapeo, etc.) |
| Error del sistema | `500` | `SYSTEM_ERROR` | Error interno inesperado del servidor |
| No autorizado | `401` | `UNAUTHORIZED` | Token JWT ausente, expirado o inválido |
| No encontrado | `404` | `NOT_FOUND` | Recurso no encontrado |
| Timeout | `408` | `REQUEST_TIMEOUT` | La petición excedió el tiempo límite (14 segundos) |

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
| `POST` | `/api/v1/dte/invalidation` | Anulación (invalidación) de DTE | No |

### DTE — Consulta (requieren autenticación)

| Método | Path | Descripción |
|--------|------|-------------|
| `GET` | `/api/v1/dte` | Listar todos los DTEs del sistema |
| `GET` | `/api/v1/dte/{id}` | Obtener DTE por código de generación (UUID) |
| `GET` | `/api/v1/dte/{id}/archivo` | Obtener el Archivo DTE (estructura firmada + `firmaElectronica` + `selloRecibido`) |
| `GET` | `/api/v1/dte/{id}/eventos` | Obtener los eventos del DTE (invalidación) con su Archivo DTE |

### Otros

| Método | Path | Autenticación | Descripción |
|--------|------|:---:|-------------|
| `GET` | `/health` | No | Estado del servicio. No lleva el prefijo `/api/v1` |
| `GET` | `/api/v1/metrics` | Sí | Métricas de uso de endpoints (JSON) del sistema autenticado; admite `endpoint` y `method` como filtros |
| `GET` | `/api/v1/test` | Sí | Prueba de componentes del sistema |
| `POST` | `/api/v1/debug/notify-test` | Sí | Solo disponible con `DEBUG=true`; envía un correo de prueba al administrador |

---

## Contingencia

Cuando un DTE que soporta contingencia no puede transmitirse a Hacienda (por problemas de red, indisponibilidad del servicio, timeout, etc.), el sistema lo almacena automáticamente y retorna **HTTP 201** con la estructura del DTE — pero sin `reception_stamp`.

### Cuándo se activa

Los tipos corresponden al catálogo CAT-005 (Tipo de Contingencia): `1` No disponibilidad de sistema del MH, `2` No disponibilidad de sistema del emisor, `3` Falla en el suministro de servicio de Internet del emisor, `4` Falla en el suministro de energía eléctrica del emisor, `5` Otro motivo.

| Condición | Tipo de contingencia |
|-----------|---------------------|
| Error de red (connection refused, connection reset, broken pipe) | `2` |
| Red no disponible (no route to host, network unreachable, i/o timeout, no such host) | `3` |
| Otro error de red | `2` |
| HTTP 502/503 de Hacienda | `1` |
| HTTP 408/504 (timeout de petición) | `1` |
| HTTP 429 (rate limiting) | `1` |
| Mantenimiento o sobrecarga (según descripción de Hacienda) | `1` |
| Timeout de contexto (deadline exceeded) | `1` |
| HTTP 401/403 con "token" en la descripción | `2` |
| Cualquier otro error no clasificado | `5` |

El tipo `4` (energía eléctrica) no se asigna automáticamente.

> La clasificación depende del tipo de error recibido: los errores HTTP que no vienen en el cuerpo de respuesta de Hacienda solo clasifican como `1` los códigos 502, 503 y 504; el resto (por ejemplo 408 o 429) se clasifica como `5`.

### Cuándo NO se activa

- El documento tiene errores de validación (campos incorrectos)
- Hacienda devuelve estado `RECHAZADO`
- Errores de negocio internos (mapeo, número secuencial, etc.)

### Respuesta durante contingencia

La respuesta es `HTTP 201` con la estructura del DTE (sin firma ni sello) y `reception_stamp: null`. El documento queda en estado `PENDING` con transmisión `CONTINGENCY`:

```json
{
  "success": true,
  "reception_stamp": null,
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?...",
  "data": {
    "identificacion": {
      "tipoModelo": 2,
      "tipoOperacion": 2,
      "tipoContingencia": 1,
      "motivoContin": "Servicio del Ministerio de Hacienda no disponible",
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

- **NIT:** 14 o 9 dígitos; si se envían guiones, se eliminan.
- **DUI:** 9 dígitos; si se envían guiones, se eliminan.
- **NRC:** De 1 a 8 dígitos.
- **Textos:** Evitar caracteres especiales no estándar o emojis; respetar los límites de longitud.
- **Cálculos:** La API valida los totales y tributos de cada documento con una tolerancia de ±$0.01 (en la Factura, el CCF y la FSE también la venta de cada ítem). Enviar los cálculos ya verificados para evitar rechazos por redondeo. Las fórmulas se detallan en [Reglas de cálculo](./calculations.md).

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
- [Reglas de cálculo](./calculations.md)

### Plazo de 72 horas y estado `EXPIRED`

Los documentos en contingencia se retransmiten en segundo plano. El plazo para transmitirlos es de **72 horas contadas desde el sello de recepción del evento de contingencia**. Si transcurre ese plazo sin transmitirse, el documento pasa a estado `EXPIRED`, se considera no emitido y se notifica al administrador. Los documentos cuyo evento aún no tiene sello no vencen.
