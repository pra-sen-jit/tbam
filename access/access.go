package access

import (
	"fmt"
	"tbam/models"
	"github.com/go-ldap/ldap/v3"
)

func GrantAccess(conn *ldap.Conn, req models.UserRequest) error {
	groupSearch := ldap.NewSearchRequest(
		"dc=example,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s))", req.GroupAssociated),
		[]string{"dn"}, nil,
	)
	gs, err := conn.Search(groupSearch)
	if err != nil || len(gs.Entries) == 0 {
		return fmt.Errorf("associated group %s not found", req.GroupAssociated)
	}
	baseGroupDN := gs.Entries[0].DN

	userSearch := ldap.NewSearchRequest(
		baseGroupDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", req.UID),
		[]string{"dn"}, nil,
	)
	us, err := conn.Search(userSearch)
	if err != nil || len(us.Entries) == 0 {
		return fmt.Errorf("user %s not found in group %s", req.UID, req.GroupAssociated)
	}
	userDN := us.Entries[0].DN

	privSearch := ldap.NewSearchRequest(
		"dc=example,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s))", req.PrivilegeAccess),
		[]string{"dn"}, nil,
	)
	ps, err := conn.Search(privSearch)
	if err != nil || len(ps.Entries) == 0 {
		return fmt.Errorf("privilege group %s not found", req.PrivilegeAccess)
	}
	privGroupDN := ps.Entries[0].DN

	modifyReq := ldap.NewModifyRequest(privGroupDN, nil)
	modifyReq.Add("member", []string{userDN})
	modifyReq.Add("accessStartDate", []string{req.StartDate})
	modifyReq.Add("accessStartTime", []string{req.StartTime})
	modifyReq.Add("accessEndDate", []string{req.EndDate})
	modifyReq.Add("accessEndTime", []string{req.EndTime})

	err = conn.Modify(modifyReq)
	if err != nil {
		return fmt.Errorf("failed to update privilege group: %w", err)
	}

	return nil
}
