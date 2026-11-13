# Constantes y Enumeraciones del Dominio

> **Paquete:** `internal/domain/dte/common/constants`

## Descripción General

Las constantes del dominio definen los valores permitidos para campos regulados por el Ministerio de Hacienda de El Salvador. Estos valores están normados y no deben modificarse sin una actualización de la regulación fiscal.

---

## Tipos de DTE

| Constante | Código | Nombre | Descripción |
|---|---|---|---|
| `FacturaElectronica` | `01` | Factura Electrónica | Venta a consumidor final |
| `CCFElectronico` | `03` | Crédito Fiscal | Venta entre contribuyentes (B2B) |
| `NotaRemisionElectronica` | `04` | Nota de Remisión | Traslado de mercadería |
| `NotaCreditoElectronica` | `05` | Nota de Crédito | Ajuste a favor del receptor |
| `NotaDebitoElectronica` | `06` | Nota de Débito | Ajuste a favor del emisor |
| `ComprobanteRetencionElectronico` | `07` | Retención | Registro de retenciones |
| `ComprobanteLiquidacionElectronico` | `08` | Liquidación | Comprobante de liquidación |
| `DocContableLiquidacionElectronico` | `09` | Doc. Contable Liquidación | Documento contable |
| `FacturaExportacionElectronica` | `11` | Factura Exportación | Venta de exportación |
| `FacturaSujetoExcluidoElectronica` | `14` | FSE | Compra a sujeto excluido |
| `ComprobanteDonacionElectronico` | `15` | Donación | Comprobante de donación |

`ValidDTETypes` agrupa los once tipos.

---

## Estados de Documento

| Constante | Valor | Descripción |
|---|---|---|
| `DocumentReceived` | `"RECEIVED"` | Recibido por Hacienda exitosamente |
| `DocumentRejected` | `"REJECTED"` | Rechazado por Hacienda |
| `DocumentInvalid` | `"INVALIDATED"` | Invalidado después de ser recibido |
| `DocumentPending` | `"PENDING"` | Pendiente de transmisión (contingencia) |
| `DocumentExpired` | `"EXPIRED"` | Contingencia no transmitida dentro de 72 horas desde el sello del evento |

`ValidReceiverDocumentStates` agrupa los cinco estados y se usa para validar el filtro de estado en las consultas.

### Transiciones de Estado

```
[Nuevo] ──transmisión──→ RECEIVED
                          │
                          ├──invalidación──→ INVALIDATED
                          │
[Nuevo] ──transmisión──→ REJECTED

[Nuevo] ──contingencia──→ PENDING
                          │
                          ├──retransmisión exitosa──→ RECEIVED
                          ├──retransmisión fallida──→ REJECTED
                          └──ventana de 72 horas vencida──→ EXPIRED
```

Ver [contingency.md](dte/contingency.md) para el detalle de la ventana de 72 horas.

---

## Tipos de Transmisión

| Constante | Valor | Descripción |
|---|---|---|
| `TransmissionNormal` | `"NORMAL"` | Transmisión directa a Hacienda |
| `TransmissionContingency` | `"CONTINGENCY"` | Almacenamiento local por falla |

`ValidTransmissionTypes` agrupa ambos valores y se usa para validar el filtro de transmisión en las consultas.

Los valores numéricos del catálogo de tipo de transmisión del DTE (`TransmisionNormal = 1`, `TransmisionContingencia = 2`, lista `AllowedTransmisionTypes`) se definen en `transsmision_type.go`.

---

## Ambientes

| Constante | Valor | Descripción |
|---|---|---|
| `Testing` | `"00"` | Ambiente de pruebas |
| `Production` | `"01"` | Ambiente de producción |

Lista permitida: `AllowedAmbientValues`.

---

## Modelo de Facturación

| Constante | Valor | Descripción |
|---|---|---|
| `ModeloFacturacionPrevio` | `1` | Facturación en tiempo real (transmisión normal) |
| `ModeloFacturacionDiferido` | `2` | Facturación diferida (contingencia) |

**Regla:** El modelo debe ser consistente con el tipo de transmisión (`model_type_strategy.go`). Al transmitir en contingencia, `pkg/shared/utils/updates.go` asigna `ModeloFacturacionDiferido` al documento. Lista permitida: `AllowedModeloFacturacion`.

---

## Condición de Operación

| Constante | Valor | Descripción |
|---|---|---|
| `Cash` | `1` | Operación al contado |
| `Credit` | `2` | Operación a crédito |
| `Other` | `3` | Otra condición |

Lista permitida: `ValidPaymentConditions`.

---

## Tipos de Documento del Receptor

| Constante | Código | Tipo |
|---|---|---|
| `NIT` | `"36"` | Número de Identificación Tributaria |
| `DUI` | `"13"` | Documento Único de Identidad |
| `CarnetResidente` | `"02"` | Carnet de Residente |
| `Pasaporte` | `"03"` | Pasaporte |
| `OtroDocumento` | `"37"` | Otro tipo de documento |

La lista permitida es `ValidReceiverDTETypes`; también se usa para el tipo de documento del responsable y del solicitante en la invalidación.

---

## Tipos de Documento (Físico/Electrónico)

| Constante | Valor | Descripción |
|---|---|---|
| `PhysicalDocument` | `1` | Documento físico (papel) |
| `ElectronicDocument` | `2` | Documento electrónico |

---

## Formas de Pago

| Constante | Código | Descripción |
|---|---|---|
| `BilletesMonedas` | `"01"` | Efectivo |
| `TarjetaDebito` | `"02"` | Tarjeta de débito |
| `TarjetaCredito` | `"03"` | Tarjeta de crédito |
| `Cheque` | `"04"` | Cheque |
| `TransBancaria` | `"05"` | Transferencia bancaria |
| `TarjetaPrePago` | `"06"` | Tarjeta prepago |
| `Vales` | `"07"` | Vales |
| `CriptoMoneda` | `"08"` | Criptomoneda |
| `PagosElect` | `"09"` | Pagos electrónicos |
| `GiftCard` | `"10"` | Gift card |
| `NotaAbono` | `"11"` | Nota de abono |
| `OtraFormaPago` | `"12"` | Otra forma de pago |
| `ContoPrepago` | `"13"` | Cuenta prepago |
| `AplicaARete` | `"14"` | Aplica a retención |
| `NoAplica` | `"99"` | No aplica |

Lista permitida: `AllowedPaymentTypes`.

---

## Tipos de Impuesto

| Constante | Código | Nombre | Constante de valor | Valor usado en la validación |
|---|---|---|---|---|
| `TaxIVA` | `"20"` | IVA | `TaxIvaAmount` | `0.13` |
| `TaxIVAExport` | `"C3"` | IVA Exportación | `TaxIVAExportAmount` | `0.0` |
| `TaxTourism` | `"59"` | Turismo | `TaxTourismAmount` | `0.05` |
| `TaxTourismAirport` | `"71"` | Turismo Aeropuerto | `TaxTourismAirportAmount` | `7.0` (monto fijo) |
| `TaxFOVIAL` | `"D1"` | FOVIAL | `TaxFOVIALAmount` | `0.20` |
| `TaxCOTRANS` | `"C8"` | COTRANS | `TaxCOTRANSAmount` | `0.10` (monto fijo) |
| `TaxSpecialOther` | `"D5"` | Especial/Otro | - | Sin validación |

Listas permitidas: `AllowedTaxTypes` y `MapAllowedTaxTypes`. Las fórmulas con que se usan estos valores se describen en [calculations.md](../../api/calculations.md).

---

## Tipos de Contingencia

| Código | Constante | Descripción |
|---|---|---|
| `1` | `NoDisponibilidadMH` | Ministerio de Hacienda no disponible |
| `2` | `FallaConexionSistema` | Falla de conexión del sistema |
| `3` | `FallaServicioInternet` | Falla del servicio de Internet |
| `4` | `FallaEnergiaElectrica` | Falla de energía eléctrica |
| `5` | `OtroMotivo` | Otro motivo |

`ContingencyReasons` asocia a cada tipo el texto de motivo por defecto y `GetContingencyReason` lo devuelve (el del tipo `5` si el tipo no existe). Lista permitida: `AllowedContingencyTypes`.

---

## Tipos de Establecimiento

| Constante | Código | Descripción |
|---|---|---|
| `Sucursal` | `"01"` | Sucursal |
| `CasaMatriz` | `"02"` | Casa Matriz |
| `DepositoBodega` | `"04"` | Depósito o Bodega |
| `PredioOPatio` | `"07"` | Predio o Patio |
| `Otro` | `"20"` | Otro tipo |

Lista permitida: `AllowedEstablishmentTypes`.

---

## Tipos de Invalidación

| Código | Nombre | ReplacementCode |
|---|---|---|
| `1` | Reemplazo | Requerido |
| `2` | Anulación | Debe ser null |
| `3` | Otro motivo | Requerido |

El plazo y la fecha del evento de invalidación se describen en [invalidation.md](dte/invalidation.md).

---

## Tipos de Ítem, Documento Asociado y Domicilio Fiscal

| Constantes | Valores | Descripción |
|---|---|---|
| `Producto`, `Servicio`, `ProductoYServicio`, `Impuesto` | `1`, `2`, `3`, `4` | Tipo de ítem (`AllowedItemTypes`) |
| `DocumentoEmisor`, `DocumentoReceptor`, `DocumentoMedico`, `DocumentoTransporte` | `1`, `2`, `3`, `4` | Código de documento asociado (`AllowedAssociatedDocumentCodes`) |
| `Domiciled`, `NonDomiciled` | `1`, `2` | Domicilio fiscal (`ValidFiscalDomiciles`), usado en la venta por cuenta de terceros |

---

## Códigos de Retención

| Código | Descripción | Factor |
|---|---|---|
| `22` | Retención IVA 1% | 0.01 |
| `C4` | Retención IVA 13% | 0.13 |
| `C9` | Retención IVA otros | Sin factor fijo (no figura en `GetRetentionAmount`) |

`AllowedRetentionCodes` incluye los tres códigos; `ListRetentionCodes` solo `22` y `C4`.

---

## Estados de Reserva de Número

| Constante | Valor | Descripción |
|---|---|---|
| `ReservationStatusReserved` | `"RESERVED"` | Número reservado |
| `ReservationStatusConfirmed` | `"CONFIRMED"` | Número confirmado (documento transmitido) |
| `ReservationStatusReleased` | `"RELEASED"` | Número liberado (documento rechazado) |

Estas constantes se definen en `dte_documents/reserved_sequence_model.go`, no en `constants`.

---

## Formato de Número de Control

```
DTE-{TipoDTE}-{CódigoEstablecimiento}{CódigoPOS}-{Secuencia de 15 dígitos}
```

Cuando se configura el año en el número de control, la secuencia es `{Año}{secuencia de 11 dígitos}`; ver [sequential-numbers.md](sequential-numbers.md).

**Ejemplo:**
```
DTE-03-00010001-000000000000001
│   │   │       │
│   │   │       └── Secuencia 1 (15 dígitos)
│   │   └── Establecimiento 0001, POS 0001
│   └── CCF (tipo 03)
└── Prefijo fijo
```

El value object `ControlNumber` valida el patrón `^DTE-[0-9]{2}-[A-Z0-9]{8}-[0-9]{15}$`.

---

## DTEs Válidos por Contexto

### Para Retención (`ValidRetentionDTETypes`)

| Código | Tipo |
|---|---|
| `01` | Factura |
| `03` | CCF |
| `14` | Factura de Sujeto Excluido |

### Para Documentos Relacionados del CCF (`ValidCCFDTETypesRelateDoc`)

| Código | Tipo |
|---|---|
| `04` | Nota de Remisión |
| `08` | Comprobante de Liquidación |
| `09` | Doc. Contable Liquidación |

### Para Documentos Relacionados de Factura (`ValidInvoiceDTETypesRelateDoc`)

| Código | Tipo |
|---|---|
| `04` | Nota de Remisión |
| `08` | Comprobante de Liquidación |
| `09` | Doc. Contable Liquidación |

### Para Documentos Relacionados de la Nota de Remisión (`ValidRemissionNoteDTETypesRelateDoc`)

| Código | Tipo |
|---|---|
| `01` | Factura |
| `03` | CCF |
| `11` | Factura Exportación |

### Para Contingencia (`ValidDTETypesForContingency`)

`01`, `03`, `04`, `05`, `06`, `11` y `14`.

### Ajustables con notas (`ValidAdjustmentDTETypes`)

`03` (CCF) y `07` (Comprobante de Retención): no se pueden invalidar mientras tengan notas de crédito o débito vigentes.

---

## Otros

- **Estados del circuit breaker (`State`)**: `StateClosed`, `StateOpen` y `StateHalfOpen`, definidos en `ciurcuit_states.go`.
- **Mensajes de error**: `GetErrorMessage(code, params...)` traduce `validation_errors.<code>` según el idioma configurado y antepone `[code]` en modo debug.
