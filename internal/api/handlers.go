package api

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/scheduler"
	"tbam/internal/worker"
)

type NatsEnvelope struct {
	RequestID string             `json:"requestId"`
	Body      models.UserRequest `json:"body"`
}

func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	nc.Subscribe("access.grant", func(msg *nats.Msg) {
		wPool.Submit(func() {
			var env NatsEnvelope
			json.Unmarshal(msg.Data, &env)

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
