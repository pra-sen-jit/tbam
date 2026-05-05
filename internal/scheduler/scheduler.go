package scheduler

import (
	"fmt"
	"os"
	"strconv"
	"tbam/internal/ldap"
	"tbam/internal/models"
	"time"

	"github.com/nats-io/nats.go"
)

func ScheduleNatsRevocation(grant models.AccessGrant, client *ldap.Client, nc *nats.Conn) {
	expiryTime := time.Unix(grant.AccessExpiryTime, 0)
	bufVal, _ := strconv.Atoi(os.Getenv("REVOCATION_BUFFER_SECONDS"))
	buffer := time.Duration(bufVal) * time.Second
	wakeUpTime := expiryTime.Add(-buffer)

	time.AfterFunc(time.Until(wakeUpTime), func() {
		revokeCmd := fmt.Sprintf("CMD:REVOKE|USER:%s|GRP:%s|TARGET_TIME:%d|ATTR:%s", 
			grant.UserDN, grant.GroupDN, grant.AccessExpiryTime, grant.RawAttribute)

		nc.Publish("ldap.console.execute", []byte(revokeCmd))
	})
}
