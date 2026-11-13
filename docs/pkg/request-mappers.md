# Request Mappers — HTTP Request → Modelo de Dominio

> **Paquete:** `pkg/mapper/request_mapper`
> **Estructuras:** `pkg/mapper/request_mapper/structs`
> **Mappers comunes:** `pkg/mapper/request_mapper/common`

## Descripción General

Los request mappers transforman las estructuras JSON recibidas en los endpoints HTTP a modelos de dominio que los servicios de validación y cálculo pueden procesar. Cada tipo de DTE tiene su propio mapper de nivel superior que orquesta el mapeo de sus componentes.

---

## Proceso de Mapeo General

```
CreateXXXRequest (JSON deserializado)
  │
  ├── [1] Validar campos requeridos del request (validateXXXRequest)
  │
  ├── [2] Mapear Ítems (específico por DTE)
  │     invoice.MapInvoiceItems() / ccf.MapCCFItems() / etc.
  │
  ├── [3] Mapear Receptor (varía por DTE, ver "Receptor por tipo de DTE")
  │
  ├── [4] Mapear Identificación
  │     common.MapCommonRequestIdentification(modelo, versión, tipo)
  │     → Version, Ambiente, TipoDTE, TipoModelo (previo), TipoOperacion (normal)
  │     → FechaEmision y HoraEmision (hora actual, utils.TimeNow()), Moneda (USD)
  │
  ├── [5] Mapear Resumen (específico por DTE)
  │     → Totales, descuentos, tributos, formas de pago; TotalEnLetras si no viene
  │
  ├── [6] Mapear Emisor (inyectado desde BD)
  │     common.MapCommonIssuer(issuer)
  │
  ├── [7] Mapear campos opcionales
  │     common.MapCommonOptionalToInputData()
  │     ├── Venta a terceros
  │     ├── Otros documentos
  │     ├── Documentos relacionados
  │     └── Apéndices
  │     (más las formas de pago de nivel raíz `payments`, que reemplazan a las del resumen)
  │
  └── [8] Retornar modelo de dominio
```

Los errores de cada paso se envuelven con `shared_error.NewFormattedGeneralServiceWithError(mapper, método, err, "ErrorMapping", "<DTE>-><Componente>")`. Ver [Errores (ServiceError)](shared-error.md).

Versión de esquema por tipo de DTE (valores fijos del código):

| DTE | Versión |
|---|---|
| Factura (01) | 2 |
| CCF (03) | 4 |
| Nota de Remisión (04) | 4 |
| Nota de Crédito (05) | 4 |
| Nota de Débito (06) | 4 |
| Retención (07) | 2 |
| FSE (14) | 2 |
| Invalidación | 3 (tipo de modelo 1; el tipo es el del DTE original) |

Los cálculos y fórmulas de montos no se documentan aquí; ver [Cálculos de DTE](../api/calculations.md).

---

## Estructuras de Request

### Estructuras Comunes (compartidas por múltiples DTEs)

| Estructura | Campos Principales |
|---|---|
| `ReceiverRequest` | DocumentType, DocumentNumber, Name, NRC, NIT, Address, Phone, Email, ActivityCode, ActivityDesc, CommercialName (todos punteros opcionales) |
| `ItemRequest` | Number, Type, Description, Quantity, UnitMeasure, UnitPrice, Discount, Code, TaxCode, RelatedDoc, Taxes |
| `SummaryRequest` | TotalNonSubject, TotalExempt, TotalTaxed, SubTotal, NonSubjectDiscount, ExemptDiscount, DiscountPercentage, TotalDiscount, TotalOperation, TotalNonTaxed, SubTotalSales, TotalToPay, OperationCondition, Taxes, PaymentTypes, TotalInWords, Observations |
| `AddressRequest` | Department, Municipality, District, Complement |
| `ExtensionRequest` | Remanente sin uso: ningún request lo incluye y no se envía a Hacienda (ver nota en "Campos Opcionales") |
| `RelatedDocRequest` | DocumentType, GenerationType, DocumentNumber, EmissionDate |
| `OtherDocRequest` | DocumentCode, Description, Detail, Doctor (`DoctorRequest`: Name, NIT, IdentificationDoc, ServiceType) |
| `PaymentRequest` | Code, Amount, Period, Term, Reference |
| `TaxRequest` | Code, Description, Value |
| `AppendixRequest` | Field, Label, Value |
| `ThirdPartySaleRequest` | NIT, Name, FiscalDomicile (CAT-032: 1 Domiciliado, 2 No Domiciliado) |

### Estructuras Específicas por DTE

#### Factura (01)

```go
type CreateInvoiceRequest struct {
    Items          []InvoiceItemRequest
    Receiver       *ReceiverRequest         // opcional
    ModelType      int
    Summary        *InvoiceSummaryRequest
    ThirdPartySale *ThirdPartySaleRequest   // opcional
    Payments       []PaymentRequest         // opcional
    OtherDocs      []OtherDocRequest        // opcional
    RelatedDocs    []RelatedDocRequest      // opcional
    Appendixes     []AppendixRequest        // opcional
}

type InvoiceItemRequest struct {
    ItemRequest
    NonSubjectSale, ExemptSale, TaxedSale, SuggestedPrice, NonTaxed, IVAItem float64
}

type InvoiceSummaryRequest struct {
    SummaryRequest
    TaxedDiscount, IVARetention, IncomeRetention, TotalIVA, BalanceInFavor float64
}
```

**Validaciones** (`validateInvoiceRequest`): `Items` y `Summary` requeridos. Si viene `Receiver`, `DocumentType` y `DocumentNumber` deben venir juntos (error `InvalidDocumentTypeAndNumber`) y `NIT` no se admite (`InvalidFieldValue`).

#### CCF (03)

```go
type CreateCreditFiscalRequest struct {
    Items          []CreditItemRequest
    Receiver       *ReceiverRequest         // requerido
    ModelType      int
    Summary        *CreditSummaryRequest
    ThirdPartySale *ThirdPartySaleRequest   // opcional
    Payments       []PaymentRequest         // opcional
    OtherDocs      []OtherDocRequest        // opcional
    RelatedDocs    []RelatedDocRequest      // opcional
    Appendixes     []AppendixRequest        // opcional
}
// CreditItemRequest: igual a InvoiceItemRequest sin IVAItem
// CreditSummaryRequest: igual a InvoiceSummaryRequest más IVAPerception
```

**Validaciones** (`validateCCFRequest`): `Items`, `Summary` y `Receiver` requeridos; `Summary.TotalIVA` debe ser 0; `Receiver.DocumentType` y `Receiver.DocumentNumber` no deben estar presentes (`InvalidField`). El receptor del CCF se identifica por NIT y NRC.

#### Nota de Crédito (05) / Nota de Débito (06)

```go
type CreateCreditNoteRequest struct {
    Items          []CreditNoteItemRequest   // NonSubjectSale, ExemptSale, TaxedSale
    Receiver       *ReceiverRequest          // requerido
    ModelType      int
    Summary        *CreditNoteSummaryRequest // TaxedDiscount, IVAPerception, IVARetention, IncomeRetention
    ThirdPartySale *ThirdPartySaleRequest    // opcional
    Payments       []PaymentRequest          // opcional
    OtherDocs      []OtherDocRequest         // opcional
    RelatedDocs    []RelatedDocRequest       // requerido (al menos uno)
    Appendixes     []AppendixRequest         // opcional
}
```

`CreateDebitNoteRequest` tiene la misma forma (`DebitNoteItemRequest`, `DebitNoteSummaryRequest`).

**Validaciones**: `Items`, `Summary`, `Receiver` y al menos un elemento de `RelatedDocs` son requeridos; además se ejecuta `common.ValidateRelatedDocs`. En NC/ND los documentos relacionados se mapean directamente con `MapCommonRequestRelatedDocuments` y no se pasan a `MapCommonOptionalToInputData`.

#### Retención (07)

```go
type CreateRetentionRequest struct {
    Items      []RetentionItem
    Summary    *RetentionSummary
    Receiver   *ReceiverRequest    // requerido, con DocumentType y DocumentNumber
    Appendixes []AppendixRequest
}

type RetentionItem struct {
    DocumentType   int        // json "type": tipo de generación del documento (1 físico, 2 electrónico)
    DocumentNumber string
    Description    string
    RetentionCode  string     // código CAT-006 (22, C4, C9)
    IvaAmount      *float64   // requerido
    TaxedAmount    *float64   // requerido
    EmissionDate   *string    // requerido
    DTEType        *string    // requerido
}

type RetentionSummary struct {
    TotalRetentionAmount float64   // requerido (distinto de 0)
    TotalIVA             float64   // opcional; si es 0 se calcula
    TotalRetentionIVA    float64   // requerido (distinto de 0)
    Observations         *string   // máximo 3000 caracteres
}
```

**Validaciones**: `Summary` y `Receiver` requeridos (con `DocumentType` y `DocumentNumber`); todos los ítems deben compartir el mismo código de retención (`retention.SingleRetentionCode`, error `MixedRetentionCodes`). Si `TotalIVA` no se envía, se calcula con `retention_models.ExpectedTotalIVA`, que solo aplica cuando todos los documentos son electrónicos (`retention.AllElectronic`) y el código es C4. El request no incluye extensión.

#### FSE (14)

```go
type CreateFSERequest struct {
    Items      []FSEItemRequest
    Receiver   *FSEReceiverRequest   // json "excluded_subject"
    Summary    *FSESummaryRequest
    Appendixes []AppendixRequest
}

type FSEItemRequest struct {
    ItemRequest
    Purchase float64    // debe ser > 0
}

type FSESummaryRequest struct {
    SummaryRequest
    TotalPurchase, IVARetention, IncomeRetention float64
    Observations *string
}

type FSEReceiverRequest struct {
    ReceiverRequest
    DocumentType        string   // requerido
    DocumentNumber      string   // requerido
    ActivityCode        *string
    ActivityDescription *string  // máximo 150 caracteres
}
```

**Validaciones**: `Items`, `Summary`, `Receiver` requeridos; tipo de documento válido (NIT, DUI, carnet de residente, pasaporte u otro); longitud del número según tipo (NIT 9 o 14 dígitos, DUI 9, otros máximo 20); nombre y dirección del receptor requeridos (el esquema `fe-fse-v2` no admite `direccion` nula).

#### Nota de Remisión (04)

```go
type CreateRemissionNoteRequest struct {
    Receiver       *RemissionNoteReceiverRequest
    Items          []*RemissionNoteItemRequest
    Summary        *RemissionNoteSummaryRequest
    Appendixes     []*AppendixRequest
    ThirdPartySale *ThirdPartySaleRequest
    RelatedDocs    []*RelatedDocRequest
}

type RemissionNoteReceiverRequest struct {
    *ReceiverRequest
    BienTitulo *string    // json "real_state", exactamente 2 caracteres
}
```

El resumen usa nombres propios (`non_subject_total`, `subtotal_sales`, `total_amount`, `amount_in_words`, `payments`, `tributes`, etc.) y los ítems usan punteros y `tributes` en lugar de `taxes`.

**Validaciones**: `Receiver`, `Items`, `Summary` y `BienTitulo` requeridos; el receptor exige nombre, tipo y número de documento, correo y dirección; con tipo de documento `36` también NIT, código y descripción de actividad. Cada ítem debe tener al menos un tipo de venta con valor.

#### Invalidación

```go
type CreateInvalidationRequest struct {
    GenerationCode            string          // UUID del DTE a invalidar
    Reason                    *ReasonRequest
    ReplacementGenerationCode *string         // UUID del DTE de reemplazo
}

type ReasonRequest struct {
    Type               int
    ResponsibleName    string
    ResponsibleDocType string
    ResponsibleNumDoc  string   // json "responsible_num_doc"
    RequestorName      string
    RequestorDocType   string
    RequestorNumDoc    string   // json "requestor_num_doc"
    Reason             *string  // json "reason_field", requerido para tipo 3
}
```

**Validaciones** (`ValidateInvalidationReRequest`):
- Tipo 1 o 3 requiere `ReplacementGenerationCode` (`InvalidReplacementCode1And3`).
- Tipo 2 NO admite `ReplacementGenerationCode` (`InvalidInvalidationType2`).
- Tipo 3 requiere `Reason` (`InvalidInvalidationType3`); el motivo solo se mapea cuando el tipo es 3.
- `GenerationCode`, `Type`, y nombre, tipo y número de documento de responsable y solicitante son requeridos.

---

## Receptor por tipo de DTE

| DTE | Función | Campos requeridos | Notas |
|---|---|---|---|
| Factura | `common.MapCommonRequestReceiver` | ninguno (receptor opcional) | Todos los campos son opcionales; DUI con NRC es inválido (`InvalidFieldValue`) |
| CCF | `ccf.MapCCFRequestReceiver` | Name, Address, NIT, NRC, ActivityCode, ActivityDesc | Se identifica por NIT y NRC; no admite `DocumentType`/`DocumentNumber`. `CommercialName` es opcional |
| NC y ND | `common.MapAdjustmentRequestReceiver` (vía `MapCreditNoteRequestReceiver` / `MapDebitNoteRequestReceiver`) | Name, Email, Address, NRC, NIT, ActivityCode, ActivityDesc | El request identifica al receptor por NIT (se compara con el del CCF relacionado); a Hacienda se envía como tipo de documento `36` (NIT) y número igual al NIT (Anexo IV, campos 38.3 y 39.6, cuando el documento a ajustar es un CCF) |
| Retención | `retention.MapRetentionRequestReceiver` | Name, Email, Address, DocumentType, DocumentNumber, ActivityCode, ActivityDesc | NRC y teléfono opcionales |
| FSE | `fse.MapFSERequestReceiver` | Name, Address, DocumentType, DocumentNumber | Ver validaciones de FSE |
| Remisión | `remission_note.MapRemissionNoteRequestReceiver` | Name, DocumentType, DocumentNumber, Email, Address, BienTitulo | Con tipo `36` también NIT, ActivityCode y ActivityDesc |

---

## Mappers Comunes

> **Directorio:** `pkg/mapper/request_mapper/common/`

Se reutilizan funciones de mapeo para los componentes compartidos por múltiples DTEs.

### Identificación

```go
func MapCommonRequestIdentification(model, versionDoc int, typeToEmit string) (*models.Identification, error)
```

Se genera la identificación del DTE con la fecha/hora actual (`utils.TimeNow()`), moneda USD, tipo de operación normal y los parámetros del tipo de documento.

### Emisor

```go
func MapCommonIssuer(client *dte.IssuerDTE) (*models.Issuer, error)
```

Se mapea la información del emisor desde la entidad de BD al modelo de dominio (incluye códigos de establecimiento y punto de venta, locales y de Hacienda).

### Receptor

```go
func MapCommonRequestReceiver(receiver *ReceiverRequest) (*models.Receiver, error)
func MapAdjustmentRequestReceiver(receiver *ReceiverRequest) (*models.Receiver, error)
```

`MapCommonRequestReceiver` mapea el receptor con campos opcionales (devuelve `nil` si el receptor es nulo; si no hay dirección se usa una dirección vacía). **Validación**: DUI + NRC es una combinación inválida.

`MapAdjustmentRequestReceiver` (archivo `adjustment_receiver_map.go`) es la variante de NC/ND: exige Name, Email, Address, NRC, NIT, ActivityCode y ActivityDesc, y fija el tipo de documento en NIT con el NIT como número. Ver tabla "Receptor por tipo de DTE".

### Ítems

```go
func MapCommonRequestItems(items []ItemRequest) ([]models.Item, error)
func MapCommonRequestItem(item ItemRequest, index int) (*models.Item, error)
```

Se mapean ítems base con código, tipo impositivo, descuento, cantidad, unidad de medida y precio. El número de ítem se asigna como `index + 1`, sin tomarlo del request. Los tributos del ítem (`Taxes`) se validan como códigos de tributo.

### Resumen

```go
func MapCommonRequestSummary(summary SummaryRequest) (*models.Summary, error)
```

Se mapea el resumen financiero con validación de campos requeridos: SubTotal, SubTotalSales, TotalOperation, TotalToPay (distintos de 0) y `Observations` de máximo 3000 caracteres. Los montos de totales usan `financial.NewAmountForTotal` (2 decimales).

### Campos Opcionales

```go
func MapCommonOptionalToInputData(fields CommonOptionalFields, dest *models.InputDataCommon) error
```

Se orquesta el mapeo de los campos opcionales: venta a terceros, extensión, otros documentos, documentos relacionados y apéndices. Se detiene en el primer error.

> **Nota:** `CommonOptionalFields.Extension` y `MapCommonRequestExtension` (`extension_map.go`) siguen existiendo, pero ningún request de creación incluye el campo `extension`, por lo que ningún mapper de DTE lo puebla ni pasa ese campo a `MapCommonOptionalToInputData`. Es código remanente sin efecto: la extensión no se envía a Hacienda (tampoco existe mapper de respuesta de extensión).

### Otros Mappers Comunes

| Función | Propósito |
|---|---|
| `MapCommonRequestAddress()` | Se mapea dirección; departamento, municipio y complemento son requeridos (error `AddressWithReceiver`); `District` toma `"01"` si no viene |
| `MapClientAddress()` | Se mapea la dirección del emisor desde BD (distrito por defecto `"01"`) |
| `MapCommonRequestExtension()` | Se mapea extensión (entrega/recepción), placa 1-10 caracteres (sin uso por los request actuales) |
| `MapCommonRequestPaymentsType()` | Se mapean formas de pago (código y monto distintos de vacío/0) |
| `MapCommonRequestSummaryTaxes()` | Se mapean tributos del resumen (código y descripción requeridos; valor distinto de 0, salvo IVA exportación) con 2 decimales |
| `MapCommonRequestAppendix()` | Se mapean apéndices (field, label, value requeridos) |
| `MapCommonRequestRelatedDocuments()` | Se mapean documentos relacionados; si no viene fecha de emisión se usa la fecha actual |
| `MapCommonRequestOtherDocuments()` | Se mapean otros documentos (médico requerido si código=3) |
| `MapCommonRequestDoctorInfo()` | Se mapea el médico: nombre y tipo de servicio requeridos, y NIT o identificación |
| `MapCommonRequestThirdPartySale()` | Se mapea venta a terceros (NIT y nombre requeridos; `FiscalDomicile` CAT-032 requerido) |
| `ValidateRelatedDocs()` | Se validan documentos relacionados (tipo, número, tipo de generación; fecha de emisión obligatoria si el documento es físico) |

---

## Mappers Específicos por DTE

### Factura

> `pkg/mapper/request_mapper/invoice/`

| Función | Particularidad |
|---|---|
| `MapInvoiceItems()` | Se mapean ventas no sujetas, exentas, gravadas, precio sugerido, IVA por ítem |
| `MapInvoiceRequestSummary()` | Se genera `TotalEnLetras` desde `TotalToPay` si no se proporciona |

### CCF

> `pkg/mapper/request_mapper/ccf/`

| Función | Particularidad |
|---|---|
| `MapCCFItems()` | Similar a factura sin campo `IVAItem` |
| `MapCCFRequestReceiver()` | Se requieren NIT, NRC, nombre, dirección, código y descripción de actividad |
| `MapCCFRequestSummary()` | Se mapean campos de percepción y retención de IVA; `TotalEnLetras` desde `TotalToPay` |

El nombre comercial del receptor (`CommercialName`, Anexo IV campo 45) es opcional; se envía `null` si no viene.

### Notas de Crédito/Débito

> `pkg/mapper/request_mapper/credit_note/` y `debit_note/`

| Función | Particularidad |
|---|---|
| `MapCreditNoteRequestReceiver()` / `MapDebitNoteRequestReceiver()` | Delegan en `common.MapAdjustmentRequestReceiver` |
| `MapCreditNoteRequestSummary()` / `MapDebitNoteRequestSummary()` | `TotalEnLetras` desde `TotalOperation` si no se proporciona |

### Retención

> `pkg/mapper/request_mapper/retention/`

| Función | Particularidad |
|---|---|
| `MapRetentionItemList()` | Se mapean código de retención, monto IVA, monto gravado, fecha de emisión y tipo de DTE (todos requeridos) |
| `SingleRetentionCode()` | Se rechazan códigos de retención distintos en un mismo CRE |
| `AllElectronic()` | Se indica si todos los documentos son electrónicos |
| `MapRetentionSummary()` | Se calcula `TotalIVA` si no viene (solo documentos electrónicos con código C4) |

### FSE

> `pkg/mapper/request_mapper/fse/`

| Función | Particularidad |
|---|---|
| `MapFSEItems()` | Se mapea campo `Purchase` (compra, debe ser > 0) en lugar de ventas |
| `MapFSERequestReceiver()` | Se requiere tipo y número de documento del sujeto excluido |
| `MapFSERequestSummary()` | Se mapean total de compra y retenciones IVA/renta, observaciones |

### Nota de Remisión

> `pkg/mapper/request_mapper/remission_note/`

| Función | Particularidad |
|---|---|
| `MapRemissionNoteRequestReceiver()` | Se requiere `BienTitulo` (2 caracteres) |
| `MapRemissionNoteItems()` | Se exige al menos un tipo de venta con valor por ítem |
| `MapRemissionNoteSummaryFromRequest()` | `TotalEnLetras` desde `TotalAmount` si no se proporciona |

### Invalidación

> `pkg/mapper/request_mapper/invalidation/`

| Función | Particularidad |
|---|---|
| `MapInvalidationReasonRequest()` | Se mapea motivo, responsable y solicitante; el texto del motivo solo aplica al tipo 3 |
| `MapInvalidatedDocument()` | Se extrae receptor (o sujeto excluido), monto IVA y actividad del emisor del JSON del DTE original; requiere sello de recepción (`DocumentWithoutStamp`) |

`MapInvalidatedDocument` obtiene la fecha de generación del DTE original con `utils.GenerationDateOrDefault` (usa la `emissionDate` recibida si el JSON no la trae) y la fecha del sello con `utils.ParseProcessingDate` sobre `ProcessingDate` del DTE (si no se puede interpretar, usa `emissionDate`). Ver [Utilidades](utils.md).

`InvalidationMapper.MapToInvalidationData` genera un nuevo código de generación (UUID en mayúsculas) y fija la fecha y hora del evento con `policy.EventDateTime(tipoDTE, fechaGeneración, ahora)`, en lugar de la hora actual directa.

---

## Notas

1. **Validación en dos niveles**: Los request mappers validan la estructura del request (campos requeridos, formatos). Los servicios de dominio validan las reglas de negocio (cálculos, rangos, consistencia).
2. **Value objects**: Los mappers crean value objects del dominio (`Money`, `NIT`, `Location`, etc.) que son inmutables y se auto-validan en construcción.
3. **Fecha/hora automáticas**: La identificación siempre usa la fecha y hora actuales del servidor — no se aceptan del request (salvo la invalidación, ver arriba).
4. **Emisor nunca del request**: El emisor se carga de la BD por `branchID` del token JWT. Nunca se confía en datos del emisor provenientes del request HTTP.
5. **Total en letras**: Si `TotalInWords` no se proporciona, se genera con `utils.InLetters()`. El monto base es `TotalToPay` (Factura, CCF, FSE), `TotalOperation` (NC, ND) o `TotalAmount` (Nota de Remisión).
