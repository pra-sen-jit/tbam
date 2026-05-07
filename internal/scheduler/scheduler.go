package scheduler

import (
	"log"
	"time"
	"tbam/internal/ldap"
	"tbam/internal/models"
)

// ScheduleRevocations takes a list of access grants and sets an in-memory timer for each one.
func ScheduleRevocations(grants []models.AccessGrant, client *ldap.Client) {
	log.Printf("Scheduling %d specific access revocations...", len(grants))

	for _, grant := range grants {
		expiryTime := time.Unix(grant.AccessExpiryTime, 0)
		timeRemaining := time.Until(expiryTime)
		targetGrant := grant

		if timeRemaining <= 0 {
			log.Printf("[URGENT] Grant for %s to group %s is already expired! Flagging for immediate revocation.", targetGrant.UserDN, targetGrant.GroupDN)
			err := client.RevokeSpecificAccess(targetGrant)
			if err != nil {
				log.Printf("❌ ERROR revoking specific access for %s: %v", targetGrant.UserDN, err)
			}
			continue
		}
		log.Printf("Timer set: %s will be removed from %s in %v", targetGrant.UserDN, targetGrant.GroupDN, timeRemaining)

		time.AfterFunc(timeRemaining, func() {
			log.Printf("⏰ ALARM RINGING! Time to revoke access for: %s from %s", targetGrant.UserDN, targetGrant.GroupDN)
			err := client.RevokeSpecificAccess(targetGrant)
			if err != nil {
				log.Printf("❌ ERROR revoking specific access for %s: %v", targetGrant.UserDN, err)
			}
		})
	}
}
