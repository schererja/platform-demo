package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Manifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

func manifestHandler(w http.ResponseWriter, r *http.Request) {
	manifest := Manifest{
		Version:   "1.1.1",
		URL:       "http://ota-server:8081/firmware.bin",
		Signature: "2dc9e2c8-c63c-441b-b805-ecfd5624f8ee",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func firmwareHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "firmware.bin")
}

func main() {
	http.HandleFunc("/manifest", manifestHandler)
	http.HandleFunc("/firmware.bin", firmwareHandler)
	log.Println("OTA server listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
