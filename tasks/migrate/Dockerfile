FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . .

WORKDIR /app/tasks/migrate

RUN GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -o bin/blue-health-go-srv-migrate-linux-amd64 main.go

# ---

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/tasks/migrate/bin/blue-health-go-srv-migrate-linux-amd64 /

CMD ["/blue-health-go-srv-migrate-linux-amd64"] 
