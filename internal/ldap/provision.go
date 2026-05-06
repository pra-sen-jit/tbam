package ldap

import (
	"fmt"
	"time"
	"tbam/internal/models"
)

func (c *Client) PrepareGrantCommand(req models.UserRequest) (*models.AccessGrant, error) {
	// 1. BLIND MODE: No c.Borrow() and no conn.Search(). We skip LDAP entirely.

	// 2. IST TIMEZONE FIX: Set the exact timezone for Kolkata (UTC+5:30)
	loc := time.FixedZone("IST", 5*60*60+30*60) 
	timeLayout := "2006-01-02 15:04:05"
	dateTimeStr := fmt.Sprintf("%s %s", req.EndDate, req.EndTime)
	
	// Parse the time explicitly in IST
	parsedTime, err := time.ParseInLocation(timeLayout, dateTimeStr, loc)
	if err != nil {
		return nil, fmt.Errorf("time parsing failed: %v", err)
	}
	expiryUnix := parsedTime.Unix()

	// 3. RAW DATA MAPPING: Pass the text exactly as received in the JSON
	// Assuming a standard UID format since we can't search for the real DN
	userDN := fmt.Sprintf("uid=%s,dc=example,dc=com", req.UID) 
	groupDN := req.PrivilegeAccess

	return &models.AccessGrant{
		UserDN:           userDN,
		GroupDN:          groupDN,
		AccessExpiryTime: expiryUnix,
		RawAttribute:     fmt.Sprintf("%s|%d", groupDN, expiryUnix),
	}, nil
}
