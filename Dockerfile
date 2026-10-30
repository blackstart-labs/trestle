FROM node:24.21.0-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

FROM rust:1.99.0-bookworm@sha256:114c7a4425406451c2866b6aafe69fe29b1b298832db1277d411ac73c82d04d6 AS build
WORKDIR /build
COPY Cargo.toml Cargo.lock rust-toolchain.toml ./
COPY src ./src
COPY migrations ./migrations
COPY LICENSE ./LICENSE
COPY build.rs ./
COPY --from=web /web/dist ./web/dist
RUN cargo build --locked --release

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587
WORKDIR /app
COPY --from=build /build/target/release/trestle /app/trestle
COPY LICENSE THIRD_PARTY_NOTICES.md /app/
USER 10001:10001
ENV TRESTLE_BIND_ADDR=0.0.0.0:3000
EXPOSE 3000
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/trestle"]
