# Mapper Core — Interfaces, Factory y Adaptadores

> **Paquete:** `pkg/mapper`
> **Archivos:** `mapper_interface.go`, `mapper_factory.go`

## Descripción General

El núcleo del sistema de mapeo define las abstracciones y la factory que conectan los request mappers con los response mappers. Se usan generics de Go para garantizar type-safety en tiempo de compilación.

---

## DTEMapper Interface

```go
type DTEMapper interface {
    MapToDomainModel(
        req interface{},
        issuer *dte.IssuerDTE,
        params ...interface{},
    ) (interface{}, error)
}
```

Se define el contrato para transformar un request HTTP a un modelo de dominio, inyectando la información del emisor.

---

## ResponseMapperFunc

```go
type ResponseMapperFunc func(domain interface{}) interface{}
```

Se define el tipo de función para transformar un modelo de dominio al formato JSON de Hacienda.

---

## MapperAdapter

```go
type MapperAdapter struct {
    MapFunc func(req interface{}, issuer *dte.IssuerDTE, params ...interface{}) (interface{}, error)
}
```

Se envuelve una función de mapeo concreta en la interfaz `DTEMapper`. El adapter maneja el type assertion internamente.

---

## Constructores Tipados (Generics)

### `newTypedMapper`

```go
func newTypedMapper[Req any, Res any](
    mapFn func(req *Req, issuer *dte.IssuerDTE) (Res, error),
) DTEMapper
```

Se crea un mapper type-safe que:
1. Recibe un `interface{}` como request
2. Hace type assertion a `*Req`
3. Ejecuta la función de mapeo tipada
4. Retorna el resultado como `interface{}`

### `newTypedMapperWithParams`

```go
func newTypedMapperWithParams[Req any, Res any](
    mapFn func(req *Req, issuer *dte.IssuerDTE, params ...interface{}) (Res, error),
) DTEMapper
```

Variante que pasa parámetros adicionales (usada por el mapper de invalidación que necesita `DTEDetails` y `time.Time`).

---

## MapperFactory

```go
type MapperFactory struct{}
func NewMapperFactory() *MapperFactory
```

Factory centralizada que crea mappers para cada tipo de DTE.

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

| Método | Función Interna | Formato de Salida |
|---|---|---|
| `GetInvoiceResponseMapper()` | `ToMHInvoice` | JSON Hacienda Factura |
| `GetCCFResponseMapper()` | `ToMHCreditFiscalInvoice` | JSON Hacienda CCF |
| `GetCreditNoteResponseMapper()` | `ToMHCreditNote` | JSON Hacienda NC |
| `GetDebitNoteResponseMapper()` | `ToMHDebitNote` | JSON Hacienda ND |
| `GetRetentionResponseMapper()` | `ToMHRetention` | JSON Hacienda Retención |
| `GetRemissionNoteResponseMapper()` | `ToMHRemissionNote` | JSON Hacienda NR |
| `GetFSEResponseMapper()` | `ToMHFSE` | JSON Hacienda FSE |
| `GetInvalidationResponseMapper()` | `ToMHInvalidation` | JSON Hacienda Invalidación |

---

## Uso desde la Factory de Casos de Uso

```
DTEUseCaseFactory.CreateInvoiceUseCase()
  │
  ├── mapper := mapperFactory.CreateInvoiceMapperAdapter()
  │     → newTypedMapper[CreateInvoiceRequest, InvoiceData](MapToInvoiceData)
  │
  ├── responseMapper := mapperFactory.GetInvoiceResponseMapper()
  │     → ToMHInvoice
  │
  └── NewGenericDTEUseCase(..., mapper, responseMapper, ...)
```

---

## Notas

1. **Generics para type-safety**: Los constructores `newTypedMapper[Req, Res]` eliminan la necesidad de type assertions manuales en cada mapper.
2. **Invalidación es especial**: Es el único mapper que usa `newTypedMapperWithParams` porque necesita parámetros adicionales (detalles del DTE original y fecha de emisión).
3. **Factory stateless**: `MapperFactory` no tiene estado. Se puede instanciar una sola vez y reutilizar.
4. **Emisor inyectado**: El emisor (`IssuerDTE`) no proviene del request HTTP; se carga desde la BD por `branchID` y se inyecta en el mapper por el caso de uso.
