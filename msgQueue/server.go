package msgQueue

import (
	"fmt"
	"net/http"
	models "tbam/models"
	access "tbam/access"
	ldapool "tbam/connection_pool"
	workerpool "tbam/worker_pool"

	"github.com/gin-gonic/gin"
)

type UserRequest struct {
	UID              string `json:"uid"`
	GroupAssociated  string `json:"grp_associated"`
	PrivilegeAccess  string `json:"privilege_access"`
	StartDate        string `json:"start_date"`
	StartTime        string `json:"start_time"`
	EndDate          string `json:"end_date"`
	EndTime          string `json:"end_time"`
}


func StartServer(wPool *workerpool.Pool, cPool *ldapool.Pool) {
	r := gin.Default()

	r.POST("/update", func(c *gin.Context) {
		var req models.UserRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H {"error": "Invalid JSON format"})
			return
		}

		wPool.Submit(func() {
			conn, err := cPool.Borrow()
			if err != nil {
				fmt.Printf("Error borrowing connection for %s: %v\n", req.UID, err)
				return
			}

			defer cPool.Return(conn)

			err = access.GrantAccess(conn, req)

			// fmt.Printf("[LDAP] Updating %s: Access=%s, Group=%s, Until=%s %s\n", req.UID, req.PrivilegeAccess, req.GroupAssociated, req.EndDate, req.EndTime)
		})

			c.JSON(http.StatusAccepted, gin.H{"status": "queued", "user": req.UID})
	})

	r.Run(":5000")
}

