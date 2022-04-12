FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY ./ ./

WORKDIR /app/tasks/seed

RUN apk add --update make && make compile

# ---

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/tasks/seed/bin/blue-health-go-srv-seed-linux-amd64 /

CMD ["/blue-health-go-srv-seed-linux-amd64"] 
