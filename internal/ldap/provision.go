package ldap

import (
	"fmt"
	"log"
	"os"
	"tbam/internal/db"
	"tbam/internal/models"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// GrantAccess provisions the LDAP group and sets the timer attribute
func (c *Client) GrantAccess(req models.UserRequest, dbClient *db.DBClient) (*models.AccessGrant, error) {
	conn, err := c.Borrow()
	if err != nil {
		return nil, fmt.Errorf("failed to borrow connection: %w", err)
	}
	defer c.Return(conn)

	timeLayout := "2006-01-02 15:04:05" // Go's magic reference date
	dateTimeStr := fmt.Sprintf("%s %s", req.EndDate, req.EndTime)
	parsedTime, err := time.Parse(timeLayout, dateTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date/time format: %w", err)
	}
	expiryUnix := parsedTime.Unix()

	// Use this code if full DN arrives in JSON request body
	// privSearch := ldap.NewSearchRequest(
	// 	req.PrivilegeAccess,
	// 	ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
	// 	"(objectClass=groupOfUniqueNames)",
	// 	[]string{"dn"}, nil,
	// )
	// Use this code block if only Group CN arrives in JSON request body
	privSearch := ldap.NewSearchRequest(
		os.Getenv("LDAP_BASE_DN"),
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=groupOfUniqueNames)(cn=%s))", req.PrivilegeAccess),
		[]string{"dn"}, nil,
	)
	ps, err := conn.Search(privSearch)
	if err != nil || len(ps.Entries) == 0 {
		return nil, fmt.Errorf("privilege group %s not found", req.PrivilegeAccess)
	}
	privGroupDN := ps.Entries[0].DN

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

	// --- EXECUTES THE 1:N PROVISIONING ---
	groupModifyReq := ldap.NewModifyRequest(privGroupDN, nil)
	groupModifyReq.Add("uniqueMember", []string{userDN})
	if err := conn.Modify(groupModifyReq); err != nil {
		return nil, fmt.Errorf("failed to add user to group: %w", err)
	}
	grantString := fmt.Sprintf("%s|%d", privGroupDN, expiryUnix)
	userModifyReq := ldap.NewModifyRequest(userDN, nil)
	userModifyReq.Add("businessCategory", []string{grantString})
	if err := conn.Modify(userModifyReq); err != nil {
		return nil, fmt.Errorf("failed to attach expiry timer to user: %w", err)
	}
	log.Printf("Provisioned %s into %s until %s", req.UID, req.PrivilegeAccess, dateTimeStr)
	dbClient.LogEvent("GRANT", req.UID, privGroupDN, expiryUnix, "SUCCESS", grantString)

	return &models.AccessGrant{
		UserDN:           userDN,
		GroupDN:          privGroupDN,
		AccessExpiryTime: expiryUnix,
		RawAttribute:     grantString,
	}, nil
}
