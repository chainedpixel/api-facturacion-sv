# Tests — Pruebas del Sistema

**Ubicación:** `tests/`

## Descripción General

El directorio `tests/` contiene todas las pruebas del proyecto, organizadas por capa y tipo. Se utilizan mocks generados con GoMock, fixtures con Builder Pattern, y pruebas table-driven. La cobertura abarca servicios de dominio, mappers de request/response, utilidades y un test de integración que valida el flujo completo de todos los tipos de DTE.

---

## Estructura del Directorio

```
tests/
├── test_main.go                    # Inicialización global del ambiente de testing
├── asset_error_code.go             # Helper de aserción de códigos de error
├── fixtures/                       # Builders y datos de prueba
│   ├── base_common_builder.go      # DTEBuilder base (Builder Pattern)
│   ├── invoice_builder.go          # Builder específico de facturas
│   ├── invoice_fixture.go          # Fixtures de request/response de facturas
│   ├── common_fixture.go           # Fixtures compartidos (receivers, items, etc.)
│   └── invalid_fixtures.go         # Datos intencionalmente inválidos
├── mocks/                          # Mocks generados con GoMock
│   ├── mocks_generation.go         # Directivas //go:generate
│   ├── auth_manager_mock.go        # MockAuthManager
│   ├── dte_manager_mock.go         # MockDTEManager
│   ├── dte_service_mock.go         # MockDTEService
│   ├── dte_repository_mock.go      # MockDTERepositoryPort
│   ├── transmitter_mock.go         # MockBaseTransmitter
│   ├── seq_number_manager_mock.go  # MockSequentialNumberManager
│   ├── seq_number_repo_mock.go     # MockSequentialNumberRepositoryPort
│   ├── contingency_manager_mock.go # MockContingencyManager
│   ├── contingency_repo_mock.go    # MockContingencyRepositoryPort
│   ├── contingency_event_mock.go   # MockContingencyEventSender
│   └── reserved_seq_repo_mock.go   # MockReservedSequenceRepositoryPort
├── services/                       # Tests de servicios de dominio
│   ├── invoice_manager_test.go
│   ├── ccf_manager_test.go
│   ├── creditnote_manager_test.go
│   ├── debit_note_manager_test.go
│   ├── fse_manager_test.go
│   ├── retention_manager_test.go
│   ├── remission_note_manager_test.go
│   ├── invalidation_service_test.go
│   ├── dte_manager_test.go
│   ├── dte_seqnumber_test.go
│   ├── contingency_service_test.go
│   ├── contingency_models_test.go
│   ├── contingency_helpers_test.go
│   ├── circuit_breaker_test.go
│   └── batch_transmitter_circuit_breaker_test.go
├── mappers/                        # Tests de mapeo request/response
│   ├── invoice_reqmapper_test.go
│   ├── ccf_reqmapper_test.go
│   ├── creditnote_reqmapper_test.go
│   ├── debit_note_reqmapper_test.go
│   ├── fse_reqmapper_test.go
│   ├── retention_reqmapper_test.go
│   ├── remission_note_reqmapper_test.go
│   ├── invalidation_reqmapper_test.go
│   └── common_reqmapper_test.go
├── utils/                          # Tests de utilidades
│   ├── extractors_test.go
│   ├── pointer_test.go
│   └── updates_test.go
├── events/                         # Bus de eventos de dominio
│   └── in_memory_bus_test.go
├── notifier/                       # Mailer, cooldown y renderer de correos al admin
│   ├── cooldown_test.go            # miniredis: cooldown TTL y dedupe por agregado
│   ├── template_renderer_test.go   # plantillas HTML/plain con API v3.0.0
│   └── smtp_mailer_test.go         # servidor SMTP fake + degradación graceful
├── handlers/                       # Handlers de aplicación
│   ├── admin_email_handler_test.go # cooldown, render y errores propagados
│   └── test_notify_handler_test.go # endpoint GET /notify-test para verificar el correcto envio de correos
├── health/                         # Checkers del servicio de health
│   ├── domain_events_checker_test.go
│   └── smtp_checker_test.go
├── repositories/                   # ya incluye event_repository_test.go (sqlmock)
└── integration/                    # Tests de integración
    └── all_dte_test.go
```

---

## Infraestructura de Testing

### test_main.go — Inicialización Global

```go
func TestMain(t *testing.T)
```

Se ejecuta al inicio de cada archivo de test. Se inicializa:

1. Se localiza la raíz del proyecto
2. Se carga la configuración de testing (`InitEnvTesting`)
3. Se desactiva la validación MX (correos) para pruebas
4. Se inicializa el sistema de tiempo (`TimeInit`)
5. Se cargan traducciones en inglés
6. Se configura el logger en modo debug

### asset_error_code.go — Aserción de Errores

```go
func AssertErrorCode(t *testing.T, err error, expectedCode string)
```

Se verifica que un error contenga el código esperado. Se soportan tres tipos:
- `ValidationError` — Errores de validación de campos
- `ServiceError` — Errores de servicios de dominio
- `DTEError` — Errores específicos de DTE

---

## Mocks (GoMock)

> **Directorio:** `tests/mocks/`
> **Framework:** `github.com/golang/mock`

Se generan mocks para las 12 interfaces principales del sistema:

| Mock | Interfaz que implementa |
|---|---|
| `MockAuthManager` | `auth.AuthManager` |
| `MockAuthRepositoryPort` | `auth.AuthRepositoryPort` |
| `MockDTEManager` | `dte_documents.DTEManager` |
| `MockDTEService` | `ports.DTEService` |
| `MockDTERepositoryPort` | `dte_documents.DTERepositoryPort` |
| `MockBaseTransmitter` | `ports.BaseTransmitter` |
| `MockSequentialNumberManager` | `dte_documents.SequentialNumberManager` |
| `MockSequentialNumberRepositoryPort` | `ports.SequentialNumberRepositoryPort` |
| `MockContingencyManager` | `contingency.ContingencyManager` |
| `MockContingencyRepositoryPort` | `contingency.ContingencyRepositoryPort` |
| `MockContingencyEventSender` | `contingency.ContingencyEventSender` |
| `MockReservedSequenceRepositoryPort` | `dte_documents.ReservedSequenceRepositoryPort` |

### Patrón de Uso

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

mockAuth := mocks.NewMockAuthManager(ctrl)
mockAuth.EXPECT().
    GetIssuer(gomock.Any(), gomock.Any()).
    Return(issuerData, nil)
```

---

## Fixtures (Builder Pattern)

> **Directorio:** `tests/fixtures/`

### DTEBuilder — Builder Base

Se construyen documentos DTE completos y válidos paso a paso:

```go
builder := NewDTEBuilder()
builder.AddIdentification()     // Identificación con version, ambiente, tipo
builder.AddIssuer()             // Emisor con NIT, NRC, dirección
builder.AddReceiver()           // Receptor
builder.AddItems()              // 2 ítems (producto + servicio)
builder.AddSummary()            // Totales calculados, impuestos, pagos
doc, err := builder.Build()     // Build con validación completa
```

### Métodos de Construcción por Tipo de DTE

| Método | Tipo |
|---|---|
| `BuildElectronicInvoice()` | Factura válida |
| `BuildInvalidElectronicInvoice()` | Factura con IVA incorrecto |
| `BuildCreditFiscalDocument()` | CCF válido |
| `BuildInvalidCreditFiscalDocument()` | CCF sin NRC |
| `BuildCreditNote()` | Nota de crédito con documentos relacionados |
| `BuildInvalidCreditNote()` | Nota de crédito sin documentos |
| `BuildRetentionDocumentWithPhysicalItems()` | Retención con ítems físicos |
| `BuildRetentionDocumentWithElectronicItems()` | Retención con ítems electrónicos |
| `BuildRetentionDocumentWithMixedItems()` | Retención con ítems mixtos |
| `BuildInvalidRetentionDocument()` | Retención sin resumen |
| `BuildInvalidationDocumentWithReplacement()` | Invalidación tipo 1 (con reemplazo) |
| `BuildInvalidationDocumentWithAnnulment()` | Invalidación tipo 2 (anulación) |
| `BuildInvalidationDocumentWithDefinitive()` | Invalidación tipo 3 (definitiva) |
| `BuildInvalidInvalidationDocument()` | Invalidación inválida (tipo 2 con reemplazo) |

### InvoiceBuilder — Builder Específico

Se extiende `DTEBuilder` con métodos específicos de factura:

| Método | Descripción |
|---|---|
| `BuildValidInvoice()` | Se crea una factura completa válida |
| `BuildNaturalReceiverInvoice()` | Se crea una factura para consumidor final |
| `BuildCreditInvoice()` | Se crea una factura a crédito |
| `BuildAsInvoiceData()` | Se convierte a estructura `InvoiceData` |

### Fixtures de Request/Response

Se proveen funciones factory para datos de prueba de la API:

| Función | Propósito |
|---|---|
| `CreateDefaultInvoiceRequest()` | Request completo válido de factura |
| `CreateDefaultCreditFiscalRequest()` | Request completo de CCF |
| `CreateDefaultCreditNoteRequest()` | Request completo de nota de crédito |
| `CreateDefaultRetentionRequest()` | Request completo de retención |
| `CreateDefaultFSERequest()` | Request completo de FSE |
| `CreateDefaultInvalidationRequest()` | Request completo de invalidación |
| `CreateExpectedInvoiceResponse()` | Response esperado de Hacienda |

### Fixtures Inválidos

Se crean datos intencionalmente incorrectos para pruebas negativas:

| Función | Defecto |
|---|---|
| `CreateAddressWithEmptyFields()` | Campos vacíos |
| `CreateAddressWithInvalidMunicipality()` | Municipio `"99"` |
| `CreateReceiverWithInvalidEmail()` | Email `"not-an-email"` |
| `CreateReceiverWithoutRequiredFields()` | Sin nombre ni NRC |
| `CreateExtensionWithInvalidFields()` | Campos de entrega con valores inválidos (longitud excedida) |
| `CreatePaymentWithInvalidCode()` | Código `"100"` |
| `CreateThirdPartySaleWithEmptyNIT()` | NIT vacío |

---

## Tests por Categoría

### Tests de Servicios de Dominio (14 archivos)

| Archivo | Servicio Testeado | Casos Clave |
|---|---|---|
| `invoice_manager_test.go` | InvoiceService | Factura válida, consumidor final, crédito, ventas exentas, ventas no sujetas, múltiples ítems, escenarios inválidos |
| `ccf_manager_test.go` | CCFService | CCF válido, percepción IVA, receptor sin NRC |
| `creditnote_manager_test.go` | CreditNoteService | Nota de crédito, sin documentos relacionados, balance |
| `debit_note_manager_test.go` | DebitNoteService | Nota de débito, cálculo de totalToPay |
| `fse_manager_test.go` | FSEService | FSE, sujeto excluido, sin IVA |
| `retention_manager_test.go` | RetentionService | Ítems físicos, electrónicos, mixtos |
| `remission_note_manager_test.go` | RemissionNoteService | Nota de remisión, BienTitulo |
| `invalidation_service_test.go` | InvalidationService | Tipos 1/2/3, ventanas temporales |
| `dte_manager_test.go` | DTEManager | Creación con/sin contingencia, estados |
| `dte_seqnumber_test.go` | SequentialNumberManager | Formato por tipo DTE, códigos por defecto, generación por sucursal |
| `contingency_service_test.go` | ContingencyService | Almacenamiento, fallos de repositorio |
| `contingency_models_test.go` | Modelos de contingencia | Validación de estructuras |
| `contingency_helpers_test.go` | Helpers de contingencia | Funciones auxiliares |
| `circuit_breaker_test.go` | CircuitBreaker | Closed→Open→HalfOpen, threshold, reset |

### Tests de Mappers (9 archivos)

| Archivo | Mapper Testeado | Casos Clave |
|---|---|---|
| `invoice_reqmapper_test.go` | MapToInvoiceData | Request válido/nulo, ítems faltantes, email inválido, municipio inválido |
| `ccf_reqmapper_test.go` | MapToCCFData | CCF válido, campos del receptor |
| `creditnote_reqmapper_test.go` | MapToCreditNoteInput | Request válido, sin documentos relacionados |
| `debit_note_reqmapper_test.go` | MapToDebitNoteInput | Request válido, campos faltantes |
| `fse_reqmapper_test.go` | MapToFSEData | Request válido, sujeto excluido |
| `retention_reqmapper_test.go` | MapToRetentionData | Ítems de retención, códigos |
| `remission_note_reqmapper_test.go` | MapToRemissionNoteInput | BienTitulo, receptor |
| `invalidation_reqmapper_test.go` | MapToInvalidation | Tipos 1/2/3, motivos |
| `common_reqmapper_test.go` | Mappers comunes | Dirección, extensión, pagos, apéndices |

### Tests de Utilidades (3 archivos)

| Archivo | Funciones Testeadas |
|---|---|
| `extractors_test.go` | `ExtractAuxiliarIdentification()` desde objetos Go y JSON strings |
| `pointer_test.go` | Funciones de creación de punteros y conversiones |
| `updates_test.go` | Funciones de actualización de datos |

### Test de Integración (1 archivo)

> **Archivo:** `tests/integration/all_dte_test.go`

```go
func TestAllDTETypes(t *testing.T)
```

Se ejecuta un test end-to-end parametrizado para **los 8 tipos de DTE**:

| Tipo | Endpoint | Config |
|---|---|---|
| Factura | `/invoice` | Request + Builder + Mappers |
| CCF | `/ccf` | Request + Builder + Mappers |
| Nota de Crédito | `/creditnote` | Request + Builder + Mappers |
| Nota de Débito | `/debitnote` | Request + Builder + Mappers |
| Nota de Remisión | `/remissionnote` | Request + Builder + Mappers |
| Retención | `/retention` | Request + Builder + Mappers |
| FSE | `/fse` | Request + Builder + Mappers |
| Invalidación | `/invalidation` | Request + Builder + Mappers |

Se usa `httptest` para simular el servidor HTTP con:
- Router mock con handlers reales
- `MockAuthManager` para información del emisor
- `MockDTEManager` para creación de documentos
- `MockBaseTransmitter` para simulación de transmisión
- `MockSequentialNumberManager` para números de control
- `MockContingencyManager` para contingencia

---

## Patrones de Testing

### Table-Driven Tests

```go
tests := []struct {
    name      string
    setupData func() interface{}
    setupMock func(*MockXXX)
    wantErr   bool
    errorCode string
}{
    { name: "valid invoice", ... },
    { name: "missing items", wantErr: true, errorCode: "ITEMS_REQUIRED" },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Arrange + Act + Assert
    })
}
```

### Librerías de Aserción

- **testify/assert** — Aserciones que no detienen el test
- **testify/require** — Aserciones que detienen el test inmediatamente
- **AssertErrorCode** — Helper personalizado para verificar códigos de error

---

## Notas

1. **TestMain obligatorio**: Cada archivo de test debe llamar a `test.TestMain(t)` para inicializar el ambiente.
2. **Sin BD real**: Todas las pruebas usan mocks. No se requiere base de datos ni Redis para ejecutar tests.
3. **GoMock generation**: Para regenerar mocks: `go generate ./tests/mocks/...`
4. **Builder Pattern**: Se prefiere usar `DTEBuilder` para crear datos de prueba en lugar de construir structs manualmente.
5. **Pruebas negativas**: Por cada caso válido, se incluyen pruebas con datos inválidos para verificar que las validaciones funcionan.
6. **Integración ligera**: El test de integración usa `httptest`, no levanta servicios reales.
7. **Ejecución**: `go test ./tests/...` ejecuta todos los tests del proyecto.
