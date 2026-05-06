package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"
)

type NatsEnvelope struct {
	RequestID string            `json:"requestId"`
	Body      models.UserRequest `json:"body"`
}

func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	// ----- Start‑up recovery: re‑schedule any missed revocations -----
	go func() {
		grants, err := ldapClient.FetchExpiringGrants()
		if err != nil {
			log.Printf("ERROR fetching expiring grants on startup: %v", err)
			return
		}
		for _, g := range grants {
			scheduler.ScheduleNatsRevocation(g, ldapClient, nc)
			log.Printf("Recovered grant: %s -> %s (expires %d)", g.UserDN, g.GroupDN, g.AccessExpiryTime)
		}
	}()

	// Subscribe to incoming access requests
	nc.Subscribe("access.grant", func(msg *nats.Msg) {
		wPool.Submit(func() {
			var env NatsEnvelope
			if err := json.Unmarshal(msg.Data, &env); err != nil {
				log.Printf("Invalid JSON: %v", err)
				reply, _ := json.Marshal(map[string]interface{}{"ok": false, "error": "invalid JSON"})
				msg.Respond(reply)
				return
			}

			grant, err := ldapClient.PrepareGrantCommand(env.Body)
			if err != nil {
				reply, _ := json.Marshal(map[string]interface{}{"ok": false, "error": err.Error()})
				msg.Respond(reply)
				return
			}

			commandStr := fmt.Sprintf("CMD:GRANT|USER:%s|GRP:%s|EXP:%d", grant.UserDN, grant.GroupDN, grant.AccessExpiryTime)
			nc.Publish("ldap.console.execute", []byte(commandStr))

			scheduler.ScheduleNatsRevocation(*grant, ldapClient, nc)

			reply, _ := json.Marshal(map[string]interface{}{"ok": true, "status": 200, "reqId": env.RequestID})
			msg.Respond(reply)
		})
	})
}
