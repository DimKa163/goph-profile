FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG APP_NAME

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Name=goph-${APP_NAME} -X main.Version=v1.0.0 -X 'main.BuildDate=$(date +'%Y/%m/%d %H:%M:%S')'" -o bin/app ./cmd/${APP_NAME}

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /src/bin/app .
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/web ./web

EXPOSE 8080

CMD ["./app" ]
