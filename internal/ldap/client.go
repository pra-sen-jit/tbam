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

	for i := 0; i < size; i++ {
		c.pool <- nil
	}
	log.Printf("LDAP Connection pool initialized with %d workers.", size)
	return c, nil
}

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
