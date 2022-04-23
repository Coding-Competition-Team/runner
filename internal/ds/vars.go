package ds

import (
	"crypto/sha256"
	"math/rand"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
)

var RunnerPort = 10000

var ActiveUserInstance map[int]int = make(map[int]int)                                               //UserId -> InstanceId
var InstanceMap map[int]Instance = make(map[int]Instance)                                            //InstanceId -> Instance
var InstanceQueue *treebidimap.Map = treebidimap.NewWith(utils.Int64Comparator, utils.IntComparator) //Unix (Nano) Timestamp of Instance Timeout -> InstanceId
var UsedPorts map[int]bool = make(map[int]bool)

var MaxInstanceCount int = 3
var NextInstanceId int = 1
var DefaultSecondsPerInstance int64 = 300
var DefaultNanosecondsPerInstance int64 = DefaultSecondsPerInstance * 1e9

var ChallengeMap map[string]Challenge = make(map[string]Challenge) //Challenge ID -> Challenges

func GenerateChallengeId(challenge_name string) string {
	h := sha256.New()
	h.Write([]byte(challenge_name))
	return string(h.Sum(nil))
}

var PS string = "/"
var ChallDataFolder string = "../../configs/CTF Challenge Data"
var CredentialsJsonFile string = "../../configs/Credentials/credentials.json"

func ReserveDefaultPorts() {
	UsedPorts[RunnerPort] = true //Runner
	UsedPorts[8000] = true       //Portainer
	UsedPorts[9443] = true       //Portainer
	UsedPorts[5432] = true       //Runner DB
	UsedPorts[22] = true         //SSH
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