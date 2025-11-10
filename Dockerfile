FROM golang:1-trixie AS base

WORKDIR /app

COPY go.mod go.sum ./

RUN apt-get update && apt-get install -y pkg-config libasound2-dev && rm -rf /var/lib/apt/lists/*

RUN go mod download

COPY . .
RUN go build -v -o /dist/fish ./cmd/fish
RUN go build -v -o /dist/sounds ./cmd/sounds

FROM debian:trixie as fish

RUN apt-get update && \
    apt-get install -y libgpiod-dev ca-certificates libasound2-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=base /dist/fish /app/fish

CMD ["./fish"]

FROM debian:trixie as sounds

WORKDIR /app

COPY --from=base /dist/sounds /app/sounds

CMD ["./sounds"]
