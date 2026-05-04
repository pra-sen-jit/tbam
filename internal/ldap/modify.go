package ldap

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// RevokeAccess removes the time-bound privilege from the user.
func (c *Client) RevokeAccess(userDN string) error {
	log.Printf("Executing LDAP Revocation for: %s", userDN)

	// 1. Create a Modify Request for the specific user
	modifyReq := ldap.NewModifyRequest(userDN, nil)

	// 2. Define the action: Delete the attribute
	modifyReq.Delete("employeeNumber", []string{}) 

	// Note: If you wanted to remove them from a group, you would add another line like:
	// modifyReq.Delete("memberOf", []string{"cn=PrivilegedAdmins,dc=nextgen,dc=local"})

	// 3. Send the command to the LDAP server
	err := c.Conn.Modify(modifyReq)
	if err != nil {
		return fmt.Errorf("failed to modify LDAP user %s: %w", userDN, err)
	}

	log.Printf("✅ SUCCESS: Access cleanly revoked for %s", userDN)
	return nil
}
