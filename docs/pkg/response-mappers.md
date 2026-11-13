# Response Mappers — Modelo de Dominio → Formato Hacienda

> **Paquete:** `pkg/mapper/response_mapper`
> **Estructuras:** `pkg/mapper/response_mapper/structs`
> **Mappers comunes:** `pkg/mapper/response_mapper/common`

## Descripción General

Los response mappers transforman los modelos de dominio (resultado del procesamiento de servicios y validaciones) al formato JSON que la API de Hacienda espera recibir. Cada tipo de DTE tiene su propio mapper de nivel superior y mappers específicos para componentes como emisor, receptor, ítems y resumen.

Los montos y totales ya vienen calculados desde el dominio; los response mappers no recalculan los totales del documento. Las excepciones puntuales (IVA por ítem y totales de IVA en NC/ND) se indican más abajo. Las fórmulas de cálculo se documentan en [Cálculos de DTE](../api/calculations.md).

---

## Funciones de Nivel Superior

Cada función recibe un `interface{}` (modelo de dominio) y retorna la estructura JSON de Hacienda correspondiente:

| Función | Dominio Input | Hacienda Output |
|---|---|---|
| `ToMHInvoice(doc)` | `*ElectronicInvoice` | `*InvoiceDTEResponse` |
| `ToMHCreditFiscalInvoice(doc)` | `*CreditFiscalDocument` | `*CCFDTEResponse` |
| `ToMHCreditNote(doc)` | `*CreditNoteModel` | `*CreditNoteDTEResponse` |
| `ToMHDebitNote(doc)` | `*DebitNoteModel` | `*DebitNoteDTEResponse` |
| `ToMHRetention(doc)` | `*RetentionModel` | `*RetentionDTEResponse` |
| `ToMHRemissionNote(doc)` | `*RemissionNoteModel` | `*MHRemissionNote` (devuelve `nil` si el tipo no coincide) |
| `ToMHFSE(doc)` | `*FSEModel` | `*FSEDTEResponse` |
| `ToMHInvalidation(doc)` | `*InvalidationDocument` | `*InvalidationResponse` (devuelve `nil` si `doc` es `nil`) |

---

## Mappers Comunes

Se reutilizan para los componentes compartidos entre múltiples tipos de DTE.

| Función | Entrada | Salida |
|---|---|---|
| `MapCommonResponseIdentification()` | `interfaces.Identification` | `*DTEIdentification` |
| `MapCommonResponseIssuer()` | `interfaces.Issuer` | `DTEIssuer` |
| `MapCommonResponseReceiver()` | `interfaces.Receiver` | `DTEReceiver` (receptor del CCF, identificado por `nit`) |
| `MapCommonResponseDocumentReceiver()` | `interfaces.Receiver` | `DTEDocumentReceiver` (receptor de NC, ND y Retención, identificado por `tipoDocumento` y `numDocumento`) |
| `MapCommonResponseAddress()` | `interfaces.Address` | `DTEAddress` |
| `MapCommonItems()` | `interfaces.Item` | `DTEItem` |
| `MapItemCodes()` | `interfaces.Item` | `codigo` y `codTributo` del ítem |
| `MapCommonResponseSummary()` | `interfaces.Summary` | `*DTESummary` |
| `MapTaxes()` | `[]interfaces.Tax` | `[]DTETax` (`nil` si no hay tributos) |
| `MapCommonResponsePayments()` | `[]interfaces.PaymentType` | `[]DTEPayment` |
| `MapCommonResponseRelatedDocuments()` | `[]interfaces.RelatedDocument` | `[]DTERelatedDocument` |
| `MapCommonResponseOtherDocuments()` | `[]interfaces.OtherDocuments` | `[]DTEOtherDocument` |
| `MapCommonResponseThirdPartySale()` | `interfaces.ThirdPartySale` | `*DTEThirdPartySale` |
| `MapCommonResponseAppendix()` | `[]interfaces.Appendix` | `[]DTEApendice` (`nil` si no hay elementos) |
| `AdjustsRetentionVoucher()` | `[]interfaces.RelatedDocument` | `bool`: si la NC/ND ajusta un Comprobante de Retención (07) |
| `AdjustmentItemIVA()` | venta gravada del ítem y monto del resumen | IVA percibido o retenido del ítem de NC/ND |

No existe un response mapper de extensión: ningún DTE incluye la sección `extension` en el JSON enviado a Hacienda.

### Transformaciones Clave

- **Fechas**: Formato `"2006-01-02"` (YYYY-MM-DD)
- **Horas**: Formato `"15:04:05"` (HH:MM:SS)
- **Campos opcionales**: Se convierten a puntero; si están vacíos se asigna `nil` (se serializa `null`)
- **Dirección del receptor**: Se retorna `nil` si departamento, municipio y complemento están vacíos
- **Código de ítem y tributo (`MapItemCodes`)**: `codigo` va `null` si no se informó; `codTributo` solo se informa en ítems tipo 4 (impuesto sujeto a cálculo de IVA)
- **Tributos del resumen (`MapTaxes`)**: Sin tributos se devuelve `nil` (se envía `null`); los valores se redondean a 2 decimales usando `shopspring/decimal`
- **Apéndice (`MapCommonResponseAppendix`)**: Sin elementos se devuelve `nil` (se envía `null`), porque los esquemas exigen `null` o un arreglo con `minItems` 1; un arreglo vacío sería rechazado
- **Venta a terceros (`MapCommonResponseThirdPartySale`)**: Incluye `codDomiciliado` (Domicilio Fiscal, CAT-032)
- **Nombre comercial del emisor**: Se envía solo si no está vacío

---

## Mappers Específicos por DTE

### Factura (01)

| Mapper | Particularidad |
|---|---|
| `MapInvoiceResponseReceiver()` | Estructura `InvoiceReceiver`: identificado por `tipoDocumento`/`numDocumento`, todos los campos son punteros; sin `nit` ni `nombreComercial` |
| `MapInvoiceResponseItem()` | Se agregan: `VentaNoSuj`, `VentaExenta`, `VentaGravada`, `PSV`, `NoGravado`, `IvaItem` |
| `MapInvoiceResponseSummary()` | Se agregan: `DescuGravada`, `IvaRete`, `TotalIva`, `SaldoFavor` |

### CCF (03)

| Mapper | Particularidad |
|---|---|
| `MapCommonResponseReceiver()` | Receptor identificado por `nit` y `nrc`, con `nombreComercial` (opcional, `null` si no viene) |
| `MapCCFResponseItem()` | Se agregan: `VentaNoSuj`, `VentaExenta`, `VentaGravada`, `PSV`, `NoGravado` (sin `IvaItem`) |
| `MapCCFResponseSummary()` | Se agregan: `DescuGravada`, `IvaRete`, `IvaPerci`, `SaldoFavor` |

### Nota de Crédito (05) y Nota de Débito (06)

La estructura de ambas es la misma; la `identificacion` usa `DTEAdjustmentIdentification`, que agrega el campo `fusion` (siempre `null`).

| Mapper | Particularidad |
|---|---|
| `MapCreditNoteIssuer()` / `MapDebitNoteIssuer()` | Emisor sin `codEstable` ni `codPuntoVenta`; `nit` con `omitempty` |
| `MapCommonResponseDocumentReceiver()` | Receptor con `tipoDocumento` y `numDocumento` (en el request se identifica por NIT; ver [Request Mappers](request-mappers.md)) |
| `MapCreditNoteResponseItem()` / `MapDebitNoteResponseItem()` | `codTributo` y `codigo` como punteros; sin `PSV` ni `IvaItem`; `NoGravado` en 0; `TotalIva` del ítem = venta gravada x 13% redondeada a 2 decimales |
| `MapCreditNoteResponseSummary()` / `MapDebitNoteResponseSummary()` | Se agregan `IvaPerci`, `IvaRete`, `TotalIva` y `CodigoRetencionMH` (siempre `null`); sin `descuGravada`, `reteRenta` ni pagos. `TotalIva` es el valor del tributo `20` del resumen o, si no existe, total gravado x 13%. La ND agrega `NumPagoElectronico` |

Ajustes aplicados por `ToMHCreditNote` y `ToMHDebitNote` después de mapear:

1. `ivaPerci` e `ivaRete` por ítem se calculan con `AdjustmentItemIVA`: venta gravada x 1% cuando el resumen informa percepción o retención respectivamente (0 en caso contrario).
2. Si ninguno de los documentos relacionados es un Comprobante de Retención (07), `totalIva` del resumen y de cada ítem se envía en `0` (Anexo IV, campos 100 y 146: el IVA por ítem y su total solo aplican al ajustar un CRE; al ajustar un CCF el IVA va únicamente como tributo `20` del resumen).
3. `documentoRelacionado` se mapea siempre; `ventaTercero` y `apendice` solo si existen. No hay `otrosDocumentos`.

### Retención (07)

| Mapper | Particularidad |
|---|---|
| `MapRetentionResponseIssuer()` | Emisor con `codEstable` y `codPuntoVenta` (punteros) y nombre comercial opcional |
| `MapCommonResponseDocumentReceiver()` | Receptor con `tipoDocumento` y `numDocumento` |
| `MapRetentionResponseItem()` | Estructura propia: `numItem`, `tipoDte`, `tipoGeneracion`, `numeroDocumento`, `fechaEmision`, `montoSujetoGrav`, `codigoRetencionMH`, `ivaRetenido`, `descripcion` |
| `MapRetentionResponseSummary()` | `totalSujetoRetencion`, `totalIva`, `totalIvaRetenido`, `totalLetras`, `observaciones` |

La Retención no incluye documentos relacionados, venta a terceros ni extensión; solo `apendice` opcional.

### Nota de Remisión (04)

| Mapper | Particularidad |
|---|---|
| `MapRemissionNoteIssuer()` | Reutiliza `MapCommonResponseIssuer()` |
| `MapRemissionNoteReceiver()` | Estructura `MHRemissionNoteReceiver` con `bienTitulo` (si el receptor no es del tipo específico, `"99"`); `nrc` = `nil` si `DocumentType == "13"` |
| `MapRemissionNoteItems()` | Se agregan: `ventaNoSuj`, `ventaExenta`, `ventaGravada`, `codigo`; `codTributo` solo para ítems tipo 4 (primer tributo del ítem) |
| `MapRemissionNoteSummary()` | `descuGravada` toma el descuento gravado del modelo (0 si no existe); `porcentajeDescuento` usa el valor del modelo o el del resumen base; `totalLetras` por defecto `"CERO"` si no hay resumen base; `tributos` es `null` si no hay tributos; sin pagos |
| `MapRemissionNoteRelatedDocuments()`, `MapRemissionNoteThirdPartySale()`, `MapRemissionNoteAppendix()` | Devuelven `nil` si no hay datos |

### FSE (14)

| Mapper | Particularidad |
|---|---|
| `MapFSEResponseIssuer()` | Sin `TipoEstablecimiento` ni `NombreComercial`; `codActividad` y `descActividad` como punteros |
| `MapFSEResponseReceiver()` | Estructura `FSESubjectExcluded` (no `DTEReceiver`), publicada en la clave `receptor`; se remueven guiones del número de documento |
| `MapFSEResponseItems()` | Campo `Compra` en lugar de ventas; sin tributos; reenumerados desde 1 |
| `MapFSEResponseSummary()` | Campos: `totalCompra`, `descu`, `totalDescu`, `subTotal`, `reteRenta`, `totalPagar`, `totalLetras`, `condicionOperacion`, `pagos` (puntero, `null` si no hay), `observaciones` (`null` si vacío) |

El `apendice` de FSE es un puntero a arreglo: `nil` cuando no hay elementos.

### Invalidación

| Mapper | Particularidad |
|---|---|
| `MapIdentificationResponse()` | Campos `version`, `ambiente`, `codigoGeneracion`, `fecEmi`, `horEmi` y `fusion` (siempre `null`); no incluye `tipoDte` ni `numeroControl` |
| `MapIssuerResponse()` | Emisor reducido: `nit`, `nombre`, `codEstableMH`, `codEstable`, `codPuntoVentaMH`, `codPuntoVenta`, `telefono`, `correo` (sin NRC, dirección ni código de actividad) |
| `MapInvalidatedDocumentResponse()` | Campos propios: `tipoDte`, `codigoGeneracion`, `selloRecibido`, `numeroControl`, `fecEmi`, `codigoGeneracionR` (reemplazo, opcional), `tipoDocumento`, `numDocumento`, `nombre`, `telefono`, `correo` |
| `MapInvalidationReasonResponse()` | `tipoAnulacion`, `motivoAnulacion` (opcional), responsable (`nombreResponsable`, `tipDocResponsable`, `numDocResponsable`) y solicitante (`nombreSolicita`, `tipDocSolicita`, `numDocSolicita`) |

---

## Estructura Base JSON Hacienda

### DTE Estándar (Factura, CCF)

```json
{
  "identificacion": { "version", "ambiente", "tipoDte", "numeroControl", "codigoGeneracion", "tipoModelo", "tipoOperacion", "tipoContingencia", "motivoContin", "fecEmi", "horEmi", "tipoMoneda" },
  "emisor": { "nit", "nrc", "nombre", "codActividad", "descActividad", "direccion", "codEstable", "codPuntoVenta", ... },
  "receptor": { "nombre", "nrc", "direccion", ... },
  "cuerpoDocumento": [ { "numItem", "tipoItem", "descripcion", "cantidad", "precioUni", ... } ],
  "resumen": { "totalGravada", "subTotal", "totalPagar", "totalLetras", "tributos", "pagos", ... },
  "documentoRelacionado": [ ... ],
  "otrosDocumentos": [ ... ],
  "ventaTercero": { ... },
  "apendice": [ ... ]
}
```

El receptor de la Factura se identifica con `tipoDocumento`/`numDocumento`; el del CCF con `nit`. NC, ND y Retención usan `tipoDocumento`/`numDocumento` y no incluyen `otrosDocumentos`.

### Invalidación

```json
{
  "identificacion": { "version", "ambiente", "codigoGeneracion", "fecEmi", "horEmi", "fusion" },
  "emisor": { "nit", "nombre", "codEstableMH", "codEstable", "codPuntoVentaMH", "codPuntoVenta", "telefono", "correo" },
  "documento": { "tipoDte", "codigoGeneracion", "selloRecibido", "numeroControl", "fecEmi", "codigoGeneracionR", ... },
  "motivo": { "tipoAnulacion", "motivoAnulacion", "nombreResponsable", "tipDocResponsable", ... }
}
```

---

## Archivos de Estructuras de Salida

> **Directorio:** `pkg/mapper/response_mapper/structs/`

| Archivo | Estructuras |
|---|---|
| `common_structs.go` | DTEIdentification, DTEAdjustmentIdentification, DTEIssuer, DTEReceiverBase, DTEReceiver, DTEDocumentReceiver, DTEAddress, DTEItem, DTESummary, DTETax, DTEPayment, DTEApendice, DTERelatedDocument, DTEOtherDocument, DTEDoctor, DTEThirdPartySale |
| `invoice_structs.go` | InvoiceDTEResponse, InvoiceSummary, InvoiceReceiver, InvoiceItem |
| `ccf_structs.go` | CCFDTEResponse |
| `credit_note_structs.go` | CreditNoteDTEResponse, CreditNoteDTEIssuer, CreditNoteDTEItem, CreditNoteDTESummary |
| `debit_note_structs.go` | DebitNoteDTEResponse, DebitNoteDTEIssuer, DebitNoteDTEItem, DebitNoteDTESummary |
| `remission_note_structs.go` | MHRemissionNote, MHRemissionNoteReceiver, MHRemissionNoteItem, MHRemissionNoteSummary |
| `fse_structs.go` | FSEDTEResponse, FSEIssuer, FSESubjectExcluded, FSEItemResponse, FSESummaryResponse |
| `retention_struct.go` | RetentionDTEResponse, RetentionSummary, RetentionItem, RetentionIssuer |
| `invalidation_structs.go` | InvalidationResponse, InvalidationIdentification, InvalidationIssuer, DocumentResponse, ReasonResponse |

---

## Notas

1. **Interfaces como entrada**: Todos los mappers comunes usan interfaces (`interfaces.Issuer`, `interfaces.Item`, etc.) para desacoplar del modelo concreto. Los mappers específicos usan tipos concretos cuando necesitan campos exclusivos del tipo de DTE.
2. **Value objects**: Los montos financieros se extraen con `.GetValue()` sobre value objects inmutables del dominio.
3. **Redondeo de tributos**: `MapTaxes` usa `shopspring/decimal` con redondeo a 2 decimales.
4. **Campos nulos en JSON**: Los campos requeridos por el esquema que no aplican se serializan como `null` (punteros sin `omitempty`). Solo `nit` del emisor de NC/ND usa `omitempty`.
5. **FSE es diferente**: El JSON de FSE usa el objeto de sujeto excluido en la clave `receptor`, `compra` en lugar de ventas, y no tiene tributos por ítem.
6. **Invalidación es diferente**: Usa `documento` + `motivo` en lugar de `cuerpoDocumento` + `resumen`. La identificación usa `fecEmi`/`horEmi` (fecha y hora del evento).
