package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

type Manifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block type: <nil>")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid private key type in PKCS#8: %T", parsed)
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("invalid PEM block type: %v", block.Type)
	}
}

func signFirmware(privateKey *rsa.PrivateKey, firmwarePath string) (string, error) {
	f, err := os.Open(firmwarePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil

}

func manifestHandler(privateKey *rsa.PrivateKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sig, err := signFirmware(privateKey, "./firmware.bin")
		if err != nil {
			http.Error(w, "Failed to sign firmware", http.StatusInternalServerError)
			return
		}
		manifest := Manifest{
			Version:   "1.1.2",
			URL:       "http://ota-server:8081/firmware.bin",
			Signature: sig,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}
}

func firmwareHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "firmware.bin")
}
func otaCheckHandler(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer("ota-server").Start(r.Context(), "otaCheckHandler")
	defer span.End()

	deviceID := r.URL.Query().Get("device_id")
	currentVersion := r.URL.Query().Get("current_version")
	latestVersion := "1.1.2"

	updateAvailable := currentVersion != latestVersion
	response := map[string]interface{}{
		"update_available": updateAvailable,
		"latest_version":   latestVersion,
		"url":              "http://ota-server:8081/firmware/" + latestVersion,
	}
	w.Header().Set("Content-Type", "application/json")
	span.SetAttributes(
		semconv.ClientAddress(deviceID),
	)
	json.NewEncoder(w).Encode(response)
	log.Printf("Received OTA check from device %s with current version %s", deviceID, currentVersion)

}
func otaApplyHandler(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer("ota-server").Start(r.Context(), "otaApplyHandler")
	defer span.End()

	var body struct {
		DeviceID string `json:"device_id"`
		Version  string `json:"version"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	resp := map[string]string{
		"status":  "success",
		"version": body.Version,
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	endpoing := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoing == "" {
		endpoing = "otel-collector:4318"
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoing),
		otlptracehttp.WithURLPath("/v1/traces"),
	)
	if err != nil {
		log.Fatalf("failed to initialize OTLP exporter: %v", err)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ota-server"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
func main() {
	privateKey, err := loadPrivateKey("/keys/private.pem")
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
		os.Exit(1)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()
	http.HandleFunc("/manifest", manifestHandler(privateKey))
	http.HandleFunc("/firmware.bin", firmwareHandler)
	http.HandleFunc("/ota/check", otaCheckHandler)
	http.HandleFunc("/ota/apply", otaApplyHandler)
	log.Println("OTA server listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
