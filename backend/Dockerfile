# --- build stage
FROM golang:1.22 AS builder

WORKDIR /app
COPY . .

RUN go mod init backend || true
RUN go mod tidy
RUN go build -o backend .

# --- final stage
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/backend .

EXPOSE 8080
ENTRYPOINT [ "./backend" ]