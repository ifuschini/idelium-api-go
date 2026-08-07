FROM golang:1.26.5-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599 AS builder

ARG VERSION=dev
ARG SOURCE_REVISION=unknown

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X github.com/idelium/idelium-api-go/internal/buildinfo.Version=${VERSION} -X github.com/idelium/idelium-api-go/internal/buildinfo.Commit=${SOURCE_REVISION}" \
    -o /out/idelium-api-go ./cmd/api

FROM scratch

ARG SOURCE_REVISION=unknown
LABEL org.opencontainers.image.source="https://github.com/idelium/idelium-api-go" \
      org.opencontainers.image.revision="${SOURCE_REVISION}"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/idelium-api-go /idelium-api-go

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/idelium-api-go"]
