package creds

import (
	"math/rand"

	"github.com/emirpasic/gods/maps/treemap"

	"runner/internal/ds"
	"runner/internal/log"
)

var PortainerInstanceCounts map[string]int = make(map[string]int) //PortainerUrl -> InstanceCount (No. of instances running on that Portainer)
var PortainerQueue *treemap.Map = treemap.NewWithIntComparator() //InstanceCount -> {PortainerUrls}

func getPortainerQueueSet(instanceCount int) map[string]bool {
	val, ok := PortainerQueue.Get(instanceCount)
	var set map[string]bool
	if ok {
		set = val.(map[string]bool)
	} else {
		set = make(map[string]bool)
	}
	return set
}

func AddPortainerQueue(instanceCount int, url string) {
	set := getPortainerQueueSet(instanceCount)
	set[url] = true
	PortainerQueue.Put(instanceCount, set)
	_debug("ADD")
}

func RemovePortainerQueue(instanceCount int, url string) {
	set := getPortainerQueueSet(instanceCount)
	delete(set, url)
	if len(set) == 0 {
		PortainerQueue.Remove(instanceCount)
	} else {
		PortainerQueue.Put(instanceCount, set)
	}
	_debug("REMOVE")
}

func _debug(mode string){
	json, _ := PortainerQueue.ToJSON()
	log.Debug(mode, "PortainerQueue", string(json))
}

func GetBestPortainer() string { //TODO: Returns the Url of the least loaded Portainer
	if ds.PortainerBalanceStrategy == "RANDOM" {
		return PortainerUrls[rand.Intn(len(PortainerUrls))]
	} else if ds.PortainerBalanceStrategy == "DISTRIBUTE" {
		_, val := PortainerQueue.Min()
		set := val.(map[string]bool)

		for url := range set { //Get arbitrary url from set
			return url
		}
	}
	panic("Unexpected behavior") //TODO
}