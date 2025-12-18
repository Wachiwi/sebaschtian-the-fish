FROM otel/opentelemetry-collector-contrib:0.96.0

COPY config.yaml /etc/otelcol-contrib/config.yaml
COPY entrypoint.sh /entrypoint.sh

RUN mkdir -p /etc/otel && chmod 777 /etc/otel && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--config=/etc/otelcol-contrib/config.yaml"]

