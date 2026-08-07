# Puertos del Dominio (Domain Ports)

> **Paquete:** `internal/domain/ports`
> **Paquete auth:** `internal/domain/auth`

## Descripción General

Los puertos son interfaces que definen las capacidades que el dominio necesita del mundo exterior. Siguiendo el principio de inversión de dependencias, el dominio **define** estas interfaces pero **no las implementa**. Las implementaciones viven en la capa de infraestructura.

---

## Puertos de Servicio DTE

### DTEService — Puerto Principal

```go
type DTEService interface {
    Create(ctx context.Context, data interface{}, branchID uint) (interface{}, error)
}
```

La interfaz que implementan todos los servicios de DTE. El `data` es un `interface{}` para permitir diferentes tipos de input por DTE.

**Implementaciones:**
- `invoiceService` (DTE 01)
- `creditFiscalService` (DTE 03)
- `remissionNoteService` (DTE 04)
- `creditNoteService` (DTE 05)
- `debitNoteService` (DTE 06)
- `retentionService` (DTE 07)
- `fseService` (DTE 14)

---

## Puertos de Persistencia

### SequentialNumberRepositoryPort

```go
type SequentialNumberRepositoryPort interface {
    GetNext(ctx context.Context, dteType string, branchID uint) (int, error)
}
```

Se obtiene el siguiente número secuencial para un tipo de DTE y sucursal.

### FailedSequenceNumberRepositoryPort

```go
type FailedSequenceNumberRepositoryPort interface {
    RegisterFailedSequence(
        ctx context.Context,
        branchID uint, dteType string,
        sequenceNumber uint, year uint,
        failureReason string, responseCode string,
        originalRequestData interface{}, mhResponse string,
    ) error
    GetFailedSequences(ctx context.Context, branchID uint, dteType string, limit int) ([]FailedSequenceNumber, error)
    GetFailedSequencesByYear(ctx context.Context, branchID uint, dteType string, year uint, limit int) ([]FailedSequenceNumber, error)
}
```

Se registran y consultan números secuenciales fallidos para auditoría y recuperación.

---

## Puertos de Seguridad

### CryptManager

```go
type CryptManager interface {
    GenerateAPIKey() (string, error)
    GenerateAPISecret() (string, error)
    EncryptStruct(token string, data HaciendaCredentials) (string, error)
    DecryptStruct(token string, data string) (HaciendaCredentials, error)
    GenerateBulkAPIKeys(amount int) ([]string, []string, error)
}
```

Se gestiona la generación de API keys, secrets, y el cifrado/descifrado de credenciales de Hacienda.

### TokenManager

```go
type TokenManager interface {
    GenerateToken(claims *AuthClaims, tokenLifetime time.Duration) (string, error)
    ValidateToken(token string) (*AuthClaims, error)
    RevokeToken(token string) error
    SaveTimestampsForContingency(issuedAt, expiresAt time.Time, tokenLifetime time.Duration, claims *AuthClaims) error
    GetSecretKey() string
}
```

Se generan, validan y revocan tokens JWT. `SaveTimestampsForContingency` preserva los timestamps del token para poder reconstruirlo en contingencia.

---

## Puertos de Infraestructura

### CircuitManager (Circuit Breaker)

```go
type CircuitManager interface {
    AllowRequest() bool
    RecordSuccess()
    RecordFailure()
    GetState() constants.State
    GetFailureCount() int32
}
```

Se implementa el patrón Circuit Breaker para proteger llamadas a la API de Hacienda:

| Estado | Descripción |
|---|---|
| `Closed` | Normal — las solicitudes pasan |
| `Open` | Protección activa — las solicitudes se rechazan |
| `HalfOpen` | Prueba — se permite una solicitud de prueba |

### CacheManager

```go
type CacheManager interface {
    Set(key string, claims []byte, ttl time.Duration) error
    SetCredentials(token string, cipherInfo *HaciendaCredentials, ttl time.Duration) error
    GetCredentials(token string) (*HaciendaCredentials, error)
    Get(key string) (string, error)
    Delete(token string) error
    GetRedisClient() *redis.Client
    CacheListManager
}

type CacheListManager interface {
    RPush(key string, value []byte) error
    LPush(key string, value []byte) error
    LRange(key string, start, stop int64) ([]string, error)
    LLen(key string) (int64, error)
    LTrim(key string, start, stop int64) error
}
```

Se gestiona el caché en Redis para tokens, credenciales y listas.

### TimeProvider

```go
type TimeProvider interface {
    Now() time.Time
    Sleep(d time.Duration)
}
```

Se abstrae el acceso al tiempo para facilitar testing.

---

## Puertos de Autenticación

### AuthRepositoryPort

```go
type AuthRepositoryPort interface {
    GetAuthTypeByApiKey(ctx context.Context, apiKey string) (string, error)
    GetAuthTypeByNIT(ctx context.Context, nit string) (string, error)
    GetByNIT(ctx context.Context, nit string) (*User, error)
    GetIssuerInfoByBranchID(ctx context.Context, branchID uint) (*IssuerDTE, error)
    GetByBranchID(ctx context.Context, branchID uint) (*User, error)
    GetBranchByBranchID(ctx context.Context, branchID uint) (*BranchOffice, error)
    GetBranchByBranchApiKey(ctx context.Context, apiKey string) (*BranchOffice, error)
    GetByBranchApiKey(ctx context.Context, apiKey string) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    UpdateBranchOffices(ctx context.Context, userID uint, branches []BranchOffice) error
    DeleteBranchOffice(ctx context.Context, userID uint, branchID uint) error
    GetMatrixBranch(ctx context.Context, userID uint) (*BranchOffice, error)
}
```

Se accede a datos de autenticación: usuarios, sucursales, credenciales.

### AuthStrategy

```go
type AuthStrategy interface {
    GetAuthType() string
    Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthClaims, error)
    ValidateCredentials(credentials *AuthCredentials) error
    GetHaciendaCredentials(token string) (*HaciendaCredentials, error)
    GetTokenLifetime(credentials *AuthCredentials) (time.Duration, error)
}
```

Se implementa el patrón Strategy para diferentes tipos de autenticación.

### AuthManager

```go
type AuthManager interface {
    Login(ctx context.Context, credentials *AuthCredentials) (string, error)
    GetByNIT(ctx context.Context, nit string) (*User, error)
    GetBranchByBranchID(ctx context.Context, branchID uint) (*BranchOffice, error)
    GetIssuer(ctx context.Context, branchID uint) (*IssuerDTE, error)
    GetHaciendaCredentials(ctx context.Context, nit, token string) (*HaciendaCredentials, error)
    Create(ctx context.Context, user *User) error
}
```

Se orquesta el flujo de autenticación completo.

---

## Puertos de DTE Documents

### DTEManager

```go
type DTEManager interface {
    Create(ctx context.Context, document interface{}, transmission, status string, receptionStamp *string) error
    UpdateDTE(ctx context.Context, branchID uint, document DTEDetails) error
    VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
    GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*DTEDocument, error)
    GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, id, originalDTE string, document interface{}) error
    GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error
    ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
    GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*DTEResponse, error)
    GetAllDTEs(ctx context.Context, filters *DTEFilters) (*DTEListResponse, error)
}
```

Se gestiona el ciclo de vida completo de los DTE: creación, actualización, consulta, y control de balance.

### SequentialNumberManager

```go
type SequentialNumberManager interface {
    GetNextControlNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string) (string, error)
    ReserveNextNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string, documentData interface{}, isContingency bool) (string, error)
    ConfirmReservation(ctx context.Context, controlNumber, documentID string, branchID uint) error
    ReleaseReservation(ctx context.Context, controlNumber, rejectionReason, haciendaCode string, branchID uint) error
    ConfirmReservationByDocumentID(ctx context.Context, documentID string) error
    ReleaseReservationByDocumentID(ctx context.Context, documentID, rejectionReason, haciendaCode string) error
    MarkReservationAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error
    MarkReservationAsContingencyByControlNumber(ctx context.Context, controlNumber, documentID, contingencyDocID string, branchID uint) error
}
```

Se gestiona el ciclo de vida de números de control: reserva, confirmación, liberación, y marcado de contingencia.

### ContingencyManager

```go
type ContingencyManager interface {
    StoreDocumentInContingency(ctx context.Context, document interface{}, dteType string, contingencyType int8, reason string) error
    RetransmitPendingDocuments(ctx context.Context) error
}
```

### InvalidationManager

```go
type InvalidationManager interface {
    Validate(ctx context.Context, branchID uint, document interface{}) error
    ValidateStatus(ctx context.Context, branchID uint, req interface{}) error
    InvalidateDocument(ctx context.Context, branchID uint, originalCode string) error
}
```

---

## Diagrama de Dependencias

```
Capa de Dominio (define interfaces)
│
├── ports.DTEService ──────────────── implementado por ──→ invoiceService, ccfService, etc.
├── ports.SequentialNumberRepositoryPort ────────────── implementado por ──→ GORM Repository
├── ports.CryptManager ───────────── implementado por ──→ Crypto Service
├── ports.CircuitManager ─────────── implementado por ──→ Circuit Breaker
├── ports.CacheManager ───────────── implementado por ──→ Redis Adapter
├── ports.TokenManager ───────────── implementado por ──→ JWT Service
├── ports.TimeProvider ───────────── implementado por ──→ Real/Mock Time
├── auth.AuthRepositoryPort ──────── implementado por ──→ GORM Auth Repository
├── auth.AuthStrategy ────────────── implementado por ──→ API Key Strategy
│
└── Todos apuntan hacia el dominio (Dependency Inversion)
```

---

## Regla de Oro

> El dominio **define** interfaces. La infraestructura **implementa** interfaces. Las flechas de dependencia siempre apuntan **hacia adentro** (hacia el dominio).
