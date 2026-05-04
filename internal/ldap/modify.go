package ldap

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// RevokeAccess completely removes the user from the privileged group and clears their expiry timer.
func (c *Client) RevokeAccess(userDN string) error {
	log.Printf("Executing LDAP Revocation for: %s", userDN)

	// --- 1. REMOVE FROM PRIVILEGED GROUP ---
	groupDN := "cn=PrivilegedGroup,ou=Groups,dc=example,dc=com"
	groupModifyReq := ldap.NewModifyRequest(groupDN, nil)
	groupModifyReq.Delete("member", []string{userDN})
	err := c.Conn.Modify(groupModifyReq)
	if err != nil {
		// Note: Error code 16 means "No Such Attribute" (they are already not in the group).
		return fmt.Errorf("failed to remove user from group %s: %w", groupDN, err)
	}
	log.Printf("-> Step 1 Complete: Kicked out of PrivilegedAdmins group")

	// --- 2. CLEAR THE TIMER ATTRIBUTE ---
	userModifyReq := ldap.NewModifyRequest(userDN, nil)
	userModifyReq.Delete("employeeNumber", []string{}) // Our hijacked timestamp attribute

	err = c.Conn.Modify(userModifyReq)
	if err != nil {
		return fmt.Errorf("failed to clear expiry attribute for user %s: %w", userDN, err)
	}
	log.Printf("-> Step 2 Complete: Stripped time-bound timer attribute")

	log.Printf("✅ SUCCESS: Total Privileged Access cleanly revoked for %s", userDN)
	return nil
}
