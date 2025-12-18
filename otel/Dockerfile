FROM otel/opentelemetry-collector-contrib:0.96.0

USER root
COPY config.yaml /etc/otelcol-contrib/config.yaml

# Ensure we can read the socket
RUN mkdir -p /var/run && chmod 777 /var/run
