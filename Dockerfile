FROM golang:1-trixie AS base

WORKDIR /app

COPY go.mod go.sum ./

RUN apt-get update && apt-get install -y pkg-config libasound2-dev && rm -rf /var/lib/apt/lists/*

RUN go mod download

COPY . .
RUN go build -v -o /dist/fish ./cmd/fish
RUN go build -v -o /dist/sounds ./cmd/sounds

FROM debian:trixie AS fish

RUN apt-get update && \
    apt-get install -y libgpiod-dev ca-certificates libasound2-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=base /dist/fish /app/fish

CMD ["./fish"]

FROM debian:trixie AS sounds

RUN apt-get update && \
    apt-get install -y ca-certificates gnupg wget && \
    wget -qO - https://archive.raspberrypi.com/debian/raspberrypi.gpg.key | gpg --dearmor -o /usr/share/keyrings/raspberrypi-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/raspberrypi-archive-keyring.gpg] https://archive.raspberrypi.com/debian/ trixie main" > /etc/apt/sources.list.d/raspberrypi.list && \
    apt-get update && \
    apt-get install -y rpicam-apps && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=base /dist/sounds /app/sounds

CMD ["./sounds"]
