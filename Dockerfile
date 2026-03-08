FROM golang:1.25.0-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o admin-service ./cmd/server

FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=builder /workspace/admin-service /admin-service

EXPOSE 8080

ENTRYPOINT ["/admin-service"]
