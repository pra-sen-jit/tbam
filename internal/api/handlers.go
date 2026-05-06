package api

import (
	"encoding/json"
	"fmt"
	"log"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"

	"github.com/nats-io/nats.go"
)

type NatsEnvelope struct {
	RequestID string             `json:"requestId"`
	Body      models.UserRequest `json:"body"`
}

func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	log.Println("✅ BRAIN IS ALIVE: Listening for JSON on 'access.grant'")

	nc.Subscribe("access.grant", func(msg *nats.Msg) {
		log.Printf("📥 RECEIVED RAW JSON: %s", string(msg.Data))

		wPool.Submit(func() {
			var env NatsEnvelope
			if err := json.Unmarshal(msg.Data, &env); err != nil {
				log.Printf("❌ JSON PARSE ERROR: %v", err)
				return
			}

			grant, err := ldapClient.PrepareGrantCommand(env.Body)
			if err != nil {
				log.Printf("❌ GRANT PREP ERROR: %v", err)
				return
			}

			// Publish to Terminal 2
			commandStr := fmt.Sprintf("CMD:GRANT|USER:%s|GRP:%s|EXP:%d", grant.UserDN, grant.GroupDN, grant.AccessExpiryTime)
			nc.Publish("ldap.console.execute", []byte(commandStr))
			log.Printf("📤 PUBLISHED TO HAND: CMD:GRANT for %s", env.Body.UID)

			// Schedule the Revocation
			scheduler.ScheduleNatsRevocation(*grant, ldapClient, nc)
			log.Printf("⏱️ SCHEDULED REVOCATION: Set timer for %s %s", env.Body.EndDate, env.Body.EndTime)
		})
	})
}
