package ds

import (
	"math/rand"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
)

var ActiveUserInstance map[int]int = make(map[int]int)                                               //UserId -> InstanceId
var InstanceMap map[int]InstanceData = make(map[int]InstanceData)                                    //InstanceId -> InstanceData
var InstanceQueue *treebidimap.Map = treebidimap.NewWith(utils.Int64Comparator, utils.IntComparator) //Unix (Nano) Timestamp of Instance Timeout -> InstanceId
var UsedPorts map[int]bool = make(map[int]bool)

var MaxInstanceCount int = 3
var NextInstanceId int = 1
var DefaultSecondsPerInstance int64 = 300
var DefaultNanosecondsPerInstance int64 = DefaultSecondsPerInstance * 1e9

var ChallengeMap map[int]Challenge = make(map[int]Challenge) //Challenge ID -> Challenges
var ChallengeNamesMap map[string]int = make(map[string]int)  //Challenge Name -> Challenge ID

var PS string = "/"
var ChallDataFolder string = "../../configs/CTF Challenge Data"
var CredentialsJsonFile string = "../../configs/Credentials/credentials.json"

func ReserveDefaultPorts() {
	UsedPorts[8000] = true //Portainer
	UsedPorts[9443] = true //Portainer
	UsedPorts[3306] = true //Runner DB
	UsedPorts[22] = true   //SSH
}

func GetRandomPort() int { //Returns an (unused) random port from [1024, 65536)
	for {
		port := rand.Intn(65536-1024) + 1024
		if !UsedPorts[port] {
			UsedPorts[port] = true
			return port
		}
	}
}