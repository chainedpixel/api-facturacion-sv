# Validación de Retención (DTE 07)

> **Validador:** `RetentionRulesValidator`
> **Paquete:** `internal/domain/dte/retention/validator`

## Estrategias Compuestas

1. **RetentionItemStrategy** — Validación de ítems de retención y cálculos de IVA
2. **RetentionTotalStrategy** — Validación de totales

---

## 1. RetentionItemStrategy

**Archivo:** `internal/domain/dte/retention/validator/retention_item_strategy.go`

### Validación de IVA Retenido

Por cada ítem de retención, se valida que el `RetentionIVA` coincida con el cálculo basado en el código de retención:

```
RetentionIVA = RetentionAmount * GetRetentionAmount(ReceptionCodeMH)
```

| Código | Descripción | Factor |
|---|---|---|
| `22` | Retención IVA 1% | 0.01 |
| `C4` | Retención IVA 13% | 0.13 |
| `C9` | Retención IVA otros | Variable |

Tolerancia: ±0.01

### Validación de Fechas

La fecha de emisión del documento retenido debe estar dentro del período permitido:

#### Regla del Período

```
1. La fecha puede ser del MISMO MES que la retención
2. La fecha puede ser del MES ANTERIOR si está dentro de 10 días hábiles
   del inicio del mes de la retención
3. Los días hábiles excluyen sábados y domingos
```

#### Ejemplo

```
Retención emitida: 15 de marzo 2024

Documentos válidos:
  OK: 1 de marzo 2024 - 15 de marzo 2024 (mismo mes)
  OK: 15 de febrero 2024 - 29 de febrero 2024 (mes anterior, dentro de período)

Documentos inválidos:
  NO: 1 de enero 2024 (fuera de período)
  NO: 16 de marzo 2024 (futuro)
```

---

## 2. RetentionTotalStrategy

### Totales

```
TotalSubjectRetention = Sum(item.RetentionAmount)     para cada ítem
TotalIVARetention     = Sum(item.RetentionIVA)        para cada ítem
```

El método `GetTotalByItems()` del `RetentionModel` calcula ambos totales.

---

## Tipos de DTE Válidos para Retención

Solo se pueden incluir documentos de los siguientes tipos:

| Código | Tipo |
|---|---|
| `01` | Factura Electrónica |
| `03` | CCF Electrónico |
| `11` | Factura de Exportación Electrónica |

---

## Errores Específicos de Retención

| Error | Causa |
|---|---|
| `RequiredField` | Campos obligatorios faltantes |
| `InvalidRetentionIVA` | IVA retenido no coincide con fórmula |
| `DateOutOfAllowedRange` | Fecha fuera del período permitido |
