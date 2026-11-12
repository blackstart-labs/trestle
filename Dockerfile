FROM node:24.21.0-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY main.go ./
COPY internal ./internal
COPY migrations ./migrations
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trestle .

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/trestle /app/trestle
COPY LICENSE /app/
USER 10001:10001
ENV TRESTLE_BIND_ADDR=0.0.0.0:3000
EXPOSE 3000
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/trestle"]
