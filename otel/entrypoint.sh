#!/bin/sh

if [ -n "$GCP_KEY" ]; then
    echo "Detected GCP_KEY environment variable. configuring Google Cloud credentials..."
    echo "$GCP_KEY" > /etc/otel/gcp-key.json
    export GOOGLE_APPLICATION_CREDENTIALS=/etc/otel/gcp-key.json
fi

# Run the collector
exec /otelcol-contrib "$@"
