FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w" \
  -o /server \
  ./cmd/api

FROM alpine:3.22

RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /server ./server

RUN chown -R app:app /app

USER app

EXPOSE 8080

ENTRYPOINT [ "./server" ]
