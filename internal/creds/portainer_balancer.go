package creds

import (
	"math/rand"
)

var PortainerInstanceCounts map[string]int = make(map[string]int) //PortainerUrl -> No. of instances running on that Portainer

func GetBestPortainer() string { //TODO: Returns the Url of the least loaded Portainer
	return PortainerUrls[rand.Intn(len(PortainerUrls))]
}