FROM golang:1.26.6-bookworm@sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976 AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X github.com/idelium/idelium-api-go/internal/buildinfo.Version=${VERSION} -X github.com/idelium/idelium-api-go/internal/buildinfo.Commit=${SOURCE_REVISION}" \
    -o /out/idelium-worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X github.com/idelium/idelium-api-go/internal/buildinfo.Version=${VERSION} -X github.com/idelium/idelium-api-go/internal/buildinfo.Commit=${SOURCE_REVISION}" \
    -o /out/idelium-migrate ./cmd/migrate

FROM scratch

ARG SOURCE_REVISION=unknown
LABEL org.opencontainers.image.source="https://github.com/idelium/idelium-api-go" \
      org.opencontainers.image.revision="${SOURCE_REVISION}"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/idelium-api-go /idelium-api-go
COPY --from=builder /out/idelium-worker /idelium-worker
COPY --from=builder /out/idelium-migrate /idelium-migrate

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/idelium-api-go"]
