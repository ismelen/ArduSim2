FROM rust:1.67 as build

COPY ./ ./

#build communication model
WORKDIR ./
RUN cargo build --release

# our final base
FROM debian:buster-slim

COPY --from=build /target/release/broker .

EXPOSE 3400
# Command to run the binary
CMD ["./broker"]
