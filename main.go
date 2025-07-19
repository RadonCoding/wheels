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

var cache *ristretto.Cache

func init() {
	var err error
	cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     200 << 20,
		BufferItems: 64,
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

	cache.SetWithTTL(cacheKey, gif, int64(len(gif)), 24*time.Hour)

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
		fmt.Println("No port has been configured in environment variables")
		return
	}

	addr := fmt.Sprintf(":%s", port)

	http.HandleFunc("/", handler)

	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
