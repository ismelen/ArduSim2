FROM rust:1.85-slim AS build

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src/ ./src/

RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

COPY --from=build /app/target/release/network_module .

EXPOSE 3000/udp
EXPOSE 3001/udp

CMD ["./network_module", "3000", "3001"]
