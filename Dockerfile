# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bunny-ddns ./cmd/bunny-ddns

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /bunny-ddns /bunny-ddns
ENTRYPOINT ["/bunny-ddns"]
