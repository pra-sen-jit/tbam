package ldap

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
	"tbam/internal/models"
)

// RevokeSpecificAccess removes the user from the specific privileged group and clears just that grant's timer.
func (c *Client) RevokeSpecificAccess(grant models.AccessGrant) error {
	conn, _ := c.Borrow()
	defer c.Return(conn)

	log.Printf("Executing Targeted LDAP Revocation for: %s from Group: %s", grant.UserDN, grant.GroupDN)

	// --- 1. REMOVE FROM SPECIFIC PRIVILEGED GROUP ---
	groupModifyReq := ldap.NewModifyRequest(grant.GroupDN, nil)
	groupModifyReq.Delete("uniqueMember", []string{grant.UserDN})
	
	err := conn.Modify(groupModifyReq)
	if err != nil {
		// Note: Error code 16 means "No Such Attribute" (they are already not in the group).
		return fmt.Errorf("failed to remove user from group %s: %w", grant.GroupDN, err)
	}
	log.Printf("-> Step 1 Complete: Kicked out of group: %s", grant.GroupDN)

	// --- 2. SURGICALLY CLEAR THE SPECIFIC TIMER ATTRIBUTE ---
	userModifyReq := ldap.NewModifyRequest(grant.UserDN, nil)
	
	// Notice we are passing the EXACT RawAttribute string (e.g. "cn=Group...|123456789")
	// LDAP will search the businessCategory array and delete ONLY this matching string.
	userModifyReq.Delete("businessCategory", []string{grant.RawAttribute})

	err = conn.Modify(userModifyReq)
	if err != nil {
		return fmt.Errorf("failed to clear specific expiry attribute for user %s: %w", grant.UserDN, err)
	}
	log.Printf("-> Step 2 Complete: Stripped targeted time-bound timer attribute")

	log.Printf("✅ SUCCESS: Specific Privileged Access cleanly revoked for %s", grant.UserDN)
	return nil
}
