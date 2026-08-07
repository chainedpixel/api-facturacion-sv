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

---

## Estados de Documento

| Constante | Valor | Descripción |
|---|---|---|
| `DocumentReceived` | `"RECEIVED"` | Recibido por Hacienda exitosamente |
| `DocumentRejected` | `"REJECTED"` | Rechazado por Hacienda |
| `DocumentInvalid` | `"INVALIDATED"` | Invalidado después de ser recibido |
| `DocumentPending` | `"PENDING"` | Pendiente de transmisión (contingencia) |

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
                          └──retransmisión fallida──→ REJECTED
```

---

## Tipos de Transmisión

| Constante | Valor | Descripción |
|---|---|---|
| `TransmissionNormal` | `"NORMAL"` | Transmisión directa a Hacienda |
| `TransmissionContingency` | `"CONTINGENCY"` | Almacenamiento local por falla |

---

## Ambientes

| Constante | Valor | Descripción |
|---|---|---|
| `Testing` | `"00"` | Ambiente de pruebas |
| `Production` | `"01"` | Ambiente de producción |

---

## Modelo de Facturación

| Constante | Valor | Descripción |
|---|---|---|
| `ModeloFacturacionPrevio` | `1` | Facturación en tiempo real (transmisión normal) |
| `ModeloFacturacionDiferido` | `2` | Facturación diferida (contingencia) |

**Regla:** El modelo debe ser consistente con el tipo de transmisión.

---

## Condición de Operación

| Constante | Valor | Descripción |
|---|---|---|
| `Cash` | `1` | Operación al contado |
| `Credit` | `2` | Operación a crédito |
| `Other` | `3` | Otra condición |

---

## Tipos de Documento del Receptor

| Constante | Código | Tipo |
|---|---|---|
| `NIT` | `"36"` | Número de Identificación Tributaria |
| `DUI` | `"13"` | Documento Único de Identidad |
| `CarnetResidente` | `"02"` | Carnet de Residente |
| `Pasaporte` | `"03"` | Pasaporte |
| `OtroDocumento` | `"37"` | Otro tipo de documento |

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

---

## Tipos de Impuesto

| Constante | Código | Nombre | Tasa |
|---|---|---|---|
| `TaxIVA` | `"20"` | IVA | 13% |
| `TaxIVAExport` | `"C3"` | IVA Exportación | 0% |
| `TaxTourism` | `"59"` | Turismo | 1% |
| `TaxTourismAirport` | `"71"` | Turismo Aeropuerto | Monto fijo |
| `TaxFOVIAL` | `"D1"` | FOVIAL | 0.5% |
| `TaxCOTRANS` | `"C8"` | COTRANS | Monto fijo |
| `TaxSpecialOther` | `"D5"` | Especial/Otro | Sin validación |

---

## Tipos de Contingencia

| Código | Constante | Descripción |
|---|---|---|
| `1` | `NoDisponibilidadMH` | Ministerio de Hacienda no disponible |
| `2` | `FallaConexionSistema` | Falla de conexión del sistema |
| `3` | `FallaServicioInternet` | Falla del servicio de Internet |
| `4` | `FallaEnergiaElectrica` | Falla de energía eléctrica |
| `5` | `OtroMotivo` | Otro motivo |

---

## Tipos de Establecimiento

| Valor | Descripción |
|---|---|
| `Sucursal` | Sucursal |
| `CasaMatriz` | Casa Matriz |
| `DepositoBodega` | Depósito o Bodega |
| `PredioOPatio` | Predio o Patio |
| `Otro` | Otro tipo |

---

## Tipos de Invalidación

| Código | Nombre | ReplacementCode |
|---|---|---|
| `1` | Reemplazo | Requerido |
| `2` | Anulación | Debe ser null |
| `3` | Otro motivo | Requerido |

---

## Códigos de Retención

| Código | Descripción | Factor |
|---|---|---|
| `22` | Retención IVA 1% | 0.01 |
| `C4` | Retención IVA 13% | 0.13 |
| `C9` | Retención IVA otros | Variable |

---

## Estados de Reserva de Número

| Constante | Valor | Descripción |
|---|---|---|
| `ReservationStatusReserved` | `"RESERVED"` | Número reservado |
| `ReservationStatusConfirmed` | `"CONFIRMED"` | Número confirmado (documento transmitido) |
| `ReservationStatusReleased` | `"RELEASED"` | Número liberado (documento rechazado) |

---

## Formato de Número de Control

```
DTE-{TipoDTE}-{CódigoEstablecimiento}{CódigoPOS}-{Año}{Secuencia}
```

**Ejemplo:**
```
DTE-03-00010001-2024000000001
│   │   │       │
│   │   │       └── Año 2024, secuencia 1
│   │   └── Establecimiento 0001, POS 0001
│   └── CCF (tipo 03)
└── Prefijo fijo
```

---

## DTEs Válidos por Contexto

### Para Retención

| Código | Tipo |
|---|---|
| `01` | Factura |
| `03` | CCF |
| `11` | Factura Exportación |

### Para Documentos Relacionados del CCF

| Código | Tipo |
|---|---|
| `04` | Nota de Remisión |
| `08` | Comprobante de Liquidación |
| `09` | Doc. Contable Liquidación |

### Para Documentos Relacionados de Factura

| Código | Tipo |
|---|---|
| `04` | Nota de Remisión |
| `09` | Doc. Contable Liquidación |

### Para Contingencia

Todos los DTEs excepto:
- `08` — Comprobante de Liquidación
- `09` — Doc. Contable Liquidación
