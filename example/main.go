package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fllarpy/apm-probe"
)

func main() {
	// Create a new ServeMux to register our handlers to.
	mux := http.NewServeMux()

	// Register the handlers directly. The middleware will be applied to the mux.
	mux.HandleFunc("/", helloHandler)

	// This handler demonstrates an instrumented client request.
	mux.HandleFunc("/client-request", clientRequestHandler)

	// This handler will always return a 500 error to test error collection.
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("This is a simulated error."))
	})

	// Register the metrics handler if the probe is enabled.
	metricsHandler := apm_probe.MetricsHandler()
	if metricsHandler != nil {
		debugEndpointPath := getEnv("APM_DEBUG_ENDPOINT", "/debug/apm")
		mux.Handle(debugEndpointPath, metricsHandler)
		log.Printf("Metrics endpoint enabled at http://localhost:8080%s", debugEndpointPath)
	} else {
		log.Println("APM probe is disabled, not serving metrics.")
	}

	log.Println("Starting server on :8080")
	log.Println("Test endpoint: http://localhost:8080/")
	log.Println("Client request test endpoint: http://localhost:8080/client-request")
	log.Println("Error test endpoint: http://localhost:8080/error")

	// Start the server, wrapping our mux with the APM middleware.
	// This ensures all requests, including 404s handled by the mux itself, are captured.
	if err := http.ListenAndServe(":8080", apm_probe.Middleware()(mux)); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate some work
	time.Sleep(50 * time.Millisecond)

	// Chance to return a 4xx or 5xx error
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(100)
	if n < 5 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Oh no, a 500 error!")
		return
	} else if n < 10 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Oops, a 400 error!")
		return
	}

	fmt.Fprintln(w, "Hello, world!")
}

func clientRequestHandler(w http.ResponseWriter, r *http.Request) {
	// Create an instrumented client. We pass nil to use a client based on http.DefaultClient.
	client := apm_probe.NewClient(nil)

	// Make a request to an external service.
	resp, err := client.Get("https://httpbin.org/delay/1")
	if err != nil {
		http.Error(w, "Failed to make client request", http.StatusInternalServerError)
		log.Printf("Client request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Fprintf(w, "Made a client request to httpbin.org, status: %s", resp.Status)
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsBool reads a boolean environment variable or returns a default value.
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
