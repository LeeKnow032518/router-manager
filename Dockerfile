FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o router-manager main.go

FROM alpine:latest AS final
WORKDIR /root/

COPY --from=builder /app/router-manager .
COPY --from=builder /app/db/migration /root/db/migration
COPY --from=builder /app/logs/app.log /var/log/app.log

EXPOSE 8080 50051
CMD ["./router-manager"]