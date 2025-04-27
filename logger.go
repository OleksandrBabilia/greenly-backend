package main

import (
	"bytes"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		log.WithFields(log.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"ip":     r.RemoteAddr,
			"body":   string(bodyBytes),
		}).Info("Incoming request")

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		log.WithFields(log.Fields{
			"duration": duration,
			"method":   r.Method,
			"path":     r.URL.Path,
			"ip":       r.RemoteAddr,
		}).Info("Request handled")
	})
}
