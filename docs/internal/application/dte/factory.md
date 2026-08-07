# DTEUseCaseFactory — Factory de Casos de Uso

> **Paquete:** `internal/application/dte`
> **Archivo:** `dte_use_case_factory.go`

## Descripción General

El `DTEUseCaseFactory` implementa el **patrón Factory** para crear instancias de `GenericDTEUseCase` configuradas para cada tipo de DTE. Cada variante recibe un mapper, response mapper, y operaciones adicionales específicos.

---

## Estructura

```go
type DTEUseCaseFactory struct {
    authService       auth.AuthManager
    dteService        dte_documents.DTEManager
    transmitter       ports.BaseTransmitter
    sequentialManager dte_documents.SequentialNumberManager
    mapperFactory     *mapper.MapperFactory
    operationsFactory *DTEOperations
}
```

### Constructor

```go
func NewDTEUseCaseFactory(
    authService auth.AuthManager,
    dteService dte_documents.DTEManager,
    transmitter ports.BaseTransmitter,
    sequentialManager dte_documents.SequentialNumberManager,
) *DTEUseCaseFactory
```

El `MapperFactory` y `DTEOperations` se instancian internamente.

---

## Métodos Factory

Cada método recibe el servicio de dominio específico del DTE y retorna un `GenericDTEUseCase` completamente configurado.

### DTEs Estándar (sin operaciones adicionales)

| Método | DTE | Mapper | Response Mapper |
|---|---|---|---|
| `CreateInvoiceUseCase(service)` | Factura (01) | `InvoiceMapper` | `ToMHInvoice` |
| `CreateCCFUseCase(service)` | CCF (03) | `CCFMapper` | `ToMHCreditFiscalInvoice` |
| `CreateRetentionUseCase(service)` | Retención (07) | `RetentionMapper` | `ToMHRetention` |
| `CreateRemissionNoteUseCase(service)` | Nota Remisión (04) | `RemissionNoteMapper` | `ToMHRemissionNote` |
| `CreateFSEUseCase(service)` | FSE (14) | `FSEMapper` | `ToMHFSE` |

Todos estos usan `GetNoOperation()` como operaciones adicionales (no-op).

### DTEs con Operaciones de Balance

| Método | DTE | Mapper | Response Mapper | Operaciones |
|---|---|---|---|---|
| `CreateCreditNoteUseCase(service)` | Nota Crédito (05) | `CreditNoteMapper` | `ToMHCreditNote` | `GetCreditNoteOperations` |
| `CreateDebitNoteUseCase(service)` | Nota Débito (06) | `DebitNoteMapper` | `ToMHDebitNote` | `GetDebitNoteOperations` |

### Invalidación (Caso Especial)

```go
func (f *DTEUseCaseFactory) CreateInvalidationUseCase(
    invalidationManager invalidation.InvalidationManager,
) *InvalidationUseCase
```

La invalidación NO usa `GenericDTEUseCase` — tiene su propio caso de uso dedicado (`InvalidationUseCase`) porque su flujo es fundamentalmente diferente.

---

## Diagrama de la Factory

```
DTEUseCaseFactory
  │
  ├── CreateInvoiceUseCase(invoiceService)
  │     └── GenericDTEUseCase {
  │           mapper:        InvoiceMapper
  │           responseMapper: ToMHInvoice
  │           additionalOps:  NoOp
  │         }
  │
  ├── CreateCCFUseCase(ccfService)
  │     └── GenericDTEUseCase {
  │           mapper:        CCFMapper
  │           responseMapper: ToMHCreditFiscalInvoice
  │           additionalOps:  NoOp
  │         }
  │
  ├── CreateCreditNoteUseCase(creditNoteService)
  │     └── GenericDTEUseCase {
  │           mapper:        CreditNoteMapper
  │           responseMapper: ToMHCreditNote
  │           additionalOps:  CreditNoteOps  ← Balance
  │         }
  │
  ├── CreateDebitNoteUseCase(debitNoteService)
  │     └── GenericDTEUseCase {
  │           mapper:        DebitNoteMapper
  │           responseMapper: ToMHDebitNote
  │           additionalOps:  DebitNoteOps   ← Balance
  │         }
  │
  ├── CreateRetentionUseCase(retentionService)
  │     └── GenericDTEUseCase { ... NoOp }
  │
  ├── CreateRemissionNoteUseCase(remissionNoteService)
  │     └── GenericDTEUseCase { ... NoOp }
  │
  ├── CreateFSEUseCase(fseService)
  │     └── GenericDTEUseCase { ... NoOp }
  │
  └── CreateInvalidationUseCase(invalidationManager)
        └── InvalidationUseCase { ... } ← Caso de uso independiente
```

---

## Notas

1. **Extensibilidad**: Para agregar un nuevo tipo de DTE, se crea un nuevo método factory que inyecte el mapper, response mapper, y operaciones apropiadas.
2. **Dependencias compartidas**: `authService`, `dteService`, `transmitter` y `sequentialManager` se comparten entre todos los casos de uso.
3. **MapperFactory y DTEOperations**: Se crean una vez en el constructor y se reutilizan en todos los métodos factory.
