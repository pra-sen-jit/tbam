package ldap

import (
	"fmt"
	"log"
	"os"

	"github.com/go-ldap/ldap/v3"
)

// Client holds our active LDAP connection
type Client struct {
	Conn *ldap.Conn
}

// Connect initializes the connection and authenticates (Binds) to the server
func Connect() (*Client, error) {
	url := os.Getenv("LDAP_URL")
	bindDN := os.Getenv("LDAP_BIND_DN")
	password := os.Getenv("LDAP_PASSWORD")

	log.Printf("Attempting to connect to LDAP at %s...", url)

	// 1. Dial the LDAP server
	l, err := ldap.DialURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}

	// 2. Authenticate using the Admin credentials (Bind)
	err = l.Bind(bindDN, password)
	if err != nil {
		l.Close() // Always clean up if the bind fails
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	log.Println("Successfully connected and bound to LDAP!")

	return &Client{Conn: l}, nil
}

// Close cleanly shuts down the connection when the app stops
func (c *Client) Close() {
	if c.Conn != nil {
		log.Println("Closing LDAP connection...")
		c.Conn.Close()
	}
}
