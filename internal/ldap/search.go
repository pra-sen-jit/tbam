package ldap

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	
	"tbam/internal/models"
)

// Queries the LDAP directory for any user with the employeeNumber attribute
// employeeNumber is being used to hold accessExpiryTime
func (c *Client) FetchExpiringUsers() ([]models.ExpiringAccess, error) {
	log.Println("Searching LDAP for users with time-bound access...")

	// 1. Define the search request
	searchRequest := ldap.NewSearchRequest(
		"dc=nextgen,dc=local", // Base DN
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(employeeNumber=*)", // The Filter: "Has this attribute"
		[]string{"dn", "employeeNumber"}, // The attributes we want returned
		nil,
	)

	// 2. Execute the search
	result, err := c.Conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	// 3. Parse the results into our Go structs
	var expiringUsers []models.ExpiringAccess

	for _, entry := range result.Entries {
		expiryStr := entry.GetAttributeValue("employeeNumber")
		
		// Convert the string timestamp to an integer
		expiryInt, err := strconv.ParseInt(expiryStr, 10, 64)
		if err != nil {
			log.Printf("Warning: Failed to parse timestamp for %s. Value: %s", entry.DN, expiryStr)
			continue
		}

		user := models.ExpiringAccess{
			DN:               entry.DN,
			AccessExpiryTime: expiryInt,
		}
		expiringUsers = append(expiringUsers, user)
	}

	log.Printf("Found %d users with scheduled access expirations.", len(expiringUsers))
	return expiringUsers, nil
}
