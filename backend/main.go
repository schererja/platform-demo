package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Telemtry struct {
	DeviceID string `json:"device_id"`
	Message  string `json:"message"`
}

func telemetryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var t Telemtry
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		log.Printf("failed to decode telemetry: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("received telemtry from %s: %s", t.DeviceID, t.Message)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/telemetry", telemetryHandler)
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
