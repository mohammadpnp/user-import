FROM golang:1.25-alpine AS api-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o /out/api ./cmd/api

FROM alpine:3.21 AS api

RUN addgroup -S app && adduser -S -G app app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=api-builder /out/api /usr/local/bin/api

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]

FROM alpine:3.21 AS migrate-builder

ARG MIGRATE_VERSION=v4.18.3
ARG TARGETARCH

RUN apk add --no-cache ca-certificates curl tar
RUN case "${TARGETARCH}" in \
    amd64|arm64) ARCH="${TARGETARCH}" ;; \
    *) echo "unsupported target arch: ${TARGETARCH}" && exit 1 ;; \
  esac \
  && curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${ARCH}.tar.gz" \
  | tar -xz -C /tmp

FROM alpine:3.21 AS migrate

RUN addgroup -S app && adduser -S -G app app

COPY --from=migrate-builder /tmp/migrate /usr/local/bin/migrate
COPY migrations /migrations

USER app

ENTRYPOINT ["/usr/local/bin/migrate", "-path", "/migrations"]
