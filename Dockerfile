FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG APP_NAME
ARG APP_VERSION
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Name=goph-${APP_NAME} -X main.Version=${APP_VERSION} -X 'main.BuildDate=$(date +'%Y/%m/%d %H:%M:%S')'" -o bin/app ./cmd/${APP_NAME}

FROM alpine:3.24

RUN adduser -D -u 10001 appuser

USER appuser

WORKDIR /app

COPY --from=builder /src/bin/app .
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/web ./web
COPY --from=builder /src/docs ./docs

EXPOSE 8080

CMD ["./app" ]
