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
	godotenv.Load(".env")

	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatal(err)
	}
	defer ldapClient.CloseAll()

	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	workerCount, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	workerPool := worker.NewPool(workerCount, 500)

	go api.StartNatsListener(nc, workerPool, ldapClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
