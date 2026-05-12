package api

import (
	"encoding/json"
	"log"
	"net/http"
	"tbam/internal/db"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func HandleProvisionTime(nc *nats.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserRequest
		if size := c.Request.ContentLength; size == -1 || size > 25 * 1024 * 1024 {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body too large"})
				return
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		js, err := nc.JetStream()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not enabled"})
			return
		}

		data, _ := json.Marshal(req)
		_, err = js.Publish("events.provision.time", data)
		if err != nil {
			log.Printf("❌ JetStream Publish Error: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Persistence failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Request persisted in JetStream"})
	}
}

func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client, dbClient *db.DBClient) {
	nc.Subscribe("events.provision.time", func(msg *nats.Msg) {
		log.Printf("📥 Received from channel [events.provision.time]")
		
		wPool.Submit(func() {
			var req models.UserRequest
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				log.Printf("❌ Error unmarshaling request: %v", err)
				return
			}
			grant, err := ldapClient.GrantAccess(req, dbClient)
			if err != nil {
				log.Printf("❌ Error provisioning access: %v", err)
				return
			}
			scheduler.ScheduleRevocations([]models.AccessGrant{*grant}, ldapClient, dbClient)
		})
	})
}
