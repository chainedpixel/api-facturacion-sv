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
    CheckStatus(ctx context.Context, document interface{}, nit string) (*models.TransmitResult, error)
}
```

Esta es la interfaz que llaman los casos de uso. Abstrae el comportamiento de reintento con backoff para que los casos de uso no necesiten preocuparse por conteos de reintentos ni intervalos.

**`RetryTransmission`**

Llamado por `GenericDTEUseCase.Create` e `InvalidationUseCase.InvalidateDocument`. Acepta:

- `document interface{}` — el struct del documento en formato MH completamente poblado (o struct de anulación)
- `token string` — el JWT del sistema en crudo, reenviado a `HaciendaAuthManager` para obtener un token de Hacienda
- `nit string` — el NIT del emisor, necesario por el servicio de firma

Devuelve `*models.TransmitResult` en caso de éxito, que contiene `ReceptionStamp`, `Status`, `ProcessingDate` y mensajes de observación.

**`CheckStatus`**

Llamado durante la retransmisión por contingencia para verificar si un documento enviado previamente fue procesado por Hacienda. Ver `ContingencyService.verifyAndUpdateExistingDocuments`.

**Implementación de infraestructura:** `RetryTransmitterService` (envuelve `MHTransmitter`)

---

## `SignerManager`

**Archivo:** `internal/application/ports/base_transmit.go`

```go
type SignerManager interface {
    SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
```

Usado dentro de `MHTransmitter.Transmit` (indirectamente a través del pipeline de procesadores) para firmar el JSON del documento serializado antes de la transmisión. La firma es manejada por un microservicio externo de Spring Boot.

**Implementación de infraestructura:** `signer.DTESigner` en `internal/infrastructure/adapters/signing/signer/dte_signer.go`

---

## `DTETransmitter`

**Archivo:** `internal/application/ports/dte_hacienda_transmitter_port.go`

```go
type DTETransmitter interface {
    Transmit(context.Context, interface{}, string, string) (*models.TransmitResult, error)
    CheckDocumentStatus(context.Context, interface{}, string) (*models.TransmitResult, error)
    SendToHacienda(context.Context, *models.HaciendaRequest, string) (*models.HaciendaResponse, error)
}
```

Esta es la interfaz de transmisor de nivel más bajo. A diferencia de `BaseTransmitter`, no incluye lógica de reintento — eso es responsabilidad de `BaseTransmitter` que envuelve a este.

- `Transmit` — firma el documento, construye la petición a Hacienda y llama a `SendToHacienda`
- `CheckDocumentStatus` — consulta el endpoint de consulta de Hacienda para el estado actual de un documento
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

Esta interfaz gestiona el token OAuth de Hacienda de forma independiente al JWT del sistema. Los tokens de Hacienda son de corta duración y se almacenan en caché en Redis.

**`GetOrCreateHaciendaToken`**

Usado durante la transmisión normal (no por lotes). La implementación:
1. Busca en Redis un token de Hacienda en caché con clave `hacienda:token:<systemToken>`
2. Si no se encuentra, extrae `AuthClaims` del contexto para obtener el NIT
3. Llama a `AuthManager.GetHaciendaCredentials` para recuperar el usuario/contraseña de Hacienda almacenado para ese NIT
4. Hace un POST a la URL de autenticación de Hacienda usando campos form-encoded `user` + `pwd`
5. Almacena en caché el token devuelto en Redis con un TTL de 24 horas

**`GetOrCreateHaciendaTokenWithCreds`**

Usado durante la retransmisión por lotes/contingencia, donde las credenciales se pasan explícitamente en lugar de consultarse desde el contexto.

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
// Adaptador de firma → satisface SignerManager
signer := signer.NewDTESigner(authRepo)

// Adaptador de autenticación de Hacienda → satisface HaciendaAuthManager
haciendaAuth := signing.NewHaciendaAuthService(cache, authManager)

// Transmisor de bajo nivel → satisface DTETransmitter
mhTransmitter := transmitter.NewMHTransmitter(haciendaAuth, failedSeqRepo)

// Wrapper de reintento → satisface BaseTransmitter
retryTransmitter := retry.NewRetryTransmitterService(mhTransmitter, signer)
```

Desde la perspectiva del caso de uso, solo `BaseTransmitter` es visible. La estratificación garantiza que los detalles de firma, autenticación y HTTP estén completamente encapsulados.
