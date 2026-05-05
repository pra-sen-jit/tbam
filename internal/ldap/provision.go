package ldap

import (
	"fmt"
	"os"
	"time"
	"tbam/internal/models"
	"github.com/go-ldap/ldap/v3"
)

func (c *Client) PrepareGrantCommand(req models.UserRequest) (*models.AccessGrant, error) {
	conn, err := c.Borrow()
	if err != nil {
		return nil, err
	}
	defer c.Return(conn)

	timeLayout := "2006-01-02 15:04:05"
	dateTimeStr := fmt.Sprintf("%s %s", req.EndDate, req.EndTime)
	parsedTime, err := time.Parse(timeLayout, dateTimeStr)
	if err != nil {
		return nil, err
	}
	expiryUnix := parsedTime.Unix()

	privSearch := ldap.NewSearchRequest(
		req.PrivilegeAccess,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=groupOfNames)",
		[]string{"dn"}, nil,
	)
	ps, err := conn.Search(privSearch)
	if err != nil || len(ps.Entries) == 0 {
		return nil, fmt.Errorf("privilege group not found")
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
		return nil, fmt.Errorf("user not found")
	}

	return &models.AccessGrant{
		UserDN:           us.Entries[0].DN,
		GroupDN:          privGroupDN,
		AccessExpiryTime: expiryUnix,
		RawAttribute:     fmt.Sprintf("%s|%d", privGroupDN, expiryUnix),
	}, nil
}
