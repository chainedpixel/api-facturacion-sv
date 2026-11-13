# Sistema de Errores del Dominio

> **Paquete:** `internal/domain/dte/common/dte_errors`
> **Paquete core:** `internal/domain/core/error`

## Descripción General

El sistema de errores del dominio proporciona tipos de error especializados que permiten diferenciar entre errores de validación, errores de negocio, y errores centinela. Todos implementan la interfaz `error` de Go.

---

## Tipos de Error

### DTEError — Error Principal

```go
type DTEError struct {
    ValidationErrors []error
    BusinessErrors   []*DTEError
    ErrorType        string
    Message          string
    Code             string
}
```

El error principal del dominio. Se puede crear de dos formas:

#### Constructor Simple

```go
func NewDTEErrorSimple(errorType string, params ...interface{}) *DTEError
```

Se crea un error con un tipo y parámetros opcionales. El `Message` se resuelve con `constants.GetErrorMessage` desde `validation_errors` en `assets/i18n/es.yaml`, y el `Code` es el tipo en mayúsculas (`strings.ToUpper(errorType)`).

```go
// Ejemplo
err := NewDTEErrorSimple("InvalidIVAItemCalculation", itemNumber, expected, actual)
```

#### Constructor Compuesto

```go
func NewDTEErrorComposite(businessErrors []*DTEError) *DTEError
```

Se agrupan múltiples errores de negocio en uno solo (se descartan los nulos). El `Message` une los mensajes con `; ` y el `Code` es `MANY_ERRORS`. Se usa cuando una validación genera varios errores que se deben reportar juntos.

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Implementa `error`, retorna `Message` ("Unknown DTE error" si el receptor es nulo) |
| `GetValidationErrorsString()` | `[]string` | Extrae los mensajes de `ValidationErrors` y de `BusinessErrors` |
| `GetCode()` | `string` | Retorna `Code`, o el tipo en mayúsculas si `Code` está vacío (`UNKNOWN_DTE_ERROR` si el receptor es nulo) |
| `GetMessage()` | `string` | Si hay errores de validación o de negocio retorna la traducción de `service_errors.FailedToCreateDTE`; en otro caso busca `service_errors.<ErrorType>` y, si no existe, retorna `Message` |

---

### ValidationError — Error de Validación

```go
type ValidationError struct {
    ErrorType string
    Message   string
}
```

Se usa para errores de validación individual (un campo, una regla). El mensaje se resuelve igual que en `DTEError`.

#### Constructor

```go
func NewValidationError(errorType string, params ...interface{}) *ValidationError
func NewFormattedValidationError(err error) *ValidationError // ErrorType vacío; Message es err.Error()
```

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Retorna `Message` |
| `GetType()` | `string` | Retorna `ErrorType` (`UnknownError` si el receptor es nulo) |

---

### ServiceError — Error de Servicio (`pkg/shared/shared_error`)

```go
type ServiceError struct {
    Type      string // servicio que lo origina
    Operation string // operación
    Message   string
    Code      string
    Err       error
}
```

Se crea con `NewGeneralServiceError(serviceType, op, msg, err)` (mensaje libre), `NewFormattedGeneralServiceError(serviceType, op, code, args...)` o `NewFormattedGeneralServiceWithError(serviceType, op, err, code, args...)`; estos dos últimos traducen la clave `service_errors.<code en minúsculas>`. Con `Server.Debug` activo, `Error()` antepone `[Code]` o `[Type] Operation`.

---

### CompositeError — Error Compuesto

```go
type CompositeError struct {
    Errors []error
}
```

Se agrupan múltiples errores de cualquier tipo.

#### Constructor

```go
func NewCompositeError(errors ...error) *CompositeError
```

#### Métodos

| Método | Retorno | Descripción |
|---|---|---|
| `Error()` | `string` | Todos los mensajes unidos por `;` |

---

## Errores Centinela (Core)

**Paquete:** `internal/domain/core/error/sentinels_error.go`

Errores predefinidos para condiciones específicas del negocio:

| Variable | Mensaje | Contexto |
|---|---|---|
| `ErrBranchMatrixNotFound` | `branch matrix not found` | La sucursal matriz no existe |
| `ErrAtLeastOneBranch` | `at least one branch is required` | El usuario no tiene sucursales |
| `ErrDontHaveBranchMatrix` | `don't have branch matrix, is required that the user have a branch matrix` | No hay sucursal tipo matriz |
| `ErrMoreThanOneBranchMatrix` | `more than one branch matrix, is required that the user have only one branch matrix` | Más de una sucursal matriz |
| `ErrBranchMatrixWithoutAddress` | `for the branch matrix is required that have an address associated` | La sucursal matriz no tiene dirección |

Se envuelven con `dte_errors.NewFormattedValidationError`.

---

## Catálogo de Códigos de Error

Los mensajes se definen en `assets/i18n/es.yaml` (y `en.yaml`) en dos secciones. La columna **Parámetros** indica la cantidad de marcadores de formato (`%s`, `%d`, `%f`) del mensaje; quien construye el error debe pasar exactamente esa cantidad de argumentos. La búsqueda de la clave se hace en minúsculas.

- `validation_errors`: usados por `NewDTEErrorSimple` y `NewValidationError` (a través de `constants.GetErrorMessage`, que antepone el código entre corchetes si `Server.Debug` está activo).
- `service_errors`: usados por `NewFormattedGeneralServiceError` y `NewFormattedGeneralServiceWithError` (paquete `pkg/shared/shared_error`).

Si la clave no existe en el idioma activo ni en `en`, el mensaje resultante es `[clave] <mensaje de unknownerror>`.

### validation_errors

| Código | Parámetros | Mensaje (es) |
|---|---|---|
| `RequiredField` | 1 | El campo %s es requerido |
| `InvalidLength` | 3 | El campo %s no cumple con la longitud requerida, debe ser de %s caracteres, recibido %s |
| `InvalidNumberRange` | 3 | El campo %s no cumple con el rango requerido, debe ser de %s dígitos, recibido %s |
| `InvalidFormat` | 3 | El campo %s no cumple con el formato requerido, debe ser %s, recibido %s |
| `InvalidPattern` | 3 | El campo %s no cumple con el patrón requerido, debe ser %s, recibido %s |
| `InvalidField` | 1 | El campo %s no cumple con las reglas de negocio definidas |
| `InvalidFieldValue` | 1 | El campo %s no está permitido. Por favor elimínelo y vuelva a intentarlo |
| `InvalidDocumentNumber` | 1 | Número de documento inválido, debe ser: DUI con guión, NIT (con o sin guión), o un documento de 3-20 caracteres. Recibido: %s |
| `NegativeDiscount` | 1 | El descuento %s es negativo |
| `InvalidValue` | 3 | El valor %d no es válido, debe ser %s para el campo %s |
| `InvalidEmail` | 1 | El correo %s no es válido. Debe ser una dirección de correo válida y existente, y debe tener al menos 3 caracteres y no más de 100 |
| `InvalidRetentionCode` | 1 | El código de retención '%s' no es válido. Debe ser uno de los códigos de retención permitidos: (22 -> IVA 1 por ciento, C4 -> IVA 13 por ciento, C9 -> Otros Casos) |
| `InvalidRetentionIVA` | 3 | Para el ítem %d, la retención de IVA %f no coincide con el valor esperado %f.  |
| `InvalidTotalSubjectRetention` | 2 | La retención total sujeta %f no coincide con el valor esperado %f |
| `FSEItemInvalidEmpty` | 0 | Es requerido al menos un ítem para FSE |
| `FSEItemInvalidPurchase` | 0 | El campo compra debe ser mayor a cero |
| `FSEItemInvalidDescription` | 0 | La descripción del ítem no puede estar vacía |
| `FSEItemInvalidQuantity` | 0 | La cantidad debe ser mayor a cero |
| `FSEItemInvalidUnitPrice` | 0 | El precio unitario debe ser mayor a cero |
| `FSETaxInvalidDocument` | 0 | El documento FSE es requerido para validación de impuestos |
| `FSETaxInvalidIVARetention` | 0 | La retención de IVA debe ser mayor o igual a cero |
| `FSETaxInvalidIncomeRetention` | 0 | La retención de renta debe ser mayor o igual a cero |
| `FSETaxInvalidTotalToPay` | 2 | El total a pagar %f no es correcto; debe ser subtotal - retención de renta = %f |
| `FSETaxInvalidTaxedSale` | 0 | Los sujetos excluidos no pueden tener ventas gravadas |
| `FSEReceiverInvalidDocument` | 0 | El documento FSE es requerido para validación del receptor |
| `FSEReceiverRequiredDocumentType` | 0 | El tipo de documento es obligatorio para sujeto excluido |
| `FSEReceiverRequiredDocumentNumber` | 0 | El número de documento es obligatorio para sujeto excluido |
| `FSEReceiverRequiredName` | 0 | El nombre es obligatorio para sujeto excluido |
| `FSEReceiverRequiredAddress` | 0 | La dirección es obligatoria para sujeto excluido |
| `FSEReceiverInvalidDocumentType` | 0 | Tipo de documento no válido para sujeto excluido |
| `FSEReceiverInvalidNIT` | 0 | El NIT debe tener 9 o 14 dígitos |
| `FSEReceiverInvalidDUI` | 0 | El DUI debe tener exactamente 9 dígitos |
| `FSEReceiverInvalidDocumentLength` | 0 | El número de documento no puede exceder 20 caracteres |
| `FSEReceiverInvalidActivityCode` | 0 | El código de actividad económica no es válido |
| `FSEReceiverInvalidActivityDescription` | 0 | La descripción de actividad no puede exceder 150 caracteres |
| `FSEValidatorNullDocument` | 0 | El documento FSE no puede ser nulo |
| `InvalidTotalIVARetention` | 2 | La retención total de IVA %f no coincide con el valor esperado %f |
| `DateOutOfAllowedRange` | 2 | El ítem %d con número de documento %s tiene una fecha fuera del rango permitido. El documento debe ser del período actual o del período inmediato anterior, y el documento de retención debe emitirse dentro de los primeros 10 días hábiles del mes siguiente |
| `InvalidAmbientCode` | 1 | El código de ambiente %s no es válido, debe ser: 00 -> (Pruebas) y 01 -> (Producción) |
| `InvalidDateTime` | 1 | La fecha %s no es válida, debe ser una fecha menor o igual a la fecha actual |
| `InvalidEmissionTime` | 1 | La hora de emisión %s no es válida, debe ser una hora menor o igual a la hora actual |
| `InvalidTransmissionType` | 1 | El tipo de transmisión %s no es válido, debe ser: 1 -> (Normal) y 2 -> (Contingencia) |
| `InvalidAmount` | 1 | El monto %s no es válido, debe ser un número entre 0 y 99999999999.99 |
| `InvalidQuantity` | 1 | La cantidad %s no es válida, debe ser un número entre 1 y 99999999999.99 |
| `InvalidDiscount` | 1 | El descuento %s no es válido, debe ser un número entre 0 y 100 |
| `InvalidTax` | 1 | El impuesto %s no es válido, debe ser un número entre 0 y 99999999999.99 |
| `InvalidCurrency` | 1 | La moneda %s no es válida, debe ser USD |
| `InvalidItemType` | 1 | El tipo de ítem %d no es válido, debe ser: 1 -> (Producto), 2 -> (Servicio), 3 -> (Ambos) y 4 -> (Impuesto) |
| `InvalidTaxType` | 1 | El tipo de impuesto %s no es válido, debe estar dentro del catálogo de impuestos permitidos |
| `InvalidMunicipality` | 3 | Código de municipio %s inválido para el departamento %s. El código de municipio debe ser un número de dos dígitos que siga el patrón del catálogo oficial. Por ejemplo, códigos válidos para este departamento incluyen: %s. Consulte el catálogo oficial de municipios. |
| `InvalidEstablishmentType` | 1 | El tipo de establecimiento %s no es válido, debe ser: 01 -> (Casa Matriz), 02 -> (Sucursal), 04 -> (Bodega), 07 -> (Propiedad o Terreno) y 20 -> (Otro) |
| `InvalidDocumentNumberItem` | 1 | El número de documento %s no es válido, cuando el tipo de documento es '1' (físico), debe ser un número entre 1 y 20 caracteres, cuando el tipo de documento es '2' (electrónico), debe ser un UUID válido |
| `InvalidEmissionDateForPhysicalDocument` | 1 | La fecha de emisión %s no es válida, debe ser una fecha menor o igual a la fecha actual cuando el tipo de generación es '1' (física) |
| `InvalidPaymentTotal` | 2 | El total de pagos (%f) no coincide con el monto de operación (%f) |
| `InvalidTotalIVA` | 2 | El total de IVA %f no coincide con la suma de impuestos IVA %f |
| `InvalidTotalTaxed` | 2 | El total gravado %f no coincide con la suma de ventas gravadas %f |
| `InvalidTotalDiscount` | 2 | El total de descuentos %f no coincide con la suma de descuentos %f |
| `UnsupportedTaxCode` | 1 | Código de impuesto no soportado %s, debe estar en el catálogo de impuestos permitidos |
| `ExceededItemsLimit` | 1 | El número de ítems (%d) excede el límite permitido de 2000 |
| `RequiredFieldMissing` | 2 | El campo requerido '%s' no está presente para el tipo de documento %s |
| `InvalidBienTitulo` | 1 | El campo bienTitulo '%s' no es válido. Debe ser: 01 (Venta), 02 (Consignación), 03 (Exhibición), 04 (Traslado interno), 05 (Préstamo), 99 (Otros) |
| `RequiredBienTitulo` | 0 | El campo bienTitulo es obligatorio para Notas de Remisión |
| `InvalidRemissionNoteReceiver` | 0 | El receptor debe ser de tipo RemissionNoteReceiver para Notas de Remisión |
| `RemissionNoteItemValidationFailed` | 0 | Error de validación en los items de la Nota de Remisión |
| `RemissionNoteSummaryValidationFailed` | 0 | Error de validación en el resumen de la Nota de Remisión |
| `RemissionNoteReceiverValidationFailed` | 0 | Error de validación en el receptor de la Nota de Remisión |
| `AtLeastOneSaleTypeRequired` | 0 | Al menos un tipo de venta debe tener valor mayor a cero |
| `InvalidSequence` | 0 | Los números de item deben ser secuenciales empezando desde 1 |
| `RequiredSummary` | 1 | Campo de resumen requerido para el tipo de documento %s |
| `InvalidSubTotal` | 2 | El subtotal %f no coincide con la suma de totales gravados, exentos y no sujetos %f |
| `InvalidItemNumber` | 1 | El número de ítem %d no es válido, debe ser un número entre 1 y 2000 |
| `InvalidTaxRulesCCF` | 0 | Las reglas de impuestos no coinciden con el monto gravado, solo se permite el impuesto IVA para ítems de tipo 4 |
| `InvalidUnitMeasure` | 1 | La unidad de medida %d no es válida, debe ser 99 para ítems de tipo 4 |
| `InvalidNonTaxedAmount` | 0 | El total_non_taxed del Resumen no puede ser mayor que 0 cuando non_taxed de los ítems es 0 y viceversa |
| `InvalidTotalNonTaxed` | 2 | La suma de los ítems de monto no gravado del resumen %f no coincide con las ventas total_non-taxed %f |
| `MissingNRC` | 2 | NRC requerido para %s cuando el tipo de documento es %s |
| `InvalidTaxCalculation` | 3 | El cálculo de impuestos %s (total_taxed - total_discount * tax) es incorrecto. Esperado: %f, Actual: %f |
| `InvalidTotalOperation` | 2 | La operación total (%f) no coincide con la suma de totales (%f) |
| `InvalidDUIFormat` | 1 | El formato de DUI %s no es válido, debe tener 9 dígitos (con o sin guion: XXXXXXXX-X) |
| `InvalidNITFormat` | 1 | El formato de NIT %s no es válido, debe ser de 14 dígitos para personas jurídicas y 9 dígitos para personas naturales homologación |
| `MixedDocumentTypesNotAllowed` | 0 | No se permiten tipos de documentos mixtos, se encontró un tipo diferente al esperado. Los tipos de documentos relacionados deben ser iguales, no mezcle tipos de documentos |
| `MissingTaxes` | 0 | En los ítems existen impuestos pero en el resumen no existe ningún impuesto |
| `MissingTaxesItem` | 1 | El ítem %d tiene venta gravada pero no hay impuestos presentes, el impuesto (20) IVA debe estar presente para ventas gravadas |
| `MissingItemUnitPrice` | 1 | El ítem %d, el precio unitario es requerido cuando la venta gravada es mayor que cero |
| `InvalidPerceptionAmount` | 2 | El monto de percepción %s no coincide con el esperado (1%% del total gravado): %s |
| `InvalidDTETypeForInvalidation` | 1 | El tipo de documento %s no es válido para invalidación, solo (01) -> Factura Electrónica, (03) -> Comprobante de Crédito Fiscal Electrónico y (04) -> Nota de Remisión Electrónica están permitidos |
| `TransmissionFailed` | 0 | La transmisión falló, el documento no fue recibido por Hacienda |
| `ExceededRelatedDocsLimit` | 1 | El número de documentos relacionados (%d) excede el límite permitido de 50 |
| `InvalidRelatedDocTypeForCreditNote` | 1 | El tipo de documento relacionado %s no es válido para notas de crédito, debe ser (03) -> Comprobante de Crédito Fiscal o (07) -> Comprobante de Retención Electrónico |
| `InvalidRelatedDocDTEType` | 2 | El tipo de documento relacionado %s no es válido, debe ser %s |
| `InvalidRelatedDocDate` | 1 | La fecha del documento relacionado %s no es válida, debe ser una fecha menor o igual a la fecha actual |
| `InvalidRelatedDocNumberContingency` | 1 | El número de documento relacionado %s no es válido, debe ser un número entre 1 y 36 caracteres |
| `InvalidRelatedDocNumberNormal` | 1 | El número de documento relacionado %s no es válido, debe ser un número entre 1 y 20 caracteres |
| `InvalidOtherDocsCount` | 1 | El número de documentos asociados debe estar entre 1 y 10, encontrado %d |
| `InvalidMedicalDocFields` | 0 | Para documentos médicos (código 3) los campos de descripción y detalle no deben enviarse |
| `MutuallyExclusiveFields` | 2 | Los campos %s y %s son mutuamente excluyentes, solo uno de ellos puede enviarse |
| `InvalidUnitPrice` | 3 | El precio unitario debe ser igual a la venta gravada cuando non_taxed > 0. Ítem: %d, Precio Unitario: %f, Venta Gravada: %f |
| `InvalidUnitPriceZero` | 3 | El precio unitario debe ser mayor que cero cuando la venta gravada es mayor que cero. Ítem: %d, Precio Unitario: %f, Venta Gravada: %f |
| `InvalidDecimals` | 1 | El monto debe ser múltiplo de 0.01, obtenido: %s |
| `ExcessiveItemTotal` | 2 | El total del ítem %f es mayor que el total de ventas %f |
| `InvalidTotalExempt` | 2 | El total exento %f no coincide con la suma de ventas exentas %f |
| `InvalidTotalNonSubject` | 2 | El total no sujeto %f no coincide con la suma de ventas no sujetas %f |
| `InvalidMixedSalesWithNonTaxed` | 1 | El ítem %d tiene ventas mixtas con no gravado, el campo de impuestos debe estar vacío |
| `InvalidTaxesWithNonTaxed` | 1 | El ítem %d tiene impuestos con no gravado, el campo de impuestos debe estar vacío |
| `InvalidUnitPriceWithNonTaxed` | 1 | El ítem %d, el precio unitario debe ser 0 porque es un ítem no gravado |
| `InvalidMixedSalesWithExempt` | 1 | El ítem %d tiene ventas mixtas con exento, solo una |
| `InvalidTaxesWithExempt` | 1 | El ítem %d tiene impuestos con exento, el campo de impuestos debe estar vacío |
| `InvalidTaxesWithNonSubject` | 1 | El ítem %d tiene impuestos con no sujeto, el campo de impuestos debe estar vacío |
| `ValidationFailed` | 0 | Validación fallida para el documento |
| `InvalidSubTotalSales` | 2 | El subtotal de ventas %f no coincide con la suma de todos los tipos de ventas %f |
| `InvalidTaxes` | 0 | Montos de impuestos presentes con venta gravada cero |
| `MixedSalesTypesNotAllowed` | 1 | El ítem %d tiene tipos de ventas mixtos, solo se permite un tipo por ítem |
| `InvalidMonetaryAmount` | 2 | El monto %s es inválido, debe ser multiplo de 0.01, recibido %s |
| `InvalidUnitPriceForNonTaxed` | 1 | El ítem %d con monto no gravado debe tener precio unitario 0 |
| `MissingItemRelatedDoc` | 1 | El ítem %d, documento relacionado es requerido cuando la sección de Documentos Relacionados está presente |
| `MissingItemTaxes` | 1 | Para este tipo de DTE, el campo de impuestos es obligatorio en los item. El item %d no tiene impuestos |
| `InvalidItemRelatedDoc` | 2 | El ítem %d, documento relacionado %s no es válido |
| `InvalidSubTotalCalculation` | 2 | El cálculo del subtotal con descuentos (descuento_exento - descuento_no_sujeto - descuento_gravado) es incorrecto. Esperado: %f, Actual: %f |
| `InvalidTotalToPayCalculation` | 2 | El cálculo del total a pagar es incorrecto. Esperado: %f, Actual: %f |
| `InvalidMixedSalesWithNonSubject` | 1 | El ítem %d tiene ventas mixtas con no sujeto, solo una |
| `InvalidIVAItemWithoutTaxedSale` | 1 | El ítem %d tiene IVA ítem pero no tiene venta gravada |
| `InvalidIVAItemCalculation` | 3 | El ítem %d tiene un cálculo de IVA ítem inválido. Esperado: %f, Actual: %f |
| `DiscountExceedsSubtotal` | 3 | El %s, con valor %f, excede el subtotal %f |
| `DiscountExceedsBase` | 3 | El descuento %s (%f) excede su base correspondiente (%f) |
| `InvalidTotalDiscountCalculation` | 2 | El total de descuentos (%f) no coincide con la suma calculada (%f) de descuentos de ítems más descuentos del resumen |
| `MissingContingencyType` | 0 | Tipo de contingencia requerido cuando el tipo de operación es contingencia |
| `MissingContingencyReason` | 0 | Razón de contingencia requerida cuando el tipo de contingencia es 'Otra razón' |
| `InvalidModelType` | 1 | Tipo de modelo inválido %d para transmisión normal |
| `InvalidPhone` | 1 | El teléfono %s no es válido, debe ser un número entre 8 y 30 dígitos |
| `InvalidAppendixField` | 1 | El campo de apéndice no es válido, debe ser un valor entre 1 y 25 caracteres, recibido %s |
| `InvalidAppendixLabel` | 1 | El campo de etiqueta de apéndice no es válido, debe ser un valor entre 3 y 50 caracteres, recibido %s |
| `InvalidAppendixValue` | 1 | El campo de valor de apéndice no es válido, debe ser un valor entre 1 y 150 caracteres, recibido %s |
| `InvalidDeliveryDocument` | 1 | El documento de entrega no es válido, debe ser un valor entre 1 y 25 caracteres, recibido %s |
| `InvalidDeliveryName` | 1 | El nombre de entrega no es válido, debe ser un valor entre 1 y 100 caracteres, recibido %s |
| `InvalidDTEType` | 1 | El tipo de DTE no es válido, debe ser un valor válido del cátalogo CAT-002, recibido %s |
| `InvalidDocumentForReceiver` | 1 | El tipo de documento del receptor no es válido, debe ser NIT (36), DUI (13), Carnet de residente (02), Pasaporte (03) u otro documento (37), recibido %s |
| `InvalidDTETypeForRetention` | 1 | El tipo de DTE no es válido para retención, debe ser un factura electronica (01), comprobante de credito fiscal (03) o factura sujeto excluido electronica (14), recibido %s |
| `InvalidInvalidationReason` | 1 | La razón de invalidación no es válida, debe ser un valor entre 5 y 250 caracteres, cantidad actual: %d |
| `InvalidAssociatedDocumentCode` | 1 | El código de documento asociado %d no es válido, debe ser un numero entre 1 y 4 |
| `InvalidInvalidationType` | 1 | El tipo de invalidación no es válido, debe ser un valor válido del catálogo CAT-024, obtenido %d |
| `InvalidServiceType` | 1 | El tipo de servicio %d no es válido, debe ser un número entre 1 y 6 |
| `InvalidCreditNoteTransaction` | 4 | El total %s es mayor que el saldo restante del documento %s. Total obtenido: %f. Saldo restante para este total: %f. |
| `InvalidPaymentTypeOP2` | 0 | Cuando la condición de operación es 2, el código de pago NO debe ser es '01' (Billetes y monedas) porque no es una venta al contado |
| `InvalidPaymentTerms` | 0 | Cuando la condición de operación es 2, los terminos 'term' y plazo 'period' son requeridos |
| `InvalidPaymentTermsOF` | 0 | Cuando la condición de operación es 1, los terminos 'term' y plazo 'period' NO DEBEN ENVIARSE |
| `InvalidTaxCodeOnly20` | 2 | El código de impuesto %s no es válido para artículos de tipo %d, debe ser 20 (IVA) |
| `InvalidIVARetentionWithoutTaxedAmount` | 0 | Retention de IVA esta presente pero no tiene venta gravada, no puede haber IVA retenido sin venta gravada |
| `IncomeRetentionNotAllowed` | 0 | La retención de renta no aplica para FE, CCF, Nota de Crédito ni Nota de Débito (eliminada en la Normativa DTE v2.0); income_retention debe ser 0 |
| `ArchiveNotAvailable` | 1 | El Archivo DTE de %s no está disponible: el documento se emitió antes de guardar la firma electrónica o la firma no es válida |
| `GlobalDiscountNotAllowedInAdjustment` | 1 | Las Notas de Crédito y Débito no admiten descuentos globales ni porcentaje de descuento, %s debe ser 0. Aplique el descuento en cada item |
| `RelatedDocOutOfPeriod` | 3 | El documento relacionado tipo %s con fecha %s solo se puede relacionar hasta el %s |
| `MixedRetentionCodes` | 2 | Todos los documentos de un Comprobante de Retención deben tener el mismo código de retención (%s y %s) |
| `InvalidItemSaleAmount` | 3 | Item %d: la venta declarada %f no coincide con (precio unitario x cantidad) - descuento = %f |
| `TaxNotAllowedInInvoice` | 2 | El tributo %s no aplica en los items de Factura Electrónica (item %d): el IVA va incluido en el precio y se informa en iva_item |
| `TaxNotAllowedInInvoiceSummary` | 1 | El tributo %s no aplica en el resumen de Factura Electrónica: el IVA va incluido en el precio y se informa en total_iva |
| `IVARetentionNotAllowedInFSE` | 0 | La retención de IVA no aplica en Factura de Sujeto Excluido (eliminada en la Normativa DTE v2.0); iva_retention debe ser 0 |
| `InvalidNonTaxedAmountCalculation` | 2 | El cálculo de la venta no gravada es incorrecto. Total en resumen: %f, Total declarado en items: %f |
| `MissingStampDate` | 1 | No se conoce la fecha del sello de recepción del documento %s, no se puede calcular el plazo de invalidación |
| `InvalidationDeadlineExpired` | 2 | El plazo para invalidar el DTE tipo %s venció el %s. El documento conserva su validez, use una Nota de Crédito o Débito si corresponde |
| `InvalidationEventDateOutOfRange` | 2 | La fecha del evento de invalidación del DTE tipo %s no puede ser posterior al %s |
| `InvalidationEventDateMismatch` | 2 | La fecha del evento de invalidación del DTE tipo %s debe ser igual a su fecha de generación %s |
| `InvalidContingencyType` | 0 | El tipo de contingencia no es válido, debe ser un valor entre 1 y 5 |
| `ExcessiveItemDiscount` | 3 | Item %d: el descuento %f excede el monto del item (precio unitario x cantidad) %f |
| `InvalidTotalPurchase` | 2 | El total de compras %f no coincide con la suma de las compras de los items %f |
| `FSEDiscountValidatorNullDocument` | 0 | FSE: documento nulo en la validación de descuentos |
| `MissingIVAForTaxedAmount` | 0 | Existen ventas gravadas pero no se informó el IVA correspondiente |
| `InvalidEnum` | 1 | El valor del campo %s no es válido |
| `NullDocument` | 1 | Documento nulo en %s |

### service_errors

| Código | Parámetros | Mensaje (es) |
|---|---|---|
| `ErrorMapping` | 1 | Error al mapear la sección %s |
| `NotFound` | 0 | No existe registro con estos datos |
| `DuplicatedEntry` | 1 | El valor del campo %s ya existe en nuestros registros, intente con otro valor |
| `InvalidData` | 0 | Los datos enviados no son válidos, verifique el formato y los valores |
| `MissingCredentials` | 0 | Faltan credenciales, todos los campos son obligatorios |
| `FailedToGetCredentials` | 0 | No se pudieron obtener las credenciales, por favor verifique los datos o contacte al administrador |
| `FailedToLogin` | 0 | Error al iniciar sesión |
| `InvalidCredentials` | 0 | Las credenciales no son válidas, por favor verifique los datos y vuelva a intentarlo |
| `UserNotActive` | 0 | La cuenta de usuario no está activa, por favor contacte al administrador |
| `FailedToCreateUser` | 0 | No se pudo crear el usuario, por favor verifique los datos y vuelva a intentarlo |
| `Unauthorized` | 0 | No tienes permiso para realizar esta acción |
| `AuthServiceUnavailable` | 0 | El servicio de autenticación no está disponible, por favor contacte al administrador |
| `TokenNotExist` | 0 | El token no existe |
| `FailedToSetCache` | 0 | No se pudo establecer guardar el valor temporalmente, por favor contacte al administrador |
| `FailedToGetCache` | 0 | No se pudo obtener el valor temporal, por favor contacte al administrador |
| `AddressWithReceiver` | 0 | Cuando esté presente la dirección, los campos departamento, municipio y complemento son requeridos |
| `InvalidDoctorInfo` | 0 | El nombre del doctor y el tipo de servicio son requeridos, pero uno o ambos están vacíos (el tipo de servicio 0 se considera vacío) |
| `InvalidDoctorDocuments` | 0 | Se requiere la identificación o el NIT del doctor, pero ambos están vacíos |
| `InvalidPaymentTypeInfo` | 0 | El código y el monto son requeridos y no pueden estar vacíos en payment_types |
| `InvalidTaxCodeAndDescription` | 0 | El código y la descripción del impuesto son requeridos, pero uno o ambos están vacíos |
| `InvalidThirdParty` | 0 | El nombre y el NIT son requeridos y no pueden estar vacíos en third_party_sale |
| `InvalidBaseDTE` | 0 | DTE base inválido |
| `InvalidInvalidationType3` | 0 | Si el tipo de motivo es 3, el campo 'reason_field' en el objeto reason debe ser proporcionado |
| `InvalidInvalidationType2` | 0 | El código de reemplazo debe ser nulo para el tipo 2 |
| `InvalidReplacementCode1And3` | 0 | El código de reemplazo es requerido para los tipos 1 y 3 |
| `InvalidDocumentTypeAndNumber` | 0 | DocumentType y DocumentNumber. Si DocumentType está presente, DocumentNumber también debe estarlo y viceversa, estos campos |
| `FailedToExtractSummaryTotals` | 0 | No se pudo extraer los totales del resumen |
| `FailedToCreateDTE` | 0 | No se pudo crear el DTE, revise los detalles a continuación |
| `FailedToGetDTE` | 1 | No se pudo obtener el DTE con el código: %s, revise los detalles a continuación |
| `FailedToUpdateDTE` | 0 | Hubo un error al actualizar el DTE, revise los detalles a continuación |
| `FailedToVerifyDTE` | 0 | Hubo un error al verificar el DTE, revise los detalles a continuación |
| `FailedToGenerateBalanceTransaction` | 0 | Hubo un error al generar la transacción de saldo, revise los detalles a continuación |
| `FailedToGetBalanceControl` | 0 | Hubo un error al obtener el control de saldo del documento relacionado, revise los detalles a continuación |
| `FailedToGetSummaryStats` | 0 | Hubo un error al obtener las estadísticas del resumen, revise los detalles a continuación |
| `FailedToGetPagedDoc` | 0 | Hubo un error al obtener el paginado, revise los detalles a continuación |
| `MissingReceptionStamp` | 0 | El sello de recepción es requerido para transmisiones normales y no puede estar vacío |
| `MissingDTEIdentification` | 0 | El DTE no contiene los campos de identificación requeridos (código de generación, número de control y tipo de DTE) para ser persistido |
| `NoRelatedDocs` | 0 | No hay documentos relacionados |
| `ValidationFailed` | 0 | Validaciones no superadas, revise los detalles a continuación |
| `FailedToSetControlNumber` | 0 | Hubo un error al establecer el número de control, revise los detalles a continuación |
| `DocumentAlreadyInvalid` | 1 | El %s ya ha sido invalidado, no se puede invalidar nuevamente |
| `DocumentReject` | 1 | El %s fue rechazado por hacienda, no se puede invalidar |
| `DocumentPending` | 1 | El %s aun está pendiente de retransmitir, no se puede invalidar |
| `InvalidQueryParam` | 2 | El parámetro de consulta %s tiene un formato incorrecto, debe ser %s |
| `RelatedDocumentNotReceived` | 2 | El documento relacionado %s no fue recibido por Hacienda, su estado actual es: %s |
| `NotMatchingReceiverNIT` | 0 | El NIT del receptor en el documento de nota de crédito no coincide con el NIT del documento a acreditar |
| `FailedToInvalidatedDTE` | 0 | Hubo un error al invalidar el DTE, por favor contacte al administrador |
| `PendingAdjustmentNotes` | 2 | El documento %s tiene Notas de Crédito o Débito vigentes que deben invalidarse primero: %s. Invalide cada nota y luego vuelva a solicitar la invalidación del documento |
| `DocumentWithoutStamp` | 1 | El documento %s no tiene sello de recepción, solo se pueden invalidar documentos recibidos por Hacienda |
| `FailedToRecoverInvalidatedAmounts` | 0 | Hubo un error al recuperar los montos del DTE invalidado, por favor contacte al administrador |

### Claves definidas sin uso en el código

- `validation_errors`: `WithoutParams`, `ExceededParameters`, `ExcessiveDiscount`, `NegativeTaxedAmount`, `ExcessiveTaxedAmount`, `FSETaxInvalidTotalPurchase`, `InvalidVersion`, `UnknownError`, `ServerError`, `InvalidTotalAmount`, `InvalidTaxedAmount`, `InvalidDocumentState`, `RemissionNoteValidationFailed`, `TotalMismatch`, `InvalidCalculation`, `InvalidTaxRules`, `InvalidTaxCode`, `DocumentInvalid`, `MissingPaymentCondition`, `InvalidTotalToPay`, `InvalidTotalToPayNonTaxedCCF`, `InvalidItemTotal`, `MissingTaxInItems`, `InvalidRetentionAmount`, `MixedSalesNotAllowed`, `InvalidRelatedDocType`, `InvalidTotalTypeCCF`, `InvalidSaleType`, `InvalidMultipleSaleCategories`, `MissingRelatedDocWithNonTaxed`, `InvalidUnitPriceWithTaxedSale`, `InvalidIVACalculation`, `MissingIVAItem`, `InvalidContingencyReason`, `RequiredExtension`, `InvalidTransmissionModel`, `RequiredReceiver`, `InvalidTaxForProduct`, `InvalidSummaryTaxForProduct`, `MissingIVAInTaxes`.
- `service_errors`: `NotDetails`, `PresentAppendixNil`, `FailedToSetReceptionStamp`, `UnsupportedDTETypeForAppendix`, `InvalidDTETypeForRetention`, `NotMatchingRelatedDoc`, `ContingencyActiveTransmission`, `NoDetailsAvailable`, `RequestTimeOutTitle`, `RequestTimeOut`.

### Claves usadas en el código sin traducción

Se invocan con `NewFormattedGeneralServiceError` o `NewFormattedGeneralServiceWithError` pero no existen en `service_errors`: `ServerError`, `RequiredField`, `FailedToRegisterEvent`, `FailedToGetEvents`, `ReceiverIsRequired`, `FailedToGenerateControlNumber`, `SummaryIsRequired`. `ServerError` existe solo en `validation_errors`, por lo que la búsqueda en `service_errors` no la encuentra.

### Llamadas con distinta cantidad de parámetros que el mensaje

- `InvalidLength` mensaje=3 llamada=2 en internal/domain/dte/common/models/base_summary_model.go
- `InvalidContingencyType` mensaje=0 llamada=1 en internal/domain/dte/common/validator/strategy/contingency_strategy.go
- `InvalidTaxType` mensaje=1 llamada=2 en internal/domain/dte/credit_note/validator/strategy/credit_note_item_strategy.go
- `InvalidTaxType` mensaje=1 llamada=2 en internal/domain/dte/debit_note/validator/strategy/debit_note_item_strategy.go
- `NegativeDiscount` mensaje=1 llamada=2 en internal/domain/dte/debit_note/validator/strategy/debit_note_tax_strategy.go
- `FSEItemInvalidEmpty` mensaje=0 llamada=1 en internal/domain/dte/fse/validator/strategy/fse_item_strategy.go
- `FSEItemInvalidPurchase` mensaje=0 llamada=1 en internal/domain/dte/fse/validator/strategy/fse_item_strategy.go
- `FSEItemInvalidDescription` mensaje=0 llamada=1 en internal/domain/dte/fse/validator/strategy/fse_item_strategy.go
- `FSEItemInvalidQuantity` mensaje=0 llamada=1 en internal/domain/dte/fse/validator/strategy/fse_item_strategy.go
- `FSEItemInvalidUnitPrice` mensaje=0 llamada=1 en internal/domain/dte/fse/validator/strategy/fse_item_strategy.go
- `FailedToGetDTE` mensaje=1 llamada=0 en internal/domain/dte/invalidation/invalidation_service.go
- `NegativeDiscount` mensaje=1 llamada=2 en internal/domain/dte/invoice/validator/strategy/invoice_total_strategy.go
- `ValidationFailed` mensaje=0 llamada=2 en internal/domain/dte/remission_note/validator/remission_note_validator.go

---

## Flujo de Propagación de Errores

```
Value Object inválido
    │
    ▼
ValidateModel[T]() ──→ []error (ValidationErrors)
    │
    ▼
ValidateDTEDocument(doc)          (invocado por el servicio de cada DTE)
    │
    ├── doc.ValidateDTERules() ──→ *DTEError (estrategias de negocio)
    │     └── si hay error: se le agregan los errores de VO en ValidationErrors
    │
    ├── solo errores de VO ──→ *CompositeError
    │
    ▼
Capa de respuesta HTTP (ResponseWriter)
    │
    ├── DTEError / ValidationError → 400 Bad Request
    ├── ServiceError → 400 Bad Request
    ├── errores de Hacienda → tratados como errores de negocio
    └── otros errores → 500 Internal Server Error (código SYSTEM_ERROR)
```

Las validaciones específicas de cada DTE (`CCFRulesValidator`, etc.) devuelven un `DTEError` compuesto y el servicio lo envuelve en un `ServiceError` con el código `ValidationFailed`.
