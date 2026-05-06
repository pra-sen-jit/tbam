package ldap

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"tbam/internal/models"
)

func (c *Client) FetchExpiringGrants() ([]models.AccessGrant, error) {
	conn, err := c.Borrow()
	if err != nil {
		return nil, fmt.Errorf("cannot borrow connection: %w", err)
	}
	defer c.Return(conn)

	log.Println("Searching LDAP for active time-bound access grants...")

	searchRequest := ldap.NewSearchRequest(
		os.Getenv("LDAP_BASE_DN"),
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(businessCategory=*)",
		[]string{"dn", "businessCategory"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	var allGrants []models.AccessGrant
	for _, entry := range result.Entries {
		grantStrings := entry.GetAttributeValues("businessCategory")
		for _, grantStr := range grantStrings {
			parts := strings.Split(grantStr, "|")
			if len(parts) != 2 {
				log.Printf("Warning: Malformed grant string found for %s: %s", entry.DN, grantStr)
				continue
			}
			groupDN := parts[0]
			expiryStr := parts[1]
			expiryInt, err := strconv.ParseInt(expiryStr, 10, 64)
			if err != nil {
				log.Printf("Warning: Failed to parse timestamp for %s: %s", entry.DN, expiryStr)
				continue
			}
			grant := models.AccessGrant{
				UserDN:           entry.DN,
				GroupDN:          groupDN,
				AccessExpiryTime: expiryInt,
				RawAttribute:     grantStr,
			}
			allGrants = append(allGrants, grant)
		}
	}

	log.Printf("Found %d scheduled access revocations.", len(allGrants))
	return allGrants, nil
}
