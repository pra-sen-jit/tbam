package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"tbam/internal/api"
	"tbam/internal/ldap"
	"tbam/internal/worker"
)

func main() {
	godotenv.Load(".env")

	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer ldapClient.CloseAll()

	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer nc.Close()

	workerPool := worker.NewPool(150, 500)

	go api.StartNatsListener(nc, workerPool, ldapClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
