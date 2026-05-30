FROM alpine:3.23.4 AS base

# Create a non-root user and group for runtime
RUN addgroup -S projdocs && \
    adduser -S -G projdocs -H -D projdocs

# Trust anchors for TLS verification
RUN apk add --no-cache ca-certificates


FROM base AS build
ARG VERSION

RUN if [ -z "${VERSION}" ]; then \
      echo "VERSION is unset" && exit 1; \
    else \
      echo "building for VERSION=$VERSION"; \
    fi

RUN case "$(apk --print-arch)" in \
      aarch64) ARCH=arm64 ;; \
      x86_64)  ARCH=amd64 ;; \
      *) \
        echo "architecture $(apk --print-arch) is not supported" && \
        exit 1 ;; \
    esac && \
    echo "ARCH=${ARCH}" > /etc/build-env

RUN apk add --no-cache curl

RUN . /etc/build-env && \
    curl -fsSL \
      "https://github.com/projdocs/api/releases/download/${VERSION}/projdocs-api-${VERSION}-linux-${ARCH}" \
      -o /usr/local/bin/projdocs-api && \
    chmod 0755 /usr/local/bin/projdocs-api


FROM base AS run

COPY --from=build /usr/local/bin/projdocs-api /usr/local/bin/projdocs-api

# Drop to non-root for all subsequent instructions and at runtime
USER projdocs

ENTRYPOINT ["/usr/local/bin/projdocs-api"]