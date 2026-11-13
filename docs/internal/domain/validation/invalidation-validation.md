# Validación de Invalidación

> **Validador:** `InvalidationRulesValidator`
> **Paquete:** `internal/domain/dte/invalidation/validator`

## Estrategias Compuestas

1. **InvalidationBasicStrategy** — Campos requeridos
2. **InvalidationDocumentStrategy** — Validación del documento a invalidar
3. **InvalidationReasonStrategy** — Validación del motivo
4. **InvalidationDateStrategy** — Restricciones temporales

---

## 1. InvalidationBasicStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/invalidation_basic_strategy.go`

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

**Archivo:** `internal/domain/dte/invalidation/validator/invalidation_document_strategy.go`

| Campo | Requerido | Validación |
|---|---|---|
| Document | Sí (no nil) | — |
| Type | Sí | Tipo de DTE válido |
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

**Archivo:** `internal/domain/dte/invalidation/validator/invalidation_reason_strategy.go`

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

Los `ResponsibleDocType` y `RequesterDocType` deben ser tipos válidos del receptor:
- `36` (NIT), `13` (DUI), `02` (Carnet Residente), `03` (Pasaporte), `37` (Otro)

---

## 4. InvalidationDateStrategy

**Archivo:** `internal/domain/dte/invalidation/validator/invalidation_date_strategy.go`

### Ventanas Temporales

| Tipo de DTE | Plazo para transmitir la invalidación (contado desde el sello) | Fecha del evento (`fecEmi`/`horEmi`) |
|---|---|---|
| FE (`01`), FEXE (`11`), FSE (`14`) | Hasta las 23:59:59 del mismo día, 3 meses después (2 años para FE y FEXE de emisores con actividad 21001, 21008, 46482, 46484, 46491 o 47721) | Fecha y hora actual; no puede superar el mismo periodo contado desde la generación |
| Resto (`03`, `04`, `05`, `06`, `07`) | Hasta el décimo día hábil del mes siguiente al mes del sello | Misma fecha y hora de generación del documento (la hora actual si se generó hoy) |

- La fecha del sello es `processing_date` (`fhProcesamiento`); para documentos antiguos sin ese dato se usa la fecha de registro.
- Días hábiles: lunes a viernes sin asuetos nacionales fijos, Semana Santa (de Lunes Santo a Lunes de Pascua), ni los asuetos de `MH_EXTRA_HOLIDAYS`.

Se valida por separado el plazo de transmisión (fecha actual contra la fecha del sello) y la fecha del evento (contra la fecha de generación).

---

## Errores Específicos de Invalidación

| Error | Causa |
|---|---|
| `RequiredField` | Campo obligatorio faltante |
| `InvalidPattern` | Sello de recepción no cumple patrón |
| `InvalidDTETypeForInvalidation` | Tipo de DTE no válido |
| `InvalidEnum` | Tipo de invalidación fuera de rango (1-3) |
| `InvalidField` | ReplacementCode presente cuando no debe (tipo 2) |
| `InvalidationDeadlineExpired` | Plazo de transmisión vencido |
| `InvalidationEventDateOutOfRange` | Fecha del evento de FE/FEXE/FSE fuera de periodo |
| `InvalidationEventDateMismatch` | Fecha del evento distinta a la generación |
| `MissingStampDate` | No se conoce la fecha del sello |
