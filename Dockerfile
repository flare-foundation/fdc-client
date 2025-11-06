# build executable
FROM golang:1.25.1-trixie@sha256:61226c61f37cb86253c4ac486ef22c47f14bfddb8f60bb4805bfc165001be758 AS builder

WORKDIR /build

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container
COPY . .

# Build the applications
RUN go build -o /app/fdc-client main/main.go

FROM debian:trixie@sha256:fd8f5a1df07b5195613e4b9a0b6a947d3772a151b81975db27d47f093f60c6e6 AS execution

WORKDIR /app

# binary
COPY --from=builder /app/fdc-client .
# abis and system configs
COPY --from=builder /build/configs/abis /app/configs/abis
COPY --from=builder /build/configs/systemConfigs /app/configs/systemConfigs
# ssl certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


CMD ["./fdc-client" ]
