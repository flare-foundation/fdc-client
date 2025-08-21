# build executable
FROM golang:1.24.4 AS builder

WORKDIR /build

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container
COPY . .

# Build the applications
RUN go build -o /app/fdc-client main/main.go

FROM debian:latest AS execution

WORKDIR /app

# binary
COPY --from=builder /app/fdc-client .
# abis and system configs
COPY --from=builder /build/configs/abis /app/configs/abis
COPY --from=builder /build/configs/systemConfigs /app/configs/systemConfigs
# ssl certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


CMD ["./fdc-client" ]
