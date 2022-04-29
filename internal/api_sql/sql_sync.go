package api_sql

import (
	"gorm.io/gorm"

	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
)

func syncChallenges() {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	challenges := []ds.Challenge{}
	db.Find(&challenges)

	for _, ch := range challenges {
		ds.ChallengeMap[ch.Challenge_Id] = ch
	}
}

func syncInstances() {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	instances := []ds.Instance{}
	db.Find(&instances) //Fully trust DB

	for _, instance := range instances {
		if (instance.Instance_Id + 1) > ds.NextInstanceId {
			ds.NextInstanceId = instance.Instance_Id + 1
		}
		ds.ActiveUserInstance[instance.Usr_Id] = instance.Instance_Id
		ds.InstanceQueue.Put(instance.Instance_Timeout, instance.Instance_Id)

		deserialized_ports := DeserializeI(instance.Ports_Used)
		for _, port := range deserialized_ports {
			ds.UsedPorts[port] = true
		}
		ds.InstanceMap[instance.Instance_Id] = instance
	}
}

func SyncWithDB() {
	log.Info("Starting DB Sync...")
	syncChallenges()
	log.Debug("Challenge Map:", ds.ChallengeMap)
	syncInstances()
	log.Debug("Instance Map:", ds.InstanceMap)
	log.Info("DB Sync Complete!")
}