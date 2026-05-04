package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"tbam/internal/ldap"
	"tbam/internal/scheduler"
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

	// Test the Search
	users, err := ldapClient.FetchExpiringUsers()
	if err != nil {
		log.Printf("Error fetching users: %v", err)
	} else {
		for _, u := range users {
			log.Printf("-> User: %s | Expires At: %d", u.DN, u.AccessExpiryTime)
		}
	}

	// Start the timers
	if len(users) > 0 {
		scheduler.ScheduleRevocations(users, ldapClient)
	} else {
		log.Println("No expiring access found for today.")
	}

	// ----------------------------------------------------------------------

	// Setup Graceful Shutdown
	// Created a channel to listen for OS interrupt signals (like Ctrl+C in terminal)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to shut down.")
	<-quit
	log.Println("Shutdown signal received. Exiting gracefully...")
}
