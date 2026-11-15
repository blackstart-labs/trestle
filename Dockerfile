FROM --platform=$BUILDPLATFORM node:24.22.0-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY main.go ./
COPY internal ./internal
COPY migrations ./migrations
COPY --from=web /web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /out/trestle .

FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/trestle /app/trestle
COPY LICENSE /app/
USER 10001:10001
ENV TRESTLE_BIND_ADDR=0.0.0.0:3000
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -fsS "http://127.0.0.1:${TRESTLE_BIND_ADDR##*:}/health/ready" || exit 1
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 CMD ["/app/trestle", "healthcheck"]
ENTRYPOINT ["/app/trestle"]
