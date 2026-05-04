package ldap

import (
	"log"
	"os"

	"github.com/go-ldap/ldap/v3"
)

type Client struct {
	pool chan *ldap.Conn
	url  string
	user string
	pass string
}

// InitPool creates a connection pool of the specified size
func InitPool(size int) (*Client, error) {
	url := os.Getenv("LDAP_URL")
	user := os.Getenv("LDAP_BIND_DN")
	pass := os.Getenv("LDAP_PASSWORD")

	c := &Client{
		pool: make(chan *ldap.Conn, size),
		url:  url,
		user: user,
		pass: pass,
	}

	// Pre-fill the pool with nil connections to represent available capacity
	for i := 0; i < size; i++ {
		c.pool <- nil
	}
	
	log.Printf("LDAP Connection pool initialized with %d workers.", size)
	return c, nil
}

// Borrow retrieves a connection or dials a new one if necessary
func (c *Client) Borrow() (*ldap.Conn, error) {
	conn := <-c.pool

	if conn == nil || conn.IsClosing() {
		newConn, err := ldap.DialURL(c.url)
		if err != nil {
			return nil, err
		}
		err = newConn.Bind(c.user, c.pass)
		if err != nil {
			newConn.Close()
			c.pool <- nil
			return nil, err
		}
		return newConn, nil
	}
	return conn, nil
}

// Return puts the connection back into the pool
func (c *Client) Return(conn *ldap.Conn) {
	c.pool <- conn
}

func (c *Client) CloseAll() {
	close(c.pool)
	for conn := range c.pool {
		if conn != nil {
			conn.Close()
		}
	}
}
