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
	"github.com/joho/godotenv"
)

const DEFAULT_PORT = "8080"
const DEFAULT_CACHE_MAX_BYTES int64 = 2 << 30
const DEFAULT_CACHE_NUM_COUNTERS int64 = 10240
const DEFAULT_CACHE_BUFFER_ITEMS int64 = 64
const DEFAULT_CACHE_TTL_HOURS time.Duration = 24 * time.Hour

var cache *ristretto.Cache
var cacheTTL time.Duration

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Failed to load environment variables in init(). Using defaults/existing env.")
	}

	maxCost := DEFAULT_CACHE_MAX_BYTES
	if val := os.Getenv("CACHE_MAX_BYTES"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err != nil {
			log.Printf("Warning: Invalid CACHE_MAX_BYTES '%s'. Using default %d bytes.", val, maxCost)
		} else {
			maxCost = parsed
		}
	}

	numCounters := DEFAULT_CACHE_NUM_COUNTERS
	if val := os.Getenv("CACHE_NUM_COUNTERS"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err != nil {
			log.Printf("Warning: Invalid CACHE_NUM_COUNTERS '%s'. Using default %d.", val, numCounters)
		} else {
			numCounters = parsed
		}
	}

	bufferItems := DEFAULT_CACHE_BUFFER_ITEMS
	if val := os.Getenv("CACHE_BUFFER_ITEMS"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err != nil {
			log.Printf("Warning: Invalid CACHE_BUFFER_ITEMS '%s'. Using default %d.", val, bufferItems)
		} else {
			bufferItems = parsed
		}
	}

	cacheTTL = DEFAULT_CACHE_TTL_HOURS
	if val := os.Getenv("CACHE_TTL_HOURS"); val != "" {
		if parsed, err := strconv.Atoi(val); err != nil {
			log.Printf("Warning: Invalid CACHE_TTL_HOURS '%s'. Using default %s.", val, cacheTTL)
		} else {
			cacheTTL = time.Duration(parsed) * time.Hour
		}
	}

	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: bufferItems,
		Metrics:     true,
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

	cacheKey := createCacheKey(options, target, fps, duration)
	if cached, found := cache.Get(cacheKey); found {
		log.Printf("Serving GIF from cache for key: %s", cacheKey)
		gif := cached.([]byte)
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(gif)
		return
	}

	start := time.Now()

	var buf bytes.Buffer
	err = generateWheelGIF(&buf, options, target, fps, duration)
	if err != nil {
		log.Printf("Error generating wheel GIF: %v", err)
		http.Error(w, "Failed to generate wheel", http.StatusInternalServerError)
		return
	}

	elapsed := time.Since(start)

	log.Printf("Generated wheel GIF in %v (duration=%d, fps=%d)", elapsed, duration, fps)

	gif := buf.Bytes()

	cache.SetWithTTL(cacheKey, gif, int64(len(gif)), cacheTTL)

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(gif)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Failed to load environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = DEFAULT_PORT
	}

	addr := fmt.Sprintf(":%s", port)

	http.HandleFunc("/", handler)

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
