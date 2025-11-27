FROM golang:1.24 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /build/pfcp-cp ./cmd/pfcp-cp

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/pfcp-cp /usr/local/bin/pfcp-cp

EXPOSE 8805/udp 50051/tcp

ENTRYPOINT ["/usr/local/bin/pfcp-cp"]
