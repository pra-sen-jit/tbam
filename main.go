package main

import (
	"fmt"
	"tbam/connection_pool"
	"tbam/worker_pool"
	"tbam/msgQueue"
)

func main() {
	cPool := ldapool.New(100, "ldap://localhost:10389", "uid=admin,ou=system", "secret")
	wPool := workerpool.New(150, 5000)

	fmt.Println("Started.....")

	msgQueue.StartServer(wPool, cPool)
}






