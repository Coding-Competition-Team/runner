package main

import (
	"math/rand"
	"time"

	"runner/internal/api_sql"
	"runner/internal/ds"
	"runner/internal/creds"
	"runner/internal/workers"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ds.ReserveDefaultPorts()
	creds.LoadCredentials()
	api_sql.SyncWithDB()
	go workers.NewWorker(10 * time.Second).Run()
	workers.HandleRequests()
}