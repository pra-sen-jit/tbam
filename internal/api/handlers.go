package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// HandleProvisionTime is the Gin handler that receives the POST request from your friend.
func HandleProvisionTime(nc *nats.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		// Convert the request to bytes and publish to the NATS channel
		data, _ := json.Marshal(req)
		err := nc.Publish("events.provision.time", data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Request received and queued on events.provision.time"})
	}
}

// StartNatsListener subscribes to the NATS channel and processes the logic.
func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	nc.Subscribe("events.provision.time", func(msg *nats.Msg) {
		log.Printf("📥 Received message on channel [events.provision.time]")

		wPool.Submit(func() {
			var req models.UserRequest
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				log.Printf("❌ Failed to unmarshal NATS message: %v", err)
				return
			}

			// 1. Prepare the grant 
			// Note: Changed to GrantAccess to match your provision.go file
			grant, err := ldapClient.GrantAccess(req)
			if err != nil {
				log.Printf("❌ LDAP Error: %v", err)
				return
			}

			// 2. Publish to the console executor (Terminal 2)
			// Note: Matches the field names in your models.AccessGrant struct
			commandStr := fmt.Sprintf("CMD:GRANT|USER:%s|GRP:%s|EXP:%d", 
				grant.UserDN, grant.GroupDN, grant.AccessExpiryTime)
			nc.Publish("ldap.console.execute", []byte(commandStr))

			// 3. Schedule the auto-revocation
			// Note: Ensure your scheduler.go signature is: 
			// func ScheduleRevocations(grant models.AccessGrant, client *ldap.Client, nc *nats.Conn)
			scheduler.ScheduleRevocations(*grant, ldapClient, nc)
		})
	})
}
