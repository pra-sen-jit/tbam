package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"tbam/internal/api"
	"tbam/internal/db"
	"tbam/internal/ldap"
	"tbam/internal/scheduler"
	"tbam/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// If it's a preflight OPTIONS request, abort with 204 (No Content) and return
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	godotenv.Load(".env")

	dbClient, err := db.InitMySQL()
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer dbClient.Conn.Close()

	ldapClient, err := ldap.InitPool(10)
	if err != nil {
		log.Fatalf("Failed to initialize LDAP pool: %v", err)
	}
	defer ldapClient.CloseAll()

	natsURL := os.Getenv("NATS_URL")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS at %s", natsURL)

	workerCount, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	workerPool := worker.NewPool(workerCount, 500)

	// Boot-up Recovery
	grants, fetchErr := ldapClient.FetchExpiringGrants()
	if fetchErr != nil {
		log.Printf("Error fetching access grants on boot: %v", fetchErr)
	} else if len(grants) > 0 {
		scheduler.ScheduleRevocations(grants, ldapClient, dbClient)
	} else {
		log.Println("No active expirations found on boot.")
	}

	r := gin.Default()
	r.Use(CORSMiddleware())
	r.POST("/api/access/time", api.HandleProvisionTime(nc))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server is reachable"})
	})

	// Starts the NATS Subscriber in a background Goroutine
	// It listens to the "events.provision.time" channel
	go api.StartNatsListener(nc, workerPool, ldapClient, dbClient)

	go func() {
		log.Println("🚀 Gin Server starting on 0.0.0.0:8080")
		if err := r.Run("0.0.0.0:8080"); err != nil {
			log.Fatalf("Gin server failed: %v", err)
		}
	}()

	// Graceful Shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}
