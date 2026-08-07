# Response Mappers — Modelo de Dominio → Formato Hacienda

> **Paquete:** `pkg/mapper/response_mapper`
> **Estructuras:** `pkg/mapper/response_mapper/structs`
> **Mappers comunes:** `pkg/mapper/response_mapper/common`

## Descripción General

Los response mappers transforman los modelos de dominio (resultado del procesamiento de servicios y validaciones) al formato JSON que la API de Hacienda espera recibir. Cada tipo de DTE tiene su propio mapper de nivel superior y mappers específicos para componentes como emisor, receptor, ítems y resumen.

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
| `ToMHRemissionNote(doc)` | `*RemissionNoteModel` | `*MHRemissionNote` |
| `ToMHFSE(doc)` | `*FSEModel` | `*FSEDTEResponse` |
| `ToMHInvalidation(doc)` | `*InvalidationDocument` | `*InvalidationResponse` |

---

## Mappers Comunes

Se reutilizan para los componentes compartidos entre múltiples tipos de DTE.

| Función | Entrada | Salida |
|---|---|---|
| `MapCommonResponseIdentification()` | `interfaces.Identification` | `*DTEIdentification` |
| `MapCommonResponseIssuer()` | `interfaces.Issuer` | `DTEIssuer` |
| `MapCommonResponseReceiver()` | `interfaces.Receiver` | `DTEReceiver` |
| `MapCommonResponseAddress()` | `interfaces.Address` | `DTEAddress` |
| `MapCommonItems()` | `interfaces.Item` | `DTEItem` |
| `MapCommonResponseSummary()` | `interfaces.Summary` | `*DTESummary` |
| `MapTaxes()` | `[]interfaces.Tax` | `[]DTETax` |
| `MapCommonResponsePayments()` | `[]interfaces.PaymentType` | `[]DTEPayment` |
| `MapCommonResponseExtension()` | `interfaces.Extension` | `*DTEExtension` |
| `MapCommonResponseRelatedDocuments()` | `[]interfaces.RelatedDocument` | `[]DTERelatedDocument` |
| `MapCommonResponseOtherDocuments()` | `[]interfaces.OtherDocuments` | `[]DTEOtherDocument` |
| `MapCommonResponseThirdPartySale()` | `interfaces.ThirdPartySale` | `*DTEThirdPartySale` |
| `MapCommonResponseAppendix()` | `[]interfaces.Appendix` | `[]DTEApendice` |

### Transformaciones Clave

- **Fechas**: Formato `"2006-01-02"` (YYYY-MM-DD)
- **Horas**: Formato `"15:04:05"` (HH:MM:SS)
- **Campos opcionales**: Se convierten a puntero; si están vacíos se asigna `nil`
- **Dirección**: Se retorna `nil` si todos los campos están vacíos
- **Tributos**: Se redondean a 2 decimales usando `shopspring/decimal`

---

## Mappers Específicos por DTE

### Factura (01)

| Mapper | Particularidad |
|---|---|
| `MapInvoiceResponseReceiver()` | Sin campo `NIT` (a diferencia de `DTEReceiver`) |
| `MapInvoiceResponseItem()` | Se agregan: `VentaNoSuj`, `VentaExenta`, `VentaGravada`, `PSV`, `NoGravado`, `IvaItem` |
| `MapInvoiceResponseSummary()` | Se agregan: `DescuGravada`, `IvaRete1`, `TotalIva`, `SaldoFavor`, `ReteRenta` |

### CCF (03)

| Mapper | Particularidad |
|---|---|
| `MapCCFResponseItem()` | Se agregan: `VentaNoSuj`, `VentaExenta`, `VentaGravada`, `PSV`, `NoGravado` (sin `IvaItem`) |
| `MapCCFResponseSummary()` | Se agregan: `DescuGravada`, `IvaRete1`, `IvaPerci1`, `ReteRenta`, `SaldoFavor` |

### Nota de Crédito (05)

| Mapper | Particularidad |
|---|---|
| `MapCreditNoteIssuer()` | Emisor simplificado (sin códigos POS ni establecimiento) |
| `MapCreditNoteResponseItem()` | Se agrega `CodTributo` como puntero; sin `PSV`, `NoGravado`, `IvaItem` |
| `MapCreditNoteResponseSummary()` | Se agregan: `DescuGravada`, `IvaRete1`, `IvaPerci1`, `ReteRenta`; sin pagos |
| `MapCreditNoteResponseExtension()` | Sin `PlacaVehiculo` |

### Nota de Débito (06)

| Mapper | Particularidad |
|---|---|
| Estructura idéntica a Nota de Crédito | Se agrega `NumPagoElectronico` en resumen |

### Retención (07)

| Mapper | Particularidad |
|---|---|
| `MapRetentionResponseIssuer()` | Emisor completo con todos los campos opcionales |
| `MapRetentionResponseItem()` | Estructura diferente: `TipoDTE`, `TipoDoc`, `NumDoc`, `FechaEmision`, `MontoSujetoGravado`, `CodigoRetencionMH`, `IvaRetenido` |
| `MapRetentionResponseSummary()` | Solo 3 campos: `TotalIvaRetenido`, `TotalSujRetencion`, `TotalIvaRetenidoLetras` |
| `MapRetentionResponseExtension()` | Sin `PlacaVehiculo` |

### Nota de Remisión (04)

| Mapper | Particularidad |
|---|---|
| `MapRemissionNoteReceiver()` | Se agrega `BienTitulo` (default `"99"`); NRC = `nil` si `DocumentType == "13"` |
| `MapRemissionNoteItems()` | Se agregan: `VentaNoSuj`, `VentaExenta`, `VentaGravada`, `Codigo`, `CodTributo` |
| `MapRemissionNoteSummary()` | `TaxedDiscount` hardcodeado a `0.0`; `TotalLetras` default `"CERO"` |

### FSE (14)

| Mapper | Particularidad |
|---|---|
| `MapFSEResponseIssuer()` | Sin `TipoEstablecimiento` ni `NombreComercial` |
| `MapFSEResponseReceiver()` | Struct `FSESubjectExcluded` (no `DTEReceiver`); se remueven guiones del documento |
| `MapFSEResponseItems()` | Campo `Compra` en lugar de ventas; sin tributos; reenumerados 1-indexed |
| `MapFSEResponseSummary()` | Campos: `TotalCompra`, `Descu`, `IvaRete1`, `ReteRenta`; con `Observaciones` |

### Invalidación

| Mapper | Particularidad |
|---|---|
| `MapIdentificationResponse()` | Usa `FecAnula`/`HorAnula` en lugar de `FecEmi`/`HorEmi`; sin `TipoDte`, `NumeroControl`, etc. |
| `MapIssuerResponse()` | Emisor reducido: sin NRC, dirección, código actividad |
| `MapInvalidatedDocumentResponse()` | Campos únicos: `SelloRecibido`, `MontoIva`, `CodigoGeneracionR` (reemplazo) |
| `MapInvalidationReasonResponse()` | Estructura completa: tipo, motivo, responsable, solicitante |

---

## Estructura Base JSON Hacienda

### DTE Estándar (Factura, CCF, NC, ND)

```json
{
  "identificacion": { "version", "ambiente", "tipoDte", "numeroControl", "codigoGeneracion", "tipoModelo", "tipoOperacion", "fecEmi", "horEmi", "tipoMoneda" },
  "emisor": { "nit", "nrc", "nombre", "codActividad", "descActividad", "direccion", ... },
  "receptor": { "nombre", "tipoDocumento", "numDocumento", "direccion", ... },
  "cuerpoDocumento": [ { "numItem", "tipoItem", "descripcion", "cantidad", "precioUni", ... } ],
  "resumen": { "totalGravada", "subTotal", "totalPagar", "totalLetras", "tributos", "pagos", ... },
  "extension": { ... },
  "documentoRelacionado": [ ... ],
  "otrosDocumentos": [ ... ],
  "ventaTercero": { ... },
  "apendice": [ ... ]
}
```

### Invalidación

```json
{
  "identificacion": { "version", "ambiente", "codigoGeneracion", "fecAnula", "horAnula" },
  "emisor": { "nit", "nombre", "tipoEstablecimiento", "telefono", "correo" },
  "documento": { "tipoDte", "codigoGeneracion", "selloRecibido", "numeroControl", "fecEmi", "montoIva", ... },
  "motivo": { "tipoAnulacion", "motivoAnulacion", "nombreResponsable", "tipDocResponsable", ... }
}
```

---

## Archivos de Estructuras de Salida

> **Directorio:** `pkg/mapper/response_mapper/structs/`

| Archivo | Estructuras |
|---|---|
| `common_structs.go` | DTEIdentification, DTEIssuer, DTEReceiver, DTEAddress, DTEItem, DTESummary, DTETax, DTEPayment, DTEExtension, DTEApendice, DTERelatedDocument, DTEOtherDocument, DTEDoctor, DTEThirdPartySale |
| `invoice_structs.go` | InvoiceDTEResponse, InvoiceSummary, InvoiceReceiver, InvoiceItem |
| `ccf_structs.go` | CCFDTEResponse |
| `credit_note_structs.go` | CreditNoteDTEResponse, CreditNoteDTEIssuer, CreditNoteDTEItem, CreditNoteDTESummary, CreditNoteDTEExtension |
| `debit_note_structs.go` | DebitNoteDTEResponse, DebitNoteDTEIssuer, DebitNoteDTEItem, DebitNoteDTESummary, DebitNoteDTEExtension |
| `remission_note_structs.go` | MHRemissionNote, MHRemissionNoteReceiver, MHRemissionNoteItem, MHRemissionNoteSummary |
| `fse_structs.go` | FSEDTEResponse, FSEIssuer, FSESubjectExcluded, FSEItemResponse, FSESummaryResponse |
| `retention_struct.go` | RetentionDTEResponse, RetentionSummary, RetentionItem, RetentionIssuer, RetentionExtension |
| `invalidation_structs.go` | InvalidationResponse, InvalidationIdentification, InvalidationIssuer, DocumentResponse, ReasonResponse |

---

## Notas

1. **Interfaces como entrada**: Todos los mappers comunes usan interfaces (`interfaces.Issuer`, `interfaces.Item`, etc.) para desacoplar del modelo concreto. Los mappers específicos usan tipos concretos cuando necesitan campos exclusivos del tipo de DTE.
2. **Value objects**: Los montos financieros se extraen con `.GetValue()` sobre value objects inmutables del dominio.
3. **Redondeo de tributos**: Se usa `shopspring/decimal` con `Floor`/`Round` a 2 decimales para garantizar precisión monetaria.
4. **Campos nulos en JSON**: Los campos con `omitempty` en los tags JSON se omiten cuando son `nil` o zero-value. Los punteros a string se usan para campos opcionales que Hacienda no requiere.
5. **FSE es diferente**: El JSON de FSE usa `sujetoExcluido` en lugar de `receptor`, `compra` en lugar de ventas, y no tiene tributos por ítem.
6. **Invalidación es diferente**: Usa `documento` + `motivo` en lugar de `cuerpoDocumento` + `resumen`. Los campos de fecha usan prefijo `Anula` (`fecAnula`, `horAnula`).
