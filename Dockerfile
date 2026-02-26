FROM alpine:3.21 AS builder

ARG MIGRATE_VERSION=v4.18.3
ARG TARGETARCH

RUN apk add --no-cache ca-certificates curl tar
RUN case "${TARGETARCH}" in \
    amd64|arm64) ARCH="${TARGETARCH}" ;; \
    *) echo "unsupported target arch: ${TARGETARCH}" && exit 1 ;; \
  esac \
  && curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${ARCH}.tar.gz" \
  | tar -xz -C /tmp

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /tmp/migrate /usr/local/bin/migrate
COPY migrations /migrations

USER app

ENTRYPOINT ["/usr/local/bin/migrate", "-path", "/migrations"]
