package ds

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
)

var RunnerPort int //From Config

var InstanceQueue *treebidimap.Map = treebidimap.NewWith(utils.Int64Comparator, utils.IntComparator) //Unix (Nano) Timestamp of Instance Timeout -> InstanceId
var UsedPorts map[int]bool = make(map[int]bool)

var MaxInstanceCount int64 //From Config
var PortainerJWTSecondsPerRefresh int //From Config
var NextInstanceId int = 0
var DefaultSecondsPerInstance int64 //From Config
var DefaultNanosecondsPerInstance int64 //Indirectly From Config
var MaxSecondsLeftBeforeExtendAllowed int64 //From Config

var ChallengeUnsafeToLaunch map[string]bool = make(map[string]bool) //Challenges may become unsafe to launch when they are marked for removal via /removeChallenge

var Database_Max_Retry_Attempts int //From Config
var Database_Error_Wait_Seconds int //From Config

var PortainerBalanceStrategy string //From Config
var PortainerBalanceStrategies []string = []string{"RANDOM", "DISTRIBUTE"}

func GenerateChallengeId(challenge_name string) string {
	h := sha256.New()
	h.Write([]byte(challenge_name))
	return hex.EncodeToString(h.Sum(nil))
}

var PS string = "/"
var CredentialsFileName string = "credentials.json"
var ConfigFileName string = "config.json"

var ConfigFolderPath string //From args

func GetRandomPort() int { //Returns an (unused) random port from [1024, 65536)
	for {
		port := rand.Intn(65536-1024) + 1024
		if !UsedPorts[port] {
			UsedPorts[port] = true
			return port
		}
	}
}