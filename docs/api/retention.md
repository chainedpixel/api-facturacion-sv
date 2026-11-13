# Comprobante de Retención Electrónico (CRE)

## Descripción

El **Comprobante de Retención Electrónico** (código DTE `07`) documenta las retenciones de IVA que un contribuyente aplica a sus proveedores. Cada ítem referencia un documento (físico o electrónico) sobre el cual se está practicando la retención. En el Anexo IV de la Normativa de Cumplimiento DTE v2.0, el emisor del CRE es el Agente de Retención y el receptor es el Sujeto de Retención (secciones 3 y 4).

**Soporta contingencia:** No (ver [Contingencia](#contingencia))

---

## Endpoint

```
POST /api/v1/dte/retention
Authorization: Bearer <token>
Content-Type: application/json
```

**Respuesta exitosa:** `HTTP 201 Created`

> Los datos que el servicio genera por sí mismo **no se envían**: número de control, código de generación, fecha y hora de emisión, ambiente, versión y datos del emisor. La fecha de emisión es siempre la del momento de la petición. Los campos desconocidos del request (por ejemplo `extension`, que el CRE no admite) se ignoran sin generar error.

> **Normativa:** los campos de la sección Identificación (versión, ambiente, tipo de documento, código de generación, número de control, modelo de facturación, tipo de transmisión, fecha, hora y moneda) corresponden a los campos 1 a 12 del Anexo IV de la Normativa de Cumplimiento DTE v2.0; la versión de la estructura del CRE es 2 (campo 1). El código de generación y el número de control se rigen además por las reglas 7.1.1 y 7.1.2, y los datos del emisor corresponden a los campos 18 a 33 del mismo Anexo.

---

## Request Body

### Estructura principal

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `items` | `RetentionItem[]` | Sí | Documentos sobre los que se retiene (al menos 1) |
| `receiver` | `Receiver` | Sí | Sujeto al que se le retiene (proveedor) |
| `summary` | `RetentionSummary` | Sí | Totales de la retención |
| `appendixes` | `Appendix[]` | No | Apéndices informativos |

---

### `items[]` — Documentos retenidos

Cada ítem representa un documento sobre el que se practica la retención. El número de ítem lo asigna el servicio según la posición en el arreglo (`numItem` = 1, 2, 3...).

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | integer | Sí | Tipo de generación del documento: `1` = Físico, `2` = Electrónico |
| `document_number` | string | Sí | Número del documento. Físico (`type = 1`): alfanumérico de 1 a 20 caracteres. Electrónico (`type = 2`): UUID en mayúsculas (`XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX`) |
| `description` | string | Sí | Descripción de la operación |
| `retention_code` | string | Sí | Código de retención (CAT-006): `"22"` = 1 %, `"C4"` = 13 %, `"C9"` = Otros |
| `taxed_amount` | float | Sí | Monto sujeto a retención |
| `iva_amount` | float | Sí | IVA retenido del documento |
| `emission_date` | string | Sí | Fecha del documento retenido (`YYYY-MM-DD`, no puede ser futura) |
| `dte_type` | string | Sí | Tipo de DTE del documento retenido (código CAT-002; los tipos previstos para retención son FE `01`, CCF `03` y FSE `14`) |

`taxed_amount`, `iva_amount`, `emission_date` y `dte_type` son obligatorios **tanto para documentos físicos como electrónicos**: el servicio no los consulta en la base de datos, deben enviarse.

> **Normativa:** los campos del ítem corresponden a la sección Cuerpo del Anexo IV de la Normativa de Cumplimiento DTE v2.0: `dte_type` (campo 78, tipo de documento relacionado), `type` (campo 79, CAT-007), `document_number` (campo 80), `emission_date` (campo 81), `taxed_amount` (campo 103: monto sujeto a retención, sin IVA y mayor a `0`, regla 1; con holgura de ±0.01, regla 4), `retention_code` (campo 104, CAT-006) e `iva_amount` (campo 105: monto sujeto a retención multiplicado por el porcentaje legal, regla 2, con holgura de ±0.01, regla 3).

> **Regla:** Todos los ítems del CRE deben tener el mismo `retention_code`; códigos distintos se rechazan (`MixedRetentionCodes`).

> **Regla:** `iva_amount` debe coincidir con `taxed_amount × tasa del código` con tolerancia ±0.01 (`InvalidRetentionIVA`). Ver [calculations.md](./calculations.md#4-comprobante-de-retención-cr) (sección 4; Anexo IV, campo 105).

---

### `receiver` — Receptor (sujeto retenido)

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `document_type` | string | Sí | `"36"` NIT, `"13"` DUI, `"02"` Carnet de residente, `"03"` Pasaporte, `"37"` Otro |
| `document_number` | string | Sí | Número de documento. NIT y DUI se aceptan **con o sin guiones** (se eliminan). DUI: exactamente 9 dígitos. Otros: 3–20 caracteres |
| `name` | string | Sí | Nombre o razón social |
| `email` | string | Sí | Correo electrónico válido, de 3 a 100 caracteres. Se verifica que el dominio tenga registro MX |
| `address` | `Address` | Sí | Dirección |
| `activity_code` | string | Sí | Código de actividad económica (2–6 dígitos) |
| `activity_description` | string | Sí | Descripción de actividad económica |
| `nrc` | string | No | NRC (1–8 dígitos) |
| `phone` | string | No | Teléfono (8–30 caracteres) |
| `commercial_name` | string | No | Nombre comercial |

#### `address`

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `department` | string | Sí | Código de departamento (`01`–`14`) |
| `municipality` | string | Sí | Código de municipio válido para el departamento (CAT-013) |
| `district` | string | No | Código de distrito (`01`–`99`, dos dígitos). Si se omite o se envía vacío se usa `"01"` |
| `complement` | string | Sí | Dirección complementaria (1–200 caracteres) |

> **Normativa:** en el CRE el receptor es el Sujeto de Retención (Anexo IV de la Normativa de Cumplimiento DTE v2.0, sección 4): `document_type` (campo 38, CAT-022), `document_number` (campo 39; el DUI con 9 dígitos sin guiones, regla 10; cuando el documento informado es electrónico, el receptor del CRE debe ser el emisor de ese documento y el emisor del CRE su receptor, regla 8), `nrc` (campo 41), `name` (campo 42), `activity_code` (campo 43, CAT-019), `activity_description` (campo 44), `commercial_name` (campo 45), `address` (campos 47 a 50), `phone` (campo 56) y `email` (campo 57).

---

### `summary` — Totales de retención

Los montos del resumen se envían con **máximo 2 decimales**.

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `total_retention_amount` | float | Sí | Suma de todos los `taxed_amount` (distinto de `0`) |
| `total_retention_iva` | float | Sí | Suma de todos los `iva_amount` (distinto de `0`) |
| `total_iva` | float | No | IVA 13 % del resumen. Si se omite (o es `0`) el servicio lo calcula: `total_retention_amount × 0.13` cuando **todos** los documentos son electrónicos y el código es `C4`; en los demás casos `0`. Si se envía, debe coincidir con ese valor (±0.01) |
| `observations` | string | No | Observaciones (máx. 3000 caracteres) |

El total en letras (`totalLetras`) se genera a partir de `total_retention_iva`.

> **Normativa:** `total_retention_amount` (campo 140: sumatoria de los montos sujetos a retención, mayor a `0`, con holgura de ±0.01), `total_retention_iva` (campo 155: suma de los campos 'IVA retenido', mayor a `0`), `total_iva` (campo 157: `0.00` cuando la operación no es afecta a IVA 13 %, regla 3, y, con el código de retención `C4`, monto sujeto a retención por 13 %, regla 4), el total en letras (campo 156) y `observations` (campo 168). Los montos del resumen siguen las reglas 7.2 literal b) y 7.3.

---

### `appendixes[]` — Apéndices

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `field` | string | Sí | Identificador del campo (2–25 caracteres) |
| `label` | string | Sí | Etiqueta visible (3–50 caracteres) |
| `value` | string | Sí | Valor del campo (1–150 caracteres) |

El sello de recepción **no** se agrega al apéndice: se entrega en el Archivo DTE (ver [Respuesta exitosa](#respuesta-exitosa)). `field`, `label` y `value` corresponden a los campos 172, 173 y 174 del Anexo IV de la Normativa de Cumplimiento DTE v2.0 (sección 11, Apéndice). El Sello de Recepción se incorpora al Archivo DTE que se entrega al receptor (Anexo IV, sección 12, campo 175).

---

## Reglas de cálculo

Las fórmulas completas y sus tolerancias están en [calculations.md](./calculations.md), sección 4 ([Comprobante de Retención](./calculations.md#4-comprobante-de-retención-cr)). El redondeo (regla 7.2) y la holgura de ±0.01 (regla 7.3, Cuadros 1, 2 y 3) de la Normativa de Cumplimiento DTE v2.0 se describen en las secciones [1.1](./calculations.md#11-decimales-y-redondeo) y [1.2](./calculations.md#12-holgura). Resumen de las reglas que se validan:

- IVA retenido por ítem: código `"22"` → `taxed_amount × 0.01`; código `"C4"` → `taxed_amount × 0.13` (±0.01). Con el código `"C9"` la tasa esperada es `0`, por lo que `iva_amount` solo se acepta con valor `0` (±0.01) en el ítem (Anexo IV, campo 105).
- `Σ taxed_amount` = `total_retention_amount` y `Σ iva_amount` = `total_retention_iva` (±0.01; Anexo IV, campos 140 y 155).
- `total_iva` se valida contra `total_retention_amount × 0.13` si todos los documentos son electrónicos y el código es `C4`; en los demás casos debe ser `0` (±0.01; Anexo IV, campo 157).

### Rango de fechas permitido

La fecha de **cada** documento retenido (físico o electrónico) debe estar dentro de un período permitido respecto a la fecha de emisión del CRE:

```
Regla 1: La fecha del documento está en el MISMO mes que el CRE
         → Siempre válido

Regla 2: La fecha del documento está en el MES ANTERIOR al CRE
         → Válido solo si el CRE se emite dentro de los primeros
           10 días hábiles (lunes a viernes) del mes siguiente

Regla 3: Cualquier otro caso → Inválido (DateOutOfAllowedRange)
```

**Ejemplo:**
- Documento de enero 2024 → Se puede retener en enero 2024 (mismo mes).
- Documento de enero 2024 → Se puede retener hasta los primeros 10 días hábiles de febrero 2024.
- Documento de diciembre 2023 → Se puede retener en diciembre 2023 o en los primeros 10 días hábiles de enero 2024.

Los días hábiles se cuentan de lunes a viernes; no se consideran feriados.

---

## Validaciones

| Regla | Detalle |
|-------|---------|
| Ítems | Mínimo 1 |
| `retention_code` | Solo `"22"`, `"C4"` o `"C9"`; igual en todos los ítems |
| `iva_amount` | Debe coincidir con `taxed_amount × tasa` (±0.01) |
| Fecha del documento | Dentro del período permitido (todos los ítems) |
| Campos del ítem | `taxed_amount`, `iva_amount`, `emission_date` y `dte_type` obligatorios en todos los tipos |
| `document_number` | Físico: alfanumérico 1–20; electrónico: UUID en mayúsculas |
| `type` | `1` o `2` |
| Receptor | `document_type`, `document_number`, `name`, `email`, `address`, `activity_code` y `activity_description` obligatorios |
| Resumen | `total_retention_amount` y `total_retention_iva` obligatorios y distintos de `0`; máximo 2 decimales |
| Montos | Entre `0` y `99999999999.99` |

---

## Respuesta exitosa

**HTTP 201 Created**

La emisión normal devuelve en `data` el **Archivo DTE**: la estructura del documento tal como fue firmada y transmitida, más los atributos `firmaElectronica` (JWS firmado) y `selloRecibido` (sello de recepción de Hacienda). El sello de recepción se repite en `reception_stamp` en la raíz de la respuesta.

> **Normativa:** el Archivo DTE se rige por la regla 4 (Entrega) del Anexo I y el Cuadro 11 de la Normativa de Cumplimiento DTE v2.0: contiene la estructura de datos íntegra, el documento firmado y el Sello de Recepción (`selloRecibido`, Anexo I, regla 3.4). La regla 11.1 exige entregar al receptor el DTE íntegro, con su Representación Gráfica y el Sello de Recepción.

```json
{
  "success": true,
  "reception_stamp": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C",
  "qr_link": "https://admin.factura.gob.sv/consultaPublica?ambiente=01&codGen=8E3A1C66-4D5F-4B7C-9A0D-3C4D5E6F7081&fechaEmi=2026-05-15",
  "data": {
    "identificacion": {
      "version": 2,
      "ambiente": "01",
      "tipoDte": "07",
      "numeroControl": "DTE-07-N0010001-000000000000001",
      "codigoGeneracion": "8E3A1C66-4D5F-4B7C-9A0D-3C4D5E6F7081",
      "tipoModelo": 1,
      "tipoOperacion": 1,
      "tipoContingencia": null,
      "motivoContin": null,
      "fecEmi": "2026-05-15",
      "horEmi": "10:15:30",
      "tipoMoneda": "USD"
    },
    "resumen": {
      "totalSujetoRetencion": 226.5,
      "totalIva": 0,
      "totalIvaRetenido": 2.26,
      "totalLetras": "DOS 26/100",
      "observaciones": null
    },
    "emisor": {
      "nit": "06141101690011",
      "nrc": "1937",
      "nombre": "Emisor de Ejemplo S.A. de C.V.",
      "codActividad": "47190",
      "descActividad": "Venta al por menor de otros productos",
      "nombreComercial": null,
      "direccion": { "departamento": "06", "municipio": "23", "distrito": "01", "complemento": "Dirección de ejemplo" },
      "codEstable": "N001",
      "codPuntoVenta": "0001",
      "telefono": "22220000",
      "correo": "emisor@example.com"
    },
    "receptor": {
      "tipoDocumento": "36",
      "numDocumento": "08211311610013",
      "nombre": "Nombre de Ejemplo",
      "nrc": "1027018",
      "codActividad": "47522",
      "descActividad": "Venta al por menor de artículos de ferretería",
      "nombreComercial": "Comercial de Ejemplo",
      "direccion": { "departamento": "08", "municipio": "23", "distrito": "01", "complemento": "Dirección de ejemplo" },
      "telefono": "22220000",
      "correo": "ejemplo@example.com"
    },
    "cuerpoDocumento": [
      {
        "numItem": 1,
        "tipoDte": "03",
        "tipoGeneracion": 1,
        "numeroDocumento": "S221001345",
        "fechaEmision": "2026-05-05",
        "montoSujetoGrav": 226.5,
        "codigoRetencionMH": "22",
        "ivaRetenido": 2.26,
        "descripcion": "Compra de equipos informáticos"
      }
    ],
    "apendice": null,
    "firmaElectronica": "eyJhbGciOiJSUzUxMiJ9.eyJpZGVudGlmaWNhY2lvbiI6...<firma>",
    "selloRecibido": "20262E5B1C7D4A8E9F3A0B1C2D3E4F5A6B7C"
  }
}
```

Notas sobre la estructura:

- `receptor` se identifica por `tipoDocumento` y `numDocumento` (sin guiones para NIT y DUI).
- `resumen` usa `totalSujetoRetencion`, `totalIva`, `totalIvaRetenido`, `totalLetras` y `observaciones`.
- Cada ítem de `cuerpoDocumento` incluye `tipoDte`, `tipoGeneracion` (el valor de `type`), `numeroDocumento`, `fechaEmision`, `montoSujetoGrav`, `codigoRetencionMH`, `ivaRetenido` y `descripcion`.
- `emisor` solo incluye `codEstable` y `codPuntoVenta` como códigos de establecimiento y punto de venta; no existen `tipoEstablecimiento`, `codEstableMH` ni `codPuntoVentaMH`.
- El CRE no tiene `extension`, ventas ni tributos en la estructura emitida.
- Los campos opcionales sin valor se emiten como `null`; `apendice` solo trae contenido si se enviaron apéndices.
- El orden de las llaves dentro de `data` no es significativo. Para entregar el documento al receptor se debe conservar `data` íntegro, sin alterarlo.
- Si el Archivo DTE no pudiera armarse (firma no disponible), `data` contiene únicamente la estructura del documento, sin `firmaElectronica` ni `selloRecibido`.

---

## Contingencia

El CRE **no soporta contingencia**: si la transmisión a Hacienda falla, el documento no se almacena para retransmisión y se devuelve el error correspondiente (por ejemplo `HACIENDA_<codigo>` o `SYSTEM_ERROR`). El consumidor debe reintentar la petición. Por ello nunca se obtiene una respuesta con `reception_stamp: null`.

---

## Respuestas de error

Los errores de validación devuelven `HTTP 400` con la estructura `error.code`, `error.message` y `error.details`. El valor de `code` depende de la etapa en la que se detecta el problema:

| Etapa | `code` | `message` | `details` |
|-------|--------|-----------|-----------|
| Lectura del request (JSON inválido) | `BAD_REQUEST` | `Invalid request format` | vacío |
| Estructura mínima (falta `summary` o `receiver`; `receiver.document_type` o `document_number` ausentes; códigos de retención mezclados) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDFIELD`, `MIXEDRETENTIONCODES`) | Mensaje del campo | `No hay más detalles disponibles` |
| Mapeo de una sección (ítems, receptor, resumen) | Tipo del error en mayúsculas (`REQUIREDFIELD`, `INVALIDRETENTIONCODE`, ...) o `ERRORMAPPING` | `Error al mapear la sección Retention->...` | Mensaje del campo |
| Reglas específicas del CRE | `VALIDATIONFAILED` | `Validaciones no superadas, revise los detalles a continuación` | Lista de mensajes de cada regla incumplida |
| Rechazo de Hacienda | `HACIENDA_<codigo>` | Descripción de Hacienda | Estado, fecha de proceso y observaciones |
| Error interno | `SYSTEM_ERROR` (`HTTP 500`) | `Ha ocurrido un error inesperado` | vacío |

Con la depuración activa del servidor, los mensajes de validación se prefijan con la clave `[Clave]`. El CRE solo evalúa sus reglas específicas; no aplica el conjunto de reglas comunes de los demás DTE.

### Reglas específicas del CRE (`VALIDATIONFAILED`)

| Clave | Cuándo ocurre |
|-------|---------------|
| `InvalidRetentionIVA` | `iva_amount` no coincide con `taxed_amount × tasa` del código de retención |
| `DateOutOfAllowedRange` | La fecha del documento retenido está fuera del período permitido |
| `InvalidTotalSubjectRetention` | `total_retention_amount` no coincide con la suma de `taxed_amount` |
| `InvalidTotalIVARetention` | `total_retention_iva` no coincide con la suma de `iva_amount` |
| `InvalidTotalIVA` | `total_iva` no coincide con el valor esperado |
| `RequiredField` | El documento no tiene ítems |

### Errores en el mapeo

| Clave | Cuándo ocurre |
|-------|---------------|
| `RequiredField` | Falta `summary`, `receiver`, un campo obligatorio del receptor o del ítem (`taxed_amount`, `iva_amount`, `emission_date`, `dte_type`, `description`), o `total_retention_amount` / `total_retention_iva` son `0` |
| `InvalidRetentionCode` | `retention_code` distinto de `22`, `C4`, `C9` |
| `MixedRetentionCodes` | Los ítems tienen códigos de retención distintos |
| `InvalidNumberRange` | `type` distinto de `1` o `2` |
| `InvalidDocumentNumberItem` | `document_number` con formato inválido para su `type` |
| `InvalidDateTime` | `emission_date` con formato inválido o futura |
| `InvalidDTEType` | `dte_type` fuera del catálogo CAT-002 |
| `InvalidDecimals` | Un monto del resumen tiene más de 2 decimales |
| `InvalidLength` | `observations` mayor a 3000 caracteres |

---

## Ejemplos

Los valores de `emission_date` de los ejemplos son ilustrativos: deben ajustarse para cumplir el [rango de fechas permitido](#rango-de-fechas-permitido) respecto a la fecha en que se emite el CRE.

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
    }
}
```

---

### Ejemplo 2: Retención sobre documento electrónico

Con todos los documentos electrónicos y código `C4`, `total_iva` se calcula como `100.00 × 0.13 = 13.00` si se omite.

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
            "document_number": "E1E31344-D7D8-466D-BF99-FE1BCFEAD10C",
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
