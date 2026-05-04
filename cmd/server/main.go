package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"tbam/internal/api"
	"tbam/internal/ldap"
	"tbam/internal/scheduler"
	"tbam/internal/worker"
)

func main() {
	log.Println("Starting NextGen IGA Unified Time-Bound Access Service...")
	godotenv.Load(".env")

	// 1. Initialize LDAP Connection Pool (e.g., 10 concurrent connections)
	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatalf("Fatal error starting service: %v", err)
	}
	defer ldapClient.CloseAll()

	// 2. Initialize Worker Pool (e.g., 20 goroutines, queue size 100)
	workerPool := worker.NewPool(20, 100)

	// --- PHASE 1: BOOT-UP RECOVERY ---
	// Catch any access grants we missed while the server was offline
	grants, err := ldapClient.FetchExpiringGrants()
	if err != nil {
		log.Printf("Error fetching access grants: %v", err)
	} else if len(grants) > 0 {
		scheduler.ScheduleRevocations(grants, ldapClient)
	} else {
		log.Println("No active expirations found on boot.")
	}

	// --- PHASE 2: START GIN API ---
	// We run this in a goroutine so it doesn't block our graceful shutdown channel
	go api.StartServer(workerPool, ldapClient)

	// --- GRACEFUL SHUTDOWN ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Service is running and listening for provisioning requests on port 5000. Press Ctrl+C to shut down.")
	<-quit
	log.Println("Shutdown signal received. Exiting gracefully...")
}
