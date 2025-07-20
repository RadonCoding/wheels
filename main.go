package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/ristretto"
)

const DEFAULT_PORT = "8080"
const DEFAULT_WHEEL_RADIUS = 360
const DEFAULT_CACHE_MAX_BYTES = 200 * 1024 * 1024
const DEFAULT_CACHE_NUM_COUNTERS = 1024
const DEFAULT_CACHE_TTL_HOURS = 6

var cache *ristretto.Cache
var cacheTTL time.Duration

func init() {
	maxCost := getEnvInt("CACHE_MAX_BYTES", DEFAULT_CACHE_MAX_BYTES)
	numCounters := getEnvInt("CACHE_NUM_COUNTERS", DEFAULT_CACHE_NUM_COUNTERS)
	cacheTTL = time.Duration(getEnvInt("CACHE_TTL_HOURS", DEFAULT_CACHE_TTL_HOURS)) * time.Hour

	var err error
	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(numCounters),
		MaxCost:     int64(maxCost),
		BufferItems: 64,
	})
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
}

func createCacheKey(options []string, target, fps, duration int) string {
	key := fmt.Sprintf("options=%s&target=%d&fps=%d&duration=%d",
		strings.Join(options, ","), target, fps, duration)
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	optionsParam := query.Get("options")
	if optionsParam == "" {
		http.Error(w, "Missing 'options' query parameter", http.StatusBadRequest)
		return
	}
	options := strings.Split(optionsParam, ",")
	if len(options) < 2 {
		http.Error(w, "Provide at least two options", http.StatusBadRequest)
		return
	}

	targetParam := query.Get("target")
	if targetParam == "" {
		http.Error(w, "Missing 'target' query parameter", http.StatusBadRequest)
		return
	}
	target, err := strconv.Atoi(targetParam)
	if err != nil || target < 0 || target >= len(options) {
		http.Error(w, "Invalid 'target' index", http.StatusBadRequest)
		return
	}

	fps := 12
	if f := query.Get("fps"); f != "" {
		fps, err = strconv.Atoi(f)
		if err != nil {
			http.Error(w, "Invalid 'fps' value", http.StatusBadRequest)
			return
		}
		fps = clamp(fps, 1, 24)
	}

	duration := 10
	if d := query.Get("duration"); d != "" {
		duration, err = strconv.Atoi(d)
		if err != nil {
			http.Error(w, "Invalid 'duration' value", http.StatusBadRequest)
			return
		}
		duration = clamp(duration, 1, 30)
	}

	key := createCacheKey(options, target, fps, duration)
	if cached, found := cache.Get(key); found {
		log.Printf("Serving GIF from cache for key: %s", key)
		gif := cached.([]byte)
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(gif)
		return
	}

	start := time.Now()

	radius := getEnvInt("WHEEL_RADIUS", DEFAULT_WHEEL_RADIUS)

	wr := &WheelRenderer{
		OuterRadius: float64(radius),
		InnerRadius: float64(radius) * 0.95,
		Options:     options,
		Target:      target,
		FPS:         fps,
		Duration:    duration,
	}

	var buf bytes.Buffer
	err = wr.RenderGIF(&buf)
	if err != nil {
		log.Printf("Error rendering GIF: %v", err)
		http.Error(w, "Failed to render GIF", http.StatusInternalServerError)
		return
	}

	elapsed := time.Since(start)

	log.Printf("Rendered GIF in %v (duration=%d, fps=%d).", elapsed, duration, fps)

	gif := buf.Bytes()

	cache.SetWithTTL(key, gif, int64(len(gif)), cacheTTL)

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	w.Write(gif)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DEFAULT_PORT
	}

	addr := fmt.Sprintf(":%s", port)

	http.HandleFunc("/", handler)

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
