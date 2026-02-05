FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go mod init ota-server || true
RUN go mod tidy
RUN go build -o ota-server .

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/ota-server .
COPY firmware.bin .
EXPOSE 8081
ENTRYPOINT [ "./ota-server" ]