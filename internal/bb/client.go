package bb

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"
)

func NewHTTPClient() (*http.Client, error) {
	certFile := os.Getenv("BB_CERT_FILE")
	keyFile := os.Getenv("BB_KEY_FILE")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}
