FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . .

RUN GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -o bin/blue-health-go-srv-linux-amd64 main.go

# ---

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/bin/blue-health-go-srv-linux-amd64 /

EXPOSE 8080 8081 9090

CMD ["/blue-health-go-srv-linux-amd64"]
