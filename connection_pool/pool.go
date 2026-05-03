package ldapool

import (
	"github.com/go-ldap/ldap/v3"
)


type Pool struct {
	stalls chan *ldap.Conn
	addr string
	user string
	pass string
}

func New(size int, addr, user, pass string) *Pool {
	p := &Pool{
		stalls: make(chan *ldap.Conn, size),
		addr: addr,
		user: user,
		pass: pass,
	}

	for i := 0; i < size; i++ {
		p.stalls <- nil
	}

	return p
}

func (p *Pool) Borrow() (*ldap.Conn, error) {
	conn := <- p.stalls

	if conn == nil || conn.IsClosing() {
		newConn, err := ldap.DialURL(p.addr)

		if err != nil {
			return nil, err
		}

		err = newConn.Bind(p.user, p.pass)

		if err != nil {
			newConn.Close()
			p.stalls <- nil
			return nil, err
		}

		return newConn, nil
	}

	return conn, nil
}

func (p *Pool) Return(conn *ldap.Conn) {
	p.stalls <- conn
}
