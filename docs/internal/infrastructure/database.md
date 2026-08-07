# Base de Datos — Modelos y Migraciones

> **Paquetes:**
> - `internal/infrastructure/database/db_models` — Modelos de base de datos (GORM)
> - `internal/infrastructure/database` — Migraciones automáticas

## Descripción General

Los modelos de base de datos representan la estructura de persistencia del sistema. Se definen como structs de Go con tags de GORM que mapean a tablas de la base de datos. Las migraciones se ejecutan automáticamente al iniciar la aplicación.

---

## Modelos de Base de Datos

### User

Representa un usuario/contribuyente del sistema.

```go
type User struct {
    ID              uint
    NIT             string           // NIT del contribuyente
    NRC             string           // Número de Registro de Contribuyente
    Name            string           // Nombre o razón social
    ActivityCode    string           // Código de actividad económica
    AuthType        string           // Tipo de autenticación
    PrivateKeyPass  string           // Password de clave privada (firma digital)
    BranchOffices   []BranchOffice   // Sucursales (relación 1:N)
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### BranchOffice

Representa una sucursal del contribuyente.

```go
type BranchOffice struct {
    ID                uint
    UserID            uint           // FK al usuario
    Name              string         // Nombre de la sucursal
    Phone             string         // Teléfono
    Email             string         // Correo electrónico
    CommercialName    string         // Nombre comercial
    EstablishmentType string         // Tipo de establecimiento (01: casa matriz)
    EstablishmentCode string         // Código de establecimiento
    POSCode           string         // Código de punto de venta
    APIKey            string         // API key (hasheada)
    APISecret         string         // API secret (hasheado)
    IsMatrix          bool           // Es casa matriz
    Address           *Address       // Dirección (relación 1:1)
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

### Address

Dirección asociada a una sucursal.

```go
type Address struct {
    ID             uint
    BranchOfficeID uint       // FK a la sucursal
    Department     string     // Departamento
    Municipality   string     // Municipio
    Complement     string     // Complemento de dirección
}
```

### DTEDocument

Documento tributario electrónico principal.

```go
type DTEDocument struct {
    ID               uint
    BranchID         uint           // FK a la sucursal
    GenerationCode   string         // UUID (código de generación)
    DTEType          string         // Tipo de DTE (01, 03, etc.)
    TransmissionType string         // NORMAL o CONTINGENCY
    Status           string         // RECEIVED, REJECTED, INVALIDATED, PENDING
    Details          *DTEDetails    // Detalles del documento (relación 1:1)
    BalanceControl   *DTEBalanceControl // Control de balance (relación 1:1)
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### DTEDetails

Detalles y contenido completo de un DTE.

```go
type DTEDetails struct {
    ID              uint
    DTEDocumentID   uint             // FK al documento DTE
    ControlNumber   string           // Número de control (DTE-XX-XXXX-XXXXXXXXX)
    ReceptionStamp  *string          // Sello de recepción de Hacienda (40 chars)
    JSONContent     json.RawMessage  // JSON completo del DTE (formato Hacienda)
    Observations    json.RawMessage  // Observaciones de Hacienda (JSON array)
}
```

### DTEBalanceControl

Control de balance para documentos que pueden recibir notas de crédito/débito.

```go
type DTEBalanceControl struct {
    ID              uint
    DTEDocumentID   uint              // FK al documento DTE
    OriginalAmount  float64           // Monto original del documento
    CurrentBalance  float64           // Balance actual (disminuye con NC, aumenta con ND)
    Transactions    []DTEBalanceTransaction  // Historial de transacciones
}
```

### DTEBalanceTransaction

Transacción que afecta el balance de un DTE.

```go
type DTEBalanceTransaction struct {
    ID                  uint
    BalanceControlID    uint      // FK al control de balance
    TransactionType     string    // "05" (NC) o "06" (ND)
    DocumentNumber      string    // Número del documento que genera la transacción
    GenerationCode      string    // UUID de la nota de crédito/débito
    Amount              float64   // Monto de la transacción
    PreviousBalance     float64   // Balance antes de la transacción
    NewBalance          float64   // Balance después de la transacción
    CreatedAt           time.Time
}
```

### ContingencyDocument

Documento almacenado durante una contingencia (sin transmitir a Hacienda).

```go
type ContingencyDocument struct {
    ID                 uint
    UUID               string           // UUID generado automáticamente
    BranchID           uint             // FK a la sucursal
    NIT                string           // NIT del emisor
    DTEType            string           // Tipo de DTE
    ContingencyType    int              // Tipo de contingencia
    ContingencyReason  string           // Motivo de la contingencia
    SignedDocument     string           // Documento firmado (base64)
    JSONContent        json.RawMessage  // JSON del DTE
    Status             string           // PENDING o PROCESSED
    BatchID            *string          // ID del lote (asignado al procesar)
    MHBatchID          *string          // ID del lote en Hacienda
    ReceptionStamp     *string          // Sello de recepción (post-procesamiento)
    Observations       json.RawMessage  // Observaciones
    ReservedSequence   *ReservedSequenceNumber  // Reservación asociada
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

### ControlNumberSequence

Secuencia de números de control por tipo de DTE y sucursal.

```go
type ControlNumberSequence struct {
    ID         uint
    BranchID   uint     // FK a la sucursal
    DTEType    string   // Tipo de DTE
    Year       int      // Año de la secuencia
    Current    int      // Último número generado
    UpdatedAt  time.Time
}
```

### ReservedSequenceNumber

Reservación de un número secuencial.

```go
type ReservedSequenceNumber struct {
    ID                  uint
    BranchID            uint
    DTEType             string
    SequenceNumber      int
    Year                int
    Status              string     // RELEASED, RESERVED, CONFIRMED, REJECTED
    ExpiresAt           *time.Time // Expiración de la reservación
    DocumentID          *string    // ID del documento asociado
    ContingencyDocID    *uint      // FK al documento de contingencia
    FailureReason       *string    // Motivo de fallo
    HaciendaCode        *string    // Código de respuesta de Hacienda
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### FailedSequenceNumber

Registro de un intento fallido de uso de número secuencial.

```go
type FailedSequenceNumber struct {
    ID                  uint
    BranchID            uint
    DTEType             string
    SequenceNumber      int
    Year                int
    FailureReason       string           // Motivo del fallo
    ResponseCode        string           // Código de Hacienda
    OriginalRequestData json.RawMessage  // JSON del request original
    MHResponse          json.RawMessage  // JSON de la respuesta de Hacienda
    CreatedAt           time.Time
}
```

### DomainEvent

Evento de dominio para event sourcing.

```go
type DomainEvent struct {
    ID        uint
    Type      string           // Tipo de evento
    Payload   json.RawMessage  // Datos del evento
    CreatedAt time.Time
}
```

### UserNotification / NotifiableUser

Modelos para el sistema de notificaciones.

```go
type UserNotification struct {
    ID        uint
    UserID    uint
    Type      string
    Message   string
    Read      bool
    CreatedAt time.Time
}

type NotifiableUser struct {
    ID     uint
    UserID uint
    Email  string
    Active bool
}
```

---

## Migraciones

> **Archivo:** `database/migrations.go`

### Función Principal

```go
func RunMigrations(db *gorm.DB) error
```

### Flujo

```
RunMigrations(db)
  │
  ├── Definir lista de modelos:
  │   [User, BranchOffice, Address, DTEDocument, DTEDetails,
  │    ContingencyDocument, ControlNumberSequence, ReservedSequenceNumber,
  │    FailedSequenceNumber, DomainEvent, UserNotification,
  │    NotifiableUser, DTEBalanceControl, DTEBalanceTransaction]
  │
  └── Por cada modelo:
        ├── Log: "Migrando {ModelName}..."
        ├── db.AutoMigrate(&model)
        │   → Crear tabla si no existe
        │   → Agregar columnas nuevas
        │   → NO elimina columnas existentes
        └── Log: "Migración de {ModelName} completada"
```

### Comportamiento de AutoMigrate

- Se **crean** tablas que no existen
- Se **agregan** columnas nuevas a tablas existentes
- **No se eliminan** columnas que ya no están en el modelo
- **No se modifican** tipos de columnas existentes
- Las foreign keys se crean automáticamente basándose en los tags de GORM

---

## Diagrama de Relaciones

```
User (1)
  │
  ├── (N) BranchOffice
  │         ├── (1) Address
  │         ├── (N) DTEDocument
  │         │         ├── (1) DTEDetails
  │         │         └── (1) DTEBalanceControl
  │         │                   └── (N) DTEBalanceTransaction
  │         ├── (N) ContingencyDocument
  │         │         └── (1) ReservedSequenceNumber
  │         ├── (N) ControlNumberSequence
  │         ├── (N) ReservedSequenceNumber
  │         └── (N) FailedSequenceNumber
  │
  ├── (N) UserNotification
  └── (N) NotifiableUser
```

---

## Notas

1. **AutoMigrate no destructivo**: GORM AutoMigrate nunca elimina columnas ni tablas. Los cambios destructivos requieren migraciones manuales.
2. **JSON en columnas**: `DTEDetails.JSONContent`, `ContingencyDocument.JSONContent`, y los campos de `FailedSequenceNumber` almacenan JSON como `json.RawMessage` (tipo `[]byte` en Go, `JSON` o `TEXT` en BD).
3. **Soft deletes**: Los modelos no usan `gorm.DeletedAt` (soft delete). Las eliminaciones son permanentes.
4. **Índices**: Los campos frecuentemente consultados (`GenerationCode`, `BranchID`, `DTEType`, `Status`) deben tener índices para rendimiento óptimo.
5. **Balance control**: Solo se crea un `DTEBalanceControl` para documentos que pueden recibir notas de crédito/débito (facturas, CCF). Los demás tipos no lo necesitan.
