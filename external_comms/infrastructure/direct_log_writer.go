package infrastructure

import (
	"os"
	"strings"
	"time"

	"external_comms/ports"
)

type DirectLogWriter struct {
	loggerLink ports.LoggerLink
}

func NewDirectLogWriter(loggerLink ports.LoggerLink) *DirectLogWriter {
	return &DirectLogWriter{
		loggerLink: loggerLink,
	}
}

func (w *DirectLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	payload := map[string]interface{}{
		"InstanceID": hostname,
		"ServiceID":  "external_comms",
		"Level":      "INFO",
		"Timestamp":  time.Now().Format(time.RFC3339),
		"Message":    msg,
	}

	w.loggerLink.SendLog(payload)

	return len(p), nil
}
