package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	optionsParam := query.Get("options")
	targetParam := query.Get("target")
	if optionsParam == "" {
		http.Error(w, "Missing 'options' query parameter", http.StatusBadRequest)
		return
	}
	if targetParam == "" {
		http.Error(w, "Missing 'target' query parameter", http.StatusBadRequest)
		return
	}

	options := strings.Split(optionsParam, ",")
	if len(options) < 2 {
		http.Error(w, "Provide at least two options", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "image/gif")
	w.Write(buf.Bytes())
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
