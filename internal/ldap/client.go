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

	// Pre-fill the pool with nil to represent free slots
	for i := 0; i < size; i++ {
		c.pool <- nil
	}

	log.Printf("LDAP Connection pool initialized with %d potential connections.", size)
	return c, nil
}

func (c *Client) Borrow() (*ldap.Conn, error) {
	conn := <-c.pool

	// If we have a living connection, return it directly
	if conn != nil && !conn.IsClosing() {
		return conn, nil
	}

	// Dead or nil – try to create a fresh one
	newConn, err := ldap.DialURL(c.url)
	if err != nil {
		// IMPORTANT: put back a nil so the slot isn't lost
		c.pool <- nil
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

func (c *Client) Return(conn *ldap.Conn) {
	if conn != nil && !conn.IsClosing() {
		c.pool <- conn
	} else {
		c.pool <- nil
	}
}

func (c *Client) CloseAll() {
	close(c.pool)
	for conn := range c.pool {
		if conn != nil {
			conn.Close()
		}
	}
}
