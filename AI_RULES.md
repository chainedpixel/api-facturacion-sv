# Constitucion del Proyecto y Reglas de Arquitectura

## 1. Stack y Entorno
- Lenguaje: Go 1.24+
- Base de datos: Definida en `.env` (default: MySQL)
- Cache: Redis
- Dependencias: Prohibido introducir dependencias externas nuevas en `go.mod` sin autorizacion explicita.
- Documentacion base: Cualquier duda tecnica o de contexto existente debe consultarse primero en `/docs`.

## 2. Arquitectura Hexagonal Estricta
- `internal/domain`: Entidades, Value Objects, errores de dominio e interfaces de puertos (Repositorios). PROHIBIDO importar paquetes de otros niveles, solo se permiten de afuera los paquetes de `pkg/shared`.
- `internal/application`: Casos de uso. Orquestan el dominio y manejan transacciones.
- `internal/infrastructure`: Adaptadores concretos (MySQL, Redis, HTTP Handlers, clientes externos).
- `internal/bootstrap`: Inicializacion del sistema, configuracion e inyeccion de dependencias.

## 3. Reglas de Codigo, Errores y Concurrencia
- Flujo SDD: Prohibido escribir o modificar codigo de produccion sin antes haber validado la especificacion correspondiente en `/specs`.
- Manejo de errores tipado: Utilizar estrictamente los tipos de error del proyecto (`ValidationError`, `CompositeError`, `ServiceError`, `DTEError`) segun el alcance y la capa, respetando los codigos y la cantidad exacta de parametros definidos en `assets/i18n`. Envolver errores de bajo nivel con `fmt.Errorf("...: %w", err)` solo cuando corresponda propagar contexto tecnico desde infraestructura.
- Concurrencia e I/O: Todo acceso a I/O (base de datos, red, cache) debe recibir y propagar `context.Context` como primer parametro.
- Comentarios en codigo: Prohibido comentar dentro del cuerpo de las funciones o comentar en exceso; el codigo debe ser autoexplicativo. Los comentarios se reservan unicamente arriba de la declaracion de paquetes, tipos, interfaces o funciones exportadas usando formato GoDoc (`//`) y estrictamente en idioma ingles.
- Formato y estilo: Prohibido el uso de emojis en cualquier parte del proyecto (codigo, logs, commits, tests y documentacion).
- Nomenclatura: Prohibido usar nombres de variables, constantes o funciones que colisionen o se confundan con palabras reservadas de Go o funciones predeclaradas del lenguaje (`new`, `make`, `copy`, `len`, `error`, `string`, etc.). En Go no existen clases, referirse y estructurar siempre mediante `structs` e `interfaces`.

## 4. Sincronizacion de Documentacion y Contratos
- Postman: Siempre que se cree o modifique un endpoint HTTP, es obligatorio verificar y actualizar la coleccion y el environment en `/scripts/postman`.
- Documentacion tecnica: Siempre que se modifique la logica o arquitectura de un modulo, se debe verificar y actualizar el archivo correspondiente dentro de `/docs` para evitar documentacion desfasada.

## 5. QA, Auditoria y Testing
- Ejecucion de pruebas: Al finalizar la implementacion de una fase o feature, es obligatorio ejecutar la suite de tests (`go test ./...`) y asegurar que todos los casos pasen.
- Rol de Subagente de QA (Auditoria Final): Antes de dar por completada cualquier tarea, el agente debe realizar una auditoria autonoma del codigo generado (o invocar un subagente de revision si la herramienta lo soporta) verificando y reportando explicitamente el cumplimiento de:
    1. Aislamiento estricto de Arquitectura Hexagonal y reglas de DDD.
    2. Principios SOLID, DRY y KISS.
    3. Uso correcto de codigos de error en `assets/i18n`.
    4. Actualizacion de `/docs` y `/scripts/postman`.