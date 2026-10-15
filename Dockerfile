FROM rust:1.99.0-bookworm@sha256:114c7a4425406451c2866b6aafe69fe29b1b298832db1277d411ac73c82d04d6 AS build
WORKDIR /build
COPY Cargo.toml Cargo.lock rust-toolchain.toml ./
COPY src ./src
COPY migrations ./migrations
COPY LICENSE ./LICENSE
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
