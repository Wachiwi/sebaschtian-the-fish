package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/balena"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {
	logger.Setup()

	// 1. Initialize Balena Client
	// On local dev (outside Balena), this might fail if env vars aren't set.
	// We'll warn and exit or loop waiting.
	var client *balena.SupervisorClient
	var err error

	// Retry getting client (env vars might be injected late? unlikely but safe)
	if os.Getenv("BALENA_SUPERVISOR_ADDRESS") == "" {
		slog.Warn("BALENA_SUPERVISOR_ADDRESS not set. Balena monitoring disabled. (Are you running locally?)")
		// For local testing, we might just block or exit. Let's block to keep container alive but idle.
		select {}
	}

	client, err = balena.NewSupervisorClient()
	if err != nil {
		slog.Error("Failed to create Balena Supervisor client", "error", err)
		os.Exit(1)
	}

	// 2. Initialize OpenTelemetry
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("balena-monitor"),
		),
	)
	if err != nil {
		slog.Error("Failed to create resource", "error", err)
		os.Exit(1)
	}

	// Connect to OTel Collector (otel-collector:4317)
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint("otel-collector:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		slog.Error("Failed to create OTLP exporter", "error", err)
		os.Exit(1)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))),
	)
	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down meter provider", "error", err)
		}
	}()
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("balena-monitor")

	// 3. Define Metrics
	updatePendingGauge, err := meter.Int64Gauge("balena.update.pending", metric.WithDescription("1 if an update is pending, 0 otherwise"))
	if err != nil {
		slog.Error("Failed to create update_pending gauge", "error", err)
	}

	downloadProgressGauge, err := meter.Float64Gauge("balena.update.download_progress", metric.WithDescription("Percentage of update downloaded"))
	if err != nil {
		slog.Error("Failed to create download_progress gauge", "error", err)
	}

	statusGauge, err := meter.Int64Gauge("balena.status_code", metric.WithDescription("Status code mapping (1=Idle, 2=Downloading, 3=Installing, 0=Unknown)"))
	if err != nil {
		slog.Error("Failed to create status gauge", "error", err)
	}

	slog.Info("Starting Balena Monitor loop...")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			state, err := client.GetState()
			if err != nil {
				slog.Error("Failed to fetch supervisor state", "error", err)
				continue
			}

			// Record metrics
			// Update Pending
			pendingVal := int64(0)
			if state.UpdatePending {
				pendingVal = 1
			}
			updatePendingGauge.Record(ctx, pendingVal)

			// Download Progress
			downloadProgressGauge.Record(ctx, state.DownloadProgress)

			// Status Mapping
			statusVal := int64(0)
			switch state.Status {
			case "Idle":
				statusVal = 1
			case "Downloading":
				statusVal = 2
			case "Installing":
				statusVal = 3
			}
			statusGauge.Record(ctx, statusVal, metric.WithAttributes(attribute.String("status_text", state.Status)))

			slog.Info("Recorded metrics", "status", state.Status, "progress", state.DownloadProgress, "pending", state.UpdatePending)
		}
	}
}
