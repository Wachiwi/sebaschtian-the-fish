FROM golang:1-trixie as base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /dist/gpt-app ./cmd/gpt-app
RUN go build -v -o /dist/gpt-api ./cmd/gpt-api

FROM debian:trixie

RUN apt-get update && \
    apt-get install -y libgpiod-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=base /dist/gpt-app /app/gpt-app
COPY --from=base /dist/gpt-api /app/gpt-api

CMD ["./gpt-app"]
