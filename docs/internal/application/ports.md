# Puertos de Aplicación

**Ubicación:** `internal/application/ports/`

## Qué Son los Puertos de Aplicación

Los puertos de aplicación son interfaces de Go propiedad de la capa de aplicación que describen capacidades que la capa de aplicación necesita de la capa de infraestructura. Definirlos aquí mantiene la capa de aplicación independiente de cualquier detalle de implementación concreto (clientes HTTP, Redis, servicio de firma Spring Boot, etc.).

Los tres archivos de puertos en este paquete representan diferentes capacidades externas:

| Archivo | Interfaz(ces) | Propósito |
|---|---|---|
| `base_transmit.go` | `BaseTransmitter`, `SignerManager` | Transmisión de documentos con reintento y firma de DTE |
| `dte_hacienda_transmitter_port.go` | `DTETransmitter` | Transmisión HTTP de bajo nivel a Hacienda |
| `hacienda_auth_manager.go` | `HaciendaAuthManager` | Gestión del token de autenticación con Hacienda |

---

## `BaseTransmitter`

**Archivo:** `internal/application/ports/base_transmit.go`

```go
type BaseTransmitter interface {
    RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*models.TransmitResult, error)
    CheckStatus(ctx context.Context, document interface{}, nit string, systemToken string) (*models.TransmitResult, error)
}
```

Esta es la interfaz que llaman los casos de uso. Encapsula la firma del documento, la transmisión y la lógica de reintento para que los casos de uso no necesiten preocuparse por conteos de reintentos ni intervalos.

**`RetryTransmission`**

Llamado por `GenericDTEUseCase.Create` e `InvalidationUseCase.InvalidateDocument`. Acepta:

- `document interface{}` — el struct del documento en formato MH completamente poblado (o struct del evento de invalidación)
- `token string` — el JWT del sistema en crudo, reenviado a `HaciendaAuthManager` para obtener un token de Hacienda
- `nit string` — el NIT del emisor, necesario por el servicio de firma

Firma el documento (`SignerManager.SignDTE`), lo transmite y devuelve `*models.TransmitResult`, que contiene `Status`, `ReceptionStamp`, `ProcessingDate`, `MessageCode`, `MessageDesc`, `Observations` y la evidencia de la transmisión: `SignedDocument` (el JWS transmitido, que se guarda como `firmaElectronica` del Archivo DTE) y `HaciendaResponse` (la respuesta completa de Hacienda en JSON). El detalle del flujo está en [Transmitter](dte/transmitter.md).

**`CheckStatus`**

Consulta a Hacienda si un documento ya fue procesado; recibe el token del sistema (`systemToken`) para obtener el token de Hacienda. Dentro de `BaseTransmitter` solo se invoca desde `RetryTransmission`; el servicio de contingencia usa su propio método interno `checkDocumentStatus` y no pasa por este puerto.

**Implementación:** `dte.BaseTransmitter` en `internal/application/dte/base_transmit.go`, construido con `dte.NewBaseTransmitter(transmitter, signer)`. Envuelve a `DTETransmitter` (`MHTransmitter`) y a `SignerManager`.

---

## `SignerManager`

**Archivo:** `internal/application/ports/base_transmit.go`

```go
type SignerManager interface {
    SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
```

Usado por `BaseTransmitter.RetryTransmission` para firmar el JSON del documento serializado antes de la transmisión (`MHTransmitter` no firma: recibe el JWS ya firmado). También lo usan el servicio de contingencia y el transmisor por lotes al retransmitir documentos pendientes. La firma es manejada por un microservicio externo (ruta en `config.Signer.Path`, timeout HTTP de 2 segundos).

**Implementación de infraestructura:** `signer.DTESigner` en `internal/infrastructure/adapters/signing/signer/dte_signer.go`

---

## `DTETransmitter`

**Archivo:** `internal/application/ports/dte_hacienda_transmitter_port.go`

```go
type DTETransmitter interface {
    Transmit(context.Context, interface{}, string, string) (*models.TransmitResult, error)
    CheckDocumentStatus(context.Context, interface{}, string, string) (*models.TransmitResult, error)
    SendToHacienda(context.Context, *models.HaciendaRequest, string) (*models.HaciendaResponse, error)
}
```

Esta es la interfaz de transmisor de nivel más bajo. A diferencia de `BaseTransmitter`, no incluye lógica de reintento — eso es responsabilidad de `BaseTransmitter` que envuelve a este.

- `Transmit(ctx, document, signedDoc, token)` — recibe el documento y su JWS (ya firmado por `BaseTransmitter`), elige el procesador (DTE o invalidación), construye la petición a Hacienda y llama a `SendToHacienda`. Adjunta la respuesta completa en `TransmitResult.HaciendaResponse`
- `CheckDocumentStatus(ctx, document, nit, systemToken)` — consulta el endpoint de consulta de Hacienda para el estado actual de un documento; obtiene el token de Hacienda a partir del token del sistema en cada llamada
- `SendToHacienda` — el HTTP POST crudo con gestión de token; separado para que los tests puedan sustituirlo

**Implementación de infraestructura:** `MHTransmitter` en `internal/infrastructure/adapters/transmitter/mh_transmitter.go`

---

## `HaciendaAuthManager`

**Archivo:** `internal/application/ports/hacienda_auth_manager.go`

```go
type HaciendaAuthManager interface {
    GetOrCreateHaciendaToken(ctx context.Context, systemToken string) (string, error)
    GetOrCreateHaciendaTokenWithCreds(ctx context.Context, systemToken string, creds models.HaciendaCredentials) (string, error)
}
```

Esta interfaz gestiona el token de autenticación de Hacienda de forma independiente al JWT del sistema. El token de Hacienda se almacena en caché en Redis.

**`GetOrCreateHaciendaToken`**

Usado durante la transmisión normal (no por lotes). La implementación:
1. Busca en Redis un token de Hacienda en caché con clave `hacienda:token:<systemToken>`
2. Si no se encuentra, extrae `AuthClaims` del contexto para obtener el NIT
3. Llama a `AuthManager.GetHaciendaCredentials` para recuperar el usuario/contraseña de Hacienda: se leen descifrados de la caché Redis (clave `hacienda:credentials:<systemToken>`), donde `Login` los guardó cifrados con el tiempo de vida del JWT
4. Hace un POST a la URL de autenticación de Hacienda usando campos form-encoded `user` + `pwd`
5. Almacena en caché el token devuelto en Redis con un TTL de 24 horas

**`GetOrCreateHaciendaTokenWithCreds`**

Usado por el transmisor por lotes (`BatchTransmitterService`) y por el servicio HTTP de eventos de contingencia (`contingencyHTTPService`, creado con `NewContingencyEventService`), donde las credenciales se pasan explícitamente en lugar de consultarse desde el contexto. También consulta primero la caché con la misma clave.

**Implementación de infraestructura:** `HaciendaAuthService` en `internal/infrastructure/adapters/signing/hacienda_auth_service.go`

---

## Por Qué Estos Puertos Existen a Nivel de Aplicación (No de Dominio)

La capa de dominio debe permanecer completamente libre de conocimiento de infraestructura. Sin embargo, la capa de aplicación coordina llamadas externas, por lo que necesita expresar *qué* capacidades requiere sin especificar *cómo* se implementan.

Los puertos en este nivel satisfacen el principio de inversión de dependencias:

- La capa de aplicación define la interfaz (apuntando hacia adentro)
- La capa de infraestructura implementa la interfaz (apuntando hacia afuera)
- La capa de aplicación nunca importa paquetes de infraestructura

La distinción con los puertos a nivel de dominio:

| Ubicación | Ejemplo | Razón |
|---|---|---|
| `internal/domain/ports/` | `DTEService`, `SequentialNumberRepositoryPort` | Conceptos de dominio puro (sin conocimiento de sistemas externos) |
| `internal/application/ports/` | `BaseTransmitter`, `HaciendaAuthManager` | Requieren conocimiento de protocolos externos (API de Hacienda, servicio de firma) |

Los puertos a nivel de dominio describen "qué puede hacer el dominio". Los puertos a nivel de aplicación describen "qué necesita la aplicación del exterior del dominio".

---

## Cómo Se Conectan Estos Puertos

En la raíz de inyección de dependencias (`internal/bootstrap/containers/`), las implementaciones concretas se vinculan con estas interfaces:

```go
// ServicesContainer.Initialize
signerManager := signer.NewDTESigner(authRepo)                                  // SignerManager
haciendaAuth := signing.NewHaciendaAuthService(cacheManager, authManager)       // HaciendaAuthManager
transmitter := adapterTransmitter.NewMHTransmitter(haciendaAuth, failedSeqRepo) // DTETransmitter

// UseCaseContainer.Initialize
baseTransmitter := dte.NewBaseTransmitter(transmitter, signerManager)           // BaseTransmitter
```

Desde la perspectiva del caso de uso, solo `BaseTransmitter` es visible. La estratificación garantiza que los detalles de firma, autenticación y HTTP estén completamente encapsulados.
