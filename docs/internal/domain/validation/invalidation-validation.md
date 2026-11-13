# Validación de Invalidación

> **Validador:** `InvalidationRulesValidator`
> **Paquete:** `internal/domain/dte/invalidation/validator`
> **Plazos y fecha del evento:** `internal/domain/dte/invalidation/policy/`
> **Calendario de días hábiles:** `internal/domain/dte/common/calendar/`

## Orden de la Validación

El caso de uso `InvalidationUseCase.InvalidateDocument` ejecuta, deteniéndose en el primer paso que falle:

| Paso | Qué se valida | Error |
|---|---|---|
| 1. Solicitud (`ValidateInvalidationReRequest`) | Motivo, tipo de invalidación, código de generación, datos del responsable y del solicitante, y coherencia del código de reemplazo | `RequiredField`, `InvalidReplacementCode1And3`, `InvalidInvalidationType2`, `InvalidInvalidationType3` |
| 2. Estado (`ValidateStatus`) | El documento a invalidar (y el de reemplazo, en tipos 1 y 3) no pueden estar invalidados, rechazados ni pendientes de retransmitir | `DocumentAlreadyInvalid`, `DocumentReject`, `DocumentPending` |
| 3. Dependencias (`ValidateDependencies`) | Un CCF o CR no puede tener NC o ND vigentes | `PendingAdjustmentNotes` |
| 4. Mapeo del documento a invalidar | El documento debe tener sello de recepción | `DocumentWithoutStamp` |
| 5. Reglas del evento (`InvalidationRulesValidator`) | Estrategias de las secciones siguientes | Ver cada estrategia |
| 6. Transmisión | Envío a Hacienda; solo si fue recibido se marca el documento como invalidado | `TransmissionFailed` |

### Dependencias entre documentos

El documento a invalidar no puede tener documentos que dependan de él:

- **CCF (`03`) o CR (`07`)**: si tiene notas de crédito o débito **recibidas o pendientes** registradas en su control de saldo, la invalidación se rechaza y el error lista los códigos de generación de las notas. El cliente debe invalidar primero cada nota y luego solicitar la invalidación del CCF o CR. La API no invalida notas de forma automática (Normativa, Anexo V, campo "Código de generación" del evento de invalidación, regla 3).
- Al invalidar una **NC o ND**, el servicio registra una transacción en el control de saldo del documento relacionado electrónico para restituir los montos (`FailedToRecoverInvalidatedAmounts` si falla).

## Estrategias Compuestas

Todas se ejecutan y los errores se agrupan:

1. **InvalidationBasicStrategy** — Campos requeridos
2. **InvalidationDocumentStrategy** — Validación del documento a invalidar
3. **InvalidationReasonStrategy** — Validación del motivo
4. **InvalidationDateStrategy** — Plazo de transmisión y fecha del evento

---

## 1. InvalidationBasicStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/strategy/invalidation_basic_strategy.go`

### Campos de Identificación

| Campo | Requerido |
|---|---|
| Identification | Sí (no nil) |
| Version | Sí |
| Ambient | Sí |
| GenerationCode | Sí |
| EmissionDate | Sí |
| EmissionTime | Sí |

### Campos del Emisor

| Campo | Requerido |
|---|---|
| Issuer | Sí (no nil) |
| NIT | Sí |
| Name | Sí |
| EstablishmentType | Sí |
| Email | Sí |

---

## 2. InvalidationDocumentStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/strategy/invalidation_document_strategy.go`

| Campo | Requerido | Validación |
|---|---|---|
| Document | Sí (no nil) | — |
| Type | Sí | Tipo de DTE válido del catálogo de tipos (`InvalidDTETypeForInvalidation`) |
| GenerationCode | Sí | No vacío |
| ReceptionStamp | Sí | Patrón `^[A-Z0-9]{40}$` |
| ControlNumber | Sí | No vacío |
| EmissionDate | Sí | No vacío |

### Patrón del Sello de Recepción

```regex
^[A-Z0-9]{40}$
```

Exactamente 40 caracteres alfanuméricos en mayúsculas. Este sello se recibe de Hacienda al momento de la transmisión exitosa del documento original.

---

## 3. InvalidationReasonStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/strategy/invalidation_reason_strategy.go`

### Campos Obligatorios del Motivo

| Campo | Requerido | Descripción |
|---|---|---|
| Reason | Sí (no nil) | — |
| Type | Sí | 1, 2, o 3 |
| ResponsibleName | Sí | Nombre del responsable |
| ResponsibleDocType | Sí | Tipo de documento (NIT, DUI, etc.) |
| ResponsibleDocNum | Sí | Número de documento |
| RequesterName | Sí | Nombre del solicitante |
| RequesterDocType | Sí | Tipo de documento |
| RequesterDocNum | Sí | Número de documento |

### Reglas por Tipo de Invalidación

| Tipo | Nombre | Reason | ReplacementCode |
|---|---|---|---|
| 1 | Reemplazo | No requerido | **Requerido** |
| 2 | Anulación | No requerido | **Debe ser null** |
| 3 | Otro motivo | **Requerido** | **Requerido** |

### Validación de Tipos de Documento

Los `ResponsibleDocType` y `RequesterDocType` deben ser tipos válidos del receptor (`InvalidEnum`):
- `36` (NIT), `13` (DUI), `02` (Carnet Residente), `03` (Pasaporte), `37` (Otro)

---

## 4. InvalidationDateStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/strategy/invalidation_date_strategy.go`

Se valida por separado el **plazo de transmisión** (instante actual contra la fecha del sello) y la **fecha del evento** (contra la fecha de generación del documento). Si el documento no tiene fecha de sello se devuelve `MissingStampDate`.

### Plazo de Transmisión (`policy.Deadlines.TransmissionDeadline`)

El plazo se cuenta desde la fecha del sello de recepción (`processing_date` / `fhProcesamiento`; para documentos antiguos sin ese dato se usa la fecha de registro) y vence a las **23:59:59** del último día:

| Tipo de DTE | Plazo máximo | Referencia en la Normativa |
|---|---|---|
| FE (`01`), FEXE (`11`), FSE (`14`) | 3 meses desde el sello. Si el día no existe en el mes destino se usa el último día de ese mes | Cuadro 6, fila 2 |
| FE y FEXE de emisores con actividad económica `21001`, `21008`, `46482`, `46484`, `46491` o `47721` | 2 años desde el sello (no aplica a FSE) | Cuadro 6, fila 3 (venta de medicamentos y especialidades farmacéuticas perecederos) |
| Resto de tipos (`03`, `04`, `05`, `06`, `07`) | Décimo día hábil del mes siguiente al mes del sello | Cuadro 6, fila 1 |

El código del emisor para la regla de 2 años se toma de `codActividad` del emisor del documento original.

Error: `InvalidationDeadlineExpired` (parámetros: tipo de DTE y fecha límite `YYYY-MM-DD HH:MM:SS`). El documento conserva su validez; el mensaje sugiere emitir una NC o ND si corresponde.

### Días Hábiles (`calendar.BusinessCalendar`)

Un día es hábil si es lunes a viernes y no es:

- Asueto nacional fijo: 1 de enero, 1 y 10 de mayo, 17 de junio, 6 de agosto, 15 de septiembre, 2 de noviembre y 25 de diciembre.
- Semana Santa: desde el Lunes Santo hasta el Lunes de Pascua (inclusive).
- Una fecha configurada en la variable `MH_EXTRA_HOLIDAYS` (lista separada por comas, formato `YYYY-MM-DD`).

El décimo día hábil se cuenta desde el día 1 del mes siguiente (el día 1 cuenta si es hábil).

### Fecha del Evento (`fecEmi` / `horEmi`)

La fecha y hora del evento se asignan al mapear la solicitud (`policy.EventDateTime`):

| Tipo de DTE | Fecha y hora asignadas al evento | Validación | Error |
|---|---|---|---|
| FE, FEXE, FSE | El instante actual | La fecha del evento no puede ser posterior a la fecha de generación más el período de validez (3 meses o 2 años, igual que el plazo de transmisión) | `InvalidationEventDateOutOfRange` |
| Resto (`03`, `04`, `05`, `06`, `07`) | La fecha y hora de generación del documento; si fue generado hoy, el instante actual | La fecha del evento debe ser el mismo día de la fecha de generación | `InvalidationEventDateMismatch` |

La fecha de generación se toma del documento original (`fecEmi`); si no se conoce, la validación de fecha del evento se omite.

---

## Errores Específicos de Invalidación

| Error | Causa |
|---|---|
| `RequiredField` | Campo obligatorio faltante |
| `InvalidPattern` | Sello de recepción no cumple patrón |
| `InvalidDTETypeForInvalidation` | Tipo de DTE no pertenece al catálogo de tipos |
| `InvalidEnum` | Tipo de invalidación fuera de rango (1-3) o tipo de documento del responsable o solicitante no válido |
| `InvalidField` | ReplacementCode presente cuando no debe (tipo 2) |
| `InvalidReplacementCode1And3` | Tipos 1 y 3 sin código de reemplazo (solicitud) |
| `InvalidInvalidationType2`, `InvalidInvalidationType3` | Tipo 2 con código de reemplazo; tipo 3 sin campo `reason_field` (solicitud) |
| `DocumentAlreadyInvalid`, `DocumentReject`, `DocumentPending` | Estado del documento a invalidar o del de reemplazo |
| `PendingAdjustmentNotes` | CCF o CR con NC o ND vigentes |
| `DocumentWithoutStamp` | El documento no tiene sello de recepción |
| `MissingStampDate` | No se conoce la fecha del sello |
| `InvalidationDeadlineExpired` | Plazo de transmisión vencido |
| `InvalidationEventDateOutOfRange` | Fecha del evento de FE/FEXE/FSE fuera de período |
| `InvalidationEventDateMismatch` | Fecha del evento distinta a la generación |
| `TransmissionFailed` | Hacienda no recibió el evento |
