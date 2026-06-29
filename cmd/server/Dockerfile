FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:3.20

WORKDIR /server

COPY --from=builder /src/server .
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

CMD ["./server" ]
