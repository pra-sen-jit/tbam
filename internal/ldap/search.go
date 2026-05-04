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

// FetchExpiringGrants queries the LDAP directory for any user with the businessCategory attribute
// businessCategory holds an array of strings formatted as "GroupDN|Timestamp"
func (c *Client) FetchExpiringGrants() ([]models.AccessGrant, error) {
	conn, _ := c.Borrow()
	defer c.Return(conn)

	log.Println("Searching LDAP for active time-bound access grants...")

	// 1. Define the search request
	searchRequest := ldap.NewSearchRequest(
		os.Getenv("LDAP_BASE_DN"), // Base DN
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(businessCategory=*)", // The Filter: "Has this attribute"
		[]string{"dn", "businessCategory"}, // The attributes we want returned
		nil,
	)

	// 2. Execute the search
	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	// 3. Parse the results into our Go structs
	var allGrants []models.AccessGrant

	for _, entry := range result.Entries {
		// Use GetAttributeValues (plural) to get the array of strings
		grantStrings := entry.GetAttributeValues("businessCategory")

		for _, grantStr := range grantStrings {
			// Split the string by the pipe "|"
			parts := strings.Split(grantStr, "|")
			if len(parts) != 2 {
				log.Printf("Warning: Malformed grant string found for %s: %s", entry.DN, grantStr)
				continue
			}

			groupDN := parts[0]
			expiryStr := parts[1]

			// Convert the string timestamp to an integer
			expiryInt, err := strconv.ParseInt(expiryStr, 10, 64)
			if err != nil {
				log.Printf("Warning: Failed to parse timestamp in grant for %s. Value: %s", entry.DN, expiryStr)
				continue
			}

			grant := models.AccessGrant{
				UserDN:           entry.DN,
				GroupDN:          groupDN,
				AccessExpiryTime: expiryInt,
				RawAttribute:     grantStr, // We save this so we can delete it specifically later
			}
			allGrants = append(allGrants, grant)
		}
	}

	log.Printf("Found %d scheduled access revocations.", len(allGrants))
	return allGrants, nil
}
