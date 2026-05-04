package scheduler

import (
	"log"
	"time"

	"tbam/internal/ldap"
	"tbam/internal/models"
)

// ScheduleRevocations takes a list of users and sets an in-memory timer for each one.
func ScheduleRevocations(users []models.ExpiringAccess, client *ldap.Client) {
	log.Printf("Scheduling %d access revocations...", len(users))

	for _, user := range users {
		// 1. Convert the raw Unix integer into a Go time.Time object
		expiryTime := time.Unix(user.AccessExpiryTime, 0)
		
		// 2. Calculate exactly how much time is left from 'Right Now'
		timeRemaining := time.Until(expiryTime)
		targetDN := user.DN

		// 3. Handle edge cases: What if the access is already expired?
		if timeRemaining <= 0 {
			log.Printf("[URGENT] Access for %s is already expired! Flagging for immediate revocation.", user.DN)
			// Trigger immediate revocation
			err := client.RevokeAccess(targetDN)
			if err != nil {
				log.Printf("❌ ERROR revoking access for %s: %v", targetDN, err)
			}
			continue
		}

		log.Printf("Timer set: %s will be revoked in %v", user.DN, timeRemaining)

		// 4. Set the Alarm
		time.AfterFunc(timeRemaining, func() {
			// --- THIS CODE RUNS IN THE FUTURE ---
			// Go automatically spawns a new background goroutine when this timer hits 0.
			log.Printf("⏰ ALARM RINGING! Time to revoke access for: %s", targetDN)
			err := client.RevokeAccess(targetDN)
			if err != nil {
				log.Printf("❌ ERROR revoking access for %s: %v", targetDN, err)
			}
		})
	}
}
