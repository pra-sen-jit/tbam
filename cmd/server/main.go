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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {
	// 1. Load configuration from .env file
	godotenv.Load(".env")

	// 2. Initialize LDAP Connection Pool
	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatalf("Failed to initialize LDAP pool: %v", err)
	}
	defer ldapClient.CloseAll()

	// 3. Connect to NATS Server (Pointed to port 4223 or your specific IP)
	natsURL := os.Getenv("NATS_URL")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS at %s", natsURL)

	// 4. Initialize Worker Pool for handling heavy LDAP tasks
	workerCount, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	workerPool := worker.NewPool(workerCount, 500)

	// 5. Setup Gin Router for the HTTP Endpoint
	r := gin.Default()

	// The endpoint your friend will hit
	r.POST("/api/provision/time", api.HandleProvisionTime(nc))

	// Optional: Heartbeat check for your friend
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server is reachable"})
	})

	// 6. Start the NATS Subscriber in a background Goroutine
	// It listens to the "events.provision.time" channel
	go api.StartNatsListener(nc, workerPool, ldapClient)

	// 7. Run the Gin Server
	// Use "0.0.0.0" to ensure it accepts external connections from your friend
	go func() {
		log.Println("🚀 Gin Server starting on 0.0.0.0:8080")
		if err := r.Run("0.0.0.0:8080"); err != nil {
			log.Fatalf("Gin server failed: %v", err)
		}
	}()

	// 8. Graceful Shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}
