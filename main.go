package main

import (
	"fmt"
	"time"

	"tbam/ldapool"
)

func main() {
	pool := ldapool.New(10, "ldap://localhost:10389", "uid=admin,ou=system", "secret")

	fmt.Println("exection\n")

	go func() {
		conn, err := pool.Borrow()

		if err != nil {
			fmt.Println("Error Connecting\n")
			pool.Return(nil)
			return
		}

		fmt.Println("Borrowed and Connected\n")
		pool.Return(conn)
	}()

	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
