package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"
)

func StartServer(wPool *worker.Pool, ldapClient *ldap.Client) {
	r := gin.Default()

	r.POST("/provision", func(c *gin.Context) {
		var req models.UserRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		// Submit the heavy lifting to the background worker pool
		wPool.Submit(func() {
			// 1. Provision the Access in LDAP
			grant, err := ldapClient.GrantAccess(req)
			if err != nil {
				fmt.Printf("❌ Failed to provision %s: %v\n", req.UID, err)
				return
			}

			// 2. IMMEDIATELY SET THE DEPROVISIONING TIMER
			// We pass a slice containing our single new grant to your existing scheduler
			scheduler.ScheduleRevocations([]models.AccessGrant{*grant}, ldapClient)
		})

		// Return fast to the UI
		c.JSON(http.StatusAccepted, gin.H{"status": "queued", "user": req.UID})
	})

	r.Run(":5000")
}
