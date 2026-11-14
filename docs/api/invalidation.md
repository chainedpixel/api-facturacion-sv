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
| `replacement_generation_code` | string | No* | UUID del DTE de reemplazo. **Obligatorio para tipos `1` y `3`**, excepto cuando el documento anulado es una Nota de Crédito (`05`) o un Comprobante de Liquidación (`08`), en cuyo caso debe omitirse o ser `null`. Prohibido para tipo `2` |

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
| `1` | Error en datos — se emitió un DTE de reemplazo | **Obligatorio** (nulo si el documento anulado es `05` u `08`) | Opcional |
| `2` | Anulación sin reemplazo — el DTE se elimina sin sustituir | **Prohibido** | Opcional |
| `3` | Otro motivo — anulación con reemplazo y justificación | **Obligatorio** (nulo si el documento anulado es `05` u `08`) | **Obligatorio** |

---

## Validaciones

### Del request

| Regla | Detalle |
|-------|---------|
| `generation_code` | Obligatorio. Debe corresponder a un DTE de la sucursal autenticada |
| `reason` | Obligatorio, con `type`, `responsible_name`, `responsible_doc_type`, `responsible_num_doc`, `requestor_name`, `requestor_doc_type` y `requestor_num_doc` |
| `reason.type` | `1`, `2` o `3` |
| `reason.responsible_doc_type`, `reason.requestor_doc_type` | `36` (NIT), `13` (DUI), `02` (Carnet de residente), `03` (Pasaporte) o `37` (Otro) |
| `reason.responsible_num_doc`, `reason.requestor_num_doc` | NIT y DUI se aceptan con o sin guiones (se eliminan); el DUI debe tener 9 dígitos. Longitud de 3 a 20 caracteres |
| `replacement_generation_code` | **Prohibido** para tipo `2` (`INVALIDINVALIDATIONTYPE2`). Para tipos `1` y `3` se valida en las reglas del evento (ver abajo): obligatorio salvo que el documento anulado sea una Nota de Crédito (`05`) o un Comprobante de Liquidación (`08`) |
| `reason.reason_field` | **Obligatorio** para tipo `3` (`INVALIDINVALIDATIONTYPE3`), de 5 a 250 caracteres. Para los tipos `1` y `2` se ignora aunque se envíe |

### Del documento a invalidar

| Regla | Detalle |
|-------|---------|
| Estado | No puede estar `INVALIDATED` (`DOCUMENTALREADYINVALID`), `REJECTED` (`DOCUMENTREJECT`) ni `PENDING` (`DOCUMENTPENDING`). Si hay documento de reemplazo (tipos `1` y `3`), se aplican las mismas verificaciones de estado al reemplazo |
| Sello de recepción | Debe tener sello de recepción de 40 caracteres alfanuméricos; sin sello se rechaza con `DOCUMENTWITHOUTSTAMP` (un documento en contingencia sin transmitir no tiene sello) |
| CCF o CR con notas vigentes | Se rechaza con `PENDINGADJUSTMENTNOTES`, que lista las NC/ND en estado `RECEIVED` o `PENDING` que lo ajustan. El consumidor invalida cada nota y luego vuelve a solicitar la invalidación del CCF o CR; el microservicio no invalida notas automáticamente |

### Código de reemplazo según el tipo de documento

La regla se evalúa en las reglas del evento (`InvalidationReasonStrategy`, mediante `policy.RequiresReplacement`), con el tipo del documento invalidado. Corresponde al campo `codigoGeneracionR` del Anexo V.

| Tipo de anulación | Documento invalidado | `replacement_generation_code` | Error si no se cumple |
|---|---|---|---|
| `1` o `3` | Cualquiera excepto `05` y `08` | Obligatorio | `InvalidReplacementCode1And3` (dentro de `MANY_ERRORS`) |
| `1` o `3` | Nota de Crédito (`05`) o Comprobante de Liquidación (`08`) | Debe omitirse o ser `null` | `ReplacementCodeNotAllowedForType` (dentro de `MANY_ERRORS`) |
| `2` | Cualquiera | Siempre `null` | `INVALIDINVALIDATIONTYPE2` |

### Plazos y fecha del evento

Las reglas siguen el Cuadro 6 de la Normativa de Cumplimiento DTE v2.0 (plazos para transmitir el evento de invalidación) y las reglas de fecha del evento del Anexo V.

| Tipo de DTE | Plazo para transmitir la invalidación (contado desde el sello) | Fecha del evento (`fecEmi`/`horEmi`) |
|---|---|---|
| FE (`01`), FEXE (`11`), FSE (`14`) | Hasta las 23:59:59 del mismo día, 3 meses después (2 años para FE y FEXE de emisores con actividad económica 21001, 21008, 46482, 46484, 46491 o 47721; la excepción no aplica a FSE) | Fecha y hora actual; no puede superar el mismo periodo (3 meses o 2 años) contado desde la fecha de generación |
| Resto (`03`, `04`, `05`, `06`, `07`, `08`, `09`, `15`) | Hasta las 23:59:59 del décimo día hábil del mes siguiente al mes del sello | Misma fecha de generación del documento (la fecha y hora actual si se generó hoy) |

- La fecha del sello es `processing_date` (`fhProcesamiento` de Hacienda); para documentos antiguos sin ese dato se usa la fecha de registro del documento.
- La fecha de generación se toma de `fecEmi`/`horEmi` del documento guardado.
- Días hábiles: lunes a viernes, excluyendo los asuetos nacionales fijos (1 de enero, 1 de mayo, 10 de mayo, 17 de junio, 6 de agosto, 15 de septiembre, 2 de noviembre y 25 de diciembre), la Semana Santa (de Lunes Santo a Lunes de Pascua) y los asuetos de la variable `MH_EXTRA_HOLIDAYS`.
- Si el plazo venció, el documento conserva su validez y corresponde emitir una Nota de Crédito o Débito si procede.

### Orden de verificación

1. Campos del request y reglas de tipo (`reason.type`, reemplazo en tipo `2`, `reason_field`).
2. Estado del documento a invalidar y del reemplazo.
3. Dependencias (`PENDINGADJUSTMENTNOTES`) y sello de recepción.
4. Datos del evento, plazo y fecha del evento.
5. Transmisión a Hacienda.

---

## Respuesta exitosa

**HTTP 200 OK**

La respuesta es un `EventArchiveResponse`: los datos del evento registrado, la respuesta completa de Hacienda (`hacienda_response`) y el **Archivo DTE del evento** (`archivo_dte`), listo para entregar al receptor. El evento queda disponible después en `GET /api/v1/dte/{id}/eventos` (ver [Consulta de DTEs](./dte-query.md)).

```json
{
  "success": true,
  "data": {
    "generation_code": "A1B2C3D4-0000-4000-8000-1234567890AB",
    "branch_id": 1,
    "event_type": "INVALIDATION",
    "related_dte_id": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
    "replacement_dte_id": "9A7B6C5D-3E2F-1A0B-CDEF-1234567890AB",
    "reception_stamp": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
    "processing_date": "15/01/2026 14:30:05",
    "status": "RECEIVED",
    "created_at": "2026-01-15T14:30:06-06:00",
    "hacienda_response": { },
    "archivo_dte": {
      "identificacion": {
        "version": 3,
        "ambiente": "00",
        "codigoGeneracion": "A1B2C3D4-0000-4000-8000-1234567890AB",
        "fecEmi": "2026-01-15",
        "horEmi": "14:30:00",
        "fusion": null
      },
      "emisor": {
        "nit": "XXXXXXXXXXXXXX",
        "nombre": "Emisor S.A. de C.V.",
        "codEstableMH": "",
        "codEstable": "0001",
        "codPuntoVentaMH": "",
        "codPuntoVenta": "0001",
        "telefono": "XXXXXXXX",
        "correo": "emisor@ejemplo.com"
      },
      "documento": {
        "tipoDte": "01",
        "codigoGeneracion": "3B8F1A2D-4E5C-6F7A-8B9C-0D1E2F3A4B5C",
        "selloRecibido": "20260110AAFEEE1A566A44F19A622C0C35C8A1B6FAZM",
        "numeroControl": "DTE-01-N0010001-000000000000001",
        "fecEmi": "2026-01-10",
        "codigoGeneracionR": "9A7B6C5D-3E2F-1A0B-CDEF-1234567890AB",
        "tipoDocumento": "13",
        "numDocumento": "012345678",
        "nombre": "Receptor",
        "telefono": "XXXXXXXX",
        "correo": "receptor@ejemplo.com"
      },
      "motivo": {
        "tipoAnulacion": 1,
        "motivoAnulacion": null,
        "nombreResponsable": "Carlos García",
        "tipDocResponsable": "13",
        "numDocResponsable": "012345678",
        "nombreSolicita": "María López",
        "tipDocSolicita": "13",
        "numDocSolicita": "087654321"
      },
      "firmaElectronica": "<JWS>",
      "selloRecibido": "20260115AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"
    }
  }
}
```

### Campos del evento

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `generation_code` | string | Código de generación del evento (distinto al del documento invalidado) |
| `branch_id` | integer | Sucursal |
| `event_type` | string | `INVALIDATION` |
| `related_dte_id` | string | Código de generación del DTE invalidado |
| `replacement_dte_id` | string o null | Código de generación del DTE de reemplazo |
| `reception_stamp` | string o null | Sello de recepción del evento |
| `processing_date` | string | Fecha de procesamiento del evento devuelta por Hacienda |
| `status` | string | `RECEIVED` |
| `created_at` | string (ISO 8601) | Fecha de registro del evento |
| `hacienda_response` | object | Respuesta completa de Hacienda al evento |
| `archivo_dte` | object | Archivo DTE del evento (se omite si no hay firma guardada) |

### Estructura del evento dentro de `archivo_dte`

La estructura firmada se devuelve sin alteraciones; `firmaElectronica` (JWS) y `selloRecibido` se agregan al final.

**`identificacion`**

| Campo | Descripción |
|-------|-------------|
| `version` | Versión del esquema del evento (`3`) |
| `ambiente` | Código de ambiente (`00` pruebas, `01` producción) |
| `codigoGeneracion` | UUID del evento |
| `fecEmi`, `horEmi` | Fecha y hora del evento (ver regla de fecha del evento) |
| `fusion` | `null` |

**`emisor`:** `nit`, `nombre`, `codEstableMH`, `codEstable`, `codPuntoVentaMH`, `codPuntoVenta`, `telefono`, `correo`.

**`documento`**

| Campo | Descripción |
|-------|-------------|
| `tipoDte` | Tipo del DTE invalidado |
| `codigoGeneracion` | Código de generación del DTE invalidado |
| `selloRecibido` | Sello de recepción del DTE invalidado |
| `numeroControl` | Número de control del DTE invalidado |
| `fecEmi` | Fecha de generación del DTE invalidado |
| `codigoGeneracionR` | Código de generación del reemplazo; `null` en tipo `2` y cuando el documento invalidado es una Nota de Crédito (`05`) o un Comprobante de Liquidación (`08`) |
| `tipoDocumento`, `numDocumento`, `nombre`, `telefono`, `correo` | Datos del receptor (o sujeto excluido) tomados del DTE invalidado |

**`motivo`:** `tipoAnulacion`, `motivoAnulacion` (solo tipo `3`; `null` en los demás), `nombreResponsable`, `tipDocResponsable`, `numDocResponsable`, `nombreSolicita`, `tipDocSolicita`, `numDocSolicita`.

> La invalidación no incluye `reception_stamp` ni `qr_link` en el nivel raíz de la respuesta, ya que no genera un nuevo DTE sino un evento sobre uno existente.

### Efectos sobre el documento

- El DTE original pasa a estado `INVALIDATED`.
- Si el documento invalidado es una Nota de Crédito, se registra una transacción de control de saldo `INVALIDATED` que suma sus montos (gravado, exento y no sujeto) al saldo del CCF relacionado.
- Si el documento invalidado es una Nota de Débito, se registra una transacción de control de saldo `DEBIT_NOTE_INVALID` que resta sus montos del saldo del CCF relacionado (la nota de débito los había sumado).

---

## Respuestas de error

Los errores usan la estructura descrita en [Formato de respuestas](./README.md#respuesta-de-error); `error.code` se devuelve en mayúsculas.

| HTTP | `code` | Cuándo ocurre |
|------|--------|---------------|
| `400` | `REQUIREDFIELD` | Falta `generation_code`, `reason` o alguno de sus campos obligatorios |
| `400` | `INVALIDINVALIDATIONTYPE2` | Tipo `2` con `replacement_generation_code` |
| `400` | `INVALIDINVALIDATIONTYPE3` | Tipo `3` sin `reason_field` |
| `400` | `FAILEDTOVERIFYDTE` | El `generation_code` (o el reemplazo) no existe en la sucursal |
| `400` | `DOCUMENTALREADYINVALID`, `DOCUMENTREJECT`, `DOCUMENTPENDING` | El documento (o el reemplazo) está invalidado, rechazado o pendiente de retransmitir |
| `400` | `PENDINGADJUSTMENTNOTES` | El CCF o CR tiene Notas de Crédito o Débito vigentes; el mensaje lista sus códigos |
| `400` | `DOCUMENTWITHOUTSTAMP` | El documento no tiene sello de recepción |
| `400` | `MANY_ERRORS` | Falló una regla del evento; `error.details` incluye `InvalidReplacementCode1And3` (tipo `1` o `3` sin `replacement_generation_code`, salvo NC `05` y CLE `08`), `ReplacementCodeNotAllowedForType` (`replacement_generation_code` enviado para una NC `05` o CLE `08`), `InvalidationDeadlineExpired` (plazo vencido), `InvalidationEventDateOutOfRange`, `InvalidationEventDateMismatch`, `MissingStampDate` o errores de formato de los datos del evento |
| `400` | `TRANSMISSIONFAILED` | Hacienda no procesó el evento |
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

Para consultar los eventos de un DTE (por ejemplo su invalidación):

```
GET /api/v1/dte/{codigoGeneracion}/eventos
Authorization: Bearer <token>
```
