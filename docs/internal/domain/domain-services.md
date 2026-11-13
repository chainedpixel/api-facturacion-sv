# Servicios de Dominio

## Rol de los Servicios de Dominio

Los servicios de dominio orquestan la lógica de negocio que no puede residir de forma natural dentro de una sola entidad o value object. En este código cada tipo de DTE (Factura, CCF, FSE, etc.) tiene su propio servicio de dominio que:

1. Acepta modelos de datos de entrada crudos
2. Ensambla un agregado tipado (`ElectronicInvoice`, `CreditFiscalDocument`, etc.)
3. Ejecuta validación específica del tipo (estrategias propias de cada tipo agrupadas en un `<Tipo>RulesValidator`)
4. Ejecuta la validación general (`ValidateDTEDocument`: validación de value objects por reflexión más `DTERulesValidator` con las estrategias comunes). La Nota de Retención no ejecuta este paso
5. Asigna un número de control (reservado mediante `SequentialNumberManager`) y un código de generación

Todos los servicios de dominio DTE implementan la única interfaz de puerto definida en `internal/domain/ports/dte_services_port.go`:

```go
type DTEService interface {
    Create(ctx context.Context, data interface{}, branchID uint) (interface{}, error)
}
```

Los inputs y outputs `interface{}` son intencionales: la capa de aplicación hace cast al tipo concreto después de consultar el registro de tipos. Dentro de cada servicio el cast del input es directo (`input.(*Tipo)`), sin comprobación.

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

    // Paso 5: validación general (reflexión de VO + estrategias comunes)
    if err := buisnessValidator.ValidateDTEDocument(invoice); err != nil {
        return nil, err
    }

    // Paso 6: obtener número de control (reserva) y generar UUID
    if err := s.generateCodeAndIdentifiers(ctx, invoice, branchID); err != nil {
        return nil, err
    }

    return invoice, nil
}
```

La función `createBaseDocument` convierte slices de punteros de `InputDataCommon` en slices de `[]interfaces.X` para que el agregado genérico `DTEDocument` pueda contener cualquier implementación de esas interfaces.

El orden de los pasos 5 y 6 varía levemente entre servicios: en Factura, CCF, FSE y Retención se genera primero el número de control y luego el código de generación; en Nota de Crédito, Nota de Débito y Nota de Remisión se genera primero el código de generación. La Nota de Débito calcula además el total a pagar al final (ver [dte/debit-note.md](dte/debit-note.md)).

## `SequentialNumberManager`

`SequentialNumberManager` está definido en `internal/domain/dte/dte_documents/sequential_number_interface.go` e implementado por `sequentialNumberService` en `sequential_number_service.go`. El detalle completo se documenta en [sequential-numbers.md](sequential-numbers.md).

Su responsabilidad principal es generar números de control en el formato `DTE-{tipo}-{est}{pos}-{secuencia}` y llevar el ciclo de vida de cada reserva, de modo que un número rechazado antes de ser aceptado por Hacienda pueda reutilizarse.

### Flujo Reservar → Confirmar / Liberar

```
1.  GetNextControlNumber() / ReserveNextNumber()
    │  ├── Busca en reservationRepo el número "liberado" más antiguo del año, sin código de Hacienda
    │  │   └── Si se encuentra: lo marca como "reservado" con nueva expiración (1 hora) → usa ese número
    │  └── Si no se encuentra: llama a sequentialRepo.GetNext() → crea nueva entrada "reservada"
    │
    │  ReserveNextNumber devuelve: controlNumber (string), reservationID (uint)
    │
2.  El DTE se transmite a Hacienda
    │
    ├── Éxito → ConfirmReservation(ctx, controlNumber, documentID, branchID)
    │              marca la entrada como "confirmada", vincula al UUID del documento
    │
    ├── Contingencia → MarkReservationAsContingencyByControlNumber(...)
    │                   la reserva deja de expirar hasta que se confirme o se libere
    │
    └── Rechazo → ReleaseReservation(ctx, controlNumber, reason, haciendaCode, branchID)
                    marca la entrada como "liberada" y guarda el motivo y el código de Hacienda
                    (los números liberados con código de Hacienda no se reutilizan)
```

La interfaz completa:

```go
// internal/domain/dte/dte_documents/sequential_number_interface.go

type SequentialNumberManager interface {
    // Ruta simple: reserva y devuelve el número de control (sin exponer reservationID)
    GetNextControlNumber(ctx context.Context, dteType string, branchID uint,
        posCode, establishmentCode *string) (string, error)

    // Ruta de reserva completa: devuelve reservationID para marcar contingencia
    ReserveNextNumber(ctx context.Context, dteType string, branchID uint,
        posCode, establishmentCode *string,
        documentData string, isContingency bool) (string, uint, error)

    ConfirmReservation(ctx context.Context, controlNumber string, documentID string, branchID uint) error
    ReleaseReservation(ctx context.Context, controlNumber string, rejectionReason string, haciendaCode string, branchID uint) error

    ConfirmReservationByDocumentID(ctx context.Context, documentID string) error
    ReleaseReservationByDocumentID(ctx context.Context, documentID string, rejectionReason string, haciendaCode string) error

    MarkReservationAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error
    MarkReservationAsContingencyByControlNumber(ctx context.Context, controlNumber string, documentID string, contingencyDocID string, branchID uint) error
}
```

**Formato del número de control:**

Cuando `user.YearInDTE` es true:
```
DTE-01-00010001-2024000000001
    │  └── códigos est+pos
    └── tipo DTE
```
Cuando `user.YearInDTE` es false:
```
DTE-01-00010001-000000000000001
```
La porción de secuencia se rellena con ceros a 11 dígitos (precedidos por el año) o a 15 dígitos.

## Puerto `DTEManager`

`DTEManager` está definido en `internal/domain/dte/dte_documents/dte_service_interface.go` e implementado por `DTEService` en `dte_service.go` (constructor `NewDTEService(repo DTERepositoryPort, eventRepo DTEEventRepositoryPort)`). Es la abstracción de persistencia para documentos DTE completados.

```go
type DTEManager interface {
    Create(ctx context.Context, document interface{}, transmission, status string, evidence *dte.TransmissionEvidence) error
    UpdateDTE(ctx context.Context, branchID uint, document dte.DTEDetails) error
    VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
    GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error)
    GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, id, originalDTE string, document interface{}) error
    GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error
    ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*dte.DTEResponse, error)
    GetAllDTEs(ctx context.Context, filters *dte.DTEFilters) (*dte.DTEListResponse, error)
    RegisterEvent(ctx context.Context, event *dte.DTEEvent) error
    GetArchive(ctx context.Context, branchID uint, generationCode string) (json.RawMessage, error)
    GetEvents(ctx context.Context, branchID uint, generationCode string) ([]dte.EventArchiveResponse, error)
    GetActiveAdjustmentNotes(ctx context.Context, branchID uint, originalDTE string) ([]string, error)
}
```

Métodos clave:
- `Create` — persiste el documento validado junto con su evidencia de transmisión (`dte.TransmissionEvidence`: sello, JWS firmado, fecha de procesamiento y respuesta de Hacienda). Si la transmisión no es `CONTINGENCY` exige un sello de recepción no vacío (`MissingReceptionStamp`). La estructura firmada no se altera: el sello y la firma se guardan aparte
- `GenerateBalanceTransaction` — extrae montos gravados/exentos/no sujetos del resumen del documento y registra una transacción contra el `BalanceControl` del documento original
- `GenerateBalanceTransactionWithAmounts` — igual, con montos explícitos (lo usa la invalidación de notas)
- `ValidateForCreditNote` — verifica que los montos de la nota de crédito no excedan el saldo restante del documento original (`InvalidCreditNoteTransaction`)
- `ValidateForDebitNote` — confirma que el documento original tiene un registro de saldo
- `RegisterEvent` — guarda un evento transmitido a Hacienda (invalidación, contingencia) con su evidencia (`FailedToRegisterEvent` si falla)
- `GetArchive` — arma el Archivo DTE de un documento: estructura firmada tomada del payload del JWS, más `firmaElectronica` y `selloRecibido` (`BuildArchive` en `archive.go`); retorna `ArchiveNotAvailable` si el documento no tiene JWS guardado
- `GetEvents` — lista los eventos relacionados con un documento (validando antes que el documento exista en la sucursal), cada uno con la respuesta de Hacienda y su Archivo DTE
- `GetActiveAdjustmentNotes` — retorna los códigos de generación de las notas de crédito y débito en estado `RECEIVED` o `PENDING` que ajustan el documento; lo usa la invalidación para rechazar la anulación de un CCF o Comprobante de Retención con notas vigentes

## Cobertura Detallada de Cada Servicio DTE

### InvoiceService (`internal/domain/dte/invoice/invoice_service.go`)

**Tipo de documento:** `01` — Factura Electrónica. Detalle en [dte/invoice.md](dte/invoice.md).

**Modelo clave:** `ElectronicInvoice` embebe `*models.DTEDocument` más `[]InvoiceItem` e `InvoiceSummary`.

**Campos únicos en `InvoiceItem`:**
- `NonSubjectSale`, `ExemptSale`, `TaxedSale` — montos de tipo de venta
- `IVAItem` — la porción de IVA a nivel de ítem (el IVA va incluido en el precio)
- `SuggestedPrice`, `NonTaxed` — montos adicionales opcionales

La fórmula de `IVAItem` y su tolerancia se documentan en [calculations.md](../../api/calculations.md). Una discrepancia mayor a la tolerancia se rechaza con `InvalidIVAItemCalculation`.

**Exclusividad de tipo de venta:** Un ítem con `NonTaxed > 0` no puede tener `TaxedSale`, `ExemptSale` ni `NonSubjectSale` (`InvalidMixedSalesWithNonTaxed`), y no se permiten tipos de venta mixtos en un ítem (`MixedSalesTypesNotAllowed`).

### CCFService (`internal/domain/dte/ccf/`)

**Tipo de documento:** `03` — Comprobante de Crédito Fiscal Electrónico. Detalle en [dte/ccf.md](dte/ccf.md).

**Diferencia clave con la Factura:** El IVA se calcula en el resumen sobre la base gravada menos el descuento, no a nivel de ítem. El modelo de ítem CCF tiene `ExemptSale`, `NonSubjectSale`, `TaxedSale`, `SuggestedPrice` y `NonTaxed`, y el resumen lleva `TaxedDiscount`, `IVAPerception` e `IVARetention`.

**Restricción del receptor:** El receptor del CCF debe tener NRC, código y descripción de actividad. Se aplica en `CCFReceiverStrategy`, y además en `DocumentTypeStrategy` (estrategia común que verifica `receiver.GetNRC() != nil` para el tipo CCF).

**Tipos de documentos relacionados válidos para CCF:** `04` (Nota de Remisión), `08` (Comprobante Liquidación), `09` (Doc. Contable Liquidación).

### FSEService (`internal/domain/dte/fse/`)

**Tipo de documento:** `14` — Factura de Sujeto Excluido. Detalle en [dte/fse.md](dte/fse.md).

**Restricciones clave:**
- Los ítems deben tener un campo `Purchase` (el monto total de compra por línea, no una venta)
- Nunca hay IVA; la retención de IVA debe ser `0` (`IVARetentionNotAllowedInFSE`)
- El receptor FSE (`FSEReceiver`) es obligatorio y debe tener tipo y número de documento, nombre y dirección
- `TotalTaxed` del resumen debe ser cero (`FSETaxInvalidTaxedSale`)

### CreditNoteService (`internal/domain/dte/credit_note/`)

**Tipo de documento:** `05` — Nota de Crédito Electrónica. Detalle en [dte/credit-note.md](dte/credit-note.md).

**Requisito de documento relacionado:** Una nota de crédito debe referenciar al menos un documento relacionado de tipo CCF (`03`) o Comprobante de Retención (`07`). El servicio verifica que existan, estén `RECEIVED` y que el NIT del receptor coincida.

**Sin validación de pago:** `PaymentTotalStrategy` omite las notas de crédito, débito y remisión (devuelve nil cuando el tipo DTE es `04`, `05` o `06`).

**Verificación de saldo:** Dentro de `Create`, para cada documento relacionado electrónico se llama a `DTEManager.ValidateForCreditNote`, que verifica que los totales gravado, exento y no sujeto de la nota no excedan el saldo restante del documento original. Esto previene el sobre-creditado.

### DebitNoteService (`internal/domain/dte/debit_note/`)

**Tipo de documento:** `06` — Nota de Débito Electrónica. Detalle en [dte/debit-note.md](dte/debit-note.md).

**Mismas reglas de documentos relacionados que la Nota de Crédito** (tipos `03` y `07`, estado `RECEIVED`, coincidencia de NIT).

**Verificación de registro de saldo:** Dentro de `Create`, `DTEManager.ValidateForDebitNote` verifica que el documento original tenga un registro `BalanceControl`; no compara montos.

**Diferencia clave con CreditNote:** El servicio fija el total a pagar con `calculateTotalToPay` al final de `Create`, y la transacción de balance es de tipo `06` en lugar de `05`.

### RetentionService (`internal/domain/dte/retention/`)

**Tipo de documento:** `07` — Comprobante de Retención Electrónico. Detalle en [dte/retention.md](dte/retention.md).

**Códigos de retención** (value object `document.RetentionCode`, constantes en `constants`):
- `"22"` — Retención IVA 1%
- `"C4"` — Retención IVA 13%
- `"C9"` — Otros casos (sin porcentaje definido en `GetRetentionAmount`)

**Validación de rango de fechas:** La fecha de emisión de cada documento retenido debe caer dentro del mes de emisión del comprobante o, si el comprobante se emite dentro de los primeros 10 días hábiles (lunes a viernes) del mes siguiente, en el mes anterior. Se aplica mediante el código de error `DateOutOfAllowedRange`.

**Tipos de DTE retenidos:** `constants.ValidRetentionDTETypes` (`01`, `03`, `14`), expuestos por `NewDTETypeForRetention`.

### RemissionNoteService (`internal/domain/dte/remission_note/`)

**Tipo de documento:** `04` — Nota de Remisión Electrónica. Detalle en [dte/remission-note.md](dte/remission-note.md).

**Campo `BienTitulo`:** El receptor de la nota de remisión (`RemissionNoteReceiver`) tiene un campo `BienTitulo` obligatorio con valores `01`, `02`, `03`, `04`, `05` o `99`. El receptor es obligatorio.

**Monto total:** Cuando se envían `Subtotal` y `TotalAmount`, el monto total debe ser el subtotal más los tributos del resumen.

**Sin validación de pago:** `PaymentTotalStrategy` omite el tipo `04`.

### InvalidationService (`internal/domain/dte/invalidation/`)

**Propósito:** Valida y registra la invalidación de un DTE previamente transmitido. Detalle en [dte/invalidation.md](dte/invalidation.md).

**Tipos de invalidación:**
- Tipo 1 — Reemplazo (requiere código del documento de reemplazo)
- Tipo 2 — Anulación (sin código de reemplazo)
- Tipo 3 — Otro motivo (requiere motivo y código de reemplazo)

**Restricción temporal:** El plazo de transmisión y la fecha del evento dependen del tipo de DTE y se calculan en `invalidation/policy` con el calendario `common/calendar` (3 meses desde el sello para FE, FEXE y FSE; décimo día hábil del mes siguiente para el resto).

**Estados y dependencias:** `ValidateStatus` rechaza documentos en estado `INVALIDATED`, `REJECTED` o `PENDING` (`DocumentAlreadyInvalid`, `DocumentReject`, `DocumentPending`). `ValidateDependencies` rechaza la invalidación de un CCF o Comprobante de Retención con notas de crédito o débito vigentes (`PendingAdjustmentNotes`).

## ContingencyService

`ContingencyManager` está definido en `internal/domain/dte/contingency/contingency_service_interface.go`. Detalle en [dte/contingency.md](dte/contingency.md).

### `StoreDocumentInContingency`

Cuando un DTE no puede transmitirse por un error de conectividad o de servidor de Hacienda (ver criterio en contingency.md), se almacena con tipo de transmisión `"CONTINGENCY"` y estado `PENDING`. El método:
1. Extrae el código de generación y el número de control (`MissingDTEIdentification` si falta alguno)
2. Crea el DTE con transmisión `CONTINGENCY` y estado `PENDING` mediante `DTEManager.Create`
3. Persiste el registro `ContingencyDocument` mediante `ContingencyRepositoryPort.Create`
4. Marca la reserva de secuencia como contingencia mediante `SequentialNumberManager.MarkReservationAsContingencyByControlNumber`
5. Publica `ContingencyActivatedEvent` si hay bus de eventos configurado

### `RetransmitPendingDocuments`

Lo invoca periódicamente el job `internal/infrastructure/jobs/contingency_retransmission_job.go`. Realiza:
1. Obtiene los documentos `PENDING` (con sello de evento primero, luego por antigüedad), hasta el tamaño de lote configurado
2. Marca como `EXPIRED` los documentos con más de 72 horas desde el sello del evento y publica la alerta `contingency_72h_expired`
3. Publica la alerta `contingency_technical_report` si la contingencia supera 3 días
4. Por cada NIT, envía el evento de contingencia a Hacienda
5. Por cada tipo de DTE, firma los documentos, los transmite por lotes y verifica el estado; el transmisor por lotes actualiza el estado de cada documento (`RECEIVED` o `REJECTED`) y confirma o libera la reserva del número

La política de reintentos (`RetryPolicy`) la construye `TransmissionConfig.GetRetryPolicy()` y la consume el transmisor por lotes:

```go
// internal/domain/dte/contingency/models/retry_policy_model.go

type RetryPolicy struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    BackoffFactor   float64
}
```

El circuit breaker (`ports.CircuitManager`) es el puerto que abstrae el control de llamadas salientes a Hacienda:
```go
type CircuitManager interface {
    AllowRequest() bool
    RecordSuccess()
    RecordFailure()
    GetState() constants.State
    GetFailureCount() int32
}
```

Los estados (`constants.StateClosed`, `StateOpen`, `StateHalfOpen`) progresan: `CLOSED` → (fallos > umbral) → `OPEN` → (timeout transcurrido) → `HALF_OPEN` → (éxito) → `CLOSED`.
