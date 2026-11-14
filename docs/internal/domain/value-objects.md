# Value Objects

## Qué Es un Value Object

Un Value Object (VO) es un primitivo de dominio que porta un significado más allá de un tipo Go crudo. Tres propiedades lo definen:

1. **Inmutable** — el valor interno se establece en el momento de construcción y nunca se modifica en el lugar. Si necesitas un valor diferente, construyes un nuevo VO.
2. **Auto-validante** — el constructor rechaza entradas inválidas antes de devolver. Un estado inválido nunca puede existir en memoria.
3. **Igualdad por valor** — dos VOs son iguales cuando sus valores contenidos son iguales, no cuando apuntan a la misma dirección de memoria.

En este código los VOs son el mecanismo principal para mantener datos incorrectos fuera del dominio. En lugar de validar strings crudos dispersos por los servicios, el sistema de tipos garantiza la corrección en el punto de construcción.

## La Interfaz `ValueObject[T]`

Todos los Value Objects implementan la interfaz genérica definida en `internal/domain/dte/common/interfaces/validator_interface.go`:

```go
// Interfaz Validator que define los métodos que deben implementar los objetos que validan un campo
type Validator interface {
    IsValid() bool
}

// Interfaz ValueObject que define los métodos que deben implementar los value objects
type ValueObject[T any] interface {
    Validator
    ToString() string
    Equals(value ValueObject[T]) bool
    GetValue() T
}
```

| Método | Contrato |
|--------|---------|
| `IsValid() bool` | Devuelve `true` si el valor interno cumple las reglas de negocio. Usado por `BaseValidator` durante la reflexión de structs. |
| `GetValue() T` | Devuelve el valor subyacente crudo (string, float64, int, time.Time). |
| `ToString() string` | Representación legible por humanos del valor. |
| `Equals(ValueObject[T]) bool` | Comparación de igualdad basada en valor. |

## Dos Constructores: `New*` vs `NewValidated*`

La mayoría de los VOs tiene dos constructores (ver la excepción al final de la tabla):

### `New*(value)` — el constructor validado

Usar cuando se construye un VO desde entrada del usuario o cualquier fuente no confiable:

```go
// internal/domain/dte/common/value_objects/identification/nit.go

func NewNIT(value string) (*NIT, error) {
    if strings.Contains(value, "-") {
        value = strings.ReplaceAll(value, "-", "")
    }
    value = strings.TrimSpace(value)

    nit := &NIT{Value: value}
    if nit.IsValid() {
        return nit, nil
    }
    return &NIT{}, dte_errors.NewValidationError("InvalidPattern", "nit",
        "12345678901234 o 123456789", value)
}
```

El patrón: construir tentativamente, llamar a `IsValid()`, devolver un `ValidationError` si falla.

### `NewValidated*(value)` — el constructor de bypass

Usar cuando el valor proviene de una fuente confiable (base de datos, generación interna) y la validación sería redundante o incorrecta:

```go
func NewValidatedNIT(value string) *NIT {
    return &NIT{Value: value}
}
```

Este constructor nunca devuelve error. Es una asignación directa. **No usar para datos suministrados por el usuario.**

### Cuándo usar cada uno

| Escenario | Constructor |
|---------|-------------|
| Parsear un body de petición HTTP | `New*` |
| Mapear una fila de BD de vuelta al modelo de dominio | `NewValidated*` |
| Reconstruir desde un payload de cola de mensajes que persistes tú mismo | `NewValidated*` |
| Establecer un campo via decodificador JSON para una API no confiable | `New*` |
| Generar un UUID para `GenerationCode` | `NewGenerationCode()` (solo existe el constructor validado, genera internamente) |

El error más común es usar `NewValidated*` en input del usuario. Si un NIT está en la BD porque pasó validación antes, `NewValidatedNIT` es correcto. Si un NIT llegó en una nueva petición de API, se requiere `NewNIT`.

## El `BaseValidator` — Validación de Structs Basada en Reflexión

`internal/domain/dte/common/validator/base_validator.go` provee `ValidateModel[T any](model T) []error`.

Usa `reflect` para recorrer cada campo exportado de un struct y verificar si el campo implementa `interfaces.ValueObject[any]`. Si lo hace, llama a `IsValid()`. Para slices recursa en cada elemento. Para structs anidados recursa en el struct.

```go
// internal/domain/dte/common/validator/base_validator.go

func ValidateModel[T any](model T) []error {
    var validationErrors []error
    v := reflect.ValueOf(model)

    if v.Kind() == reflect.Ptr {
        if v.IsNil() {
            return validationErrors
        }
        v = v.Elem()
    }

    if v.Kind() != reflect.Struct {
        return validationErrors
    }

    for i := 0; i < v.NumField(); i++ {
        field := v.Field(i)
        fieldType := v.Type().Field(i)

        if !fieldType.IsExported() {
            continue
        }

        if field.Kind() == reflect.Slice {
            sliceErrors := validateSlice(field)
            validationErrors = append(validationErrors, sliceErrors...)
            continue
        }

        if field.Kind() == reflect.Ptr {
            if field.IsNil() {
                continue
            }
            field = field.Elem()
        }

        if field.Kind() == reflect.Struct {
            if field.CanInterface() {
                structErrors := ValidateModel(field.Interface())
                validationErrors = append(validationErrors, structErrors...)
            }
            continue
        }

        if field.CanInterface() {
            if validator, ok := field.Interface().(interfaces.ValueObject[any]); ok {
                if !validator.IsValid() {
                    validationErrors = append(validationErrors,
                        dte_errors.NewValidationError("InvalidField", fieldType.Name))
                }
            }
        }
    }

    return validationErrors
}
```

**Reglas de recorrido:**
1. Se omiten los campos no exportados.
2. Los campos de puntero nulo se omiten (los VOs opcionales se representan como punteros).
3. Campos de tipo slice: recursa en cada elemento.
4. Campos de tipo struct: recursa recursivamente (maneja structs anidados como `Identification` dentro de `DTEDocument`).
5. Campos hoja: verifica si satisfacen `ValueObject[any]`. Si `IsValid()` devuelve false, emite `NewValidationError("InvalidField", fieldName)`.

Esto significa que si agregas un campo VO a cualquier struct de modelo, se validará automáticamente sin tocar ningún código de validador.

## Tabla Completa de Value Objects

### base/ — `internal/domain/dte/common/value_objects/base/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `Email` | `string` | Formato válido (`checkmail.ValidateFormat`), registro MX alcanzable salvo que `base.SkipMXValidation` sea `true`, y 3–100 caracteres | `admin@empresa.sv` | `notanemail` |
| `Phone` | `string` | 8–30 caracteres (solo se valida la longitud) | `22345678` | `1234` |

### document/ — `internal/domain/dte/common/value_objects/document/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `Ambient` | `string` | `"00"` (pruebas) o `"01"` (producción); `NewAmbient()` toma el valor de `config.Server.AmbientCode` | `"01"` | `"02"` |
| `AppendixField` | `string` | 2–25 caracteres | `"campo1"` | `""` |
| `AppendixLabel` | `string` | 3–50 caracteres | `"Número de Orden"` | `"ab"` |
| `AppendixValue` | `string` | 1–150 caracteres | `"ORD-2024-001"` | `""` |
| `AssociatedDocumentCode` | `int` | 1–4 (`constants.AllowedAssociatedDocumentCodes`) | `3` | `5` |
| `ContingencyReason` | `string` | 5–150 caracteres | `"Fallo de internet"` | `""` |
| `ContingencyType` | `int` | 1–5 | `1` | `99` |
| `DeliveryDocument` | `string` | 1–25 caracteres | `"01234567-8"` | `""` |
| `DeliveryName` | `string` | 1–100 caracteres | `"Juan Pérez"` | `""` |
| `DocumentNumber` (paquete `document`) | `string` + tipo de generación `int` | Documento físico (`1`): `^[a-zA-Z0-9]{1,20}$`; electrónico: UUID en mayúsculas hexadecimales con el formato `XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX` | `"A1001"` | `""` |
| `DTEType` | `string` | Uno de `constants.ValidDTETypes`; `NewDTETypeForReceiver` valida contra `ValidReceiverDTETypes` y `NewDTETypeForRetention` contra `ValidRetentionDTETypes` | `"01"` | `"99"` |
| `EstablishmentType` | `string` | `01`, `02`, `04`, `07` o `20` (`constants.AllowedEstablishmentTypes`) | `"01"` | `"03"` |
| `InvalidationReason` | `string` | 5–250 caracteres | `"Error en receptor"` | `""` |
| `InvalidationType` | `int` | 1–3 | `1` | `0` |
| `ModelType` | `int` | 1 (previo) o 2 (diferido) | `1` | `3` |
| `Observation` | `string` | 0–3000 caracteres | `"Entrega en bodega"` | (>3000 caracteres) |
| `OperationType` | `int` | 1 (normal) o 2 (contingencia) | `1` | `3` |
| `RetentionCode` | `string` | `"22"`, `"C4"` o `"C9"` | `"C4"` | `"XX"` |
| `ServiceType` | `int` | 1–6 | `2` | `7` |
| `TransmissionType` | `int` | 1 o 2 | `1` | `3` |
| `Version` | `int` | 1–4 | `1` | `0` |

### financial/ — `internal/domain/dte/common/value_objects/financial/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `Amount` | `float64` | `0 ≤ value ≤ 99999999999.99`; `NewAmount` redondea a 8 decimales y `NewAmountForTotal` exige a lo sumo 2 decimales | `1150.00` | `-0.01` |
| `Currency` | `string` | Debe ser `"USD"` | `"USD"` | `"EUR"` |
| `Discount` | `float64` | `0 ≤ value ≤ 100` (porcentaje, evaluado con 2 decimales); se usa en `discount_percentage` del resumen, no en el `discount` por ítem (que usa `Amount`) | `10.0` | `101.0` |
| `PaymentCondition` | `int` | 1 (contado), 2 (crédito) o 3 (otro) | `1` | `4` |
| `PaymentTerm` | `string` | Patrón `^0[1-3]$` | `"01"` | `"30"` |
| `PaymentType` | `string` | Código `01`–`14` o `99` (expresión regular `^(0[1-9]||1[0-4]||99)$`, que por sus alternativas vacías acepta también la cadena vacía) | `"01"` | `"15"` |
| `Tax` | `float64` | `0 ≤ value < 100000000000` (evaluado con 2 decimales) | `149.56` | `-1.0` |
| `TaxType` | `string` | Uno de `constants.AllowedTaxTypes` | `"20"` (IVA) | `"ZZ"` |

### identification/ — `internal/domain/dte/common/value_objects/identification/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `ActivityCode` | `string` | `^[0-9]{2,6}$` | `"47211"` | `"ABC"` |
| `ControlNumber` | `string` | Regex `^DTE-[0-9]{2}-[A-Z0-9]{8}-[0-9]{15}$` y longitud 31 | `"DTE-01-00000001-000000000000001"` | `"DTE-01-X"` |
| `DocumentNumber` (paquete `identification`) | `string` | 3–20 caracteres; `NewDocumentNumber(value, dteType)` elimina guiones si el tipo es NIT o DUI y exige 9 dígitos para DUI | `"012345678"` (DUI) | `"12"` |
| `GenerationCode` | `string` | UUID válido en mayúsculas (`google/uuid`); `NewGenerationCode()` lo genera | `"A1B2C3D4-E5F6-7890-ABCD-EF1234567890"` | `"not-a-uuid"` |
| `NIT` | `string` | 9 o 14 dígitos (guiones eliminados en construcción) | `"06140101991011"` | `"1234"` |
| `NRC` | `string` | `^[0-9]{1,8}$` | `"1234567"` | `""` |

### item/ — `internal/domain/dte/common/value_objects/item/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `ItemCode` | `string` | 1–24 caracteres (`0 < len < 25`) | `"SKU-001"` | (≥25 caracteres) |
| `ItemNumber` | `int` | 1–1999 (`0 < value < 2000`) | `1` | `0` |
| `ItemType` | `int` | 1 (producto), 2 (servicio), 3 (ambos) o 4 (impuesto) | `2` | `5` |
| `Quantity` | `float64` | `0 < value ≤ 99999999999.99` (evaluado con 2 decimales) | `1.5` | `0.0` |
| `UnitMeasure` | `int` | 1–99 | `59` | `0` |

### location/ — `internal/domain/dte/common/value_objects/location/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `Address` | `string` | 1–200 caracteres | `"Calle Principal, Edificio A"` | `""` |
| `Department` | `string` | Código `01`–`14` | `"06"` | `"99"` |
| `District` | `string` | Dos dígitos distintos de `00` | `"01"` | `"00"` |
| `Municipality` | `string` + `Department` | Código válido dentro del departamento dado (patrón por departamento) | `"14"` | `"99"` |

### temporal/ — `internal/domain/dte/common/value_objects/temporal/`

| Nombre | Tipo param Go | Regla de validación | Ejemplo válido | Ejemplo inválido |
|------|--------------|----------------|---------------|-----------------|
| `EmissionDate` | `time.Time` | No cero y anterior a la hora actual más 24 horas | `time.Now()` | `time.Time{}` |
| `EmissionTime` | `time.Time` | No cero | `time.Now()` | `time.Time{}` |

Algunos VOs no tienen constructor `NewValidated*` (por ejemplo `InvalidationReason`, `InvalidationType` y `DocumentNumber` del paquete `document`) ni siguen exactamente la forma de campo único `Value` (`AssociatedDocumentCode` y `ServiceType` usan un campo no exportado).

## Ejemplos de Código

### Crear un NIT

```go
import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"

// Desde input del usuario — valida
nit, err := identification.NewNIT("06140101991011")
if err != nil {
    // err es *dte_errors.ValidationError
    // err.Error() -> "[InvalidPattern] The field nit does not meet the required pattern, ..."
    return err
}
fmt.Println(nit.GetValue())  // "06140101991011"
fmt.Println(nit.ToString())  // "06140101991011"

// Con guiones — se eliminan automáticamente
nit2, err := identification.NewNIT("0614010199-1011")
// nit2.GetValue() == "06140101991011"

// Desde BD — bypass de validación
nit3 := identification.NewValidatedNIT("06140101991011")
```

### Crear un Amount monetario

```go
import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"

// Monto normal — redondea a 8 decimales internamente
amount, err := financial.NewAmount(1150.00)
if err != nil {
    return err
}

// Monto total — requiere exactamente 2 decimales
totalAmount, err := financial.NewAmountForTotal(1150.00)
if err != nil {
    // err.Error() si el valor tiene >2 decimales
}

// Monto negativo — falla
_, err = financial.NewAmount(-1.0)
// err != nil, ValidationError: "InvalidAmount"

// Desde BD
validated := financial.NewValidatedAmount(1150.00)
```

### Manejar fallos de validación

```go
// Cuando BaseValidator.ValidateModel se llama en un struct con un VO inválido:
type MyModel struct {
    NIT identification.NIT
}

m := MyModel{NIT: identification.NIT{Value: "INVALIDO"}}
errs := validator.ValidateModel(m)
// errs[0].Error() -> "[InvalidField] The field NIT does not meet the defined business rules"
```

### Comparación de igualdad

```go
a, _ := identification.NewNIT("06140101991011")
b, _ := identification.NewNIT("06140101991011")
c, _ := identification.NewNIT("06140101000000")

fmt.Println(a.Equals(b))  // true
fmt.Println(a.Equals(c))  // false
```

## Cómo Agregar un Nuevo Value Object

1. **Crear el archivo** en el sub-paquete apropiado bajo `internal/domain/dte/common/value_objects/`. Usar un VO existente como plantilla (ej. `nit.go`).

2. **Definir el struct** con un único campo exportado `Value` del tipo apropiado:
   ```go
   type MyVO struct {
       Value string `json:"value"`
   }
   ```

3. **Implementar `IsValid()`** — codifica la regla de negocio aquí:
   ```go
   func (m *MyVO) IsValid() bool {
       return len(m.Value) > 0 && len(m.Value) <= 50
   }
   ```

4. **Implementar la interfaz completa `ValueObject[T]`:**
   ```go
   func (m *MyVO) GetValue() string { return m.Value }
   func (m *MyVO) ToString() string { return m.Value }
   func (m *MyVO) Equals(other interfaces.ValueObject[string]) bool {
       return m.GetValue() == other.GetValue()
   }
   ```

5. **Escribir `New*(value)` y `NewValidated*(value)`:**
   ```go
   func NewMyVO(value string) (*MyVO, error) {
       vo := &MyVO{Value: value}
       if vo.IsValid() {
           return vo, nil
       }
       return &MyVO{}, dte_errors.NewValidationError("InvalidMyVO", value)
   }

   func NewValidatedMyVO(value string) *MyVO {
       return &MyVO{Value: value}
   }
   ```

6. **Agregar una clave de mensaje de error** en `i18n/en.yaml` (y `es.yaml`) bajo `validation_errors`:
   ```yaml
   validation_errors:
     InvalidMyVO: "El campo my_vo no es válido: %s"
   ```

7. **Usar el VO** en un struct de modelo. `BaseValidator` lo recogerá automáticamente por reflexión — no requiere registro.

8. **(Opcional) Agregar una constante** en `internal/domain/dte/common/constants/` si el VO necesita un mapa de valores permitidos.
