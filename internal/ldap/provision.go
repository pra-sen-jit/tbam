package ldap

import (
	"fmt"
	"log"
	// "os"
	"time"

	"tbam/internal/models"

	"github.com/go-ldap/ldap/v3"
)

// GrantAccess provisions the LDAP group and sets the timer attribute
func (c *Client) GrantAccess(req models.UserRequest) (*models.AccessGrant, error) {
	conn, err := c.Borrow()
	if err != nil {
		return nil, fmt.Errorf("failed to borrow connection: %w", err)
	}
	defer c.Return(conn)

	// 1. Parse Date and Time into Unix Timestamp
	timeLayout := "2006-01-02 15:04:05" // Go's magic reference date
	dateTimeStr := fmt.Sprintf("%s %s", req.EndDate, req.EndTime)
	parsedTime, err := time.Parse(timeLayout, dateTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date/time format: %w", err)
	}
	expiryUnix := parsedTime.Unix()

	// 2. Find the target Privileged Group DN
	// Use this code if full DN arrives in JSON request body
	privSearch := ldap.NewSearchRequest(
		req.PrivilegeAccess,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=groupOfNames)",
		[]string{"dn"}, nil,
	)
	// Use this code block if only Group CN arrives in JSON request body
	// privSearch := ldap.NewSearchRequest(
	// 	os.Getenv("LDAP_BASE_DN"),
	// 	ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
	// 	fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s))", req.PrivilegeAccess),
	// 	[]string{"dn"}, nil,
	// )
	ps, err := conn.Search(privSearch)
	if err != nil || len(ps.Entries) == 0 {
		return nil, fmt.Errorf("privilege group %s not found", req.PrivilegeAccess)
	}
	privGroupDN := ps.Entries[0].DN

	// 3. Find the User DN based on UID
	userSearch := ldap.NewSearchRequest(
		"dc=example,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", req.UID),
		[]string{"dn"}, nil,
	)
	us, err := conn.Search(userSearch)
	if err != nil || len(us.Entries) == 0 {
		return nil, fmt.Errorf("user %s not found", req.UID)
	}
	userDN := us.Entries[0].DN

	// --- 4. EXECUTE THE 1:N PROVISIONING ---
	
	// A. Add User to Group
	groupModifyReq := ldap.NewModifyRequest(privGroupDN, nil)
	groupModifyReq.Add("member", []string{userDN})
	if err := conn.Modify(groupModifyReq); err != nil {
		return nil, fmt.Errorf("failed to add user to group: %w", err)
	}

	// B. Append the Timer String to the User
	grantString := fmt.Sprintf("%s|%d", privGroupDN, expiryUnix)
	userModifyReq := ldap.NewModifyRequest(userDN, nil)
	userModifyReq.Add("businessCategory", []string{grantString})
	
	if err := conn.Modify(userModifyReq); err != nil {
		return nil, fmt.Errorf("failed to attach expiry timer to user: %w", err)
	}

	log.Printf("Provisioned %s into %s until %s", req.UID, req.PrivilegeAccess, dateTimeStr)

	// Return the structured grant so the API can schedule the timer immediately
	return &models.AccessGrant{
		UserDN:           userDN,
		GroupDN:          privGroupDN,
		AccessExpiryTime: expiryUnix,
		RawAttribute:     grantString,
	}, nil
}
