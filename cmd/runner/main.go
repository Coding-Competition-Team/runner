package main

import (
	"math/rand"
	"os"
	"time"

	"runner/internal/api_sql"
	"runner/internal/ds"
	"runner/internal/creds"
	"runner/internal/workers"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 2 {
		panic("Usage: ./runner <absolute config folder path>")
	}
	folder_path := os.Args[1]

	_, err := os.Stat(folder_path) //Ensure folder_path exists
	if err != nil {
		panic(err)
	}
	ds.ConfigFolderPath = folder_path

	ds.LoadConfig()
	creds.LoadCredentials()
	api_sql.SyncWithDB()
	go workers.NewWorker(10 * time.Second).Run()
	go workers.JWTRefreshWorker()
	workers.HandleRequests()
}