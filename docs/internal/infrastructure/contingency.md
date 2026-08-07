# Servicios de Contingencia — Preparación y Transmisión de Eventos

> **Paquete:** `internal/infrastructure/adapters/contingency`

## Descripción General

Los servicios de contingencia de infraestructura gestionan la preparación, firma y envío de eventos de contingencia a Hacienda. Se componen de tres servicios internos que colaboran para completar el flujo:

1. **ContingencyEventService** — Orquestador principal
2. **ContingencyHTTPService** — Comunicación HTTP con Hacienda
3. **ContingencyTokenService** — Generación de tokens para firma de eventos

---

## ContingencyEventService

> **Archivo:** `contingency_event_service.go`

### Estructura

```go
type ContingencyEventService struct {
    authManager   auth.AuthManager
    repo          contingency.ContingencyRepositoryPort
    timeProvider  authPorts.TimeProvider
    tokenSvc      *contingencyTokenService
    httpSvc       *contingencyHTTPService
    connection    *drivers.DbConnection
}
```

### Método Principal

```go
func (s *ContingencyEventService) PrepareAndSendContingencyEvent(
    ctx context.Context,
    docs []ContingencyDocument,
) error
```

### Flujo Completo

```
PrepareAndSendContingencyEvent(ctx, docs)
  │
  ├── [1] Extraer detalles de los DTEs
  │     prepareDTEDetails(docs)
  │     → Por cada documento:
  │       {
  │         NoItem:         índice + 1,
  │         CodigoGeneracion: UUID del DTE,
  │         TipoDoc:        tipo de DTE (01, 03, etc.)
  │       }
  │
  ├── [2] Preparar motivo de contingencia
  │     prepareContingencyReason(ctx, doc)
  │     │
  │     ├── Obtener timestamp del primer documento pendiente
  │     │   repo.GetFirstContingencyTimestamp(ctx, branchID)
  │     │
  │     ├── Calcular rango de tiempo:
  │     │   FInicio: firstTimestamp - 60 segundos (buffer)
  │     │   FFin:    now + 10 segundos (buffer)
  │     │   HInicio: hora de FInicio
  │     │   HFin:    hora de FFin
  │     │
  │     └── Construir ContingencyReason:
  │         {
  │           TipoContingencia: tipo del documento,
  │           MotivoContingencia: motivo del documento,
  │           FInicio, FFin, HInicio, HFin
  │         }
  │
  ├── [3] Obtener información del emisor
  │     authManager.GetIssuerInfo(ctx, branchID)
  │     → NIT, NRC, nombre, tipo establecimiento
  │
  ├── [4] Construir evento de contingencia (versión 3)
  │     {
  │       identificacion: {
  │         version: 3,
  │         ambiente: "01",
  │         codigoGeneracion: UUID nuevo,
  │         fTransmision: fecha actual,
  │         hTransmision: hora actual
  │       },
  │       emisor: { nit, nombre, tipoEstablecimiento, ... },
  │       detalleDTE: [ ...dteDetails ],
  │       motivo: contingencyReason
  │     }
  │
  ├── [5] Firmar y enviar evento
  │     httpSvc.SignAndSend(ctx, event, nit, token)
  │     │
  │     ├── Éxito → return nil
  │     └── Duplicado → return ContingencyEventExistsError
  │
  └── [6] Retornar resultado
```

### Buffers de Tiempo

```
Timeline:
  ──|──────────────────|──────────────|──────|──
    │                  │              │      │
  FInicio         1er doc          ahora   FFin
  (-60s)          pendiente               (+10s)
```

- **-60 segundos antes del primer documento**: Se asegura que el rango cubra cualquier documento que pueda haber llegado justo antes.
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
    httpClient   *http.Client  // timeout: 30s
}
```

### Método Principal

```go
func (s *contingencyHTTPService) SignAndSend(
    ctx context.Context,
    event ContingencyEvent,
    nit string,
    token string,
) (isDuplicate bool, err error)
```

### Flujo

```
SignAndSend(ctx, event, nit, token)
  │
  ├── [1] Serializar evento a JSON
  │     json.Marshal(event) → jsonData
  │
  ├── [2] Firmar evento
  │     signer.SignDTE(ctx, jsonData, nit) → signedEvent
  │
  ├── [3] Obtener token de Hacienda
  │     haciendaAuth.GetOrCreateHaciendaToken(ctx, token)
  │
  ├── [4] Enviar HTTP POST
  │     POST {contingencyURL}
  │     Headers:
  │       Authorization: Bearer {haciendaToken}
  │       Content-Type: application/json
  │     Body: { nit, documento: signedEvent }
  │
  └── [5] Manejar respuesta
        handleResponse(resp)
        ├── Éxito → return (false, nil)
        ├── Duplicado → return (true, nil)
        └── Error → return (false, error)
```

### Detección de Duplicados

Se detectan eventos de contingencia duplicados buscando frases específicas en la respuesta de Hacienda:

```go
// Se busca en el mensaje y las observaciones:
"ya existe evento"    // Mensaje estándar de duplicado
"ya existe envento"   // Typo conocido en la API de Hacienda
```

Cuando se detecta un duplicado, se retorna `isDuplicate=true` en lugar de un error. El servicio de dominio de contingencia decide qué hacer con esta información (generalmente continúa con la transmisión del lote).

---

## ContingencyTokenService

> Servicio interno referenciado en `contingency_event_service.go`

### Propósito

Se genera un token compatible con el sistema para poder firmar el evento de contingencia. El evento necesita firmarse con las credenciales del emisor, que están cifradas en cache bajo el token del sistema.

### Flujo

```
GetContingencyToken(ctx, systemToken)
  │
  ├── Recuperar credenciales del cache
  │   cache.GetCredentials(systemToken)
  │
  └── Retornar token utilizable para firma
```

---

## Modelo del Evento de Contingencia

### Estructura JSON

```json
{
  "identificacion": {
    "version": 3,
    "ambiente": "01",
    "codigoGeneracion": "UUID-DEL-EVENTO",
    "fTransmision": "2024-01-15",
    "hTransmision": "14:30:00"
  },
  "emisor": {
    "nit": "06141234567890",
    "nombre": "Empresa S.A. de C.V.",
    "nombreResponsable": "Juan Pérez",
    "tipoDocResponsable": "36",
    "numeroDocResponsable": "06141234567890",
    "tipoEstablecimiento": "01"
  },
  "detalleDTE": [
    {
      "noItem": 1,
      "codigoGeneracion": "UUID-DEL-DTE-1",
      "tipoDoc": "01"
    },
    {
      "noItem": 2,
      "codigoGeneracion": "UUID-DEL-DTE-2",
      "tipoDoc": "01"
    }
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

---

## Notas

1. **Tolerancia a duplicados**: El sistema maneja graciosamente los eventos duplicados. Si Hacienda reporta que el evento ya existe, se continúa con el flujo de transmisión del lote sin error.
2. **Typo en la API**: La API de Hacienda tiene un typo conocido (`"envento"` en lugar de `"evento"`). El sistema detecta ambas variantes.
3. **Buffers temporales**: Los márgenes de 60s y 10s previenen rechazos por diferencias de reloj entre el servidor del sistema y el de Hacienda.
4. **Firma del evento**: El evento de contingencia se firma con el mismo servicio de firma que los DTEs individuales, usando el NIT del emisor.
