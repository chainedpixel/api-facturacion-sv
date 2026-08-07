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
  ├── [1] Validar campos requeridos del request
  │
  ├── [2] Mapear Identificación
  │     common.MapCommonRequestIdentification()
  │     → Version, Ambiente, TipoDTE, TipoModelo, TipoOperacion
  │     → FechaEmision (fecha actual), HoraEmision (hora actual), Moneda
  │
  ├── [3] Mapear Emisor (inyectado desde BD)
  │     common.MapCommonIssuer(issuer)
  │     → NIT, NRC, Nombre, CodigoActividad, Dirección, Teléfono, Email
  │     → CodigoEstablecimiento, CodigoPOS, NombreComercial
  │
  ├── [4] Mapear Receptor (específico por DTE)
  │     common.MapCommonRequestReceiver(req.Receiver) o mapper específico
  │     → Nombre, TipoDocumento, NumDocumento, Dirección, Email, NRC, NIT
  │
  ├── [5] Mapear Ítems (específico por DTE)
  │     invoice.MapInvoiceItems() / ccf.MapCCFItems() / etc.
  │     → Número, Tipo, Descripción, Cantidad, UnidadMedida, Precio, Descuento
  │
  ├── [6] Mapear Resumen (específico por DTE)
  │     invoice.MapInvoiceRequestSummary() / etc.
  │     → Totales, descuentos, impuestos, formas de pago
  │
  ├── [7] Mapear campos opcionales
  │     common.MapCommonOptionalToInputData()
  │     ├── Extensión (entrega/recepción)
  │     ├── Documentos relacionados
  │     ├── Otros documentos
  │     ├── Venta a terceros
  │     └── Apéndices
  │
  └── [8] Retornar modelo de dominio
```

---

## Estructuras de Request

### Estructuras Comunes (compartidas por múltiples DTEs)

| Estructura | Campos Principales |
|---|---|
| `ReceiverRequest` | Name, DocumentType, DocumentNumber, NRC, NIT, Address, Phone, Email, ActivityCode |
| `ItemRequest` | Number, Type, Description, Quantity, UnitMeasure, UnitPrice, Discount, Code, TaxCode |
| `SummaryRequest` | SubTotal, SubTotalSales, TotalOperation, TotalToPay, TotalInLetters, Taxes, Payments |
| `AddressRequest` | Department, Municipality, Complement |
| `ExtensionRequest` | DeliveryName, DeliveryDoc, ReceiverName, ReceiverDoc, Observation, VehiclePlate |
| `RelatedDocRequest` | DTEType, GenerationType, DocumentNumber, EmissionDate |
| `OtherDocRequest` | AssociatedDocCode, Description, Detail, DoctorInfo |
| `PaymentRequest` | Code, Amount, Period, Term, Reference |
| `TaxRequest` | Code, Description, Value |
| `AppendixRequest` | Field, Label, Value |
| `ThirdPartySaleRequest` | NIT, Name |

### Estructuras Específicas por DTE

#### Factura (01)

```go
type CreateInvoiceRequest struct {
    Items          []InvoiceItemRequest
    Receiver       *ReceiverRequest
    ModelType      int
    Summary        InvoiceSummaryRequest
    ThirdPartySale *ThirdPartySaleRequest   // opcional
    Extension      *ExtensionRequest        // opcional
    Payments       []PaymentRequest         // opcional
    OtherDocs      []OtherDocRequest        // opcional
    RelatedDocs    []RelatedDocRequest      // opcional
    Appendixes     []AppendixRequest        // opcional
}

type InvoiceItemRequest struct {
    ItemRequest                    // campos base
    NonSubjectSale  float64       // venta no sujeta
    ExemptSale      float64       // venta exenta
    TaxedSale       float64       // venta gravada
    SuggestedPrice  float64       // precio sugerido de venta
    NonTaxed        float64       // no gravado
    IVAItem         float64       // IVA por ítem
}

type InvoiceSummaryRequest struct {
    SummaryRequest                 // campos base
    TaxedDiscount    float64      // descuento gravado
    IVARetention     float64      // retención IVA
    IncomeRetention  float64      // retención renta
    TotalIVA         float64      // total IVA
    BalanceInFavor   float64      // saldo a favor
}
```

#### CCF (03)

```go
type CreateCreditFiscalRequest struct {
    Items       []CreditItemRequest
    Receiver    *ReceiverRequest         // requerido con NombreComercial
    ModelType   int
    Summary     CreditSummaryRequest
    // ... campos opcionales iguales a factura
}
```

**Validaciones específicas**: Receptor requerido con nombre comercial, `TotalIVA` debe ser 0, `DocumentType` y `DocumentNumber` del receptor no deben estar presentes.

#### Nota de Crédito (05) / Nota de Débito (06)

```go
type CreateCreditNoteRequest struct {
    Items       []CreditNoteItemRequest
    Receiver    *ReceiverRequest       // requerido
    Summary     CreditNoteSummaryRequest
    RelatedDocs []RelatedDocRequest    // requerido (al menos uno)
    // ... campos opcionales
}
```

**Validación**: Receptor y documentos relacionados son obligatorios.

#### Retención (07)

```go
type CreateRetentionRequest struct {
    Items      []RetentionItem
    Receiver   *ReceiverRequest    // requerido con DocumentType y DocumentNumber
    Summary    RetentionSummary
    Extension  *ExtensionRequest   // sin placa de vehículo
    Appendixes []AppendixRequest
}

type RetentionItem struct {
    Type           int        // tipo de documento
    DocumentNumber string     // número de documento
    Description    string
    RetentionCode  string     // código de retención (22, C4, C9)
    IVAAmount      *float64   // monto IVA (opcional)
    TaxedAmount    *float64   // monto gravado (opcional)
    EmissionDate   string     // fecha de emisión del doc original
    DTEType        *string    // tipo de DTE original
}
```

#### FSE (14)

```go
type CreateFSERequest struct {
    Items      []FSEItemRequest
    Receiver   *FSEReceiverRequest    // sujeto excluido
    Summary    FSESummaryRequest
    Extension  *ExtensionRequest
    Appendixes []AppendixRequest
}

type FSEItemRequest struct {
    ItemRequest
    Purchase float64    // monto de compra (debe ser > 0)
}

type FSEReceiverRequest struct {
    ReceiverRequest
    // DocumentType y DocumentNumber son requeridos
}
```

#### Nota de Remisión (04)

```go
type CreateRemissionNoteRequest struct {
    Receiver    *RemissionNoteReceiverRequest  // con BienTitulo
    Items       []RemissionNoteItemRequest
    Summary     RemissionNoteSummaryRequest
    // ... campos opcionales
}

type RemissionNoteReceiverRequest struct {
    ReceiverRequest
    BienTitulo string    // título de bien (requerido)
}
```

#### Invalidación

```go
type CreateInvalidationRequest struct {
    GenerationCode            string           // UUID del DTE a invalidar
    Reason                    ReasonRequest     // motivo de invalidación
    ReplacementGenerationCode *string           // UUID del DTE de reemplazo
}

type ReasonRequest struct {
    Type               int       // tipo: 1, 2, o 3
    ResponsibleName    string    // nombre del responsable
    ResponsibleDocType string    // tipo de documento
    ResponsibleDocNum  string    // número de documento
    RequestorName      string    // nombre del solicitante
    RequestorDocType   string    // tipo de documento
    RequestorDocNum    string    // número de documento
    Reason             *string   // motivo (requerido para tipo 3)
}
```

**Validaciones**:
- Tipo 1 o 3 → requiere `ReplacementGenerationCode`
- Tipo 2 → NO debe tener `ReplacementGenerationCode`
- Tipo 3 → requiere `Reason`

---

## Mappers Comunes

> **Directorio:** `pkg/mapper/request_mapper/common/`

Se reutilizan funciones de mapeo para los componentes compartidos por múltiples DTEs.

### Identificación

```go
func MapCommonRequestIdentification(model int, versionDoc int, typeToEmit string) (*models.Identification, error)
```

Se genera la identificación del DTE con la fecha/hora actual y los parámetros del tipo de documento.

### Emisor

```go
func MapCommonIssuer(client *dte.IssuerDTE) (*models.Issuer, error)
```

Se mapea la información del emisor desde la entidad de BD al modelo de dominio.

### Receptor

```go
func MapCommonRequestReceiver(receiver *ReceiverRequest) (*models.Receiver, error)
```

Se mapea el receptor con manejo de campos opcionales. **Validación**: DUI + NRC es una combinación inválida.

### Ítems

```go
func MapCommonRequestItems(items []ItemRequest) ([]models.Item, error)
func MapCommonRequestItem(item ItemRequest, index int) (*models.Item, error)
```

Se mapean ítems base con código, tipo impositivo, descuento, cantidad, unidad de medida y precio.

### Resumen

```go
func MapCommonRequestSummary(summary SummaryRequest) (*models.Summary, error)
```

Se mapea el resumen financiero con validación de campos requeridos: SubTotal, SubTotalSales, TotalOperation, TotalToPay.

### Campos Opcionales

```go
func MapCommonOptionalToInputData(fields CommonOptionalFields, dest *models.InputDataCommon) error
```

Se orquesta el mapeo de todos los campos opcionales: extensión, documentos relacionados, otros documentos, venta a terceros, apéndices.

### Otros Mappers Comunes

| Función | Propósito |
|---|---|
| `MapCommonRequestAddress()` | Se mapea dirección con validación de campos |
| `MapCommonRequestExtension()` | Se mapea extensión (entrega/recepción), placa 1-10 chars |
| `MapCommonRequestPaymentsType()` | Se mapean formas de pago |
| `MapCommonRequestSummaryTaxes()` | Se mapean tributos con validación de valor > 0 |
| `MapCommonRequestAppendix()` | Se mapean apéndices (field, label, value requeridos) |
| `MapCommonRequestRelatedDocuments()` | Se mapean documentos relacionados |
| `MapCommonRequestOtherDocuments()` | Se mapean otros documentos (médico requerido si código=3) |
| `MapCommonRequestThirdPartySale()` | Se mapea venta a terceros (NIT + nombre) |
| `ValidateRelatedDocs()` | Se validan documentos relacionados |

---

## Mappers Específicos por DTE

### Factura

> `pkg/mapper/request_mapper/invoice/`

| Función | Particularidad |
|---|---|
| `MapInvoiceItems()` | Se mapean ventas no sujetas, exentas, gravadas, precio sugerido, IVA por ítem |
| `MapInvoiceRequestSummary()` | Se genera `TotalEnLetras` automáticamente si no se proporciona |

### CCF

> `pkg/mapper/request_mapper/ccf/`

| Función | Particularidad |
|---|---|
| `MapCCFItems()` | Similar a factura sin campo `IVAItem` |
| `MapCCFRequestReceiver()` | Se requieren NIT, NRC, código/descripción de actividad |
| `MapCCFRequestSummary()` | Se mapean campos de percepción IVA |

### Notas de Crédito/Débito

> `pkg/mapper/request_mapper/credit_note/` y `debit_note/`

| Función | Particularidad |
|---|---|
| `MapCreditNoteRequestReceiver()` | Se requieren nombre, email, dirección, NRC, NIT, actividad, nombre comercial |

### Retención

> `pkg/mapper/request_mapper/retention/`

| Función | Particularidad |
|---|---|
| `MapRetentionItemList()` | Se mapean campos específicos: código retención, monto IVA, monto gravado, fecha emisión |

### FSE

> `pkg/mapper/request_mapper/fse/`

| Función | Particularidad |
|---|---|
| `MapFSEItems()` | Se mapea campo `Purchase` (compra) en lugar de ventas |
| `MapFSEReceiver()` | Se requiere tipo y número de documento del sujeto excluido |
| `MapFSESummary()` | Se mapean retenciones IVA/renta, observaciones |

### Nota de Remisión

> `pkg/mapper/request_mapper/remission_note/`

| Función | Particularidad |
|---|---|
| `MapRemissionNoteReceiver()` | Se requiere `BienTitulo` |

### Invalidación

> `pkg/mapper/request_mapper/invalidation/`

| Función | Particularidad |
|---|---|
| `MapInvalidatedDocument()` | Se extrae receptor y monto IVA del JSON del DTE original |

---

## Notas

1. **Validación en dos niveles**: Los request mappers validan la estructura del request (campos requeridos, formatos). Los servicios de dominio validan las reglas de negocio (cálculos, rangos, consistencia).
2. **Value objects**: Los mappers crean value objects del dominio (`Money`, `NIT`, `Location`, etc.) que son inmutables y se auto-validan en construcción.
3. **Fecha/hora automáticas**: La identificación siempre usa la fecha y hora actuales del servidor — no se aceptan del request.
4. **Emisor nunca del request**: El emisor se carga de la BD por `branchID` del token JWT. Nunca se confía en datos del emisor provenientes del request HTTP.
5. **Total en letras**: Para facturas, si `TotalEnLetras` no se proporciona, se genera automáticamente desde el monto usando `utils.InLetters()`.
