package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"tbam/internal/ldap"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting NextGen IGA Time-Bound Access Service...")

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: No .env file found or error reading it. Relying on system environment variables.")
	}

	ldapURL := os.Getenv("LDAP_URL")
	log.Printf("Loaded configuration. LDAP Target: %s\n", ldapURL)

	// Initialize the LDAP Connection
	ldapClient, err := ldap.Connect()
	if err != nil {
		log.Fatalf("Fatal error starting service: %v", err)
	}
	defer ldapClient.Close()

	// Setup Graceful Shutdown
	// Created a channel to listen for OS interrupt signals (like Ctrl+C in terminal)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to shut down.")

	<-quit
	log.Println("Shutdown signal received. Exiting gracefully...")
}
