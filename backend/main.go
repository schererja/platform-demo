package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

var (
	mu                 sync.Mutex
	totalTelemetry     int
	telemetryPerDevice = make(map[string]int)
)

type Telemtry struct {
	DeviceID string `json:"device_id"`
	Message  string `json:"message"`
}
type LogEntry struct {
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	DeviceID  string `json:"device_id,omitempty"`
}

type Metrics struct {
	TotalTelemetry     int            `json:"total_telemetry"`
	TelemetryPerDevice map[string]int `json:"telemetry_per_device"`
}

var logger = log.New(os.Stdout, "", 0)

func logInfo(message string, deviceID string) {
	entry := LogEntry{
		Level:     "INFO",
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   message,
		DeviceID:  deviceID,
	}
	data, _ := json.Marshal(entry)
	logger.Println(string(data))
}

func logError(message string, deviceID string) {
	entry := LogEntry{
		Level:     "ERROR",
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   message,
		DeviceID:  deviceID,
	}
	data, _ := json.Marshal(entry)
	logger.Println(string(data))
}
func telemetryHandler(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer("backend").Start(r.Context(), "telemetryHandler")
	defer span.End()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var t Telemtry
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		logError("failed to decode telemetry: "+err.Error(), "")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	span.SetAttributes(
		semconv.ClientAddress(t.DeviceID),
	)
	mu.Lock()
	totalTelemetry++
	telemetryPerDevice[t.DeviceID]++
	mu.Unlock()

	logInfo("received telemetry from "+t.DeviceID+": "+t.Message, t.DeviceID)
	w.WriteHeader(http.StatusOK)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	m := Metrics{
		TotalTelemetry:     totalTelemetry,
		TelemetryPerDevice: telemetryPerDevice,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
	)

	if err != nil {
		log.Fatalf("failed to initialize OTLP exporter: %v", err)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("platform-backend"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
		os.Exit(1)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	logInfo("Backend starting", "")
	http.HandleFunc("/telemetry", telemetryHandler)
	http.HandleFunc("/metrics", metricsHandler)
	logInfo("Starting server on :8000", "")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		logError("failed to start server: "+err.Error(), "")
	}
}
