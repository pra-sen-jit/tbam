package ldap

import (
	"fmt"
	"log"
	"tbam/internal/db"
	"tbam/internal/models"

	"github.com/go-ldap/ldap/v3"
)

// RevokeSpecificAccess removes the user from the specific privileged group and clears just that grant's timer.
func (c *Client) RevokeSpecificAccess(grant models.AccessGrant, dbClient *db.DBClient) error {
	conn, _ := c.Borrow()
	defer c.Return(conn)
	log.Printf("Executing Targeted LDAP Revocation for: %s from Group: %s", grant.UserDN, grant.GroupDN)

	groupModifyReq := ldap.NewModifyRequest(grant.GroupDN, nil)
	groupModifyReq.Delete("uniqueMember", []string{grant.UserDN})
	err := conn.Modify(groupModifyReq)
	if err != nil {
		dbClient.LogEvent("REVOKE_FAILED", grant.UserDN, grant.GroupDN, grant.AccessExpiryTime, "FAILED", err.Error())
		return fmt.Errorf("failed to remove user from group %s: %w", grant.GroupDN, err)
	}

	userModifyReq := ldap.NewModifyRequest(grant.UserDN, nil)
	userModifyReq.Delete("businessCategory", []string{grant.RawAttribute})
	err = conn.Modify(userModifyReq)
	if err != nil {
		return fmt.Errorf("failed to clear specific expiry attribute for user %s: %w", grant.UserDN, err)
	}

	log.Printf("✅ SUCCESS: Specific Privileged Access cleanly revoked for %s", grant.UserDN)
	dbClient.LogEvent("REVOKE", grant.UserDN, grant.GroupDN, grant.AccessExpiryTime, "SUCCESS", "Access successfully revoked")
	return nil
}
