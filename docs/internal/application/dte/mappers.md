# Sistema de Mappers — Request y Response

> **Paquete:** `pkg/mapper`
> **Request Mappers:** `pkg/mapper/request_mapper`
> **Response Mappers:** `pkg/mapper/response_mapper`

## Descripción General

El sistema de mappers gestiona las transformaciones de datos entre tres formatos:

```
Request HTTP  ──(Request Mapper)──→  Modelo de Dominio  ──(Response Mapper)──→  Formato Hacienda
```

Cada tipo de DTE tiene su propio par de mappers (request + response).

---

## Interfaz DTEMapper

```go
type DTEMapper interface {
    MapToDomainModel(
        req interface{},
        issuer *dte.IssuerDTE,
        params ...interface{},
    ) (interface{}, error)
}
```

Se convierte un request HTTP a un modelo de dominio, inyectando la información del emisor.

---

## Interfaz ResponseMapperFunc

```go
type ResponseMapperFunc func(domain interface{}) interface{}
```

Se convierte un modelo de dominio al formato JSON esperado por la API de Hacienda.

---

## MapperAdapter — Adaptador Genérico

```go
type MapperAdapter struct {
    MapFunc func(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error)
}
```

Se envuelve una función de mapeo en la interfaz `DTEMapper`.

### Constructores Tipados

```go
func newTypedMapper[Req any, Res any](
    mapFn func(req *Req, issuer *dte.IssuerDTE) (Res, error),
) DTEMapper
```

Se usa generics para crear mappers type-safe con assert automático del tipo de request.

```go
func newTypedMapperWithParams[Req any, Res any](
    mapFn func(req *Req, issuer *dte.IssuerDTE, params ...interface{}) (Res, error),
) DTEMapper
```

Variante que acepta parámetros adicionales (usado por invalidación).

---

## MapperFactory

```go
type MapperFactory struct{}
func NewMapperFactory() *MapperFactory
```

Central factory que crea mappers para cada tipo de DTE.

### Request Mappers

| Método | Request Type | Domain Output |
|---|---|---|
| `CreateInvoiceMapperAdapter()` | `CreateInvoiceRequest` | `InvoiceData` |
| `CreateCCFMapperAdapter()` | `CreateCreditFiscalRequest` | `CCFData` |
| `CreateCreditNoteMapperAdapter()` | `CreateCreditNoteRequest` | `CreditNoteInput` |
| `CreateDebitNoteMapperAdapter()` | `CreateDebitNoteRequest` | `DebitNoteInput` |
| `CreateRetentionMapperAdapter()` | `CreateRetentionRequest` | `InputRetentionData` |
| `CreateRemissionNoteMapperAdapter()` | `CreateRemissionNoteRequest` | `RemissionNoteInput` |
| `CreateFSEMapperAdapter()` | `CreateFSERequest` | `FSEData` |
| `CreateInvalidationMapperAdapter()` | `CreateInvalidationRequest` | `InvalidationDocument` |

### Response Mappers

| Método | Función | Formato de Salida |
|---|---|---|
| `GetInvoiceResponseMapper()` | `ToMHInvoice` | Formato Hacienda Factura |
| `GetCCFResponseMapper()` | `ToMHCreditFiscalInvoice` | Formato Hacienda CCF |
| `GetCreditNoteResponseMapper()` | `ToMHCreditNote` | Formato Hacienda NC |
| `GetDebitNoteResponseMapper()` | `ToMHDebitNote` | Formato Hacienda ND |
| `GetRetentionResponseMapper()` | `ToMHRetention` | Formato Hacienda Retención |
| `GetRemissionNoteResponseMapper()` | `ToMHRemissionNote` | Formato Hacienda NR |
| `GetFSEResponseMapper()` | `ToMHFSE` | Formato Hacienda FSE |
| `GetInvalidationResponseMapper()` | `ToMHInvalidation` | Formato Hacienda Invalidación |

---

## Proceso de Mapeo (Request → Dominio)

Cada request mapper realiza los siguientes pasos:

```
CreateXXXRequest (HTTP)
  │
  ├── 1. Validar campos requeridos del request
  │
  ├── 2. Mapear Identificación
  │     common.MapCommonRequestIdentification()
  │     → Version, Ambient, DTEType, ModelType, OperationType
  │     → EmissionDate, EmissionTime, Currency
  │     → ContingencyType, ContingencyReason (si aplica)
  │
  ├── 3. Mapear Emisor (desde IssuerDTE)
  │     common.MapCommonIssuer(issuer)
  │     → NIT, NRC, Name, ActivityCode, Address, Phone, Email
  │     → EstablishmentCode, POSCode, CommercialName
  │
  ├── 4. Mapear Receptor
  │     common.MapCommonRequestReceiver(req.Receiver)
  │     → Name, DocumentType, DocumentNumber, Address
  │     → Email, Phone, NRC, NIT, ActivityCode
  │
  ├── 5. Mapear Ítems (específico por DTE)
  │     invoice.MapInvoiceItems(req.Items)
  │     → Number, Type, Description, Quantity, UnitMeasure
  │     → UnitPrice, Discount, Taxes, TaxedSale, etc.
  │
  ├── 6. Mapear Resumen (específico por DTE)
  │     invoice.MapInvoiceRequestSummary(req.Summary)
  │     → Totales, descuentos, impuestos, formas de pago
  │
  ├── 7. Mapear campos opcionales
  │     ├── Extension
  │     ├── RelatedDocuments
  │     ├── OtherDocuments
  │     ├── ThirdPartySale
  │     └── Appendixes
  │
  └── 8. Retornar modelo de dominio
```

---

## Proceso de Mapeo (Dominio → Hacienda)

Cada response mapper transforma los modelos de dominio al formato JSON de la API de Hacienda:

### Estructura JSON Base de Hacienda

```json
{
  "identificacion": { "version", "ambiente", "tipoDte", "numeroControl", "codigoGeneracion", ... },
  "emisor": { "nit", "nrc", "nombre", "codActividad", "direccion", ... },
  "receptor": { "nombre", "tipoDocumento", "numDocumento", ... },
  "cuerpoDocumento": [{ "numItem", "tipoItem", "descripcion", "cantidad", ... }],
  "resumen": { "totalNoSuj", "totalExenta", "totalGravada", "subTotal", "totalPagar", ... },
  "extension": { "nombEntrega", "docuEntrega", ... },
  "documentoRelacionado": [{ "tipoGeneracion", "numeroDocumento", ... }],
  "otrosDocumentos": [...],
  "ventaTercero": {...},
  "apendice": [{ "campo", "etiqueta", "valor" }]
}
```

### Diferencias de Formato por DTE

| DTE | Particularidades del Formato |
|---|---|
| **Factura** | `ivaItem` por ítem, `totalIva` en resumen |
| **CCF** | Sin `ivaItem`, `ivaPerci1` en resumen |
| **FSE** | `FSEIssuer` (sin tipoEstablecimiento), `sujetoExcluido` en lugar de receptor, ítems con `compra` |
| **Retención** | Sin `cuerpoDocumento` estándar, usa `cuerpoDocumento` con campos de retención |
| **Nota de Remisión** | `bienTitulo` en receptor |
| **Invalidación** | Estructura completamente diferente: `documento` + `motivo` |

---

## Mapeo de Invalidación (Caso Especial)

El mapper de invalidación recibe parámetros adicionales:

```go
func (m *InvalidationMapper) MapToInvalidationData(
    request *CreateInvalidationRequest,
    issuer *dte.IssuerDTE,
    originalDetails *dte.DTEDetails,
    emissionDate time.Time,
) (*invalidation_models.InvalidationDocument, error)
```

- `originalDetails` — Detalles del documento original (tipo DTE, sello de recepción, número de control)
- `emissionDate` — Fecha de emisión original (para validación temporal)

---

## Notas

1. **Generics en mappers**: Los constructores `newTypedMapper[Req, Res]` proporcionan type-safety en tiempo de compilación.
2. **Emisor inyectado**: El emisor no viene del request HTTP — se carga desde la BD por `branchID` y se inyecta en el mapper.
3. **Mapeo común**: Los campos compartidos (identificación, emisor, receptor) se mapean con funciones comunes reutilizadas por todos los DTEs.
4. **Formato Hacienda**: El formato JSON de Hacienda usa nombres en español (`tipoDte`, `codigoGeneracion`, etc.) diferentes a los nombres en inglés del dominio.
