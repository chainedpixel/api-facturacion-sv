# Paquete pkg/ — Componentes Compartidos

**Ubicación:** `pkg/`

## Descripción General

El directorio `pkg/` contiene componentes reutilizables que no pertenecen a una capa específica de la arquitectura. Se organizan en cinco áreas:

| Paquete | Propósito |
|---|---|
| `pkg/mapper/` | Sistema de mapeo: interfaces, factory, adaptadores genéricos |
| `pkg/mapper/request_mapper/` | Mappers HTTP Request → Modelo de Dominio |
| `pkg/mapper/response_mapper/` | Mappers Modelo de Dominio → Formato Hacienda |
| `pkg/shared/logs/` | Logger estructurado con logrus (colores, archivo, niveles) |
| `pkg/shared/shared_error/` | ServiceError: wrapper de errores con traducción y modo debug |
| `pkg/shared/utils/` | Utilidades: tiempo, punteros, números a letras, extractors JSON |
| `pkg/error/` | Errores centinela de logging |

---

## Documentación Detallada

### Sistema de Mapeo

| Documento | Descripción |
|---|---|
| [Mapper Core](mapper.md) | DTEMapper interface, ResponseMapperFunc, MapperFactory, adaptadores genéricos |
| [Request Mappers](request-mappers.md) | Transformación HTTP → Dominio para los 8 tipos de DTE, estructuras de request, mappers comunes |
| [Response Mappers](response-mappers.md) | Transformación Dominio → JSON Hacienda para los 8 tipos de DTE, estructuras de respuesta |

### Infraestructura Compartida

| Documento | Descripción |
|---|---|
| [Logs](logs.md) | Logger con logrus, CustomFormatter (colores ANSI), WriteHook (archivo), niveles de log, inicialización |
| [Errores (ServiceError)](shared-error.md) | ServiceError (wrapper con contexto/traducción), errores centinela por capa (dominio, infra, config) |
| [Utilidades](utils.md) | TimeNow (zona horaria), punteros, InLetters (montos a español), extractors JSON, FindProjectRoot |

---

## Flujo de Datos

```
HTTP Request (JSON)
  │
  ├── Deserializar a CreateXXXRequest (pkg/mapper/request_mapper/structs/)
  │
  ├── MapperFactory.CreateXXXMapperAdapter()
  │     └── DTEMapper.MapToDomainModel(request, issuer)
  │           ├── Validar campos requeridos
  │           ├── Mapear componentes comunes (identificación, emisor, receptor)
  │           ├── Mapear componentes específicos (ítems, resumen)
  │           └── Retornar modelo de dominio
  │
  ├── [Procesamiento de dominio: validación, cálculos, números de control]
  │
  ├── MapperFactory.GetXXXResponseMapper()
  │     └── ResponseMapperFunc(domainModel)
  │           ├── Mapear a estructuras de respuesta (pkg/mapper/response_mapper/structs/)
  │           └── Retornar formato JSON Hacienda
  │
  └── Transmitir a Hacienda
```

---

## Errores Centinela

Para la documentación completa del sistema de errores (ServiceError + errores centinela de todas las capas), ver [Errores (ServiceError)](shared-error.md).
