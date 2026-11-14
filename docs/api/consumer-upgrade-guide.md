# Guía de actualización para sistemas consumidores

## Alcance

Esta guía describe los cambios del contrato HTTP de la API que deben conocer los sistemas que están integrados con la versión del commit `25fbd59` (2026-11-12 19:53) y pasan a la versión actual del repositorio (commits `4598e36`, `d269375` y `4d62d39`, más los cambios aún sin commit en el árbol de trabajo).

Los cambios se obtuvieron comparando el código de `25fbd59` contra el árbol de trabajo (`git diff 25fbd59`) en las rutas del contrato (`pkg/mapper`, `internal/application`, `internal/domain`, `internal/infrastructure/api`, `assets/i18n`). Cada afirmación se respalda en ese diff. Lo que no pudo confirmarse se lista al final, en "Puntos pendientes".

Clasificación usada:

- **Rompe compatibilidad:** requests que antes funcionaban ahora fallan, o cambió un campo o forma de respuesta que el consumidor puede estar leyendo.
- **Cambio de comportamiento:** mismo contrato, resultado distinto.
- **Aditivo:** campos, códigos o capacidades nuevas que se pueden adoptar opcionalmente.

### Qué no cambió

- **Rutas, métodos y autenticación.** El diff no modifica `internal/infrastructure/api/routes`. Los endpoints siguen siendo los mismos: `POST` de cada tipo de DTE, `POST /dte/invalidation`, `GET /dte/{id}`, `GET /dte`, `GET /dte/{id}/archivo` y `GET /dte/{id}/eventos`. Los dos últimos ya existían en `25fbd59`. El único cambio en el middleware de autenticación es que dejó de escribirse el encabezado `Authorization` en el log.
- **Estructuras de request.** No hay cambios en `pkg/mapper/request_mapper/structs`: no se agregaron, quitaron ni renombraron campos de entrada. `ExtensionRequest` (extensión) sigue existiendo en el request. Lo que se eliminó es código de respuesta sin uso (`MapCommonResponseExtension` y `DTEExtension` no tienen otras referencias en el código), por lo que no hay efecto en el consumidor.
- **Respuesta de emisión.** En `25fbd59` ya se respondía el Archivo DTE (estructura firmada más `firmaElectronica` y `selloRecibido`), con respaldo a la estructura si no podía armarse. El diff solo cambia lo que ocurre antes (reintentos y errores), no la forma de la respuesta exitosa.
- **Respuesta de invalidación.** Sigue siendo el `EventArchiveResponse` (`hacienda_response` y `archivo_dte`). El diff del caso de uso solo agrega la verificación de dependencias previa.
- **Retención de renta.** `IncomeRetentionNotAllowed` ya existía en `25fbd59`; no es un cambio de esta versión.

## 1. Resumen de cambios

| # | Cambio | Tipo | Acción requerida |
|---|--------|------|------------------|
| 1 | Pagos con condición de operación crédito (`2`): código y referencia en `null`, monto `0.00`, plazo y período obligatorios | Rompe compatibilidad | Enviar `code: null`, `amount: 0`, `reference: null`, `term` y `period` en cada forma de pago a crédito |
| 2 | Plazo de invalidación validado por la API (por tipo, desde el sello) | Rompe compatibilidad | Pre-validar plazos y manejar `INVALIDATIONDEADLINEEXPIRED` |
| 3 | Invalidación de CCF/CR con NC/ND vigentes se rechaza | Rompe compatibilidad | Invalidar primero las notas |
| 4 | Invalidación exige sello de recepción | Rompe compatibilidad | No invalidar documentos sin sello; manejar `DOCUMENTWITHOUTSTAMP` |
| 5 | Código de reemplazo según tipo de documento (NC y Comprobante de Liquidación sin reemplazo) | Rompe compatibilidad | Omitir `replacement_generation_code` para `05` y `08` |
| 6 | Documentos en contingencia vencida pasan a `EXPIRED` y no se pueden invalidar | Rompe compatibilidad | Manejar el estado `EXPIRED` y `DOCUMENTEXPIRED` |
| 7 | FE (01) y CCF (03): tipos de `related_docs` restringidos en FE; si hay `related_docs`, cada ítem los referencia y cada documento debe estar referenciado por al menos un ítem; sin `related_docs`, ningún ítem lleva `related_doc` | Rompe compatibilidad | Usar tipos `04`, `08` o `09` en FE; completar `related_doc` en todos los ítems; no dejar documentos sin referenciar |
| 8 | Nota de Remisión (04): tipos de documento relacionado restringidos; el `document_number` del ítem ahora se emite y sigue la misma regla de vínculo que FE y CCF | Rompe compatibilidad | Usar solo `01`, `03` o `11`; referenciar cada documento relacionado desde el `document_number` de al menos un ítem |
| 9 | Ventana de período para documentos relacionados (`RELATEDDOCOUTOFPERIOD`) | Rompe compatibilidad | Respetar las ventanas por tipo |
| 10 | Nota de Débito: cálculo de impuestos ahora se rechaza si no cuadra | Rompe compatibilidad | Corregir el cálculo de impuestos |
| 11 | Códigos de impuesto `D1` y `C8` dejan de aceptarse | Rompe compatibilidad | No enviar esos códigos |
| 12 | FSE: ítems de tipo `4` (impuesto) rechazados | Rompe compatibilidad | No enviar ítems tipo `4` en FSE |
| 13 | `other_docs` con código `4` solo para Factura de Exportación (11) | Rompe compatibilidad | No usar código `4` en otros tipos |
| 14 | NC y ND: descuentos globales y porcentaje de descuento deben ser `0` | Rompe compatibilidad | Aplicar el descuento en cada ítem |
| 15 | Nuevo error `STAMPEDDOCUMENTNOTPERSISTED`: no re-emitir | Cambio de comportamiento (crítico) | Tratarlo como éxito en Hacienda y contactar al administrador |
| 16 | Nuevo error `ADDITIONALOPERATIONSFAILED`: no re-emitir | Cambio de comportamiento (crítico) | Igual que el anterior |
| 17 | `GET /dte?all=true` limitado al NIT del emisor autenticado | Cambio de comportamiento | Revisar si se dependía de ver otros emisores |
| 18 | Saldo disponible de NC calculado neto del descuento global del CCF | Cambio de comportamiento | Calcular NC contra el neto del CCF |
| 19 | Invalidar una ND resta de su saldo; fechas de documentos relacionados desde `fecEmi` | Cambio de comportamiento | Ninguna, informativo |
| 20 | Fecha y hora del evento de invalidación definidas por la API | Cambio de comportamiento | No enviar ni esperar fecha del evento en el request |
| 21 | Mensajes de error corregidos (`INVALIDPERCEPTIONAMOUNT`, `INVALIDTAXCALCULATION`) | Cambio de comportamiento | No depender del texto del mensaje |
| 22 | `discount` por ítem acepta montos mayores a 100 | Aditivo (más permisivo) | Opcional |
| 23 | Invalidación de NC y Comprobante de Liquidación sin reemplazo (tipos 1 y 3) | Aditivo (más permisivo) | Opcional |
| 24 | Montos de pago en `0` y código de pago vacío aceptados en el mapeo; `pagos[].codigo` puede venir `null` en el JSON del DTE | Aditivo en el request; cambio de forma en el JSON del DTE | Necesario para el cambio 1; quien lea `pagos[].codigo` del DTE debe admitir `null` |
| 25 | `processing_date` en `GET /dte/{id}` | Aditivo | Usarlo para calcular plazos de invalidación |
| 26 | Nuevos códigos de error y estado `EXPIRED` | Aditivo | Ver sección 6 |
| 27 | Total a pagar `0.00` o saldo a favor: condición `1`; en FE, CCF y FEXE además forma de pago `99` | Rompe compatibilidad | Enviar `operation_condition: 1`; en FE, CCF y FEXE `[{"code": "99", "amount": 0.00}]`. En FSE, NC y ND solo la condición `1` |
| 28 | ND: solo la venta no sujeta se limita por el saldo del CCF; gravado y exento sin tope | Cambio de comportamiento | Manejar `INVALIDDEBITNOTENOTSUBJECTAMOUNT` |
| 29 | Orden de corrección de una NC: invalidar primero y emitir después | Informativo | Ver sección 5.5 |

---

## 2. Cambios que rompen compatibilidad

### 2.1 Pagos a crédito: código y referencia en `null`, monto `0.00`, plazo y período

**Antes.** Cada forma de pago debía traer `code` y un `amount` distinto de `0` (`InvalidPaymentTypeInfo`), y la suma de los pagos debía coincidir con `total_to_pay` (`InvalidPaymentTotal`) sin importar la condición de operación. Con crédito, si ningún pago usaba `01`, se exigían `term` y `period` (`InvalidPaymentTerms`), y el código `01` no se admitía (`InvalidPaymentTypeOP2`).

**Ahora.** Con `operation_condition` crédito (`2`), en FE, CCF y FSE, **cada** forma de pago debe cumplir, antes de transmitir:

| Campo | Valor exigido | Error |
|-------|---------------|-------|
| `code` | `null` (u omitido) | `CREDITPAYMENTCODENOTALLOWED` |
| `amount` | `0.00` | `INVALIDCREDITPAYMENTAMOUNT` |
| `reference` | `null` (u omitido) | `CREDITPAYMENTREFERENCENOTALLOWED` |
| `term` | Código de plazo `01` (días), `02` (meses) o `03` (años) | `CREDITPAYMENTTERMREQUIRED` |
| `period` | Número de días, meses o años del plazo, mayor que cero | `CREDITPAYMENTPERIODREQUIRED` |

No se compara la suma contra el total. Una lista de pagos vacía con crédito y `total_to_pay` mayor que 0 se rechaza con `INVALIDPAYMENTTOTAL`. Los códigos `INVALIDPAYMENTTERMS`, `INVALIDPAYMENTTYPEOP2` e `INVALIDPAYMENTTYPEINFO` dejan de existir.

```json
"payment_types": [
  { "code": null, "amount": 0.00, "reference": null, "term": "01", "period": 30 }
]
```

En el JSON del DTE la forma de pago queda `{"codigo": null, "montoPago": 0.00, "referencia": null, "plazo": "01", "periodo": 30}`.

**Con condición `1` (contado) o `3` (otro) no cambia nada:** `code` es obligatorio en cada forma de pago (si falta: `PAYMENTCODEREQUIRED`) y la suma de `amount` debe coincidir con `total_to_pay` (`INVALIDPAYMENTTOTAL`).

**Errores.** HTTP `400`, código en `error.code` o dentro de `MANY_ERRORS` (`error.details`). La regla se aplica en el validador común, que omite NC (`05`), ND (`06`) y Nota de Remisión (`04`).

**Cambio en el consumidor.** En ventas a crédito dejar de enviar el código de forma de pago y la referencia, enviar `amount: 0` y completar siempre `term` y `period`.

### 2.2 Plazo de invalidación validado por la API

**Antes.** Regla única por tipo: FE y FEXE hasta 90 días; resto, 1 día, comparando la fecha del evento contra la fecha de emisión (`INVALIDDATEFORFEFX`, `INVALIDDATEFORALLDTE`).

**Ahora.** El plazo se cuenta desde el sello de recepción (`processing_date`) y varía por tipo (ver sección 5.2). Los códigos anteriores desaparecen. Errores nuevos, normalmente dentro de `MANY_ERRORS`:

| Código | Cuándo |
|--------|--------|
| `INVALIDATIONDEADLINEEXPIRED` | El plazo venció. El mensaje incluye el tipo y la fecha límite. El documento conserva su validez |
| `MISSINGSTAMPDATE` | No se conoce la fecha del sello del documento; no se puede calcular el plazo |
| `INVALIDATIONEVENTDATEOUTOFRANGE` | (FE, FEXE, FSE) La fecha del evento supera el período permitido desde la generación |
| `INVALIDATIONEVENTDATEMISMATCH` | (resto de tipos) La fecha del evento no es la fecha de generación del documento |

**Cambio en el consumidor.** Pre-validar el plazo antes de invocar la API (sección 5) y manejar estos códigos. Si el plazo venció, emitir una Nota de Crédito o Débito cuando corresponda (así lo indica el mensaje de la API).

### 2.3 CCF o CR con notas de crédito o débito vigentes

**Antes.** Se podía solicitar la invalidación de un CCF o CR aunque tuviera NC/ND asociadas.

**Ahora.** Si el documento es CCF (`03`) o CR (`07`) y tiene NC/ND en estado `RECEIVED` o `PENDING` que lo ajustan, la invalidación se rechaza. La API no invalida las notas automáticamente.

**Error.** HTTP `400`, `error.code` `PENDINGADJUSTMENTNOTES`. El mensaje lista los códigos de generación de las notas pendientes separados por coma: "El documento %s tiene Notas de Crédito o Débito vigentes que deben invalidarse primero: %s. Invalide cada nota y luego vuelva a solicitar la invalidación del documento".

**Cambio en el consumidor.** Invalidar cada nota listada y después reintentar con el CCF o CR. Las notas ya invalidadas no vuelven a listarse.

### 2.4 Invalidación solo de documentos con sello de recepción

**Antes.** No había verificación explícita; un documento sin sello producía un error no controlado al mapear.

**Ahora.** Un documento sin sello de recepción (por ejemplo, uno en contingencia sin transmitir) se rechaza con `DOCUMENTWITHOUTSTAMP`: "El documento %s no tiene sello de recepción, solo se pueden invalidar documentos recibidos por Hacienda". HTTP `400`.

Es un error de servicio: el código se serializa en mayúsculas (ver "Formato de errores" en la sección 6).

### 2.5 Código de reemplazo según el tipo del documento invalidado

**Antes.** Para tipos de anulación `1` y `3`, `replacement_generation_code` era siempre obligatorio.

**Ahora.** El código de reemplazo depende del tipo del documento invalidado:

| Tipo de anulación | Documento invalidado | `replacement_generation_code` |
|-------------------|----------------------|-------------------------------|
| `1` o `3` | Cualquiera excepto `05` y `08` | Obligatorio |
| `1` o `3` | Nota de Crédito (`05`) o Comprobante de Liquidación (`08`) | Debe omitirse o ser `null` |
| `2` | Cualquiera | Siempre `null` |

Errores (en las reglas del evento, normalmente dentro de `MANY_ERRORS`): `INVALIDREPLACEMENTCODE1AND3` (falta el reemplazo), `REPLACEMENTCODENOTALLOWEDFORTYPE` (se envió reemplazo para `05` u `08`). Para tipo `2` con reemplazo, el mapeo del request responde `INVALIDINVALIDATIONTYPE2`. Antes, la falta de reemplazo se devolvía como error de servicio con código propio; ahora el código puede venir dentro de `MANY_ERRORS`, por lo que conviene buscarlo también en `error.details`.

**Cambio en el consumidor.** Un cliente que enviaba siempre un reemplazo para tipos `1` y `3` fallará al invalidar NC o Comprobante de Liquidación. Omitir el campo en esos casos.

### 2.6 Estado `EXPIRED` por contingencia vencida

**Antes.** Los documentos de contingencia permanecían `PENDING` mientras se reintentaba la transmisión.

**Ahora.** El proceso de retransmisión marca como `EXPIRED` los documentos de contingencia que no se transmitieron dentro de **72 horas** desde la fecha de sello del evento de contingencia (los documentos cuyo evento aún no tiene sello nunca vencen). Un documento `EXPIRED` se considera no emitido. Al intentar invalidarlo, la API responde `DOCUMENTEXPIRED` (HTTP `400`): "El %s venció en contingencia sin obtener sello de recepción, se considera no emitido y no se puede invalidar".

El consumidor ve el cambio en `GET /dte/{id}` y `GET /dte` como `status: "EXPIRED"`. `EXPIRED` es un estado válido del catálogo de estados.

**Cambio en el consumidor.** Tratar `EXPIRED` como estado terminal: el documento no tiene sello ni validez, y si todavía se necesita la operación, emitir un documento nuevo. Quien filtraba por estado y asumía solo `RECEIVED`, `REJECTED`, `PENDING` e `INVALIDATED` debe agregar `EXPIRED`.

Además, cuando la contingencia supera 3 días, el servicio envía al administrador una alerta que indica que se requiere un informe técnico; no tiene efecto en la API.

### 2.7 Documentos relacionados en FE (01) y CCF (03)

**Antes.** En FE y CCF, si se enviaban `related_docs`, **cada ítem** debía traer `related_doc` (`MissingItemRelatedDoc`) y este debía existir entre los `related_docs` (`InvalidItemRelatedDoc`). No se verificaba que cada documento relacionado estuviera referenciado por algún ítem, ni se revisaban las referencias de los ítems cuando no había `related_docs`. En FE no se verificaba el tipo del documento relacionado.

**Ahora.**

- Tipos válidos para FE: `04` (Nota de Remisión), `08` (Comprobante de Liquidación) y `09` (Documento Contable de Liquidación). Otro tipo: `INVALIDRELATEDDOCDTETYPE`. Máximo 50 documentos (`EXCEEDEDRELATEDDOCSLIMIT`).
- Si hay `related_docs`, **cada ítem** debe traer `related_doc` (`MISSINGITEMRELATEDDOC`, igual que antes) y debe existir entre los `related_docs`; la comparación es exacta (`INVALIDITEMRELATEDDOC`).
- Nuevo: cada documento de `related_docs` debe aparecer en el `related_doc` de al menos un ítem (`UNREFERENCEDRELATEDDOC`).
- Nuevo: si no se envían `related_docs`, ningún ítem puede llevar `related_doc` (`INVALIDITEMRELATEDDOC`).

Hacienda rechaza con `HACIENDA_003` un ítem con `numeroDocumento` en `null` cuando el documento trae documentos relacionados (verificado en el ambiente de pruebas); por eso la API exige la referencia en todos los ítems y lo rechaza antes de transmitir. Un artículo que no proviene del documento relacionado debe facturarse en un documento aparte.

Ejemplo: FE que factura una NR.

```json
"related_docs": [
  { "document_type": "04", "generation_type": 2, "document_number": "CODIGO-DE-GENERACION-DE-LA-NR", "emission_date": "2026-11-13" }
],
"items": [
  { "number": 1, "description": "Mercaderia remitida", "related_doc": "CODIGO-DE-GENERACION-DE-LA-NR" },
  { "number": 2, "description": "Otra mercaderia remitida", "related_doc": "CODIGO-DE-GENERACION-DE-LA-NR" }
]
```

**Cambio en el consumidor.** Seguir completando `related_doc` en todos los ítems cuando hay `related_docs`, revisar que ningún documento de la lista quede sin referenciar y no enviar `related_doc` cuando no hay `related_docs`.

### 2.8 Nota de Remisión (04) con documentos relacionados

**Antes.** No había restricción de tipo de documento relacionado en la nota de remisión.

**Ahora.** Los documentos relacionados de una Nota de Remisión solo pueden ser `01` (FE), `03` (CCF) o `11` (FEXE). Error: `INVALIDRELATEDDOCDTETYPE` (HTTP `400`), con la lista de tipos válidos en el mensaje.

Además, el `document_number` de cada ítem (antes aceptado pero descartado, `numeroDocumento` salía siempre `null`) ahora se emite y sigue la misma regla que FE y CCF:

- Si hay `related_docs`, cada ítem debe traer `document_number` (`MISSINGITEMRELATEDDOC`) que exista entre los `related_docs` (`INVALIDITEMRELATEDDOC`).
- Cada documento de `related_docs` debe aparecer en el `document_number` de al menos un ítem (`UNREFERENCEDRELATEDDOC`).
- Sin `related_docs`, ningún ítem puede llevar `document_number` (`INVALIDITEMRELATEDDOC`).

**Cambio en el consumidor.** Una NR con `related_docs` cuyos ítems no informan `document_number` ahora se rechaza: completar `document_number` en todos los ítems. Quien enviaba `document_number` en los ítems sin `related_docs` debe dejar de enviarlo.

### 2.9 Ventana de período de documentos relacionados

**Antes.** No se validaba la antigüedad del documento relacionado.

**Ahora** se valida que la fecha de generación del documento nuevo esté dentro de la ventana permitida respecto al documento relacionado (comparando días, no horas):

| Documento relacionado | Ventana |
|-----------------------|---------|
| Nota de Remisión (`04`) | Hasta el último día de su mes, más 3 días |
| Documento Contable de Liquidación (`09`) | Hasta el último día de su mes |
| CCF o CR (`03`, `07`) relacionado desde una NC (`05`) o ND (`06`) | 3 meses desde su generación (si el día no existe en el mes destino, se usa el último día de ese mes) |

Si el documento relacionado no tiene fecha conocida, no se valida. Error: `RELATEDDOCOUTOFPERIOD` (HTTP `400`), con el tipo, la fecha del documento relacionado y la fecha límite en el mensaje.

Para NC y ND, la fecha del documento relacionado se toma ahora de `fecEmi` y `horEmi` del documento guardado, y no de la fecha de registro. En el servicio de NC, el rechazo de un documento relacionado que no fue recibido pasó a la clave `RELATEDDOCUMENTNOTRECEIVED`.

Es un error de servicio, por lo que el código se serializa en mayúsculas (ver "Formato de errores" en la sección 6).

### 2.10 Nota de Débito: cálculo de impuestos se hace cumplir

**Antes.** `DebitNoteTaxStrategy.validateIVA` detectaba la diferencia, la registraba en el log y **no devolvía error**.

**Ahora.** Si el impuesto declarado difiere del esperado, devuelve `INVALIDTAXCALCULATION` (HTTP `400`). El mensaje muestra "Esperado" y "Actual" en el orden correcto (en CCF y NC los argumentos estaban invertidos y se corrigieron).

**Cambio en el consumidor.** Verificar que el IVA y los demás tributos de la ND se calculen con la misma regla que ya aplicaba la API para CCF y NC.

### 2.11 Códigos de impuesto `D1` y `C8`

**Antes.** `D1` (FOVIAL) y `C8` (COTRANS) estaban en el catálogo aceptado, con montos fijos calculados por la API (0.20 sobre base gravada y 0.10).

**Ahora.** Ambos se rechazan al mapear el request (`UnsupportedTaxCode`), porque son impuestos por unidad de volumen que el sistema no puede calcular. Se removieron de las listas válidas y de las verificaciones de cálculo en FE, CCF, NC y ND.

**Error.** HTTP `400`, código `UNSUPPORTEDTAXCODE`: "Código de impuesto no soportado %s, debe estar en el catálogo de impuestos permitidos". Aplica al código en `tax_code`, en `taxes` de ítem y en el resumen.

**Cambio en el consumidor.** No enviar `D1` ni `C8`. Los impuestos aceptados son IVA (`20`), IVA exportación, turismo (hospedaje y salida aérea) y `D5` (otros).

### 2.12 FSE: ítems de tipo `4`

**Antes.** Un ítem tipo `4` (impuesto) se aceptaba en la Factura de Sujeto Excluido.

**Ahora.** Se rechaza con `INVALIDFSEITEMTYPE` (HTTP `400`): "El tipo de ítem %d no está permitido en la Factura de Sujeto Excluido, los ítems de tipo impuesto (4) no son soportados".

### 2.13 `other_docs` con código `4`

**Antes.** Los códigos de documento asociado `1` a `4` se aceptaban en cualquier tipo de DTE.

**Ahora.** El código `4` (documento de transporte) solo se acepta en Factura de Exportación (`11`). En otros tipos: `INVALIDASSOCIATEDDOCUMENTCODEFORTYPE` (HTTP `400`). Los códigos fuera de `1` a `4` siguen dando `INVALIDASSOCIATEDDOCUMENTCODE`.

### 2.14 NC y ND: sin descuentos globales

**Antes.** Los campos de descuento global se aceptaban en el resumen de NC y ND.

**Ahora.** `non_subject_discount`, `exempt_discount`, `taxed_discount` y `discount_percentage` deben ser `0`. Error: `GLOBALDISCOUNTNOTALLOWEDINADJUSTMENT` (HTTP `400`), con el nombre del campo en el mensaje: "Aplique el descuento en cada item". Los descuentos por ítem siguen permitidos.

---

## 3. Cambios de comportamiento

### 3.1 Documento sellado por Hacienda que no se pudo guardar (`STAMPEDDOCUMENTNOTPERSISTED`)

**Antes.** Si el guardado local fallaba tras la aceptación de Hacienda, el error del repositorio se devolvía tal cual. Si fallaba la confirmación de la reserva del número de control, la emisión devolvía error aunque Hacienda ya hubiera sellado el documento.

**Ahora.**

1. La confirmación de la reserva y el guardado del documento se reintentan una vez (tras 200 ms). Si la confirmación sigue fallando, se continúa con el guardado.
2. Si el guardado falla tras el reintento, la API responde HTTP `400` con `error.code` `STAMPEDDOCUMENTNOTPERSISTED` y el mensaje: "El documento con código de generación %s fue sellado por Hacienda (sello de recepción %s) pero no pudo guardarse localmente. NO debe emitirse nuevamente. Contacte al administrador para recuperarlo".
3. Se envía una alerta al administrador, y el documento firmado queda en el log de la aplicación para su recuperación manual. El mensaje contiene el código de generación y el sello de recepción.
4. Además, el servicio protege el número de control de documentos con sello: una reserva vencida cuyo número ya pertenece a un DTE guardado no se libera para reutilizarse.

**Cambio en el consumidor.** Ante este código: **no re-emitir**. El documento ya existe en Hacienda con ese código de generación y sello. Registrar ambos datos, no reintentar la emisión, y contactar al administrador del microservicio. Un reintento generaría un documento duplicado con otro número de control.

### 3.2 Fallo en operaciones posteriores a la emisión (`ADDITIONALOPERATIONSFAILED`)

Para NC y ND, tras el guardado se ejecutan operaciones adicionales (control de saldo). Si fallan por un error inesperado (no un error de validación o de negocio ya clasificado), la API responde HTTP `400`, código `ADDITIONALOPERATIONSFAILED`: "El documento con código de generación %s fue sellado por Hacienda y guardado, pero fallaron las operaciones posteriores. NO debe emitirse nuevamente. Contacte al administrador". Antes, ese error inesperado quedaba sin clasificar y podía tratarse como un fallo de transmisión (contingencia).

**Cambio en el consumidor.** Igual que el punto anterior: no re-emitir y contactar al administrador. Aquí el documento sí fue guardado y puede consultarse con `GET /dte/{id}`.

En este caso el documento está sellado y guardado, pero la transacción de control de saldo de la NC o ND no quedó registrada. El saldo del CCF relacionado debe ser conciliado por el administrador; el consumidor no debe re-emitir.

### 3.3 `GET /dte?all=true` limitado al emisor

**Antes.** `all=true` quitaba el filtro de sucursal sin ningún otro filtro de emisor.

**Ahora.** Con `all=true` y un token que incluye NIT, se listan solo los DTEs de las sucursales cuyo emisor (usuario con ese NIT) coincide con el del token. Si el token no incluye NIT, `all` se ignora y se lista solo la sucursal autenticada. Sin `all`, el filtro es por sucursal como antes.

**Cambio en el consumidor.** Revisar integraciones que usaban `all=true` esperando ver documentos de otros emisores; ya no aparecen. El diff no agrega verificación de permisos adicional para este parámetro.

### 3.4 Saldo de NC neto del descuento global del CCF

Al registrar un CCF, el saldo disponible para notas de crédito se crea con el total gravado, exento y no sujeto **menos** el descuento global correspondiente (`descuGravada`, `descuExenta`, `descuNoSuj`), con redondeo a centavos y sin valores negativos. Antes se usaba el total sin descuento.

**Efecto.** Una NC que supere el neto del CCF se rechaza con `INVALIDCREDITNOTETRANSACTION` (el mensaje indica el saldo restante).

Los CCF emitidos antes de la actualización conservan su saldo original calculado sin descuento (valores brutos): no hay migración ni recálculo.

### 3.5 Saldo del CCF en notas de débito

Las ventas gravadas y exentas de una ND no se comparan contra el saldo del CCF (tampoco se comparaban antes). Lo nuevo es el límite de la venta no sujeta: el `total_non_subject` de la ND, sumado al de las ND vigentes sobre el mismo CCF, no puede superar el total no sujeto del CCF menos su descuento global no sujeto. Si lo supera: `INVALIDDEBITNOTENOTSUBJECTAMOUNT` (HTTP `400`). Las ND invalidadas dejan de contar. La NC mantiene el tope en las tres categorías.

### 3.5.1 Invalidación de una ND

Al invalidar una Nota de Débito, el control de saldo registra una transacción `DEBIT_NOTE_INVALID` que **resta** del saldo lo que la ND había sumado. Es interno; no cambia el request ni la respuesta, pero afecta el saldo disponible del CCF relacionado.

### 3.6 Fecha y hora del evento de invalidación

La API fija `fecEmi` y `horEmi` del evento: la fecha y hora actual para FE, FEXE y FSE; para los demás tipos, la fecha y hora de generación del documento (o la actual si se generó hoy). El request no tiene campo para esa fecha. El consumidor no debe enviarla ni asumir que equivale a la fecha de la solicitud en documentos distintos de FE, FEXE y FSE.

### 3.7 Mensajes de error

- `INVALIDPERCEPTIONAMOUNT` ahora muestra los montos con dos decimales y su mensaje indica "esperado (1% del total gravado)".
- `INVALIDTAXCALCULATION` en CCF y NC: se corrigió el orden de "Esperado" y "Actual".
- Los argumentos de los mensajes de error de servicio ahora se pasan correctamente a la traducción (antes se pasaban como un solo valor). Mensajes con varios parámetros ahora se muestran completos.

Los consumidores deben decidir por `error.code`, no por el texto de `error.message`.

---

## 4. Cambios aditivos (opcionales)

| Cambio | Detalle |
|--------|---------|
| `processing_date` en `GET /dte/{id}` | Campo nuevo en la respuesta (`omitempty`: se omite si no está guardado). Contiene la fecha de procesamiento de Hacienda con el mismo formato con que se guarda (ejemplo en la documentación: `15/01/2026 14:30:05`). Es la fecha desde la que se cuentan los plazos de invalidación |
| `discount` por ítem sin tope de 100 | El descuento por ítem ahora usa el tipo de monto: acepta valores entre `0` y `99999999999.99`. Antes se rechazaba con `INVALIDDISCOUNT` si era mayor que `100`. Un monto negativo o fuera de rango se rechaza con `INVALIDAMOUNT` |
| Invalidación de NC (`05`) y CLE (`08`) sin reemplazo | Con tipos `1` y `3` ya no se exige reemplazo para estos documentos (ver 2.5) |
| Pagos con `amount` igual a `0` y sin `code` | El mapeo del request ya no rechaza `amount: 0` ni un `code` vacío o `null`; las reglas de la condición de operación deciden (ver 2.1) |
| Estado `EXPIRED` | Valor nuevo en `status` de documentos de contingencia vencida (ver 2.6) |
| Nuevos códigos de error | Ver sección 6 |

Rutas, requests y la forma de la respuesta de emisión no tienen campos nuevos. En el JSON del DTE, `pagos[].codigo` puede venir `null` en ventas a crédito.

---

## 5. Flujo de invalidación recomendado para el consumidor

La API valida todas estas reglas. El consumidor puede pre-validarlas para evitar solicitudes que se sabe que fallarán.

### 5.1 Pasos

1. **Consultar el documento.** `GET /api/v1/dte/{id}` y verificar:
   - `status` es `RECEIVED`. Si es `INVALIDATED`, `REJECTED`, `PENDING` o `EXPIRED`, la invalidación se rechaza.
   - Hay `reception_stamp` (sin sello: `DOCUMENTWITHOUTSTAMP`).
   - Se conoce `processing_date` (la fecha del sello para el cálculo del plazo; sin ella: `MISSINGSTAMPDATE`).
2. **Calcular el plazo** (sección 5.2) con `processing_date` y comparar con la hora actual del emisor (zona horaria configurada en el servicio). Si venció, no invalidar: emitir NC o ND si corresponde.
3. **Si es CCF (`03`) o CR (`07`)**: invalidar primero cada NC/ND vigente (estado `RECEIVED` o `PENDING`) que lo ajusta. Si se omite, `PENDINGADJUSTMENTNOTES` lista las notas pendientes. Cada nota se invalida con el mismo flujo (sus propios plazos).
4. **Determinar el reemplazo** (sección 5.4) y emitir primero el documento de reemplazo si el tipo lo requiere.
5. **Enviar** `POST /api/v1/dte/invalidation` con `generation_code`, `reason` y, si corresponde, `replacement_generation_code`. La API fija la fecha y hora del evento (sección 3.6).
6. **Leer la respuesta.** `data` es el evento con `hacienda_response` y `archivo_dte`. Guardar `archivo_dte` y el sello del evento.
7. **Manejar errores** según la sección 6. Un error de plazo no debe reintentarse.

Las verificaciones se ejecutan en este orden: request y tipo de anulación; estado del documento y del reemplazo; dependencias y sello; datos del evento, plazo y fecha del evento; transmisión a Hacienda.

### 5.2 Plazos de transmisión del evento de invalidación

Se cuentan desde la fecha del sello (`processing_date`). Si `processing_date` no está guardada o no se puede interpretar, el servicio usa la fecha de registro del documento (`created_at`), no la fecha de la solicitud.

| Tipo de DTE | Último momento para transmitir el evento |
|-------------|------------------------------------------|
| FE (`01`), FEXE (`11`), FSE (`14`) | 23:59:59 del día resultante de sumar 3 meses a la fecha del sello. Si el día no existe en el mes destino, se usa el último día de ese mes |
| FE y FEXE de emisores con actividad económica `21001`, `21008`, `46482`, `46484`, `46491` o `47721` (farmacéuticas) | 23:59:59 del día resultante de sumar **2 años** (24 meses) a la fecha del sello. La excepción no aplica a FSE |
| Resto de tipos (`03`, `04`, `05`, `06`, `07`, `08`, `09`, `15`) | 23:59:59 del **décimo día hábil** del mes siguiente al mes del sello |

El código de actividad se toma de `codActividad` del emisor en el JSON del documento guardado.

**Días hábiles.** Lunes a viernes, excluyendo:

- Asuetos fijos: 1 de enero, 1 de mayo, 10 de mayo, 17 de junio, 6 de agosto, 15 de septiembre, 2 de noviembre y 25 de diciembre.
- Semana Santa: de Lunes Santo (6 días antes del Domingo de Pascua) a Lunes de Pascua (1 día después).
- Los asuetos adicionales configurados por el administrador en la variable `MH_EXTRA_HOLIDAYS` (lista separada por comas, formato `YYYY-MM-DD`). El consumidor no puede consultar ese valor por la API; si el emisor lo usa, solicitarlo al administrador.

Se pueden invalidar todos los tipos de DTE soportados: la API solo responde `INVALIDDTETYPEFORINVALIDATION` cuando el tipo del documento no es un tipo de DTE conocido. El texto de ese mensaje menciona únicamente `01`, `03` y `04`, pero no refleja la regla real.

### 5.3 Fecha del evento

| Tipo | Fecha del evento (la fija la API) | Validación |
|------|----------------------------------|------------|
| FE, FEXE, FSE | Fecha y hora actual | No puede superar la fecha de generación más el período (3 meses, o 2 años en farmacéuticas), con ajuste a fin de mes (`INVALIDATIONEVENTDATEOUTOFRANGE`). Este límite se suma al plazo desde el sello: una FE generada el 2025-10-09 y sellada el 2025-11-25 solo puede invalidarse hasta el 2026-01-09 |
| Resto | Fecha y hora de generación del documento; la actual si se generó hoy | Debe ser el mismo día de la generación (`INVALIDATIONEVENTDATEMISMATCH`) |

### 5.4 Código de reemplazo por tipo

- Tipo de anulación `2`: sin reemplazo siempre (`INVALIDINVALIDATIONTYPE2` si se envía).
- Tipos `1` y `3`: reemplazo obligatorio, salvo que el documento invalidado sea NC (`05`) o Comprobante de Liquidación (`08`), en cuyo caso no debe enviarse.
- Tipo `3`: `reason.reason_field` obligatorio.

### 5.5 Orden de emisión e invalidación

- FE (`01`), CCF (`03`), NR (`04`), ND (`06`) y CR (`07`) con tipo `1` o `3`: primero se emite el documento nuevo y se espera su sello; después se invalida el viejo con `replacement_generation_code` igual al código de generación del nuevo. La API rechaza la invalidación si el documento de reemplazo no existe o no tiene un estado válido.
- NC (`05`) con tipo `1` o `3`: primero se invalida la NC incorrecta con `replacement_generation_code` en `null`; después se emite la NC correcta. La invalidación restituye el saldo del CCF, y la NC correcta se valida contra ese saldo.
- Tipo `2`: siempre sin reemplazo.

---

## 6. Manejo de errores nuevos

### Formato de errores

Toda respuesta de error tiene esta forma:

```json
{
  "success": false,
  "error": {
    "message": "string",
    "details": ["string"],
    "code": "string"
  }
}
```

`details` es un arreglo de **cadenas de texto**, no de objetos. Según el origen del error:

| Origen | HTTP | `error.code` | `error.details` |
|--------|------|--------------|-----------------|
| Regla de DTE (validación), una sola | 400 | Código de la regla en mayúsculas (por ejemplo `INVALIDCREDITPAYMENTAMOUNT`) | Detalle de la regla |
| Reglas de DTE agrupadas | 400 | `MANY_ERRORS`; mensaje "No se pudo crear el DTE, revise los detalles a continuación" | Una cadena por regla con el formato `[Clave] mensaje traducido` |
| Validación de valores o mapeo (por ejemplo `UNSUPPORTEDTAXCODE`, `INVALIDAMOUNT`) | 400 | Tipo en mayúsculas | `["No hay más detalles disponibles"]` |
| Error de servicio (por ejemplo `PENDINGADJUSTMENTNOTES`, `STAMPEDDOCUMENTNOTPERSISTED`, `DOCUMENTEXPIRED`) | 400 | Código del servicio en mayúsculas | Texto del error envuelto, separado por `;` |
| Rechazo de Hacienda | 400 | `HACIENDA_<código>` | `State: ...`, `Processed at: ...` y las observaciones de Hacienda |
| Error de transporte con Hacienda | el estado HTTP recibido | `HACIENDA_<estado>` | Aviso de contingencia activa |
| Error no clasificado | 400 (`BUSINESS_ERROR`) o 500 (`SYSTEM_ERROR`) | `BUSINESS_ERROR` o `SYSTEM_ERROR` | Sin detalles |

Ejemplo de errores agrupados (`MANY_ERRORS`):

```json
{
  "success": false,
  "error": {
    "message": "No se pudo crear el DTE, revise los detalles a continuación",
    "details": [
      "[InvalidCreditPaymentAmount] Con condición de operación crédito (2), el monto de cada forma de pago debe ser 0.00; se recibió 50.000000"
    ],
    "code": "MANY_ERRORS"
  }
}
```

Ejemplo de error de servicio único:

```json
{
  "success": false,
  "error": {
    "message": "El documento <codigo-generacion-ccf> tiene Notas de Crédito o Débito vigentes que deben invalidarse primero: <codigo-generacion-nota>. Invalide cada nota y luego vuelva a solicitar la invalidación del documento",
    "details": ["<texto del error envuelto, si existe>"],
    "code": "PENDINGADJUSTMENTNOTES"
  }
}
```

**Recomendación para el consumidor.** Comparar los códigos sin distinguir mayúsculas y buscar la clave tanto en `error.code` como dentro de `error.details[]` (patrón `[Clave]`), porque una misma regla puede aparecer sola o agrupada según la capa que la detecte. Decidir siempre por código y no por el texto de `message`.

En los códigos de la tabla siguiente, "(en `MANY_ERRORS`)" indica que normalmente llegan agrupados.

| Código | HTTP | Cuándo ocurre | Acción del consumidor |
|--------|------|---------------|-----------------------|
| `STAMPEDDOCUMENTNOTPERSISTED` | 400 | Hacienda selló el documento pero no pudo guardarse localmente tras un reintento | **No re-emitir.** Guardar código de generación y sello (vienen en el mensaje) y contactar al administrador |
| `ADDITIONALOPERATIONSFAILED` | 400 | El documento (NC o ND) fue sellado y guardado, pero fallaron las operaciones posteriores | **No re-emitir.** Contactar al administrador; consultar el documento con `GET /dte/{id}` |
| `INVALIDCREDITPAYMENTAMOUNT` | 400 | Condición de operación crédito con un pago distinto de `0.00` | Enviar `amount: 0` en cada pago |
| `CREDITPAYMENTCODENOTALLOWED` | 400 | Condición de operación crédito con `code` informado | Enviar `code: null` |
| `CREDITPAYMENTREFERENCENOTALLOWED` | 400 | Condición de operación crédito con `reference` informada | Enviar `reference: null` |
| `CREDITPAYMENTTERMREQUIRED` | 400 | Condición de operación crédito sin `term` | Enviar `term` (`01`, `02` o `03`) |
| `CREDITPAYMENTPERIODREQUIRED` | 400 | Condición de operación crédito sin `period` o con `period` menor o igual a cero | Enviar `period` mayor que cero |
| `PAYMENTCODEREQUIRED` | 400 | Condición de operación `1` o `3` con una forma de pago sin `code` | Enviar el código de forma de pago |
| `NOPAYMENTDUEREQUIRESCASHCONDITION` | 400 | Total a pagar `0.00` (FE, CCF, FSE, NC, ND) o saldo a favor (FE, CCF) con condición distinta de `1` | Enviar `operation_condition: 1` |
| `NOPAYMENTDUEREQUIRESCODE99` | 400 | FE o CCF con total a pagar `0.00` o saldo a favor, sin formas de pago o con un código distinto de `99` | Enviar una forma de pago `99` |
| `INVALIDDEBITNOTENOTSUBJECTAMOUNT` | 400 | ND cuya venta no sujeta, con la de las ND vigentes, supera el total no sujeto neto del CCF | Reducir la venta no sujeta de la ND |
| `INVALIDRELATEDDOCDTETYPE` | 400 | Tipo de documento relacionado no permitido (FE: `04`, `08`, `09`; NR: `01`, `03`, `11`) | Corregir el tipo |
| `MISSINGITEMRELATEDDOC` | 400 | FE, CCF, NR, NC o ND con `related_docs` y un ítem sin documento relacionado | Indicar el documento relacionado en cada ítem |
| `UNREFERENCEDRELATEDDOC` | 400 | FE, CCF o NR con un documento de `related_docs` que ningún ítem referencia | Referenciarlo desde al menos un ítem o quitarlo |
| `INVALIDITEMRELATEDDOC` | 400 | El `related_doc` del ítem no está entre los `related_docs`, o no se enviaron `related_docs` | Corregir la referencia o dejarla en `null` |
| `RELATEDDOCOUTOFPERIOD` | 400 | El documento relacionado está fuera de su ventana de período | Verificar fechas; si venció, usar otro mecanismo (por ejemplo, un documento nuevo) |
| `INVALIDTAXCALCULATION` | 400 | ND con impuesto declarado distinto del esperado | Corregir el cálculo |
| `UNSUPPORTEDTAXCODE` | 400 | Se envió `D1` o `C8` | No enviar esos códigos |
| `INVALIDFSEITEMTYPE` | 400 | Ítem tipo `4` en FSE | No enviar ítems de impuesto |
| `INVALIDASSOCIATEDDOCUMENTCODEFORTYPE` | 400 | `other_docs` código `4` fuera de FEXE | Usar código válido para el tipo |
| `GLOBALDISCOUNTNOTALLOWEDINADJUSTMENT` | 400 | NC o ND con descuento global o porcentaje de descuento distintos de `0` | Mover el descuento a cada ítem |
| `INVALIDAMOUNT` | 400 | `discount` de ítem negativo o mayor que `99999999999.99` | Corregir el valor |
| `PENDINGADJUSTMENTNOTES` | 400 | Invalidación de CCF o CR con NC/ND vigentes | Invalidar las notas listadas y reintentar |
| `DOCUMENTEXPIRED` | 400 | Invalidación de un documento `EXPIRED` | No se puede invalidar; no tiene validez |
| `DOCUMENTWITHOUTSTAMP` | 400 | Invalidación de un documento sin sello de recepción | Esperar la transmisión o no invalidar |
| `INVALIDATIONDEADLINEEXPIRED` | 400 (en `MANY_ERRORS`) | El plazo de invalidación venció | No reintentar; emitir NC o ND si corresponde |
| `MISSINGSTAMPDATE` | 400 (en `MANY_ERRORS`) | No se conoce la fecha del sello | Contactar al administrador |
| `INVALIDATIONEVENTDATEOUTOFRANGE` | 400 (en `MANY_ERRORS`) | Fecha del evento fuera del período (FE, FEXE, FSE) | No reintentar |
| `INVALIDATIONEVENTDATEMISMATCH` | 400 (en `MANY_ERRORS`) | Fecha del evento distinta de la generación (resto de tipos) | Contactar al administrador |
| `INVALIDREPLACEMENTCODE1AND3` | 400 (en `MANY_ERRORS`) | Falta el reemplazo en tipos `1` o `3` | Enviar el reemplazo |
| `REPLACEMENTCODENOTALLOWEDFORTYPE` | 400 (en `MANY_ERRORS`) | Reemplazo enviado para NC o Comprobante de Liquidación | Omitir el reemplazo |
| `INVALIDINVALIDATIONTYPE2` | 400 | Reemplazo enviado con tipo de anulación `2` | Omitir el reemplazo |

Los códigos marcados "en `MANY_ERRORS`" provienen del validador de eventos, que agrupa las reglas incumplidas; los demás se devuelven directamente. Aplicar siempre la recomendación de búsqueda en `error.code` y `error.details[]` descrita al inicio de esta sección.

---

## 7. Checklist de actualización

### Emisión

- [ ] En ventas a crédito (FE, CCF y FSE): `code: null`, `amount: 0`, `reference: null`, `term` y `period` mayor que cero en cada forma de pago.
- [ ] Con condición `1` o `3`: mantener `code` en cada forma de pago y la suma igual a `total_to_pay` (holgura 0.01).
- [ ] Ventas mixtas: condición `3` con varias formas de pago mayores que 0; la parte no cobrada con código `13`.
- [ ] Total a pagar `0.00` o saldo a favor: condición `1`; en FE, CCF y FEXE además forma de pago `99`. En FSE, NC y ND basta la condición `1`.
- [ ] En NR con `related_docs`: informar `document_number` en todos los ítems.
- [ ] En ND: la venta no sujeta acumulada no puede superar el total no sujeto neto del CCF.
- [ ] Si se lee `pagos[].codigo` del JSON del DTE, admitir `null`.
- [ ] En FE con `related_docs`: usar solo `04`, `08` o `09`.
- [ ] En FE y CCF: con `related_docs`, completar `related_doc` en todos los ítems y referenciar cada documento desde al menos un ítem; sin `related_docs`, ningún ítem lleva `related_doc`.
- [ ] En Notas de Remisión: relacionar solo documentos `01`, `03` o `11`.
- [ ] Verificar las ventanas de período de documentos relacionados (NR: fin de mes más 3 días; `09`: fin de mes; NC/ND sobre CCF o CR: 3 meses).
- [ ] No enviar los códigos de impuesto `D1` ni `C8`.
- [ ] No enviar ítems tipo `4` en FSE.
- [ ] No usar código `4` en `other_docs` fuera de FEXE.
- [ ] En NC y ND: dejar en `0` los descuentos globales y el porcentaje de descuento.
- [ ] Validar el cálculo de impuestos de la ND contra las mismas reglas que CCF y NC.
- [ ] Calcular las NC contra el saldo neto del descuento global del CCF.

### Manejo de respuestas y errores

- [ ] Tratar `STAMPEDDOCUMENTNOTPERSISTED` y `ADDITIONALOPERATIONSFAILED` como "no re-emitir": registrar y escalar.
- [ ] Evitar reintentos automáticos de emisión ante cualquier error 400 de servicio sin revisar el código.
- [ ] Decidir por `error.code` y buscar también dentro de `error.details` (`MANY_ERRORS`).
- [ ] Agregar `EXPIRED` al manejo de estados de DTE.
- [ ] Si se usa `GET /dte?all=true`, revisar que no se dependa de otros emisores.

### Invalidación

- [ ] Leer `processing_date` de `GET /dte/{id}` y calcular el plazo por tipo.
- [ ] Mantener el calendario de asuetos (fijos, Semana Santa y los de `MH_EXTRA_HOLIDAYS`).
- [ ] Invalidar las NC/ND vigentes antes que su CCF o CR.
- [ ] Omitir `replacement_generation_code` para NC (`05`) y Comprobante de Liquidación (`08`); enviarlo en el resto con tipos `1` y `3`.
- [ ] No enviar fecha del evento (la fija la API).
- [ ] No reintentar tras `INVALIDATIONDEADLINEEXPIRED`.
- [ ] Corregir FE, CCF, NR, ND y CR emitiendo primero el documento nuevo y luego invalidando el viejo con su código; corregir una NC invalidándola primero (reemplazo `null`) y emitiendo después la correcta.

### Plan de pruebas sugerido en el ambiente de pruebas

1. FE, CCF y FSE a crédito con `code: null`, `amount: 0`, `reference: null`, `term` y `period`: debe aceptarse. Con `code: "03"`: `CREDITPAYMENTCODENOTALLOWED`; sin `term`: `CREDITPAYMENTTERMREQUIRED`; con `amount` distinto de `0`: `INVALIDCREDITPAYMENTAMOUNT`.
2. FE con `related_docs` tipo `03`: `INVALIDRELATEDDOCDTETYPE`. FE con una NR y todos los ítems que la referencian: debe aceptarse. FE con una NR y un ítem con `related_doc: null`: `MISSINGITEMRELATEDDOC`.
3. NR relacionada con tipo `04`: `INVALIDRELATEDDOCDTETYPE`.
4. ND con impuesto mal calculado: `INVALIDTAXCALCULATION`.
5. Ítem con `tax_code` `D1` y otro con `C8`: `UNSUPPORTEDTAXCODE`.
6. FSE con ítem tipo `4`: `INVALIDFSEITEMTYPE`.
7. NC y ND con `taxed_discount` distinto de `0`: `GLOBALDISCOUNTNOTALLOWEDINADJUSTMENT`.
8. Ítem con `discount` mayor que `100` en FE o CCF: debe aceptarse.
9. Invalidar un CCF con una NC vigente: `PENDINGADJUSTMENTNOTES`; invalidar la NC (sin `replacement_generation_code`) y luego el CCF.
10. Invalidar un documento fuera de plazo (si el ambiente permite documentos con sello antiguo): `INVALIDATIONDEADLINEEXPIRED`.
11. Invalidar un documento en contingencia sin sello: `DOCUMENTWITHOUTSTAMP`; y uno `EXPIRED`: `DOCUMENTEXPIRED`.
12. `GET /dte/{id}`: verificar que llegue `processing_date`. `GET /dte?all=true`: verificar que solo aparezcan documentos del NIT autenticado.
13. FE con condición `3`, un pago `01` y otro `13` que suman el total: debe aceptarse. FE con `total_to_pay` `0.00` y condición `2`: `NOPAYMENTDUEREQUIRESCASHCONDITION`.
14. NR que relaciona un CCF con todos los ítems que lo referencian en `document_number`: debe aceptarse; con un ítem en `null`: `MISSINGITEMRELATEDDOC`.
15. ND gravada mayor que el saldo restante del CCF: debe aceptarse. ND no sujeta mayor que el total no sujeto del CCF: `INVALIDDEBITNOTENOTSUBJECTAMOUNT`.
16. NC incorrecta: invalidarla con tipo `1` y reemplazo `null`, y luego emitir la NC correcta: ambas deben aceptarse.
17. Verificar manualmente (con el administrador) el manejo de `STAMPEDDOCUMENTNOTPERSISTED` y `ADDITIONALOPERATIONSFAILED`, porque no se pueden provocar de forma controlada desde un consumidor.

---

## Puntos pendientes

- `MH_EXTRA_HOLIDAYS` es configuración del servidor; consultar al administrador.
- Esta guía se basa en la lectura del código y del diff; no se ejecutaron pruebas ni el servicio.
