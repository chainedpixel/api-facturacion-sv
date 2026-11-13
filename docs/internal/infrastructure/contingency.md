# Servicios de Contingencia — Preparación y Transmisión de Eventos

> **Paquete:** `internal/infrastructure/adapters/contingency`

## Descripción General

Los servicios de contingencia de infraestructura gestionan la preparación, firma y envío de eventos de contingencia a Hacienda. Se componen de tres servicios internos que colaboran para completar el flujo:

1. **ContingencyEventService** — Orquestador principal
2. **ContingencyHTTPService** — Firma y comunicación HTTP con Hacienda
3. **ContingencyTokenService** — Generación de tokens equivalentes a la sesión original

La orquestación de más alto nivel (almacenamiento de documentos, ventana de 72 horas, lotes) está en `internal/domain/dte/contingency/` y la dispara el job descrito en [jobs.md](jobs.md). Más abajo se resume ese ciclo porque determina cuándo se usan estos servicios.

---

## Ciclo de Retransmisión

`ContingencyService.RetransmitPendingDocuments` (dominio) es lo que ejecuta el `RetransmissionJob`:

```
RetransmitPendingDocuments(ctx)
  │
  ├── [1] GetPending(limit = MH_MAX_BATCH_SIZE)
  │     Documentos con dte_details.status = PENDING, ordenados por cercanía al vencimiento
  │     (primero los que ya tienen event_stamped_at, el más antiguo antes)
  │
  ├── [2] expireOverdueDocuments
  │     Para cada documento con event_stamped_at: si han pasado más de 72 horas
  │     (IsContingencyWindowExpired) →
  │       · UpdateBatch(status = EXPIRED, lote "expired") con la observación
  │         "Not transmitted within 72 hours after the contingency event stamp (fecha)"
  │       · Se publica RetransmissionJobFailedEvent (JobName = contingency_72h_expired,
  │         código CONTINGENCY_72H_EXPIRED) para avisar al administrador por correo
  │       · El documento se considera no emitido y sale del ciclo
  │     Los documentos sin sello de evento nunca vencen.
  │
  ├── [3] alertLongContingency
  │     Si el pendiente más antiguo (created_at) supera 72 horas (3 días de contingencia
  │     según la constante TechnicalReportThreshold) → log de advertencia y
  │     RetransmissionJobFailedEvent (JobName = contingency_technical_report)
  │     indicando que se requiere un informe técnico de contingencia
  │
  ├── [4] Agrupar por NIT del emisor y tipo de DTE
  │
  └── [5] Por cada NIT:
        ├── PrepareAndSendContingencyEvent (este paquete)
        │     ├── Error ContingencyEventExistsError → verifyAndUpdateExistingDocuments
        │     │     (consulta el estado de cada documento en Hacienda y actualiza)
        │     │     y se continúa con la transmisión
        │     └── Otro error → se registra y se pasa al siguiente NIT
        └── Por cada tipo de DTE: lotes de TransmissionConfig.BatchSize
              (firma → TransmitBatch → VerifyContingencyBatchStatus; ver transmitter.md)
```

Referencia normativa (Normativa de Cumplimiento DTE v2.0): el numeral 13.2.2 establece que los documentos en estado transitorio y los eventos generados en contingencia deben transmitirse en el plazo de setenta y dos horas contadas a partir de la obtención del Sello de Recepción del Evento de Contingencia; el numeral 13.2.1.1 exige un Informe Técnico de Contingencia cuando la contingencia persiste por más de tres días consecutivos.

Las constantes de la ventana están en `contingency_window.go`: `TransmissionWindow = 72h` (desde el sello del evento), `TechnicalReportThreshold = 72h` y `ContingencyExpiredCode = "CONTINGENCY_72H_EXPIRED"`.

---

## ContingencyEventService

> **Archivo:** `contingency_event_service.go`

### Estructura

```go
type ContingencyEventService struct {
    authManager  auth.AuthManager
    repo         contingency.ContingencyRepositoryPort
    timeProvider authPorts.TimeProvider
    dteManager   dte_documents.DTEManager
    tokenSvc     *contingencyTokenService
    httpSvc      *contingencyHTTPService
    connection   *drivers.DbConnection
}
```

### Método Principal

```go
func (s *ContingencyEventService) PrepareAndSendContingencyEvent(
    ctx context.Context,
    docs []dte.ContingencyDocument,
) error
```

### Flujo Completo

```
PrepareAndSendContingencyEvent(ctx, docs)
  │
  ├── [0] Validar: lista no vacía y conexión a BD (Ping)
  │
  ├── [1] Obtener datos del emisor
  │     authManager.GetIssuer(ctx, docs[0].BranchID)
  │
  ├── [2] Preparar motivo de contingencia
  │     prepareContingencyReason(ctx, docs[0])
  │     ├── repo.GetFirstContingencyTimestamp(ctx, branchID)   (created_at más antiguo pendiente)
  │     └── Rango:
  │           fInicio: fecha del primer documento
  │           hInicio: hora del primer documento - 60 s
  │           fFin / hFin: ahora / ahora + 10 s
  │           tipoContingencia y motivoContingencia: los del primer documento
  │
  ├── [3] Construir evento (ContingencyEvent)
  │     identificacion: { version: 4, ambiente, codigoGeneracion (UUID nuevo en mayúsculas),
  │                       fTransmision, hTransmision }
  │     emisor:         { nit, nombre, nombreResponsable, tipoDocResponsable (NIT),
  │                       numeroDocResponsable, tipoEstablecimiento, telefono, correo,
  │                       codEstableMH, codPuntoVentaMH }
  │     detalleDTE:     [ { noItem, codigoGeneracion, tipoDoc } ... ]
  │     motivo:         ver [2]
  │
  └── [4] sendContingencyEvent
        ├── authManager.GetByNIT → cliente
        ├── tokenSvc.GenerateTokenForUser(cliente, branchID)
        ├── httpSvc.SignAndSend(ctx, event, nit, token) → (isDuplicate, evidence, err)
        │
        ├── isDuplicate == false:
        │     ├── registerEvent  → dte_events (EventType CONTINGENCY, JWS, sello,
        │     │                    fecha de procesamiento y respuesta completa)
        │     └── markDocumentsStamped → contingency_documents.event_stamped_at
        │            (fhProcesamiento de Hacienda; si no se puede interpretar, la hora actual)
        │
        └── isDuplicate == true → ContingencyEventExistsError{ Documents }
```

Si `registerEvent` falla, solo se registra una advertencia: el evento ya fue recibido por Hacienda. `markDocumentsStamped` solo escribe `event_stamped_at` donde estaba nulo (ver [repositories.md](repositories.md)), de modo que el reenvío del evento no reinicia la ventana de 72 horas. Cuando Hacienda informa que el evento ya existe, no se registra evento ni sello.

### Buffers de Tiempo

```
Timeline:
  ──|──────────────────|──────────────|──────|──
    │                  │              │      │
  hInicio         1er doc          ahora   hFin
  (-60s)          pendiente               (+10s)
```

- **-60 segundos antes del primer documento**: Se asegura que el rango cubra cualquier documento que pueda haber llegado justo antes. El desfase se aplica solo a la hora (`hInicio`); `fInicio` es la fecha del primer documento sin ajuste.
- **+10 segundos después de ahora**: Se previenen problemas de sincronización de reloj con Hacienda.

---

## ContingencyHTTPService

> **Archivo:** `contingency_http_service.go`

### Estructura

```go
type contingencyHTTPService struct {
    haciendaAuth haciendaPorts.HaciendaAuthManager
    signer       haciendaPorts.SignerManager
    cache        authPorts.CacheManager
    httpClient   *http.Client  // timeout: 8s, pooling (MaxIdleConns 100, IdleConnTimeout 90s)
}
```

### Método Principal

```go
func (s *contingencyHTTPService) SignAndSend(
    ctx context.Context,
    event *models.ContingencyEvent,
    nit string,
    token string,
) (isDuplicate bool, evidence *dte.TransmissionEvidence, err error)
```

### Flujo

```
SignAndSend(ctx, event, nit, token)
  │
  ├── [1] Token de Hacienda
  │     cache.GetCredentials(token) → credenciales cifradas
  │     haciendaAuth.GetOrCreateHaciendaTokenWithCreds(ctx, token, creds)
  │
  ├── [2] Firmar evento
  │     signer.SignDTE(ctx, json(event), nit) → JWS
  │
  ├── [3] Enviar HTTP POST a {MH_CONTINGENCY_URL}
  │     Headers: Authorization: {haciendaToken}, Content-Type: application/json
  │     Body:    { "nit": nit, "documento": JWS }
  │
  └── [4] handleResponse(resp) + buildEventEvidence
        ├── HTTP 200 y estado distinto de RECHAZADO → éxito (false)
        ├── HTTP 200 con estado RECHAZADO:
        │     duplicado (mensaje u observaciones) → isDuplicate = true
        │     otro rechazo → error "event rejected"
        ├── HTTP != 200:
        │     mensaje con "ya existe evento" / "ya existe envento" → isDuplicate = true
        │     mensaje con "no superadas" → error "validation failed"
        │     otro → error "unexpected status"
        └── evidence: { SignedDocument = JWS, ReceptionStamp = selloRecibido,
                        ProcessingDate = fechaHora, HaciendaResponse = respuesta completa }
```

La `evidence` se devuelve incluso cuando hay error o duplicado; el orquestador solo la persiste cuando no es duplicado. La fecha de procesamiento se lee del campo `fechaHora` de la respuesta del evento (formato `dd/MM/yyyy HH:mm:ss` según el Manual Técnico de Integración v2.0; se interpreta con `utils.ParseProcessingDate`), a diferencia de las respuestas de recepción de DTE, que usan `fhProcesamiento`.

### Detección de Duplicados

Se detectan eventos de contingencia duplicados buscando frases específicas en el mensaje (y en la primera observación cuando el cuerpo es un `RECHAZADO`):

```go
"ya existe evento"    // Mensaje estándar de duplicado
"ya existe envento"   // Typo conocido en la API de Hacienda
```

Cuando se detecta un duplicado, se retorna `isDuplicate=true` en lugar de un error. El servicio de dominio decide qué hacer: consulta el estado de los documentos en Hacienda (`verifyAndUpdateExistingDocuments`) y continúa con la transmisión del lote.

---

## ContingencyTokenService

> **Archivo:** `contingency_token_service.go`

### Propósito

Se genera un JWT equivalente al de la sesión original del usuario, para poder obtener las credenciales de Hacienda cifradas en cache (que están indexadas por token) y autenticarse contra Hacienda cuando no hay una petición HTTP en curso (por ejemplo, desde el job).

### Flujo

```
GenerateTokenForUser(client, branchID)   /   GenerateTokenForBranch(branch)
  │
  ├── cache.Get("token:timestamps:{userID}") → { IssuedAt, ExpiresAt }
  │     (guardado por JWTService.GenerateToken al iniciar sesión)
  │
  └── Se firma un JWT HS256 con los mismos claims (sub, branch_sub, auth_type, nit)
      y los mismos iat/exp originales → mismo token que el de la sesión, por lo que la
      clave de cache de credenciales coincide
```

Si la sesión del usuario ya expiró (no hay marcas en cache), la generación falla y la contingencia no puede transmitirse hasta que el usuario inicie sesión de nuevo.

---

## Modelo del Evento de Contingencia

### Estructura JSON

```json
{
  "identificacion": {
    "version": 4,
    "ambiente": "01",
    "codigoGeneracion": "UUID-DEL-EVENTO",
    "fTransmision": "2024-01-15",
    "hTransmision": "14:30:00"
  },
  "emisor": {
    "nit": "06141234567890",
    "nombre": "Empresa S.A. de C.V.",
    "nombreResponsable": "Empresa S.A. de C.V.",
    "tipoDocResponsable": "36",
    "numeroDocResponsable": "06141234567890",
    "tipoEstablecimiento": "01",
    "telefono": "22222222",
    "correo": "correo@empresa.com",
    "codEstableMH": "M001",
    "codPuntoVentaMH": "P001"
  },
  "detalleDTE": [
    { "noItem": 1, "codigoGeneracion": "UUID-DEL-DTE-1", "tipoDoc": "01" },
    { "noItem": 2, "codigoGeneracion": "UUID-DEL-DTE-2", "tipoDoc": "01" }
  ],
  "motivo": {
    "fInicio": "2024-01-15",
    "fFin": "2024-01-15",
    "hInicio": "14:29:00",
    "hFin": "14:30:10",
    "tipoContingencia": 2,
    "motivoContingencia": "Falla en el servicio de Hacienda"
  }
}
```

El código de generación del evento es un UUID nuevo en mayúsculas, distinto del de los documentos. `nombreResponsable` toma la razón social del emisor y `numeroDocResponsable` su NIT. `codEstableMH` y `codPuntoVentaMH` son los códigos asignados por Hacienda de la sucursal (nulos si no están configurados). `motivoContingencia` es nulo si el documento no tiene motivo.

---

## Notas

1. **Tolerancia a duplicados**: Si Hacienda reporta que el evento ya existe, se verifica el estado de los documentos y se continúa con el flujo de transmisión del lote sin error.
2. **Typo en la API**: La API de Hacienda tiene un typo conocido (`"envento"` en lugar de `"evento"`). El sistema detecta ambas variantes.
3. **Buffers temporales**: Los márgenes de 60 s y 10 s previenen rechazos por diferencias de reloj entre el servidor del sistema y el de Hacienda.
4. **Firma del evento**: El evento de contingencia se firma con el mismo servicio de firma que los DTEs individuales, usando el NIT del emisor.
5. **Evidencia del evento**: El JWS, el sello, la fecha y la respuesta completa de Hacienda se guardan en `dte_events` (tipo `CONTINGENCY`, `related_dte_id` nulo).
6. **Ventana de 72 horas**: Se cuenta desde el sello del evento (`event_stamped_at`), no desde la creación del documento. Los documentos no transmitidos dentro de la ventana pasan a `EXPIRED` y se avisa al administrador.
