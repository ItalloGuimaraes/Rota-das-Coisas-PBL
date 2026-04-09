# Estágio de Compilação
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
# O argumento TARGET define qual pasta dentro de cmd/ vamos compilar
ARG TARGET
RUN go build -o main ./cmd/${TARGET}/main.go

# Estágio Final (Imagem Leve para o Laboratório)
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .

# Variável para o código saber que está no Docker
ENV ENV=DOCKER

# Comando padrão
CMD ["./main"]