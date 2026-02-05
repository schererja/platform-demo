package main

import (
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
			Version:   "1.1.1",
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

func main() {
	privateKey, err := loadPrivateKey("/keys/private.pem")
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	http.HandleFunc("/manifest", manifestHandler(privateKey))
	http.HandleFunc("/firmware.bin", firmwareHandler)
	log.Println("OTA server listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
