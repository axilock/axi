package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

func SendCoverageData() {
	// Placeholder for sending coverage data to the server
	resp, err := http.Post("https://github.com/coverage", "application/json", nil)
	if err != nil {
		log.Printf("Failed to send coverage data: %v", err)
		return
	}
	defer resp.Body.Close()
}

func GetSystemMetadataJson(cli_version string) string {
	type Metadata struct {
		OS         string    `json:"os"`
		Arch       string    `json:"arch"`
		Hostname   string    `json:"hostname"`
		Username   string    `json:"username"`
		GoVersion  string    `json:"go_version"`
		CliVersion string    `json:"cli_version"`
		NumCPU     int       `json:"num_cpu"`
		PID        int       `json:"pid"`
		Timestamp  time.Time `json:"timestamp"`
	}

	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	metadata := Metadata{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Hostname:   hostname,
		Username:   username,
		GoVersion:  runtime.Version(),
		CliVersion: cli_version,
		NumCPU:     runtime.NumCPU(),
		PID:        os.Getpid(),
		Timestamp:  time.Now(),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}

	return string(metadataJSON)
}
