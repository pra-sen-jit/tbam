package api

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"tbam/internal/worker"
)

type NatsEnvelope struct {
	RequestID string             `json:"requestId"`
	UserID    string             `json:"userId"`
	Body      models.UserRequest `json:"body"`
}

func StartNatsListener(nc *nats.Conn, wPool *worker.Pool, ldapClient *ldap.Client) {
	nc.Subscribe("access.grant", func(msg *nats.Msg) {
		wPool.Submit(func() {
			var env NatsEnvelope
			json.Unmarshal(msg.Data, &env)

			grant, err := ldapClient.PrepareGrantCommand(env.Body)
			
			if err != nil {
				reply, _ := json.Marshal(map[string]interface{}{
					"ok": false, "status": 500, "message": err.Error(),
				})
				msg.Respond(reply)
				return
			}

			commandStr := fmt.Sprintf("COMMAND:GRANT|USER:%s|GROUP:%s|EXPIRY:%d", 
				grant.UserDN, grant.GroupDN, grant.AccessExpiryTime)

			nc.Publish("ldap.console.execute", []byte(commandStr))

			reply, _ := json.Marshal(map[string]interface{}{
				"ok": true, "status": 200, "message": "Command Generated",
				"data": grant,
			})
			msg.Respond(reply)
		})
	})
}
