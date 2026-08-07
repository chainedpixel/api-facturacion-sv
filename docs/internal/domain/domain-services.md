# Servicios de Dominio

## Rol de los Servicios de Dominio

Los servicios de dominio orquestan la lógica de negocio que no puede residir de forma natural dentro de una sola entidad o value object. En este código cada tipo de DTE (Factura, CCF, FSE, etc.) tiene su propio servicio de dominio que:

1. Acepta modelos de datos de entrada crudos
2. Ensambla un agregado tipado (`ElectronicInvoice`, `CCFModel`, etc.)
3. Ejecuta validación específica del tipo (via `DTERulesValidator` + estrategias específicas del tipo)
4. Ejecuta la validación general de dos niveles (`ValidateDTEDocument`)
5. Reserva y asigna un número de control y código de generación

Todos los servicios de dominio DTE implementan la única interfaz de puerto definida en `internal/domain/ports/dte_services_port.go`:

```go
type DTEService interface {
    Create(ctx context.Context, data interface{}, branchID uint) (interface{}, error)
}
```

Los inputs y outputs `interface{}` son intencionales: la capa de aplicación hace cast al tipo concreto después de consultar el registro de tipos.

## Patrón Estándar de Servicio

Cada servicio DTE sigue este patrón estructural (mostrado con el servicio de Factura como ejemplo canónico):

```go
// internal/domain/dte/invoice/invoice_service.go

type invoiceService struct {
    validator        *validator.InvoiceRulesValidator
    seqNumberManager dte_documents.SequentialNumberManager
}

func NewInvoiceService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService {
    return &invoiceService{
        validator:        validator.NewInvoiceRulesValidator(nil),
        seqNumberManager: seqNumberManager,
    }
}

func (s *invoiceService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
    // Paso 1: cast del input genérico al modelo de datos concreto
    data := input.(*invoice_models.InvoiceData)

    // Paso 2: ensamblar el documento base (agregado DTEDocument)
    baseDoc := createBaseDocument(data)

    // Paso 3: envolver con campos específicos del tipo
    invoice := &invoice_models.ElectronicInvoice{
        DTEDocument:    baseDoc,
        InvoiceItems:   data.Items,
        InvoiceSummary: *data.InvoiceSummary,
    }

    // Paso 4: validación de reglas específicas del tipo (estrategias solo de factura)
    if err := s.validate(invoice); err != nil {
        return nil, err
    }

    // Paso 5: validación general de dos niveles (estrategias comunes + reflexión de VO)
    if err := buisnessValidator.ValidateDTEDocument(invoice); err != nil {
        return nil, err
    }

    // Paso 6: reservar número de control y generar UUID
    if err := s.generateCodeAndIdentifiers(ctx, invoice, branchID); err != nil {
        return nil, err
    }

    return invoice, nil
}
```

La función `createBaseDocument` convierte slices de punteros de `InputDataCommon` en slices de `[]interfaces.X` para que el agregado genérico `DTEDocument` pueda contener cualquier implementación de esas interfaces.

## `SequentialNumberManager`

`SequentialNumberManager` está definido en `internal/domain/dte/dte_documents/sequential_number_interface.go` e implementado por `sequentialNumberService` en `sequential_number_service.go`.

Su responsabilidad principal es generar números de control en el formato `DTE-{tipo}-{est}{pos}-{secuencia}` garantizando que ningún número se salte ni se use dos veces, incluso cuando una transmisión falla.

### Flujo Reservar → Confirmar / Liberar

```
1.  ReserveNextNumber()
    │  ├── Verifica reservationRepo en busca del número más antiguo "liberado" (anteriormente fallido)
    │  │   └── Si se encuentra: re-marca como "reservado" con nueva expiración → usar ese número
    │  └── Si no se encuentra: llama a sequentialRepo.GetNext() → crea nueva entrada "reservada"
    │
    │  Devuelve: controlNumber (string), reservationID (uint)
    │
2.  El DTE se transmite a Hacienda
    │
    ├── Éxito → ConfirmReservation(ctx, controlNumber, documentID)
    │              marca la entrada como "confirmada", vincula al UUID del documento
    │
    └── Fallo → ReleaseReservation(ctx, controlNumber, reason, haciendaCode)
                  marca la entrada como "liberada", almacena el motivo de rechazo
                  (el número se reutilizará la próxima vez)
```

La interfaz completa:

```go
// internal/domain/dte/dte_documents/sequential_number_interface.go

type SequentialNumberManager interface {
    // Ruta simple — reserva y devuelve inmediatamente (sin exponer reservationID)
    GetNextControlNumber(ctx context.Context, dteType string, branchID uint,
        posCode, establishmentCode *string) (string, error)

    // Ruta de reserva completa — devuelve reservationID para confirmar/liberar después
    ReserveNextNumber(ctx context.Context, dteType string, branchID uint,
        posCode, establishmentCode *string,
        documentData string, isContingency bool) (string, uint, error)

    ConfirmReservation(ctx context.Context, controlNumber string, documentID string) error
    ReleaseReservation(ctx context.Context, controlNumber string, rejectionReason string, haciendaCode string) error

    ConfirmReservationByDocumentID(ctx context.Context, documentID string) error
    ReleaseReservationByDocumentID(ctx context.Context, documentID string, rejectionReason string, haciendaCode string) error

    MarkReservationAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error
    MarkReservationAsContingencyByControlNumber(ctx context.Context, controlNumber string, documentID string, contingencyDocID string) error
}
```

**Formato del número de control:**

Cuando `user.YearInDTE` es true:
```
DTE-01-00000001-2024000000001
     │  └── códigos est+pos
     └── tipo DTE
```
Cuando `user.YearInDTE` es false:
```
DTE-01-00000001-000000000000001
```
La porción de secuencia se rellena con ceros a 11 o 15 dígitos.

## Puerto `DTEManager`

`DTEManager` está definido en `internal/domain/dte/dte_documents/dte_service_interface.go` e implementado por `DTEService` en `dte_service.go`. Es la abstracción de persistencia para documentos DTE completados.

```go
type DTEManager interface {
    Create(context.Context, interface{}, string, string, *string) error
    UpdateDTE(ctx context.Context, branchID uint, document dte.DTEDetails) error
    VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
    GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error)
    GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, id, originalDTE string, document interface{}) error
    GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error
    ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*dte.DTEResponse, error)
    GetAllDTEs(ctx context.Context, filters *dte.DTEFilters) (*dte.DTEListResponse, error)
}
```

Métodos clave:
- `Create` — persiste el documento completamente validado y transmitido; inyecta el sello de recepción en el apéndice (excepto documentos en contingencia)
- `GenerateBalanceTransaction` — extrae montos gravados/exentos/no sujetos del documento y registra una transacción de crédito o débito contra el `BalanceControl` del documento original
- `ValidateForCreditNote` — verifica que los montos de la nota de crédito no excedan el saldo restante en el CCF original
- `ValidateForDebitNote` — confirma que el documento original existe y tiene un registro de saldo

## Cobertura Detallada de Cada Servicio DTE

### InvoiceService (`internal/domain/dte/invoice/invoice_service.go`)

**Tipo de documento:** `01` — Factura Electrónica

**Modelo clave:** `ElectronicInvoice` embebe `*models.DTEDocument` más `[]InvoiceItem` e `InvoiceSummary`.

**Campos únicos en `InvoiceItem`:**
- `NonSubjectSale`, `ExemptSale`, `TaxedSale` — montos de tipo de venta mutuamente excluyentes
- `IVAItem` — la porción de IVA a nivel de ítem: `IVAItem = TaxedSale / 1.13 * 0.13`
- `SuggestedPrice`, `NonTaxed` — montos adicionales opcionales

**Regla de cálculo de IVAItem (aplicada por `InvoiceItemsStrategy`):**
```
baseGravable = TaxedSale / 1.13
IVAItem = baseGravable * 0.13
tolerancia = 0.01
```
Cualquier discrepancia mayor a 0.01 USD se rechaza con `InvalidIVAItemCalculation`.

**Exclusividad de tipo de venta:** Un ítem no puede tener `NonTaxed > 0` y cualquiera de `TaxedSale`, `ExemptSale` o `NonSubjectSale` al mismo tiempo. Los tipos de venta mixtos dentro de un solo ítem son rechazados.

### CCFService (`internal/domain/dte/ccf/`)

**Tipo de documento:** `03` — Comprobante de Crédito Fiscal Electrónico

**Diferencia clave con la Factura:** El IVA se calcula **después** del descuento a nivel de resumen, no a nivel de ítem. El modelo de ítem CCF tiene `ExemptSale`, `NonSubjectSale`, `TaxedSale` (antes de descuento) y el resumen lleva campos `IVAPerception`, `IVARetention`.

**Restricción del receptor:** El receptor del CCF debe tener NRC. Esto se aplica mediante `DocumentTypeStrategy` (la estrategia común que verifica `receiver.GetNRC() != nil` para el tipo CCF).

**Tipos de documentos relacionados válidos para CCF:** `04` (Nota de Remisión), `08` (Comprobante Liquidación), `09` (Doc. Contable Liquidación).

### FSEService (`internal/domain/dte/fse/`)

**Tipo de documento:** `14` — Factura de Sujeto Excluido

**Restricciones clave:**
- Los ítems deben tener un campo `Purchase` (el monto total de compra por línea, no una venta)
- Nunca hay IVA — los documentos FSE representan compras a sujetos excluidos
- El receptor debe tener tipo y número de documento (DUI, NIT, etc.)
- `TaxedSale` en los ítems debe ser cero — intentar incluir ventas gravadas devuelve `FSETaxInvalidTaxedSale`

### CreditNoteService (`internal/domain/dte/credit_note/`)

**Tipo de documento:** `05` — Nota de Crédito Electrónica

**Requisito de documento relacionado:** Una nota de crédito debe referenciar al menos un documento CCF relacionado. `RelatedDocsStrategy` aplica la consistencia del tipo de documento en todos los relacionados.

**Sin validación de pago:** `PaymentTotalStrategy` omite completamente las notas de crédito (devuelve nil inmediatamente cuando el tipo DTE es `05`).

**Verificación de saldo:** Antes de crear la nota de crédito, la capa de aplicación llama a `DTEManager.ValidateForCreditNote`, que verifica que `creditNote.Summary.TotalTaxed ≤ balanceControl.RemainingTaxedAmount` (igual para exento y no sujeto). Esto previene el sobre-creditado.

### DebitNoteService (`internal/domain/dte/debit_note/`)

**Tipo de documento:** `06` — Nota de Débito Electrónica

**Mismo salto que CreditNote:** `PaymentTotalStrategy` devuelve nil para el tipo `06`.

**Verificación de registro de saldo:** `DTEManager.ValidateForDebitNote` verifica que el documento original tiene un registro `BalanceControl`, confirmando que es un CCF que puede ser incrementado.

**Diferencia clave con CreditNote:** Una nota de débito incrementa el saldo adeudado; incrementa el total del documento original. Una nota de crédito lo disminuye.

### RetentionService (`internal/domain/dte/retention/`)

**Tipo de documento:** `07` — Comprobante de Retención Electrónico

**Códigos de retención** (value object `constants.RetentionCode`):
- `"22"` — Retención IVA 1%
- `"C4"` — Retención IVA 13%
- `"C9"` — Otros casos

**Validación de rango de fechas:** La fecha de emisión de cada documento retenido debe caer dentro del período de facturación actual o el inmediatamente anterior. Los documentos retenidos en los primeros 10 días hábiles de un mes pueden referenciar el mes anterior. Esto se aplica mediante el código de error `DateOutOfAllowedRange`.

**Tipos de documentos del receptor:** Debe ser uno de `constants.ValidRetentionDTETypes` (`01`, `03`, `14`).

### RemissionNoteService (`internal/domain/dte/remission_note/`)

**Tipo de documento:** `04` — Nota de Remisión Electrónica

**Campo `real_state`:** Las notas de remisión tienen un campo `BienTitulo` que describe el tipo de traslado (`"01"` venta, `"02"` consignación, `"03"` exhibición, `"04"` traslado interno, `"05"` préstamo, `"99"` otro).

**Restricción `subtotal = total_amount`:** Para notas de remisión no hay valor de venta — el `SubTotal` es igual al `TotalAmount` porque no ocurre ninguna transacción monetaria.

**Sin validación de pago:** `PaymentTotalStrategy` omite el tipo `04`.

### InvalidationService (`internal/domain/dte/invalidation/`)

**Propósito:** Crea un documento que invalida un DTE previamente transmitido.

**Tipos de invalidación:**
- Tipo 1 — Reemplazar (el original será sustituido por un nuevo documento)
- Tipo 2 — Anular (el original simplemente se anula)
- Tipo 3 — Otro

**Restricción de fecha:** La invalidación debe emitirse en el mismo año fiscal que el documento original.

**No se pueden invalidar documentos ya inválidos:** `DTEService.VerifyStatus` se llama antes del servicio de invalidación; si el estado ya es `"INVALIDATED"`, se devuelve el error `DocumentInvalid`.

## ContingencyService

`ContingencyManager` está definido en `internal/domain/dte/contingency/contingency_service_interface.go`.

### `StoreDocumentInContingency`

Cuando la API de MH (Hacienda) no está disponible (circuit breaker abierto), los documentos se almacenan con tipo de transmisión `"CONTINGENCY"` en una cola local. El método:
1. Serializa el documento de dominio completamente validado a JSON
2. Lo persiste via `ContingencyRepositoryPort` con estado `PENDING`
3. Marca la reserva de secuencia como contingencia via `SequentialNumberManager.MarkReservationAsContingency`

### `RetransmitPendingDocuments`

Una goroutine en segundo plano llama a esto periódicamente. Realiza:
1. Obtiene todos los documentos en contingencia `PENDING` en orden cronológico
2. Para cada documento, intenta la transmisión a Hacienda
3. En éxito: marca como `CONFIRMED`, actualiza el registro DTE
4. En fallo: aplica backoff exponencial usando `RetryPolicy`

```go
// internal/domain/dte/contingency/models/retry_policy_model.go

type RetryPolicy struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    BackoffFactor   float64
}
```

El intervalo de backoff es `min(InitialInterval * BackoffFactor^attempt, MaxInterval)`.

El circuit breaker (`ports.CircuitManager`) controla todas las llamadas salientes:
```go
type CircuitManager interface {
    AllowRequest() bool
    RecordSuccess()
    RecordFailure()
    GetState() constants.State
    GetFailureCount() int32
}
```

Los estados progresan: `CLOSED` → (fallos > umbral) → `OPEN` → (timeout transcurrido) → `HALF_OPEN` → (éxito) → `CLOSED`.
