FROM golang:1.25.8-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git
COPY . .

RUN mkdir -p assets/logs/ && \
    touch assets/logs/dte_microservice.log

RUN mkdir /app/release && \
    cp -r /app/assets /app/release/ && \
    cp /app/.env /app/release/

RUN CGO_ENABLED=0 go mod tidy && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o dte-app cmd/main.go

RUN cp /app/dte-app /app/release/

FROM gcr.io/distroless/static-debian12
LABEL authors="chainedpixel"

WORKDIR /app

COPY --from=builder /app/release /app/

EXPOSE 7319

CMD ["/app/dte-app"]