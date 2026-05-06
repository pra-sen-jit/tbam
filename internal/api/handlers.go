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

// 1. GIN HANDLER: This is what your friend hits
func HandleProvisionTime(nc *nats.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		// Push the data into the NATS channel your friend requested
		data, _ := json.Marshal(req)
		nc.Publish("events.provision.time", data)

		c.JSON(http.StatusOK, gin.H{"message": "Event queued on events.provision.time"})
	}
}

// 2. NATS SUBSCRIBER: This listens to that channel and does the LDAP work
func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	nc.Subscribe("events.provision.time", func(msg *nats.Msg) {
		log.Printf("📥 Received from channel [events.provision.time]")
		
		wPool.Submit(func() {
			var req models.UserRequest
			json.Unmarshal(msg.Data, &req)

			// Your existing logic to prepare LDAP command
			grant, err := ldapClient.PrepareGrantCommand(req)
			if err != nil {
				log.Printf("❌ Error: %v", err)
				return
			}

			// Send command to the executor (Terminal 2)
			commandStr := fmt.Sprintf("CMD:GRANT|USER:%s|GRP:%s|EXP:%d", grant.UserDN, grant.GroupDN, grant.AccessExpiryTime)
			nc.Publish("ldap.console.execute", []byte(commandStr))

			// Start the auto-revocation timer
			scheduler.ScheduleNatsRevocation(*grant, ldapClient, nc)
		})
	})
}
