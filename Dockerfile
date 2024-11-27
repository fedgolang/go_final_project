# Для запуска имейджа: docker run -d  -p 8080:8080 github.com/fedgolang/go_final_project
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/main .

COPY --from=builder /app/web ./web

EXPOSE 8080

ENV TODO_PORT=8080 \
    TODO_DBFILE=/app/scheduler.db \
    TODO_PASSWORD=JWT_PASS789

CMD ["./main"]