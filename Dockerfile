FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o slackbot main.go

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/slackbot .
ENTRYPOINT ["./slackbot"]
