package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"tbam/internal/api"
	"tbam/internal/ldap"
	"tbam/internal/worker"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {
	// Load .env, fail if missing
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// LDAP connection pool
	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatalf("LDAP pool init failed: %v", err)
	}
	defer ldapClient.CloseAll()

	// NATS
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
	}
	defer nc.Close()

	// Worker pool
	workerCount, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	workerPool := worker.NewPool(workerCount, 500)

	// Start NATS listener + recovery
	go api.StartNatsListener(nc, workerPool, ldapClient)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down GRIP …")
}
