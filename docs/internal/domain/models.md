# Modelos del Dominio

> **Paquete base:** `internal/domain/dte/common/models`
> **Paquete core:** `internal/domain/core`

## Descripción General

Los modelos del dominio representan las entidades y agregados del negocio. Se organizan en dos categorías: modelos comunes compartidos entre DTEs y modelos core de persistencia.

---

## Modelos Comunes del DTE

### DTEDocument — Agregado Raíz

```go
type DTEDocument struct {
    Identification   interfaces.Identification
    Issuer           interfaces.Issuer
    Receiver         interfaces.Receiver
    Items            []interfaces.Item
    Summary          interfaces.Summary
    Extension        interfaces.Extension
    Appendix         []interfaces.Appendix
    RelatedDocuments []interfaces.RelatedDocument
    OtherDocuments   []interfaces.OtherDocuments
    ThirdPartySale   interfaces.ThirdPartySale
}
```

El `DTEDocument` es el agregado raíz que todos los DTEs específicos embeben. Implementa `DTEDocumentGetter`, `DTEDocumentSetter` y `Validate()`.

**Métodos clave:**
- `Validate()` — Ejecuta `ValidateModel[T]()` para validar VOs por reflexión
- `ValidateDTERules()` — Ejecuta estrategias de validación de negocio

---

### InputDataCommon — Input Genérico

```go
type InputDataCommon struct {
    Identification *Identification
    Issuer         *Issuer
    Receiver       *Receiver
    Extension      *Extension
    RelatedDocs    []RelatedDocument
    OtherDocs      []OtherDocument
    ThirdPartySale *ThirdPartySale
    Appendixes     []Appendix
}
```

Estructura de entrada compartida que cada DTE extiende con sus ítems y resumen específicos.

---

### Identification

```go
type Identification struct {
    Version           document.Version
    Ambient           document.Ambient
    DTEType           document.DTEType
    ControlNumber     identification.ControlNumber
    GenerationCode    identification.GenerationCode
    ModelType         document.ModelType
    OperationType     document.OperationType
    EmissionDate      temporal.EmissionDate
    EmissionTime      temporal.EmissionTime
    Currency          financial.Currency
    ContingencyType   *document.ContingencyType
    ContingencyReason *document.ContingencyReason
}
```

| Campo | Value Object | Descripción |
|---|---|---|
| `Version` | `document.Version` | Versión del esquema del DTE |
| `Ambient` | `document.Ambient` | `"00"` test, `"01"` producción |
| `DTEType` | `document.DTEType` | Código del tipo de DTE (`01`, `03`, etc.) |
| `ControlNumber` | `identification.ControlNumber` | Número de control secuencial |
| `GenerationCode` | `identification.GenerationCode` | UUID v4 único del documento |
| `ModelType` | `document.ModelType` | `1` previo, `2` diferido (contingencia) |
| `OperationType` | `document.OperationType` | Tipo de transmisión |
| `EmissionDate` | `temporal.EmissionDate` | Fecha de emisión |
| `EmissionTime` | `temporal.EmissionTime` | Hora de emisión |
| `Currency` | `financial.Currency` | Código de moneda |
| `ContingencyType` | `*document.ContingencyType` | Tipo de contingencia (1-5, opcional) |
| `ContingencyReason` | `*document.ContingencyReason` | Motivo de contingencia (opcional) |

**Métodos especiales:**
- `GenerateCode()` — Genera un UUID v4 y lo asigna al `GenerationCode`
- `SetDTETypeForce()` — Asigna el tipo sin validación (uso interno)

---

### Issuer

```go
type Issuer struct {
    NIT                 identification.NIT
    NRC                 identification.NRC
    Name                string
    ActivityCode        identification.ActivityCode
    ActivityDescription string
    EstablishmentType   document.EstablishmentType
    Address             interfaces.Address
    Phone               base.Phone
    Email               base.Email
    CommercialName      string
    EstablishmentCode   *string
    EstablishmentMHCode *string
    POSCode             *string
    POSMHCode           *string
}
```

Representa al emisor del DTE (el contribuyente que genera el documento).

---

### Receiver

```go
type Receiver struct {
    Name                *string
    DocumentType        *document.DTEType
    DocumentNumber      *identification.DocumentNumber
    Address             interfaces.Address
    Email               *base.Email
    Phone               *base.Phone
    NRC                 *identification.NRC
    NIT                 *identification.NIT
    ActivityDescription *string
    ActivityCode        *identification.ActivityCode
    CommercialName      *string
}
```

Representa al receptor del DTE. La mayoría de campos son opcionales (punteros) porque no todos los DTEs requieren receptor completo.

---

### Item

```go
type Item struct {
    Number      item.ItemNumber
    Type        item.ItemType
    Description string
    Quantity    item.Quantity
    UnitMeasure item.UnitMeasure
    UnitPrice   financial.Amount
    Discount    financial.Discount
    Taxes       []string
    Code        *item.ItemCode
    TaxCode     *financial.TaxType
    RelatedDoc  *string
}
```

Ítem base que cada DTE extiende con campos específicos (tipos de venta, IVA, Purchase, etc.).

**Métodos especiales:**
- `SetForceUnitPrice()` — Asigna precio sin validación
- `SetForceRelatedDoc()` — Asigna doc relacionado sin validación

---

### Summary

```go
type Summary struct {
    TotalNonSubject    financial.Amount
    TotalExempt        financial.Amount
    TotalTaxed         financial.Amount
    SubTotal           financial.Amount
    SubTotalSales      financial.Amount
    NonSubjectDiscount financial.Amount
    ExemptDiscount     financial.Amount
    DiscountPercentage financial.Discount
    TotalDiscount      financial.Amount
    TotalOperation     financial.Amount
    TotalNonTaxed      financial.Amount
    OperationCondition financial.PaymentCondition
    TotalToPay         financial.Amount
    TotalTaxes         []interfaces.Tax
    TotalInWords       string
    ElectronicPayment  *string
    PaymentTypes       []interfaces.PaymentType
}
```

Resumen financiero del documento con totales, descuentos, impuestos y formas de pago.

**Métodos especiales:**
- `SetForceTotalToPay()` — Asigna total sin validación (usado en nota de débito)

---

### Tax

```go
type Tax struct {
    Code        financial.TaxType
    Description string
    Value       *TaxAmount
}

type TaxAmount struct {
    TotalAmount financial.Amount
}
```

### PaymentType

```go
type PaymentType struct {
    Code      financial.PaymentType
    Amount    financial.Amount
    Reference string
    Term      *financial.PaymentTerm
    Period    *int
}
```

### Address

```go
type Address struct {
    Department   location.Department
    Municipality location.Municipality
    Complement   location.Address
}
```

### Extension

```go
type Extension struct {
    DeliveryName     *document.DeliveryName
    DeliveryDocument *document.DeliveryDocument
    ReceiverName     *document.DeliveryName
    ReceiverDocument *document.DeliveryDocument
    Observation      *document.Observation
    VehiculePlate    *string
}
```

### RelatedDocument

```go
type RelatedDocument struct {
    DocumentType   document.DTEType
    GenerationType document.ModelType
    DocumentNumber string
    EmissionDate   temporal.EmissionDate
}
```

### Appendix

```go
type Appendix struct {
    Field document.AppendixField
    Label document.AppendixLabel
    Value document.AppendixValue
}
```

### OtherDocument

```go
type OtherDocument struct {
    AssociatedCode document.AssociatedDocumentCode
    Description    *string
    Detail         *string
    Doctor         interfaces.DoctorInfo
}

type DoctorInfo struct {
    Name           string
    ServiceType    document.ServiceType
    NIT            *identification.NIT
    Identification *string
}
```

### ThirdPartySale

```go
type ThirdPartySale struct {
    NIT  identification.NIT
    Name string
}
```

---

## Modelos Core de Persistencia

### User

```go
type User struct {
    ID                   uint
    Status               bool
    NIT                  string
    NRC                  string
    AuthType             string
    PasswordPri          string
    CommercialName       string
    Business             string
    EconomicActivity     string
    EconomicActivityDesc string
    Email                string
    Phone                string
    YearInDTE            bool
    TokenLifetime        int
    CreatedAt            time.Time
    UpdatedAt            time.Time
    BranchOffices        []BranchOffice
}
```

**Métodos:**
- `Validate()` — Valida el usuario y sus sucursales
- `GetBranchOfficeMatrix()` — Retorna la sucursal matriz
- `ValidateBranchOffices()` — Valida estructura de sucursales
- `SetBranchesKeysAndSecrets()` — Asigna API keys/secrets generados
- `ListBranches()` — Lista IDs de sucursales

**Validaciones de User:**
- Debe tener al menos una sucursal (`ErrAtLeastOneBranch`)
- Debe tener exactamente una sucursal matriz (`ErrDontHaveBranchMatrix`, `ErrMoreThanOneBranchMatrix`)
- La sucursal matriz debe tener dirección (`ErrBranchMatrixWithoutAddress`)

### BranchOffice

```go
type BranchOffice struct {
    ID                  uint
    UserID              uint
    EstablishmentCode   *string
    EstablishmentCodeMH *string
    Email               *string
    APIKey              string
    APISecret           string
    Phone               *string
    EstablishmentType   string
    POSCode             *string
    POSCodeMH           *string
    IsActive            bool
    Address             *Address
    User                *User
}
```

### DTEDocument (Core)

```go
type DTEDocument struct {
    ID         string
    BranchID   uint
    DocumentID string
    CreatedAt  time.Time
    UpdatedAt  time.Time
    Details    *DTEDetails
}
```

### DTEDetails

```go
type DTEDetails struct {
    ID             string
    DTEType        string
    ControlNumber  string
    ReceptionStamp *string
    Transmission   string     // "NORMAL" o "CONTINGENCY"
    Status         string     // "RECEIVED", "REJECTED", "INVALIDATED", "PENDING"
    JSONData       string
}
```

### ReservedSequence

```go
type ReservedSequence struct {
    ID                    uint
    BranchID              uint
    DTEType               string
    SequenceNumber        uint
    Year                  int
    Status                string     // "RESERVED", "CONFIRMED", "RELEASED"
    DocumentID            *string
    ReservedAt            time.Time
    ConfirmedAt           *time.Time
    ReleasedAt            *time.Time
    ExpiresAt             *time.Time
    RejectionReason       *string
    HaciendaCode          *string
    IsContingency         bool
    ContingencyDocumentID *string
}
```

### DomainEvent

```go
type DomainEvent struct {
    ID         string
    UserID     uint
    BranchID   uint
    EventType  string
    Payload    string      // JSON serializado
    OccurredAt time.Time
}
```

### DTEResponse (Consulta)

```go
type DTEResponse struct {
    ControlNumber  string
    GenerationCode string
    ReceptionStamp *string
    Transmission   string
    Status         string
    CreatedAt      string
    UpdatedAt      string
    JSONData       map[string]interface{}
}
```

### DTEListResponse

```go
type DTEListResponse struct {
    Documents  []DTEModelResponse
    Summary    ListSummary
    Pagination DTEPaginationResponse
}

type ListSummary struct {
    Total         int64
    Received      int64
    Invalid       int64
    Rejected      int64
    Pending       int64
    ByContingency int64
    ByNormal      int64
}
```

---

## Jerarquía de Modelos por DTE

Cada DTE extiende los modelos base:

```
DTEDocument (base)
├── ElectronicInvoice      → InvoiceItem, InvoiceSummary
├── CreditFiscalDocument   → CreditItem, CreditSummary
├── FSEModel               → FSEItem, FSESummary, FSEReceiver
├── CreditNoteModel        → CreditNoteItem, CreditNoteSummary
├── DebitNoteModel         → DebitNoteItem, DebitNoteSummary
├── RetentionModel         → RetentionItem, RetentionSummary (estructura diferente)
├── RemissionNoteModel     → RemissionNoteItem, RemissionNoteSummary
└── InvalidationDocument   → InvalidatedDocument, InvalidationReason (NO embebe DTEDocument)
```

> **Nota:** `InvalidationDocument` y `RetentionModel` tienen estructuras significativamente diferentes al resto.
