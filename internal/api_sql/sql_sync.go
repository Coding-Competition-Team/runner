package api_sql

import (
	"time"

	"gorm.io/gorm"

	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
)

var DB *gorm.DB

func createTableIfNotExists(x interface{}) {
	err := DB.AutoMigrate(&x)
	if err != nil {
		panic(err)
	}
}

func initalizeDB() {
	createTableIfNotExists(ds.Instance{})
	createTableIfNotExists(ds.RunnerChallenge{})
}

func validatePortainerUrl(url string) bool {
	_, ok := creds.PortainerCreds[url]
	return ok
}

func syncInstances() {
	instances := []ds.Instance{}
	DB.Find(&instances) //Fully trust DB

	for _, instance := range instances {
		if !validatePortainerUrl(instance.Portainer_Url) {
			panic("Instance " + instance.ToString() + "'s Portainer_Url is not specified in credentials")
		}

		if (instance.Instance_Id + 1) > ds.NextInstanceId {
			ds.NextInstanceId = instance.Instance_Id + 1
		}
		ds.InstanceQueue.Put(instance.Instance_Timeout, instance.Instance_Id)

		deserialized_ports := DeserializeI(instance.Ports_Used)
		for _, port := range deserialized_ports {
			ds.UsedPorts[port] = true
		}

		creds.IncrementPortainerQueue(instance.Portainer_Url)
	}
}

func SyncWithDB() {
	log.Info("Starting DB Sync...")

	var err error
	DB, err = gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	initalizeDB()
	syncInstances()

	log.Info("DB Sync Complete!")
}