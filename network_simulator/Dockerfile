FROM rust:1.85-slim as builder

WORKDIR /usr/src/app
# Context is network_simulator/
COPY . .

RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y libgcc-s1 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/src/app/target/release/network_simulator /usr/local/bin/

# Port configurations
EXPOSE 3000/udp

CMD ["network_simulator"]
