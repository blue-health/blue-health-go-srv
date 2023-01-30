FROM golang:1.19-alpine AS builder

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . .

WORKDIR /app/tasks/migrate

RUN GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -o bin/billing-srv-migrate-linux-amd64 main.go

# ---

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/tasks/migrate/bin/billing-srv-migrate-linux-amd64 /

CMD ["/billing-srv-migrate-linux-amd64"] 
