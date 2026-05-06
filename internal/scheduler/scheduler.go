package scheduler

import (
	"log"
	"time"

	"tbam/internal/ldap"
	"tbam/internal/models"
	"github.com/nats-io/nats.go" // Added NATS import
)

// ScheduleRevocations now takes 3 arguments: a single grant, the client, and nats connection
func ScheduleRevocations(grant models.AccessGrant, client *ldap.Client, nc *nats.Conn) {
	// 1. Calculate time remaining
	expiryTime := time.Unix(grant.AccessExpiryTime, 0)
	timeRemaining := time.Until(expiryTime)

	if timeRemaining <= 0 {
		log.Printf("[URGENT] Already expired: %s", grant.UserDN)
		executeRevocation(grant, nc)
		return
	}

	log.Printf("⏰ Timer set for %s: Revoking in %v", grant.UserDN, timeRemaining)

	// 2. Set the Alarm
	time.AfterFunc(timeRemaining, func() {
		log.Printf("🚨 ALARM: Revoking access for %s", grant.UserDN)
		executeRevocation(grant, nc)
	})
}

// Helper to notify the system to revoke via NATS
func executeRevocation(grant models.AccessGrant, nc *nats.Conn) {
	// You can publish to a 'revoke' channel that Terminal 2 listens to
	revokeMsg := fmt.Sprintf("CMD:REVOKE|USER:%s|GRP:%s", grant.UserDN, grant.GroupDN)
	nc.Publish("ldap.console.execute", []byte(revokeMsg))
}
