# Base de Datos — Modelos y Migraciones

> **Paquetes:**
> - `internal/infrastructure/database/db_models` — Modelos de base de datos (GORM)
> - `internal/infrastructure/database` — Migraciones automáticas

## Descripción General

Los modelos de base de datos representan la estructura de persistencia del sistema. Se definen como structs de Go con tags de GORM que mapean a tablas de la base de datos. Las migraciones **no** se ejecutan automáticamente en cada arranque: solo corren cuando la variable de entorno `RUN_MIGRATION=true` (`config.Server.RunMigration`), evaluada en `bootstrap/app.go` justo después de abrir la conexión. Con el valor por defecto (falso) no se crea ni se modifica ninguna tabla o columna.

El sistema define 15 modelos (tablas). Los nombres de tabla y de columna que se listan a continuación son los declarados en los tags `gorm` de cada struct.

---

## Modelos de Base de Datos

### User (`users`)

Contribuyente que usa la API.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | uint, PK, autoincrement | |
| `nit` | varchar(17) | único |
| `nrc` | varchar(10) | único |
| `status` | tinyint | índice |
| `auth_type` | varchar(15) | |
| `password_pri` | varchar(255) | contraseña de la llave privada del certificado (firma) |
| `commercial_name` | varchar(150) | |
| `economic_activity` | varchar(6) | |
| `economic_activity_desc` | varchar(150) | |
| `business_name` | varchar(200) | |
| `email` | varchar(100) | único |
| `phone` | varchar(30) | único |
| `year_in_dte` | tinyint | si es verdadero, el número de control incluye el año |
| `token_lifetime` | int | por defecto 14 |
| `created_at`, `updated_at` | timestamp | |

### BranchOffice (`branch_offices`)

Sucursal o establecimiento del contribuyente. Los campos terminados en `_mh` son códigos asignados por Hacienda.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | uint, PK | |
| `user_id` | uint | FK a `users`, índice |
| `establishment_code` | varchar(10), nulo | código interno |
| `establishment_code_mh` | varchar(4), nulo | código de establecimiento asignado por Hacienda |
| `email`, `phone` | varchar, nulos | |
| `api_key` | varchar(255) | único |
| `api_secret` | varchar(255) | |
| `establishment_type` | varchar(2) | índice |
| `pos_code` | varchar(15), nulo | código interno del punto de venta |
| `pos_code_mh` | varchar(4), nulo | código de punto de venta asignado por Hacienda |
| `is_active` | tinyint(1) | índice |

### Address (`addresses`)

Dirección de la sucursal (relación 1:1 por `branch_id`).

| Columna | Tipo | Notas |
|---|---|---|
| `id` | uint, PK | |
| `branch_id` | uint | FK a `branch_offices`, índice |
| `municipality` | varchar(2) | |
| `department` | varchar(2) | |
| `district` | varchar(2) | por defecto `'01'` |
| `complement` | varchar(200) | |

### DTEDocument (`dte_documents`)

Relación entre un documento y la sucursal que lo emitió.

| Columna | Tipo | Notas |
|---|---|---|
| `document_id` | varchar(36), PK | FK a `dte_details.id` (código de generación) |
| `branch_id` | uint | FK a `branch_offices`, índice |
| `created_at`, `updated_at` | timestamp | `created_at` indexado |

Hook `AfterCreate`: si el tipo del documento pertenece a `ValidAdjustmentDTETypes` (hoy `03` CCF y `07` comprobante de retención), se crea automáticamente su `DTEBalanceControl` con los totales gravado, exento y no sujeto leídos del resumen del JSON, netos del descuento global de cada categoría (`NetTaxed`, `NetExempt` y `NetNotSubject` de `AuxiliarTotalAmountsExtractor`, redondeados a centavos y nunca negativos).

### DTEDetails (`dte_details`)

Contenido y evidencia de transmisión del documento.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | varchar(36), PK | código de generación (UUID) |
| `dte_type` | varchar(2) | índice |
| `control_number` | varchar(30) | índice |
| `reception_stamp` | varchar(40), nulo | sello de recepción de Hacienda |
| `transmission` | varchar(15) | `NORMAL` o `CONTINGENCY` |
| `status` | varchar(15) | `RECEIVED`, `REJECTED`, `INVALIDATED`, `PENDING`, `EXPIRED`; índice |
| `json_data` | json | estructura del DTE (MySQL puede reordenar las llaves del tipo JSON) |
| `signed_document` | longtext, nulo | JWS transmitido a Hacienda; es la fuente del Archivo DTE (estructura firmada exacta + `firmaElectronica` + `selloRecibido`) |
| `processing_date` | varchar(30), nulo | fecha y hora de procesamiento informada por Hacienda (`fhProcesamiento`) |
| `hacienda_response` | text, nulo | respuesta completa de Hacienda en JSON |

Las columnas `signed_document`, `processing_date` y `hacienda_response` son nulas para documentos creados antes de su incorporación. Se llenan desde `TransmissionEvidence` (ver [transmitter.md](transmitter.md) y [repositories.md](repositories.md)).

### DTEEvent (`dte_events`)

Eventos transmitidos a Hacienda (invalidación y contingencia) con su evidencia.

| Columna | Tipo | Notas |
|---|---|---|
| `generation_code` | varchar(36), PK | código de generación del evento |
| `branch_id` | uint | índice |
| `event_type` | varchar(20) | `INVALIDATION` o `CONTINGENCY`; índice |
| `related_dte_id` | varchar(36), nulo | documento afectado (invalidación); índice. Es nulo en eventos de contingencia |
| `replacement_dte_id` | varchar(36), nulo | documento de reemplazo, si existe |
| `json_data` | json | estructura del evento |
| `signed_document` | longtext, nulo | JWS transmitido |
| `reception_stamp` | varchar(40), nulo | sello de recepción del evento |
| `processing_date` | varchar(30), nulo | |
| `hacienda_response` | text, nulo | respuesta completa de Hacienda |
| `status` | varchar(15) | |
| `created_at` | timestamp | |

Con `signed_document` y `reception_stamp` se arma el Archivo DTE del evento. Se consulta mediante `GET /api/v1/dte/{id}/eventos` (por `related_dte_id`).

### DTEBalanceControl (`dte_balance_control`)

Saldo remanente de un documento sobre el que se pueden emitir notas de crédito y de débito. Los montos originales y remanentes iniciales son los totales del resumen menos el descuento global de cada categoría (gravada, exenta y no sujeta).

| Columna | Tipo |
|---|---|
| `id` | uint, PK |
| `branch_id` | uint, índice |
| `original_dte_id` | varchar(36), índice (FK a `dte_details.id`) |
| `original_taxed_amount`, `original_exempt_amount`, `original_not_subject_amount` | decimal(18,2) |
| `remaining_taxed_amount`, `remaining_exempt_amount`, `remaining_not_subject_amount` | decimal(18,2) |
| `created_at`, `updated_at` | timestamp |

### DTEBalanceTransaction (`dte_balance_transactions`)

Movimiento de una nota que afecta el saldo.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | uint, PK | |
| `balance_control_id` | int | FK a `dte_balance_control` |
| `adjustment_document_id` | varchar(36) | FK a `dte_details.id` (la nota) |
| `transaction_type` | varchar(20) | tipo de DTE de la nota (`05` o `06`) o tipo de invalidación (`INVALIDATED` para una NC invalidada, `DEBIT_NOTE_INVALID` para una ND invalidada). `AfterCreate` resta los montos del saldo restante cuando el tipo es `05` o `DEBIT_NOTE_INVALID`; con cualquier otro tipo los suma |
| `taxed_amount`, `exempt_amount`, `not_subject_amount` | decimal(18,2) | |
| `created_at` | timestamp | |

Hook `AfterCreate`: si el tipo es `05` (nota de crédito) o `DEBIT_NOTE_INVALID` (nota de débito invalidada) se **resta** de los saldos remanentes; en caso contrario (nota de débito emitida o nota de crédito invalidada, `INVALIDATED`) se **suma**, y se guarda el `DTEBalanceControl`.

### ContingencyDocument (`contingency_documents`)

Documento emitido en contingencia, pendiente de transmitir.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | varchar(36), PK | UUID |
| `document_id` | varchar(36) | FK a `dte_details.id`, índice |
| `branch_id` | uint | FK, índice |
| `type` | tinyint | tipo de contingencia, índice |
| `reason` | varchar(150) | índice |
| `batch_id` | varchar(36), nulo | lote interno |
| `mh_batch_id` | varchar(36), nulo | lote de Hacienda |
| `observations` | text, nulo | observaciones (incluye el motivo de vencimiento) |
| `event_stamped_at` | timestamp, nulo, índice | fecha del sello del evento de contingencia; inicia la ventana de transmisión de 72 horas. Es nulo mientras el evento no se haya enviado |
| `created_at`, `updated_at` | timestamp | `created_at` indexado |

`event_stamped_at` solo se escribe una vez (ver `MarkEventStamped` en [repositories.md](repositories.md)), de modo que reenviar el evento no reinicia la ventana.

### ControlNumberSequence (`control_number_sequences`)

Contador del número de control por sucursal, tipo de DTE y año.

| Columna | Tipo | Notas |
|---|---|---|
| `id` | uint, PK | |
| `branch_id` | uint | parte del índice único `idx_branch_dte_year` |
| `dte_type` | varchar(2) | parte del índice único |
| `year` | int | parte del índice único |
| `last_number` | int | último número usado |
| `created_at`, `updated_at` | timestamp | |

### ReservedSequenceNumber (`reserved_sequence_numbers`)

Reserva de un correlativo antes de emitir el documento. Esta tabla no declara el nombre de columna en los tags; GORM usa la convención snake_case.

| Campo | Notas |
|---|---|
| `id` | PK |
| `branch_id`, `dte_type` (varchar(2)), `sequence_number`, `year` | índice único `idx_reservation_unique` |
| `status` | varchar(20), por defecto `RESERVED`; valores usados: `RESERVED`, `CONFIRMED`, `RELEASED` |
| `document_id` | varchar(36), nulo |
| `reserved_at` | creación automática |
| `confirmed_at`, `released_at` | nulos |
| `expires_at` | nulo, índice |
| `rejection_reason` (text), `hacienda_code` (varchar(10)) | nulos |
| `is_contingency` | bool, por defecto falso, índice |
| `contingency_document_id` | varchar(36), nulo |

### FailedSequenceNumber (`failed_sequence_numbers`)

Correlativo cuyo intento de transmisión falló.

| Columna | Tipo |
|---|---|
| `id` | uint, PK |
| `branch_id`, `dte_type` (varchar(2)), `sequence_number`, `year` | índice compuesto `idx_failed_seq` |
| `failure_reason` | text |
| `response_code` | varchar(10) |
| `original_request_data` | json |
| `mh_response` | text |
| `created_at` | timestamp |

### DomainEvent (`domain_events`)

Evento de dominio persistido (por ejemplo, contingencia activada).

| Columna | Tipo |
|---|---|
| `id` | uint, PK |
| `user_id`, `branch_id` | uint, FK, índices |
| `event_type` | varchar(50), índice |
| `payload` | json |
| `occurred_at` | timestamp, índice |

### UserNotification (`user_notifications`) y NotifiableUser (`notification_users`)

```go
type UserNotification struct {
    ID               uint
    UserID           uint     // FK a users
    EventID          uint     // FK a domain_events
    NotificationType string   // varchar(15)
    Message          string   // text
    DeliveryStatus   string   // varchar(15)
    DeliveryAt       time.Time
}

type NotifiableUser struct {   // tabla notification_users
    ID          uint
    UserID      uint
    EntityType  string  // varchar(10), polimórfico (usuario cliente o administrador)
    Email       string  // varchar(255)
    EnabledPush bool
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
RunMigrations(db)   (solo si RUN_MIGRATION=true)
  │
  ├── Lista de modelos (en este orden):
  │   [User, BranchOffice, Address, DTEDetails, DTEDocument,
  │    ContingencyDocument, ControlNumberSequence, FailedSequenceNumber,
  │    ReservedSequenceNumber, DomainEvent, UserNotification,
  │    NotifiableUser, DTEBalanceControl, DTEBalanceTransaction, DTEEvent]
  │
  └── Por cada modelo:
        ├── Log: "Starting model migration #N: {tabla}"
        ├── db.AutoMigrate(&model)
        │   → Crea la tabla si no existe
        │   → Agrega columnas nuevas
        │   → NO elimina columnas existentes
        └── Ante un error se registra y se aborta (el arranque falla con Fatal)
```

### Comportamiento de AutoMigrate

- Se **crean** tablas que no existen (incluida `dte_events`).
- Se **agregan** columnas nuevas a tablas existentes (por ejemplo `signed_document`, `processing_date` y `hacienda_response` en `dte_details`, o `event_stamped_at` en `contingency_documents`).
- **No se eliminan** columnas que ya no están en el modelo.
- Las foreign keys se crean automáticamente basándose en los tags de GORM.

> Al desplegar una versión que agrega columnas o tablas (por ejemplo `dte_events`, las columnas de evidencia o `event_stamped_at`), la base de datos debe migrarse arrancando una vez con `RUN_MIGRATION=true`; de lo contrario, las consultas que usan esas columnas fallan.

---

## Diagrama de Relaciones

```
User (1)
  │
  └── (N) BranchOffice
            ├── (1) Address
            ├── (N) DTEDocument ── (1) DTEDetails
            │                          ├── (1) DTEBalanceControl ── (N) DTEBalanceTransaction
            │                          └── (N) ContingencyDocument
            ├── (N) ContingencyDocument
            ├── (N) DTEEvent               (related_dte_id referencia lógica a dte_details.id)
            ├── (N) ControlNumberSequence
            ├── (N) ReservedSequenceNumber
            └── (N) FailedSequenceNumber

DomainEvent (N) ── (N) UserNotification        User (1) ── (N) UserNotification
```

---

## Notas

1. **AutoMigrate no destructivo**: GORM AutoMigrate nunca elimina columnas ni tablas. Los cambios destructivos requieren migraciones manuales.
2. **Evidencia de transmisión**: `signed_document` (JWS) es la fuente del Archivo DTE; `json_data` no se usa para ello porque el tipo JSON de MySQL reordena las llaves.
3. **Soft deletes**: Los modelos no usan `gorm.DeletedAt`. Las eliminaciones son permanentes.
4. **Balance control**: Solo se crea un `DTEBalanceControl` para los tipos de `ValidAdjustmentDTETypes` (hoy `03` y `07`), vía el hook `AfterCreate` de `DTEDocument`.
5. **Convención de IDs**: `dte_details.id` y `dte_documents.document_id` almacenan el código de generación (UUID), no un entero autoincremental.
