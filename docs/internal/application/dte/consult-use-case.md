# DTEConsultUseCase — Caso de Uso de Consulta

> **Paquete:** `internal/application/dte`
> **Archivo:** `dte_consult_use_case.go`

## Descripción General

El `DTEConsultUseCase` gestiona las consultas y búsquedas de documentos tributarios electrónicos. Es el caso de uso más simple, con solo una dependencia (`DTEManager`).

---

## Estructura

```go
type DTEConsultUseCase struct {
    dteService dte_documents.DTEManager
}
```

---

## Métodos

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
3. Se retorna el DTE con su JSON completo mapeado

> **Seguridad:** El documento se filtra por `branchID` del claim, previniendo acceso cross-branch.

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
1. Se parsean los filtros del query string HTTP
2. Se ejecuta la consulta con filtros
3. Se retorna la lista con paginación y resumen

---

## Filtros de Consulta

### Query Parameters

| Parámetro | Tipo | Default | Descripción |
|---|---|---|---|
| `all` | `bool` | `false` | Si `true`, consulta todas las sucursales (admin) |
| `status` | `string` | — | Estado del documento |
| `transmission` | `string` | — | Tipo de transmisión |
| `type` | `string` | — | Tipo(s) de DTE (separados por coma) |
| `startDate` | `string` | — | Fecha inicio (RFC3339) |
| `endDate` | `string` | — | Fecha fin (RFC3339) |
| `page` | `int` | `1` | Número de página |
| `page_size` | `int` | `5` | Tamaño de página |

### Valores Válidos de Status

| Valor (query) | Valor Interno |
|---|---|
| `received` | `RECEIVED` |
| `invalidated` | `INVALIDATED` |
| `rejected` | `REJECTED` |
| `pending` | `PENDING` |

### Valores Válidos de Transmission

| Valor (query) | Valor Interno |
|---|---|
| `normal` | `NORMAL` |
| `contingency` | `CONTINGENCY` |

### Valores Válidos de Type

Códigos de DTE separados por coma: `01,03,05,06,07,14`, etc.

### Estructura de Filtros

```go
type DTEFilters struct {
    IncludeAll   bool
    BranchID     uint
    StartDate    *time.Time
    EndDate      *time.Time
    Status       string
    Page         int
    PageSize     int
    Transmission string
    DTEType      string
    DTETypes     []string
}
```

---

## Respuesta

### DTEListResponse

```go
type DTEListResponse struct {
    Documents  []DTEModelResponse
    Summary    ListSummary
    Pagination DTEPaginationResponse
}
```

### ListSummary

```go
type ListSummary struct {
    Total         int64  // Total de documentos
    Received      int64  // Documentos recibidos
    Invalid       int64  // Documentos invalidados
    Rejected      int64  // Documentos rechazados
    Pending       int64  // Documentos pendientes
    ByContingency int64  // Transmitidos por contingencia
    ByNormal      int64  // Transmitidos normalmente
}
```

### DTEPaginationResponse

```go
type DTEPaginationResponse struct {
    TotalPages int
    Page       int
    PageSize   int
}
```

### DTEModelResponse

```go
type DTEModelResponse struct {
    Status           string
    TransmissionType string
    Document         json.RawMessage  // JSON completo del DTE
}
```

---

## Validación de Filtros

Los valores de los filtros se validan contra mapas de valores permitidos:

- `ValidReceiverDocumentStates` — estados válidos
- `ValidTransmissionTypes` — tipos de transmisión válidos
- `ValidDTETypes` — códigos de DTE válidos

Si un valor no es válido, se retorna un `ServiceError` formateado.

---

## Notas

1. **Filtro por sucursal**: Por defecto, solo se retornan documentos de la sucursal del usuario autenticado. El parámetro `all=true` permite consultar todas las sucursales (permisos de admin).
2. **Case insensitive**: Los valores de `status` y `transmission` se aceptan en minúsculas y se convierten internamente a mayúsculas.
3. **Tipos múltiples**: Se puede filtrar por múltiples tipos de DTE usando coma: `?type=01,03,05`.
4. **Resumen estadístico**: La respuesta incluye un resumen con conteos por estado y tipo de transmisión, calculado en la misma consulta.
