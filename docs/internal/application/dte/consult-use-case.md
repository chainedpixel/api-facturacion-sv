# DTEConsultUseCase — Caso de Uso de Consulta

> **Paquete:** `internal/application/dte`
> **Archivo:** `dte_consult_use_case.go`

## Descripción General

El `DTEConsultUseCase` gestiona las consultas y búsquedas de documentos tributarios electrónicos: el documento, su Archivo DTE, sus eventos y el listado filtrado. Es el caso de uso más simple, con solo una dependencia (`DTEManager`).

---

## Estructura

```go
type DTEConsultUseCase struct {
    dteService dte_documents.DTEManager
}
```

Se construye con `NewDTEConsultUseCase(dteService)` en `UseCaseContainer.Initialize`.

---

## Métodos

| Método | Endpoint | Delegación |
|---|---|---|
| `GetByGenerationCode` | `GET /api/v1/dte/{id}` | `DTEManager.GetByGenerationCodeConsult` |
| `GetArchive` | `GET /api/v1/dte/{id}/archivo` | `DTEManager.GetArchive` |
| `GetEvents` | `GET /api/v1/dte/{id}/eventos` | `DTEManager.GetEvents` |
| `GetAllDTEs` | `GET /api/v1/dte` | `parseDTEFilters` + `DTEManager.GetAllDTEs` |

Los cuatro métodos leen los claims del contexto (`ctx.Value("claims")`) para limitar la consulta a la sucursal autenticada (`GetAllDTEs` lo hace salvo `all=true` con NIT en los claims, caso en el que se limita a las sucursales del emisor).

### `GetByGenerationCode`

```go
func (u *DTEConsultUseCase) GetByGenerationCode(
    ctx context.Context,
    id string,
) (interface{}, error)
```

Se consulta un DTE individual por su código de generación (UUID).

**Flujo:**
1. Se extraen los claims del contexto
2. Se busca el documento por `branchID` y `generationCode`
3. Se retorna un `dte.DTEResponse`

```go
type DTEResponse struct {
    ControlNumber  string                 `json:"control_number"`
    GenerationCode string                 `json:"generation_code"`
    ReceptionStamp *string                `json:"reception_stamp"`
    Transmission   string                 `json:"transmission"`
    Status         string                 `json:"status"`
    CreatedAt      string                 `json:"created_at"`
    UpdatedAt      string                 `json:"updated_at"`
    ProcessingDate string                 `json:"processing_date,omitempty"` // fhProcesamiento de Hacienda
    JSONData       map[string]interface{} `json:"json_data"`
}
```

> **Seguridad:** El documento se filtra por `branchID` del claim, previniendo acceso cross-branch.

---

### `GetArchive`

```go
func (u *DTEConsultUseCase) GetArchive(
    ctx context.Context,
    id string,
) (json.RawMessage, error)
```

Se obtiene el Archivo DTE del documento (Normativa de Cumplimiento DTE v2.0, Anexo I, regla 4.1): la estructura de datos firmada, íntegra y sin alteraciones, más `firmaElectronica` y `selloRecibido`.

**Flujo (en `DTEService.GetArchive`):**
1. Se carga el documento por `branchID` y código de generación (`FailedToGetDTE` si no existe)
2. Si el documento no tiene JWS guardado (`SignedDocument` vacío) se devuelve `ArchiveNotAvailable`
3. `BuildArchive(signedDocument, receptionStamp)` toma la estructura del payload del JWS y agrega `firmaElectronica` y `selloRecibido` al final; si el JWS no tiene tres partes, no decodifica o no es un objeto JSON, devuelve `ArchiveNotAvailable`

El handler responde el JSON del archivo tal cual (`Content-Type: application/json`, `Content-Disposition: attachment; filename="<codigoGeneracion>.json"`), sin el envoltorio estándar de respuesta. Los documentos emitidos antes de que se guardara el JWS no tienen Archivo DTE disponible.

---

### `GetEvents`

```go
func (u *DTEConsultUseCase) GetEvents(
    ctx context.Context,
    id string,
) ([]dte.EventArchiveResponse, error)
```

Se listan los eventos de un documento (actualmente, sus invalidaciones) con su evidencia.

**Flujo (en `DTEService.GetEvents`):**
1. Se valida que el documento exista en la sucursal (un código inexistente responde no encontrado, no una lista vacía)
2. Se listan los eventos relacionados al documento (`DTEEventRepositoryPort.ListByRelatedDTE`, tabla `dte_events`); un fallo devuelve `FailedToGetEvents`
3. Cada evento se convierte con `ToEventArchiveResponse`: los datos del evento, `hacienda_response` (respuesta completa de Hacienda) y `archivo_dte` (Archivo DTE del evento, solo si hay JWS válido)

Si no hay eventos, se devuelve una lista vacía.

---

### `GetAllDTEs`

```go
func (u *DTEConsultUseCase) GetAllDTEs(
    ctx context.Context,
    r *http.Request,
) (*dte.DTEListResponse, error)
```

Se consultan DTEs con filtros, paginación y resumen estadístico.

**Flujo:**
1. Se parsean los filtros del query string HTTP (`parseDTEFilters`)
2. Se calcula el resumen estadístico con los mismos filtros; si el total es 0 se retorna la respuesta sin documentos
3. Se obtiene la página de documentos
4. Se retorna la lista con paginación y resumen

---

## Filtros de Consulta

### Query Parameters

| Parámetro | Tipo | Default | Descripción |
|---|---|---|---|
| `all` | `bool` | `false` | Si es `true` y los claims traen NIT, se listan los documentos de las sucursales del emisor (`IssuerNIT`); sin NIT se ignora y se usa la sucursal del claim |
| `status` | `string` | — | Estado del documento |
| `transmission` | `string` | — | Tipo de transmisión |
| `type` | `string` | — | Tipo(s) de DTE (separados por coma) |
| `startDate` | `string` | — | Fecha inicio (RFC3339) |
| `endDate` | `string` | — | Fecha fin (RFC3339) |
| `page` | `int` | `1` | Número de página |
| `page_size` | `int` | `5` | Tamaño de página |

### Valores Válidos de Status

| Valor (query, sin distinguir mayúsculas) | Valor Interno |
|---|---|
| `received` | `RECEIVED` |
| `invalidated` | `INVALIDATED` |
| `rejected` | `REJECTED` |
| `pending` | `PENDING` |
| `expired` | `EXPIRED` |

El mensaje de `InvalidQueryParam` para un `status` inválido lista solo `'received', 'invalidated', 'rejected'`, aunque se aceptan también `pending` y `expired`.

### Valores Válidos de Transmission

| Valor (query, sin distinguir mayúsculas) | Valor Interno |
|---|---|
| `normal` | `NORMAL` |
| `contingency` | `CONTINGENCY` |

### Valores Válidos de Type

Códigos de DTE de `constants.ValidDTETypes`, separados por coma: `01`, `03`, `04`, `05`, `06`, `07`, `08`, `09`, `11`, `14`, `15`. El mensaje de error indica el rango `01-15`, aunque no todos los códigos de ese rango son válidos (por ejemplo `02` o `10`).

### Estructura de Filtros

```go
type DTEFilters struct {
    BranchID     uint   // sucursal del claim (si no se consulta por emisor)
    IssuerNIT    string // NIT del claim, solo con all=true
    IncludeAll   bool
    StartDate    *time.Time
    EndDate      *time.Time
    Status       string
    Transmission string
    DTEType      string
    DTETypes     []string
    Page         int
    PageSize     int
}
```

---

## Respuesta

### DTEListResponse

```go
type DTEListResponse struct {
    Documents  []DTEModelResponse    `json:"documents"`
    Summary    ListSummary           `json:"summary"`
    Pagination DTEPaginationResponse `json:"pagination"`
}
```

### ListSummary

```go
type ListSummary struct {
    Total         int64 `json:"total"`          // Total de documentos que cumplen los filtros
    Received      int64 `json:"received"`       // Documentos recibidos
    Invalid       int64 `json:"invalid"`        // Documentos invalidados
    Rejected      int64 `json:"rejected"`       // Documentos rechazados
    Pending       int64 `json:"pending"`        // Documentos pendientes
    ByContingency int64 `json:"by_contingency"` // Transmitidos por contingencia
    ByNormal      int64 `json:"by_normal"`      // Transmitidos normalmente
}
```

El resumen no tiene contador para el estado `EXPIRED`: esos documentos cuentan en `total`, pero no en los contadores por estado.

### DTEPaginationResponse

```go
type DTEPaginationResponse struct {
    TotalPages int `json:"total_pages"`
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
}
```

### DTEModelResponse

```go
type DTEModelResponse struct {
    Status           string          `json:"status"`
    TransmissionType string          `json:"transmission_type"`
    Document         json.RawMessage `json:"document"`  // JSON completo del DTE
}
```

---

## Validación de Filtros

Los valores de los filtros se validan contra mapas de valores permitidos:

- `ValidReceiverDocumentStates` — estados válidos
- `ValidTransmissionTypes` — tipos de transmisión válidos
- `ValidDTETypes` — códigos de DTE válidos

Si un valor no es válido, se retorna un `ServiceError` formateado (`InvalidQueryParam`). Una fecha con formato distinto a RFC3339 devuelve el mismo error.

---

## Notas

1. **Filtro por sucursal**: Por defecto, solo se retornan documentos de la sucursal del usuario autenticado. El parámetro `all=true` consulta las sucursales cuyo usuario propietario tiene el mismo NIT que el del token (`branch_offices.user_id` → `users.nit`); sin NIT en los claims se usa la sucursal del claim. El código no comprueba ningún permiso adicional: basta con estar autenticado.
2. **Case insensitive**: Los valores de `status` y `transmission` se aceptan en minúsculas y se convierten internamente a mayúsculas.
3. **Tipos múltiples**: Se puede filtrar por múltiples tipos de DTE usando coma: `?type=01,03,05`.
4. **Resumen estadístico**: La respuesta incluye un resumen con conteos por estado y tipo de transmisión, calculado con los mismos filtros.
5. **Paginación**: `page` y `page_size` inválidos (no numéricos o menores o iguales a 0) se reemplazan por 1 y 5.
