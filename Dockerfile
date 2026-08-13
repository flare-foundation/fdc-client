# build executable
FROM golang:1.26.6-trixie@sha256:ab563819a16cfe5faff0f96a8bb598fbb0e400ab2ac751996e60abcb23b106a3 AS builder

WORKDIR /build

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container
COPY . .

# Build the applications
RUN go build -o /app/fdc-client main/main.go

FROM debian:trixie@sha256:34cd9e9fd437c0a095ec39cb2e73422c9f30821b0d0848ed74fd0d43bae4d958 AS execution

# unprivileged runtime account
# home is never created — /nonexistent keeps $HOME from naming a real path
RUN groupadd --system --gid 10001 app \
    && useradd --system --uid 10001 --gid 10001 --no-create-home \
       --home-dir /nonexistent --shell /usr/sbin/nologin app

WORKDIR /app

# binary
COPY --from=builder /app/fdc-client .
# abis and system configs
COPY --from=builder /build/configs/abis /app/configs/abis
COPY --from=builder /build/configs/systemConfigs /app/configs/systemConfigs
# ssl certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# must match the port of rest_server.addr in the mounted userConfig.toml
EXPOSE 8080

# numeric — k8s runAsNonRoot cannot verify a named user
USER 10001:10001

#checkov:skip=CKV_DOCKER_2: Health check is handled by container orchestrator

CMD ["./fdc-client" ]
