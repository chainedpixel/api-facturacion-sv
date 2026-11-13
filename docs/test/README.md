# Tests — Pruebas del Sistema

**Ubicación:** `tests/`

## Descripción General

El directorio `tests/` contiene todas las pruebas del proyecto, organizadas por capa y tipo. Se utilizan mocks generados con GoMock, fixtures con Builder Pattern, pruebas table-driven y validación del JSON enviado a Hacienda contra los JSON Schema del Ministerio de Hacienda.

Hay dos tipos de pruebas:

| Tipo | Ubicación | Cómo se ejecuta | Requisitos |
|---|---|---|---|
| Unitarias e integración ligera | `tests/` (todos los paquetes excepto `e2e`) | `go test ./...` | Ninguno: usan mocks, `sqlmock`, `miniredis`, `httptest` y un servidor SMTP falso |
| Suite E2E contra Hacienda | `tests/e2e/` | `go test -tags e2e ./tests/e2e/ -v -count=1` | API corriendo, firmador disponible y variables `E2E_*` en `.env` |

Los archivos de `tests/e2e/` llevan la restricción de compilación `//go:build e2e`, por lo que `go test ./...` no los compila ni los ejecuta.

---

## Estructura del Directorio

```
tests/
├── test_main.go                    # Inicialización global del ambiente de testing
├── asset_error_code.go             # Helper de aserción de códigos de error
├── fixtures/                       # Builders y datos de prueba
│   ├── base_common_builder.go      # DTEBuilder base (Builder Pattern) y BuildXxx por tipo
│   ├── common_fixture.go           # Fixtures compartidos (dirección, receptor, pagos, emisor, etc.)
│   ├── invalid_fixtures.go         # Datos intencionalmente inválidos
│   ├── <tipo>_builder.go           # Builder por tipo: invoice, ccf, creditnote, debit_note,
│   │                               #   remission_note, retention, fse, invalidation
│   └── <tipo>_fixture.go           # Requests de la API por tipo (mismos tipos)
├── mocks/                          # Mocks generados con GoMock (mockgen)
│   ├── mocks_generation.go         # Directivas //go:generate
│   └── *_mock.go                   # Un archivo por interfaz (ver tabla)
├── testdata/
│   └── svfe-json-schemas/          # JSON Schema de Hacienda (v1, v2, v3, v4)
├── services/                       # Servicios de dominio, contingencia, transmisor, plazos, cálculos
├── mappers/                        # Mappers request/response y validación contra JSON Schema
├── utils/                          # Utilidades (extractores, punteros, actualizaciones, ubicación)
├── events/                         # Bus de eventos de dominio
│   └── in_memory_bus_test.go
├── notifier/                       # Mailer, cooldown y renderer de correos al admin
│   ├── cooldown_test.go            # miniredis: cooldown TTL y dedupe por agregado
│   ├── template_renderer_test.go   # plantillas HTML/plain con API v3.0.0
│   └── smtp_mailer_test.go         # servidor SMTP fake + degradación graceful
├── handlers/                       # Handlers de aplicación y de debug
│   ├── admin_email_handler_test.go # cooldown, render y errores propagados
│   └── debug_notify_handler_test.go # endpoint POST /api/v1/debug/notify-test (eventos de prueba)
├── health/                         # Checkers del servicio de health
│   ├── domain_events_checker_test.go
│   └── smtp_checker_test.go
├── jobs/                           # Jobs programados (sqlmock / mocks)
│   ├── retransmission_job_test.go
│   ├── reservation_cleaner_job_test.go
│   └── metrics_cleanup_job_test.go
├── repositories/                   # Repositorios con sqlmock
│   ├── dte_repository_test.go
│   ├── event_repository_test.go
│   └── reserved_sequence_repository_test.go
├── integration/                    # Integración ligera (httptest)
│   └── all_dte_test.go
└── e2e/                            # Suite E2E (build tag e2e)
    ├── client_test.go              # Cliente HTTP, login, evidencia
    ├── calculator_test.go          # Armado de requests a partir de los items
    ├── hacienda_e2e_test.go        # FE, CCF, NC/ND, NR, FSE
    ├── retention_e2e_test.go       # Comprobante de Retención
    ├── invalidation_e2e_test.go    # Invalidaciones
    ├── archive_e2e_test.go         # Archivo DTE y eventos
    ├── rejections_e2e_test.go      # Rechazos esperados y límites
    ├── deadlines_e2e_test.go       # Plazos con documentos antiguos
    └── contingency_e2e_test.go     # Contingencia por fases
```

---

## Infraestructura de Testing

### test_main.go — Inicialización Global

```go
func TestMain(m *testing.T)
```

Es una función de ayuda del paquete `test` (`tests/`) que cada prueba que lo necesita invoca al inicio como `test.TestMain(t)`; no es el `TestMain(m *testing.M)` de Go. Inicializa:

1. Se localiza la raíz del proyecto
2. Se carga la configuración de testing (`config.InitEnvTesting`: ambiente `00`, `APP_LANG=en`, versión de API `3.0.0`, cooldown de 15 minutos)
3. Se desactiva la validación MX (correos) para pruebas (`base.SkipMXValidation = true`)
4. Se inicializa el sistema de tiempo (`TimeInit`)
5. Se cargan traducciones en inglés
6. Se configura el logger en modo debug

Los mensajes de error en las pruebas se verifican en inglés (`assets/i18n/en.yaml`).

### asset_error_code.go — Aserción de Errores

```go
func AssertErrorCode(t *testing.T, err error, expectedCode string)
```

Se verifica que un error contenga el código esperado (sin distinguir mayúsculas). Se soportan tres tipos:
- `ValidationError` — Errores de validación de campos (`GetType()`)
- `ServiceError` — Errores de servicios de dominio (`GetCode()`)
- `DTEError` — Errores específicos de DTE (`GetCode()`)

Cualquier otro tipo de error hace fallar la prueba.

---

## Mocks (GoMock)

> **Directorio:** `tests/mocks/`
> **Framework:** `github.com/golang/mock` (mockgen)

Las directivas `//go:generate` de `tests/mocks/mocks_generation.go` generan mocks para 13 interfaces:

| Mock | Interfaz que implementa | Archivo |
|---|---|---|
| `MockAuthManager` | `auth.AuthManager` | `auth_manager_mock.go` |
| `MockAuthRepositoryPort` | `auth.AuthRepositoryPort` | `auth_repository_mock.go` |
| `MockDTEService` | `ports.DTEService` | `dte_service_mock.go` |
| `MockBaseTransmitter` | `ports.BaseTransmitter` (aplicación) | `transmitter_mock.go` |
| `MockSequentialNumberManager` | `dte_documents.SequentialNumberManager` | `seq_number_mock.go` |
| `MockDTEManager` | `dte_documents.DTEManager` | `dte_creator_mock.go` |
| `MockDTERepositoryPort` | `dte_documents.DTERepositoryPort` | `dte_repository_mock.go` |
| `MockSequentialNumberRepositoryPort` | `ports.SequentialNumberRepositoryPort` | `seqnumber_repository_mock.go` |
| `MockReservedSequenceRepositoryPort` | `dte_documents.ReservedSequenceRepositoryPort` | `reserved_sequence_repository_mock.go` |
| `MockContingencyManager` | `contingency.ContingencyManager` | `contingency_manager_mock.go` |
| `MockContingencyRepositoryPort` | `contingency.ContingencyRepositoryPort` | `contingency_repository_mock.go` |
| `MockContingencyEventSender` | `contingency.ContingencyEventSender` | `contingency_event_sender_mock.go` |
| `MockCacheManager` | `ports.CacheManager` | `cache_manager_mock.go` |

Para regenerarlos se requiere tener `mockgen` instalado: `go generate ./tests/mocks/...`. Si cambia una interfaz (por ejemplo `DTEManager`), el mock correspondiente debe regenerarse o las pruebas dejarán de compilar.

### Patrón de Uso

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

mockAuth := mocks.NewMockAuthManager(ctrl)
mockAuth.EXPECT().
    GetIssuer(gomock.Any(), gomock.Any()).
    Return(issuerData, nil)
```

Además de GoMock, algunas pruebas usan dobles propios: `sqlmock` (repositorios y jobs), `miniredis` (cooldown), un servidor SMTP falso (`smtp_mailer_test.go`) y servidores `httptest` (consulta de estado del transmisor y circuit breaker del lote).

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
builder.AddItems()              // Ítems
builder.AddSummary()            // Totales calculados, impuestos, pagos
doc, err := builder.Build()     // Build con validación completa
```

`DTEBuilder` ofrece además variantes inválidas (`AddIssuerWithInvalidNIT`, `AddItemsWithInvalidTax`, `AddSummaryWithInvalidPayment`, `AddSummaryWithInvalidTotal`, `AddInvalidRelatedDocument`, `AddInvalidMedicalDocument`, `AddInvalidThirdPartySale`), secciones opcionales (`AddExtension`, `AddAppendixes`, `AddRelatedDocuments`, `AddOtherDocuments`, `AddMedicalDocument`, `AddThirdPartySale`) y `BuildWithoutValidation`.

### Métodos de Construcción por Tipo de DTE (en `DTEBuilder`)

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
| `BuildInvalidationDocumentWithDefinitive()` | Invalidación tipo 3 |
| `BuildInvalidInvalidationDocument()` | Invalidación inválida (tipo 2 con reemplazo) |

### Builders Específicos por Tipo

Cada tipo tiene su propio builder fluido con `Build()`, `BuildWithoutValidation()` y `Add*`:

| Builder | Archivo | Funciones de conveniencia |
|---|---|---|
| `InvoiceBuilder` | `invoice_builder.go` | `BuildValidInvoice`, `BuildNaturalReceiverInvoice`, `BuildCreditInvoice`, `BuildAsInvoiceData` |
| `CCFBuilder` | `ccf_builder.go` | `BuildValidCCF`, `BuildInvalidCCFWithoutNRC`, `BuildCCFWithMixedItemsType`, `BuildAsCCFData` |
| `CreditNoteBuilder` | `creditnote_builder.go` | `BuildValidCreditNote`, `BuildCreditNoteWithCompanyReceiver`, `BuildCreditNoteWithCreditOperation`, `BuildAsCreditNoteInput` |
| `DebitNoteBuilder` | `debit_note_builder.go` | `BuildValidDebitNote`, `BuildAsDebitNoteInput` |
| `RemissionNoteBuilder` | `remission_note_builder.go` | `BuildValidRemissionNote`, `BuildAsRemissionNoteInput` |
| `RetentionBuilder` | `retention_builder.go` | `BuildValidRetention`, `BuildRetentionWithElectronicItems`, `BuildRetentionWithMixedItems`, `BuildInvalidRetentionWithInconsistentSummary`, `BuildRetentionWithoutSummary`, `BuildAsInputRetentionData` |
| `FSEBuilder` | `fse_builder.go` | `BuildValidFSE`, `BuildAsFSEData` |
| `InvalidationBuilder` | `invalidation_builder.go` | `BuildInvalidationWithReplacement`, `BuildInvalidationWithAnnulment`, `BuildInvalidationDefinitive`, `BuildInvalidInvalidation`, `BuildInvalidationWithInvalidReason`, `BuildInvalidationWithMissingReason` |

### Fixtures de Request

Se proveen funciones factory para datos de prueba de la API:

| Función | Propósito |
|---|---|
| `CreateDefaultInvoiceRequest()` | Request completo válido de factura |
| `CreateDefaultCreditFiscalRequest()` | Request completo de CCF |
| `CreateDefaultCreditNoteRequest()` | Request completo de nota de crédito |
| `CreateDefaultDebitNoteRequest()` | Request completo de nota de débito |
| `CreateDefaultRemissionNoteRequest()` | Request completo de nota de remisión |
| `CreatePhysicalDocumentsRetentionRequest()`, `CreateElectronicDocumentsRetentionRequest()`, `CreateMixedDocumentsRetentionRequest()` | Requests de retención según tipo de documentos |
| `CreateDefaultFSERequest()` | Request completo de FSE |
| `CreateDefaultInvalidationRequest()` | Request completo de invalidación |
| `CreateExpectedInvoiceResponse()` | Response esperado de factura |
| `Create*RequestWithAllOptionalFields()` | Requests con todos los campos opcionales (factura, CCF, NC, ND, NR, FSE) |

### Fixtures Inválidos

Se crean datos intencionalmente incorrectos para pruebas negativas (`invalid_fixtures.go`):

| Función | Defecto |
|---|---|
| `CreateAddressWithEmptyFields()` | Campos vacíos |
| `CreateAddressWithInvalidMunicipality()` | Municipio `"99"` |
| `CreateReceiverWithInvalidEmail()` | Email `"not-an-email"` |
| `CreateReceiverWithoutRequiredFields()` | Sin nombre ni NRC |
| `CreateExtensionWithMissingFields()` | Extensión sin nombre de quien entrega ni de quien recibe |
| `CreateAppendixWithInvalidField()` | Apéndice con campo vacío |
| `CreatePaymentWithInvalidCode()` | Código `"100"` |
| `CreateThirdPartySaleWithEmptyNIT()` | NIT vacío |

Existen además fixtures inválidos por tipo, por ejemplo `CreateInvoiceRequestWithInvalidItems`, `CreateInvoiceSummaryWithMismatchedTotals`, `CreateCreditFiscalWithInvalidItems`, `CreateCreditNoteWithoutRelatedDocs`, `CreateInvalidationWithInvalidType`, `CreateInvalidationTypeWithoutReason` y `CreateInvalidationType2WithReplacementCode`.

---

## Tests por Categoría

### Tests de Servicios de Dominio y Transmisión (`tests/services/`)

| Archivo | Qué prueba |
|---|---|
| `invoice_manager_test.go`, `ccf_manager_test.go`, `creditnote_manager_test.go`, `debit_note_manager_test.go`, `fse_manager_test.go`, `retention_manager_test.go`, `remission_note_manager_test.go` | Servicios de creación por tipo: documento válido, variantes de condición de operación, ítems mixtos y escenarios inválidos |
| `calculos_normativa_test.go` | Ejemplos numéricos de cálculo (FE, CCF con y sin descuento global, FSE) y casos negativos, recorriendo request → mapper → validadores de dominio |
| `related_docs_rules_test.go` | Tipos y ventanas de generación permitidos para documentos relacionados |
| `error_message_format_test.go` | Formato de los mensajes de error del CCF |
| `dte_manager_test.go` | `DTEManager`: creación (rechazo sin sello, estructura firmada sin alterar), transacciones de balance, validación de NC, actualización, estado, consulta y listado |
| `archive_test.go` | `BuildArchive`: conserva la estructura firmada y agrega `firmaElectronica` y `selloRecibido`; sello nulo; rechazo de JWS inválido |
| `dte_seqnumber_test.go` | `SequentialNumberManager`: formato por tipo de DTE, códigos por defecto, generación por sucursal |
| `invalidation_service_test.go` | `InvalidationService`: validación, estado e invalidación |
| `invalidation_deadline_test.go` | Calendario de días hábiles, plazos de invalidación, fecha del evento y `ValidateDependencies` |
| `invalidation_processor_test.go` | Cuerpo de la petición del procesador de invalidación |
| `contingency_service_test.go`, `contingency_helpers_test.go`, `contingency_models_test.go` | Almacenamiento y retransmisión de contingencia, agrupación por sistema y tipo, constantes y serialización de modelos |
| `contingency_window_test.go` | Ventana de 72 horas, expiración de documentos y alertas al administrador |
| `mh_transmitter_test.go` | `MHTransmitter`: timeout y `CheckDocumentStatus` (HTTP 200, 202, 400 y 500) |
| `compliance_audit_fixes_test.go` | `GetDocumentRequestData` (DTE e invalidación), timeouts de los clientes HTTP y rutas de `MHPaths` sin barra final |
| `circuit_breaker_test.go` | `CircuitBreaker`: estados Closed/Open/HalfOpen, umbral y reinicio |
| `batch_transmitter_circuit_breaker_test.go` | Circuit breaker dentro de `BatchTransmitterService` |

### Tests de Mappers (`tests/mappers/`)

| Archivo | Qué prueba |
|---|---|
| `invoice_reqmapper_test.go`, `ccf_reqmapper_test.go`, `creditnote_reqmapper_test.go`, `debit_note_reqmapper_test.go`, `fse_reqmapper_test.go`, `retention_reqmapper_test.go`, `remission_note_reqmapper_test.go` | Funciones `MapTo<Tipo>Data` (request válido, nulo, campos faltantes, valores inválidos) |
| `invalidation_reqmapper_test.go` | `MapToInvalidationData`: tipos 1/2/3, motivos |
| `invalidation_event_date_test.go` | Fecha del evento y fecha del sello en `MapToInvalidationData` |
| `common_reqmapper_test.go` | Mappers comunes: dirección, extensión, pagos, apéndices |
| `item_codes_map_test.go` | Mapeo de códigos de ítem |
| `hacienda_schema_validation_test.go` | Validación del JSON completo contra los JSON Schema (ver siguiente sección) |
| `phase2_dte_schema_test.go`, `phase3_dte_schema_test.go`, `phase4_schema_test.go` | Reglas de serialización por versión (FE v2, CCF v4, retención v2, FSE v2, NR/NC/ND v4, invalidación v3, contingencia v4), observaciones y resolución de versión |

### Tests de Utilidades (`tests/utils/`)

| Archivo | Funciones Testeadas |
|---|---|
| `extractors_test.go` | Extractores de identificación, totales del resumen, documentos relacionados e ítems y receptor desde objetos Go y JSON strings |
| `pointer_test.go` | Funciones de creación de punteros y conversiones |
| `updates_test.go` | `UpdateContingencyIdentification` (desde string, bytes y objeto) |
| `location_test.go` | Value objects de distrito y versión, y mapeo de direcciones |

### Tests de Infraestructura de Soporte

| Directorio | Qué prueba |
|---|---|
| `events/` | `InMemoryBus`: entrega a suscriptores, filtrado por nombre y aislamiento ante el pánico de un handler |
| `notifier/` | Cooldown (miniredis), renderer de plantillas y mailer SMTP |
| `handlers/` | `AdminEmailHandler` (cooldown, render, errores) y `DebugNotifyHandler` |
| `health/` | Checkers de eventos de dominio y SMTP |
| `jobs/` | Ejecución y ejecución concurrente de los tres jobs |
| `repositories/` | Rechazo de identificación vacía, guardado de eventos y filtrado de secuencias rechazadas por Hacienda |

### Test de Integración (1 archivo)

> **Archivo:** `tests/integration/all_dte_test.go`

```go
func TestAllDTETypes(t *testing.T)
```

Se ejecuta un test parametrizado con `httptest` para cuatro tipos de DTE, cada uno con tres escenarios:

| Tipo | Endpoint de prueba |
|---|---|
| Factura | `/api/v1/invoice` |
| CCF | `/api/v1/ccf` |
| Nota de Crédito | `/api/v1/creditnote` |
| Comprobante de Retención | `/api/v1/retention` |

| Escenario | Qué verifica |
|---|---|
| Emisión normal | Flujo completo exitoso con transmisión simulada |
| Emisión en contingencia | Fallo de transmisión que dispara la contingencia (`UsesContingency: true`) |
| Error de validación | Rechazo de un monto inválido a nivel de dominio |

Se usa un router `mux` con el `GenericCreatorDTEHandler` real, el `GenericDTEUseCase` real y mocks de `MockAuthManager`, `MockDTEService`, `MockDTEManager`, `MockBaseTransmitter`, `MockSequentialNumberManager` y `MockContingencyManager`. Los paths de prueba son propios de la prueba (no coinciden con los endpoints reales `/dte/invoices`, etc.). No se prueban en esta suite nota de débito, nota de remisión, FSE ni invalidación a nivel HTTP.

---

## Validación de JSON contra los Schemas de Hacienda

> **Archivo:** `tests/mappers/hacienda_schema_validation_test.go`
> **Schemas:** `tests/testdata/svfe-json-schemas/` (copiados sin modificar de los JSON Schema oficiales)

La prueba serializa el documento que produce el response mapper (el mismo JSON que se firma y se transmite) y lo valida contra el schema del tipo. A diferencia de las pruebas por campo, detecta campos sobrantes (`additionalProperties: false`), campos requeridos ausentes, tipos, patrones, formatos y longitudes.

Detalles técnicos:

- Librería `github.com/santhosh-tekuri/jsonschema/v6` con borrador Draft 7 y validación de formatos activada (`AssertFormat`).
- Como los schemas usan lookahead en sus patrones (no soportado por `regexp` de Go), se inyecta un motor de expresiones regulares ECMAScript basado en `github.com/dlclark/regexp2`.
- `assertMatchesHaciendaSchema(t, schemaRelPath, document)` compila el schema desde la ruta relativa a `tests/testdata/svfe-json-schemas` y falla mostrando los errores y el JSON enviado.

| Prueba | Schema |
|---|---|
| `TestHaciendaSchema_Invoice` | `v2/fe-f-v2.json` |
| `TestHaciendaSchema_CCF` y `..._CCFWithThirdPartySaleAndAppendix` | `v4/fe-ccf-v4.json` |
| `TestHaciendaSchema_CreditNote` | `v4/fe-nc-v4.json` |
| `TestHaciendaSchema_DebitNote` | `v4/fe-nd-v4.json` |
| `TestHaciendaSchema_RemissionNote` | `v4/fe-nr-v4.json` |
| `TestHaciendaSchema_Retention` | `v2/fe-cr-v2.json` |
| `TestHaciendaSchema_FSE` | `v2/fe-fse-v2.json` |
| `TestHaciendaSchema_Invalidation` | `v3/invalidacion-schema-v3.json` |
| `TestHaciendaSchema_ContingencyEvent` | `v4/contingencia-schema-v4.json` |

El directorio `testdata` incluye además schemas que ninguna prueba utiliza actualmente: `v1/fe-eop-v1.json`, `v1/fe-eret-v1.json`, `v2/fe-cd-v2.json`, `v2/fe-cl-v2.json`, `v2/fe-dcl-v2.json` y `v3/fe-fex-v3.json`.

---

## Suite E2E contra el Ambiente de Pruebas de Hacienda

> **Directorio:** `tests/e2e/`
> **Build tag:** `e2e`

La suite habla con la API real del microservicio (que debe estar corriendo), la cual firma y transmite cada documento al ambiente de pruebas de Hacienda (`MH_AMBIENT_CODE=00`). Cada caso exitoso exige que Hacienda otorgue sello de recepción; un rechazo muestra el código y las observaciones de Hacienda. Las pruebas de rechazo exigen que la API rechace el documento antes de transmitirlo.

### Ejecución

```
go test -tags e2e ./tests/e2e/ -v -count=1
```

No se ejecuta como parte de `go test ./...`. Se debe levantar la API manualmente (con el firmador disponible) antes de ejecutar la suite.

### Variables de entorno (`.env` de la raíz o entorno)

El cliente lee el `.env` de la raíz del proyecto (sin sobrescribir las variables ya definidas en el entorno). Si falta una variable obligatoria, la prueba falla con el nombre de la variable.

| Variable | Uso |
|---|---|
| `E2E_API_URL` | URL base de la API (obligatoria) |
| `E2E_API_KEY`, `E2E_API_SECRET` | Credenciales de la sucursal para `POST /auth/login` (obligatorias) |
| `E2E_MH_USER`, `E2E_MH_PASSWORD` | Credenciales de Hacienda enviadas en el login (obligatorias); `E2E_MH_USER` se usa además como número de documento del responsable en las invalidaciones |
| `E2E_RECEPTOR_NAME`, `E2E_RECEPTOR_NIT`, `E2E_RECEPTOR_NRC`, `E2E_RECEPTOR_ACTIVITY`, `E2E_RECEPTOR_ACTIVITY_DESC` | Receptor contribuyente registrado en Hacienda (Hacienda valida que el NIT exista, no sirven NIT ficticios) |
| `E2E_EVIDENCE_DIR` | Directorio opcional donde se guarda la evidencia de cada llamada HTTP |
| `E2E_CONTINGENCY_PHASE` | `1` o `2`: fase de la prueba de contingencia (sin valor, la prueba se omite) |
| `E2E_OLD_CCF`, `E2E_OLD_CCF_RECEIVER_NIT`, `E2E_OLD_FE` | Documentos sellados hace meses para la prueba de plazos (si no se definen, la prueba se omite) |

Estas variables no están en `.env.example`.

### Evidencia (`E2E_EVIDENCE_DIR`)

Si la variable está definida, el cliente crea el directorio (permisos `0700`) y guarda un JSON por cada llamada HTTP (`0600`), con nombre `<fecha-hora>_<nombre-del-test>.json` y los campos `test`, `recorded_at`, `method`, `path`, `request`, `http_status` y `response`. Las llamadas a `/auth/` se guardan con las credenciales y el token omitidos.

### Pruebas

| Archivo | Pruebas | Qué cubre |
|---|---|---|
| `hacienda_e2e_test.go` | `TestE2E_FacturaElectronica`, `TestE2E_CreditoFiscal`, `TestE2E_NotasCreditoYDebito`, `TestE2E_NotaRemision`, `TestE2E_SujetoExcluido` | Emisión de FE, CCF, NC/ND (sobre CCF simple, con percepción, con retención y con ventas exentas), NR y FSE |
| `retention_e2e_test.go` | `TestE2E_ComprobanteRetencion` | CR con documentos físicos, código de retención distinto (rechazo) y NC sobre un CR (exploratorio) |
| `invalidation_e2e_test.go` | `TestE2E_Invalidacion` | Invalidación tipos 1, 2 y 3 de FE y CCF, y orden de dependencias (NC antes que su CCF) |
| `archive_e2e_test.go` | `TestE2E_ArchivoDTE` | Archivo DTE de cada tipo emitido (`GET /dte/{id}/archivo`) y eventos de invalidación (`GET /dte/{id}/eventos`) |
| `rejections_e2e_test.go` | `TestE2E_RechazosEsperados`, `TestE2E_LimitesNormativa` | Documentos con un único error introducido, que la API debe rechazar con el código esperado, y límites ampliados por la Normativa de Cumplimiento DTE v2.0 (descripción de ítem de 1500 caracteres) |
| `deadlines_e2e_test.go` | `TestE2E_PlazosDocumentosAntiguos` | Plazos de invalidación y de documentos relacionados con documentos de meses atrás (`InvalidationDeadlineExpired`, `RelatedDocOutOfPeriod`) |
| `contingency_e2e_test.go` | `TestE2E_Contingencia` | Contingencia por fases (ver abajo) |

`calculator_test.go` y `client_test.go` no contienen pruebas: arman los requests desde los items (cuerpo con 8 decimales y resumen con 2, calculado desde los valores ya redondeados de los items) y el cliente HTTP (login, `emit`, `expectRejected`, `invalidate`, consulta de archivo y eventos).

### Contingencia por fases

La prueba requiere reiniciar la API entre fases:

1. **Fase 1** — API con `FORCE_CONTINGENCY=true` (simula Hacienda caída; solo aplica con `MH_AMBIENT_CODE=00`):
   ```
   E2E_CONTINGENCY_PHASE=1 go test -tags e2e ./tests/e2e/ -run TestE2E_Contingencia -v
   ```
   Emite FE, FE de consumidor final, CCF y FSE, que la API acepta (HTTP 2xx) sin sello y quedan pendientes. Guarda los códigos de generación en `e2e_contingency_codes.txt` dentro del directorio temporal del sistema (`os.TempDir()`).
2. **Fase 2** — API con `FORCE_CONTINGENCY=false`: el job de retransmisión (cada 5 minutos en ambiente de pruebas) transmite el evento de contingencia y retransmite los documentos:
   ```
   E2E_CONTINGENCY_PHASE=2 go test -tags e2e ./tests/e2e/ -run TestE2E_Contingencia -v -timeout 20m
   ```
   Consulta cada documento cada 30 segundos, hasta 16 minutos. Cada documento debe obtener sello y tener Archivo DTE; si pasa a `EXPIRED` (no transmitido dentro de las 72 horas del evento) la prueba falla.

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
    { name: "missing items", wantErr: true, errorCode: "..." },
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

1. **Inicialización**: Las pruebas que dependen de la configuración, las traducciones o la zona horaria llaman a `test.TestMain(t)` al inicio; no todas las pruebas lo necesitan (por ejemplo, las de `tests/utils/` y `item_codes_map_test.go` no lo llaman).
2. **Sin BD ni Redis reales en las pruebas unitarias**: Usan mocks, `sqlmock` y `miniredis`. No se requiere base de datos ni Redis para `go test ./...`. La suite E2E sí requiere la API y sus dependencias.
3. **GoMock generation**: Para regenerar mocks: `go generate ./tests/mocks/...` (requiere `mockgen`).
4. **Builder Pattern**: Se prefiere usar los builders de `tests/fixtures` para crear datos de prueba en lugar de construir structs manualmente.
5. **Pruebas negativas**: Por cada caso válido, se incluyen pruebas con datos inválidos para verificar que las validaciones funcionan.
6. **Integración ligera**: El test de integración usa `httptest`, no levanta servicios reales.
7. **Ejecución**: `go test ./...` ejecuta las pruebas unitarias y de integración ligera; la suite E2E se ejecuta aparte con `-tags e2e`. Según las reglas del proyecto, `go test ./...` debe pasar por completo al finalizar una fase o feature.
