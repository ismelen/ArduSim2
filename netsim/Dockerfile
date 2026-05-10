FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o netsim .

FROM scratch
COPY --from=builder /app/netsim /netsim
COPY --from=builder /app/config.json /config.json
EXPOSE 3000/udp
ENTRYPOINT ["/netsim"]
