FROM golang:1.23-alpine AS builder

WORKDIR /app

# Instalar herramientas necesarias
RUN apk add --no-cache git

# Copiar proyecto
COPY . .

# Verificar que los archivos de traducción existan
RUN echo "=== Verificando archivos de traducción ===" && \
    find /app -name "*.yaml" -o -name "*.yml" 2>/dev/null && \
    ls -la /app/internal/ 2>/dev/null || echo "Directorio internal no encontrado"

WORKDIR /app
RUN go build -o dte-app cmd/main.go

# Verificar dónde quedó el binario
RUN echo "=== Buscando binario compilado ===" && \
    find / -name "dte-app" -type f 2>/dev/null

# Imagen final
FROM alpine:latest
LABEL authors="Marlon"

WORKDIR /app

# Copiar binario compilado
COPY --from=builder /app/dte-app /app/

# Copiar archivos de configuración y traducción
COPY --from=builder /app/internal/i18n /app/internal/i18n

RUN ls -la /app/

RUN apk add --no-cache ca-certificates tzdata

# Crear directorio de logs en la ubicación correcta y dar permisos
RUN mkdir -p /app/pkg/shared/logs/ && \
    touch /app/pkg/shared/logs/dte_microservice.log && \
    chmod -R 755 /app/pkg/shared/logs/

# Configuración
COPY .env /app/

RUN chmod +x /app/dte-app

EXPOSE 7319

CMD ["sh", "-c", "ls -la /app && exec /app/dte-app"]