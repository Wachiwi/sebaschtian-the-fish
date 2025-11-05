FROM golang:1-trixie AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /dist/fish ./cmd/fish

FROM debian:trixie

RUN apt-get update && \
    apt-get install -y libgpiod-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=base /dist/fish /app/fish

CMD ["./fish"]
