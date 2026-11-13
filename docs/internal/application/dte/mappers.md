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

Cada adaptador envuelve el método `MapTo<Tipo>Data` del request mapper correspondiente (`MapToInvoiceData`, `MapToCCFData`, `MapToCreditNoteData`, `MapToDebitNoteData`, `MapToRetentionData`, `MapToRemissionNoteData`, `MapToFSEData`, `MapToInvalidationData`).

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

Cada request mapper (`pkg/mapper/request_mapper`) realiza, con variaciones por tipo, los siguientes pasos. El ejemplo es `InvoiceMapper.MapToInvoiceData`:

```
CreateXXXRequest (HTTP)
  │
  ├── 1. Validar campos requeridos del request
  │     (request, items y resumen no nulos; tipo y número de documento del receptor
  │      van juntos; reglas propias de cada tipo)
  │
  ├── 2. Mapear Ítems (específico por DTE)
  │     invoice.MapInvoiceItems(req.Items)
  │
  ├── 3. Mapear Receptor
  │     common.MapCommonRequestReceiver(req.Receiver)
  │     (CCF, notas, remisión, retención y FSE usan sus variantes de receptor)
  │
  ├── 4. Generar la Identificación (no viene del request)
  │     common.MapCommonRequestIdentification(modelo, versión, tipoDte)
  │     → Version, Ambient (MH_AMBIENT_CODE), DTEType, ModelType, OperationType (normal),
  │       Currency (USD), EmissionDate y EmissionTime (hora actual de El Salvador)
  │
  ├── 5. Mapear Resumen (específico por DTE)
  │     invoice.MapInvoiceRequestSummary(req.Summary)
  │
  ├── 6. Mapear Emisor (desde IssuerDTE, cargado por branchID)
  │     common.MapCommonIssuer(issuer)
  │
  ├── 7. Mapear campos opcionales
  │     common.MapCommonOptionalToInputData(...)
  │     ├── ThirdPartySale
  │     ├── Extension (remanente sin uso: ningún request lo envía)
  │     ├── OtherDocs
  │     ├── RelatedDocs
  │     ├── Appendixes
  │     └── Payments (donde aplica)
  │
  └── 8. Retornar el modelo de entrada del dominio (InvoiceData, CCFData, ...)
```

El tipo de modelo de facturación es `constants.ModeloFacturacionPrevio` en todos los DTE; el tipo de operación es siempre transmisión normal. La contingencia se aplica después, sobre el documento ya generado, en el handler (`ContingencyHandler`).

Versiones de la estructura JSON por tipo (argumento de `MapCommonRequestIdentification`):

| DTE | Versión |
|---|---|
| Factura (01) | 2 |
| CCF (03) | 4 |
| Nota de Remisión (04) | 4 |
| Nota de Crédito (05) | 4 |
| Nota de Débito (06) | 4 |
| Comprobante de Retención (07) | 2 |
| Factura de Sujeto Excluido (14) | 2 |
| Evento de Invalidación | 3 |

---

## Proceso de Mapeo (Dominio → Hacienda)

Cada response mapper (`pkg/mapper/response_mapper`, funciones `ToMH*`) transforma el modelo de dominio validado a la estructura JSON de la API de Hacienda. Esa estructura es la que se serializa, se firma (JWS) y se guarda como evidencia.

### Estructura JSON Base de Hacienda

```json
{
  "identificacion": { "version", "ambiente", "tipoDte", "numeroControl", "codigoGeneracion", ... },
  "emisor": { "nit", "nrc", "nombre", "codActividad", "direccion", ... },
  "receptor": { "nombre", "tipoDocumento", "numDocumento", ... },
  "cuerpoDocumento": [{ "numItem", "tipoItem", "descripcion", "cantidad", ... }],
  "resumen": { "totalNoSuj", "totalExenta", "totalGravada", "subTotal", "totalPagar", ... },
  "documentoRelacionado": [{ "tipoGeneracion", "numeroDocumento", ... }],
  "otrosDocumentos": [...],
  "ventaTercero": {...},
  "apendice": [{ "campo", "etiqueta", "valor" }]
}
```

No todos los tipos incluyen todas las secciones; cada estructura sigue el JSON Schema oficial del tipo (ver [Tests](../../../test/README.md), validación contra schemas).

### Diferencias de Formato por DTE

| DTE | Particularidades del Formato |
|---|---|
| **Factura** | `ivaItem` por ítem, `totalIva` en el resumen |
| **CCF** | Sin `ivaItem` por ítem; el receptor se identifica por NIT; el resumen incluye `ivaRete` e `ivaPerci` y no define `totalIva` |
| **Notas de Crédito y Débito** | `receptor` con tipo y número de documento; `totalIva` en ítems y resumen |
| **FSE** | `FSEIssuer` (sin `tipoEstablecimiento` ni `nombreComercial`); el sujeto excluido va en `receptor`; ítems con `compra`; resumen con `totalCompra` y `reteRenta` |
| **Retención** | `cuerpoDocumento` con ítems de retención (`tipoDte`, `numeroDocumento`, `montoSujetoGrav`, `codigoRetencionMH`, `ivaRetenido`); resumen con `totalSujetoRetencion`, `totalIva` y `totalIvaRetenido` |
| **Nota de Remisión** | `bienTitulo` en el receptor; `ToMHRemissionNote` devuelve `interface{}` |
| **Invalidación** | Estructura completamente diferente: `identificacion`, `emisor`, `documento` y `motivo` (ver [invalidation-use-case.md](./invalidation-use-case.md)) |

---

## Mapeo de Invalidación (Caso Especial)

El mapper de invalidación recibe parámetros adicionales:

```go
func (i *InvalidationMapper) MapToInvalidationData(
    req *structs.CreateInvalidationRequest,
    client *dte.IssuerDTE,
    baseDte *dte.DTEDetails,
    emissionDate time.Time,
) (*invalidation_models.InvalidationDocument, error)
```

- `baseDte` — Detalles del documento original (tipo DTE, sello de recepción, número de control, JSON, fecha de procesamiento)
- `emissionDate` — Fecha de creación del documento original (`CreatedAt`), usada solo como respaldo cuando el JSON no trae `fecEmi`/`horEmi` ni hay fecha de procesamiento

La fecha y hora del evento (`identificacion.fecEmi`/`horEmi`) se calculan con `policy.EventDateTime`: el instante actual para FE, FEXE y FSEE; para el resto de DTE, la fecha y hora de generación del documento (o el instante actual si se generó hoy).

`InvalidationUseCase` no usa `MapperFactory.CreateInvalidationMapperAdapter()` ni `GetInvalidationResponseMapper()`: instancia `request_mapper.NewInvalidationMapper()` en su constructor y llama directamente a `response_mapper.ToMHInvalidation`. Los métodos de la factory existen pero no están conectados a ningún caso de uso.

---

## Notas

1. **Generics en mappers**: Los constructores `newTypedMapper[Req, Res]` proporcionan type-safety en tiempo de compilación; si el request no es del tipo esperado se devuelve `invalid request type: expected *<tipo>, got <tipo>`.
2. **Emisor inyectado**: El emisor no viene del request HTTP — se carga desde la BD por `branchID` y se inyecta en el mapper.
3. **Mapeo común**: Los campos compartidos (identificación, emisor, receptor, pagos, opcionales) se mapean con funciones comunes reutilizadas por los DTEs (`pkg/mapper/request_mapper/common`).
4. **Formato Hacienda**: El formato JSON de Hacienda usa nombres en español (`tipoDte`, `codigoGeneracion`, etc.) diferentes a los nombres en inglés del dominio.
5. **Detalle de mappers**: Ver [Request Mappers](../../../pkg/request-mappers.md) y [Response Mappers](../../../pkg/response-mappers.md).
