FROM otel/opentelemetry-collector-contrib:0.96.0 AS binary

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=binary /otelcol-contrib /otelcol-contrib

COPY config.yaml /etc/otelcol-contrib/config.yaml
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

WORKDIR /etc/otel

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--config=/etc/otelcol-contrib/config.yaml"]

