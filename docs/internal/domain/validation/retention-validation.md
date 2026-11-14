# Validación de Retención (DTE 07)

> **Validador:** `RetentionRulesValidator`
> **Paquete:** `internal/domain/dte/retention/validator`

Las fórmulas, decimales y holguras se detallan en [Cálculos que valida la API, sección 4](../../../api/calculations.md#4-comprobante-de-retención-cr); aquí se resumen por estrategia.

El Comprobante de Retención solo ejecuta este validador: **no pasa por la validación general** (estrategias comunes y validación de Value Objects por reflexión) que usan los demás DTE. Las validaciones de formato se aplican al mapear la solicitud.

## Estrategias Compuestas

Ambas se ejecutan y los errores se agrupan:

1. **RetentionItemStrategy** — Validación de ítems de retención, IVA retenido y fechas
2. **RetentionTotalStrategy** — Validación de totales

---

## Validaciones al Mapear la Solicitud

| Validación | Regla | Error |
|---|---|---|
| Código de retención | Debe ser `22`, `C4` o `C9` | `InvalidRetentionCode` |
| Código único | Todos los ítems deben tener el mismo código de retención | `MixedRetentionCodes` |
| Campos del ítem | `taxed_amount`, `iva_amount`, `emission_date` y `dte_type` son obligatorios en cada ítem | `RequiredField` |
| Tipo de documento del ítem | `dte_type` debe ser un tipo de DTE válido del catálogo; la lista restringida a `01`, `03` y `14` (`ValidRetentionDTETypes`) no se aplica en este mapeo | `InvalidDTEType` |
| Resumen | `total_retention_amount` y `total_retention_iva` obligatorios y distintos de 0; múltiplos de 0.01 | `RequiredField`, `InvalidDecimals` |
| Tipo de generación del ítem | `type` entre 1 y 2 | `InvalidNumberRange` |
| Número de documento | Según el tipo de generación del ítem (físico: 1 a 20 caracteres; electrónico: UUID) | `InvalidDocumentNumberItem` |

---

## 1. RetentionItemStrategy

**Archivo:** `internal/domain/dte/retention/validator/strategy/retention_item_strategy.go`

### Validación de IVA Retenido

Por cada ítem de retención, `RetentionIVA` (`iva_amount`) debe coincidir con el cálculo basado en el código de retención (Anexo IV, campo 105: monto sujeto a retención por el porcentaje legal):

```
RetentionIVA = RetentionAmount * GetRetentionAmount(ReceptionCodeMH)    (±0.01)    InvalidRetentionIVA
```

| Código | Descripción | Factor |
|---|---|---|
| `22` | Retención IVA 1% | 0.01 |
| `C4` | Retención IVA 13% | 0.13 |
| `C9` | Otros casos | Sin factor definido: se espera `RetentionIVA = 0` |

### Validación de Fechas

La fecha de emisión de cada documento retenido se compara con la fecha de emisión del CR (`DateOutOfAllowedRange`):

```
1. Se acepta si el documento y el CR son del MISMO MES (sin importar el día).
2. Se acepta si el documento es del MES ANTERIOR y el CR se emite a más tardar el décimo
   día hábil contado a partir del día 1 del mes del CR.
3. Cualquier otro caso se rechaza.
```

Los días hábiles de esta regla excluyen únicamente sábados y domingos; no se consideran asuetos ni los días de `MH_EXTRA_HOLIDAYS` (a diferencia de la invalidación).

#### Ejemplo

```
CR emitido el 15 de marzo de 2024 (el 1 de marzo de 2024 es viernes; el décimo día hábil
posterior es el viernes 15 de marzo)

Documentos válidos:
  OK: cualquier fecha de marzo de 2024 (mismo mes)
  OK: cualquier fecha de febrero de 2024 (el CR se emite a más tardar el 15 de marzo)

Documentos inválidos:
  NO: documentos de enero de 2024 o anteriores

CR emitido el 18 de marzo de 2024: los documentos de febrero ya no son válidos.
```

---

## 2. RetentionTotalStrategy

**Archivo:** `internal/domain/dte/retention/validator/strategy/retention_total_strategy.go`

### Totales

```
TotalSubjectRetention (total_retention_amount) = Sum(item.RetentionAmount)    (±0.01)    InvalidTotalSubjectRetention
TotalIVARetention (total_retention_iva)        = Sum(item.RetentionIVA)       (±0.01)    InvalidTotalIVARetention
```

El método `GetTotalByItems()` del `RetentionModel` calcula ambos totales.

### IVA 13% del Resumen (campo 157)

```
TotalIVA = TotalSubjectRetention * 0.13      si el código es C4 y TODOS los documentos son electrónicos (type = 2)
TotalIVA = 0.00                              en los demás casos
```

Holgura ±0.01, error `InvalidTotalIVA`. Si el cliente no envía `total_iva` (o envía `0`), la API lo calcula con esta misma regla (`retention_models.ExpectedTotalIVA`). La fórmula del campo 157 para el código C4 es "monto sujeto a retención por 13 %" (Anexo IV, campo 157, regla 4); se aplica solo con documentos electrónicos porque con documentos físicos Hacienda rechazó ese valor y aceptó `0.00`.

---

## Tipos de DTE Retenidos

El validador no restringe el tipo de documento retenido más allá de que `dte_type` pertenezca al catálogo de tipos de DTE. La constante `ValidRetentionDTETypes` (`01`, `03` y `14`) y el mensaje `InvalidDTETypeForRetention` existen pero la función que los aplica (`NewDTETypeForRetention`) no se invoca en el mapeo de la solicitud.

---

## Errores Específicos de Retención

| Error | Causa |
|---|---|
| `RequiredField` | Campos obligatorios faltantes |
| `InvalidRetentionCode` | Código de retención distinto de `22`, `C4` o `C9` |
| `MixedRetentionCodes` | Ítems con códigos de retención distintos |
| `InvalidRetentionIVA` | IVA retenido por ítem no coincide con la fórmula |
| `DateOutOfAllowedRange` | Fecha del documento fuera del período permitido |
| `InvalidTotalSubjectRetention` | Total sujeto a retención no coincide con la suma de ítems |
| `InvalidTotalIVARetention` | Total de IVA retenido no coincide con la suma de ítems |
| `InvalidTotalIVA` | `total_iva` del resumen incorrecto |
