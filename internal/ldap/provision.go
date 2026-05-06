package ldap

import (
	"fmt"
	"os"
	"time"

	"tbam/internal/models"

	"github.com/go-ldap/ldap/v3"
)

// PrepareGrantCommand matches the call in handlers.go
func (c *Client) PrepareGrantCommand(req models.UserRequest) (*models.AccessGrant, error) {
	conn, err := c.Borrow()
	if err != nil {
		return nil, fmt.Errorf("failed to borrow connection: %w", err)
	}
	defer c.Return(conn)

	// 1. Parse Date and Time into Unix Timestamp
	timeLayout := "2006-01-02 15:04:05" 
	dateTimeStr := fmt.Sprintf("%s %s", req.EndDate, req.EndTime)
	parsedTime, err := time.Parse(timeLayout, dateTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date/time format: %w", err)
	}
	expiryUnix := parsedTime.Unix()

	// 2. Find Group DN
	privSearch := ldap.NewSearchRequest(
		req.PrivilegeAccess,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=groupOfNames)",
		[]string{"dn"}, nil,
	)
	ps, err := conn.Search(privSearch)
	if err != nil || len(ps.Entries) == 0 {
		return nil, fmt.Errorf("privilege group %s not found", req.PrivilegeAccess)
	}
	privGroupDN := ps.Entries[0].DN

	// 3. Find User DN
	userSearch := ldap.NewSearchRequest(
		os.Getenv("LDAP_BASE_DN"),
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", req.UID),
		[]string{"dn"}, nil,
	)
	us, err := conn.Search(userSearch)
	if err != nil || len(us.Entries) == 0 {
		return nil, fmt.Errorf("user %s not found", req.UID)
	}
	userDN := us.Entries[0].DN

	// 4. Return the structured grant
	return &models.AccessGrant{
		UserDN:           userDN,
		GroupDN:          privGroupDN,
		AccessExpiryTime: expiryUnix,
		// We add the logic here but the worker executes the NATS publish
	}, nil
}
