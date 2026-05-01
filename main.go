package main

import (
	"fmt"
	"time"
	"tbam/connection_pool"
	"tbam/worker_pool"
)

func main() {
	cPool := ldapool.New(10, "ldap://localhost", "admin", "pass")
	wPool := workerpool.New(50, 10000)

	for i := 1; i <= 10000; i++ {
		userID := fmt.Sprintf("User-%d", i)

		wPool.Submit(func() {
			conn, err := cPool.Borrow()
			if err != nil {
				fmt.Println("Failed to get connection")
				cPool.Return(nil)
				return
			}

			fmt.Printf("Updating LDAP for %s\n", userID)
			time.Sleep(50 * time.Millisecond)

			cPool.Return(conn)
		})
	}

	time.Sleep(10 * time.Second)
}
