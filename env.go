package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Failed to load environment variables from .env file")
	}
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Warning: Invalid %s '%s'. Using default %d.", key, val, def)
		return def
	}
	return parsed
}
