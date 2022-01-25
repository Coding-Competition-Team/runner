package workers

import (
	"fmt"
	http_log "log"
	"net/http"
	"strconv"
	"time"

	"runner/internal/api_portainer"
	"runner/internal/api_sql"
	"runner/internal/ds"
	"runner/internal/log"
	"runner/internal/yaml"
)

func HandleRequests() {
	http.HandleFunc("/addInstance", addInstance)
	http.HandleFunc("/removeInstance", removeInstance)
	http.HandleFunc("/getTimeLeft", getTimeLeft)
	http.HandleFunc("/extendTimeLeft", extendTimeLeft)
	http_log.Fatal(http.ListenAndServe(":10000", nil))
}

//fmt.Fprintf() - print to web

func validateUserid(userid int) bool {
	return true
}

func validateChallid(challid int) bool {
	_, ok := ds.ChallengeIDMap[challid]
	return ok
}

func addInstance(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil {
		panic(err)
	}
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}

	_challid := params["challid"]
	if len(_challid) == 0 {
		http.Error(w, "Missing challid", http.StatusForbidden)
		return
	}
	challid, err := strconv.Atoi(_challid[0])
	if err != nil {
		panic(err)
	}
	if !validateChallid(challid) {
		http.Error(w, "Invalid challid", http.StatusForbidden)
		return
	}

	if ds.ActiveUserInstance[userid] > 0 {
		http.Error(w, "User is already running an instance", http.StatusForbidden)
		return
	}

	if len(ds.InstanceMap) >= ds.MaxInstanceCount { //Use >= instead of == just in case
		http.Error(w, "The max number of instances for the platform has already been reached, try again later", http.StatusForbidden)
		return
	}

	ch := ds.Challenges[ds.ChallengeIDMap[challid]]

	Ports := make([]int, ch.PortCount) //Cannot directly use [ch.PortCount]int
	for i := 0; i < ch.PortCount; i++ {
		Ports[i] = ds.GetRandomPort()
		fmt.Fprintln(w, strconv.Itoa(Ports[i]))
	}

	go _addInstance(userid, challid, Ports)
}

func _addInstance(userid int, challid int, Ports []int) { //Run Async
	InstanceId := ds.NextInstanceId
	ds.NextInstanceId++
	ds.ActiveUserInstance[userid] = InstanceId
	InstanceTimeLeft := time.Now().UnixNano() + ds.DefaultNanosecondsPerInstance
	ds.InstanceQueue.Put(InstanceTimeLeft, InstanceId) //Use higher precision time to (hopefully) prevent duplicates
	discriminant := strconv.FormatInt(time.Now().Unix(), 10)
	ds.InstanceMap[InstanceId] = ds.InstanceData{UserId: userid, ChallengeId: challid, InstanceTimeLeft: InstanceTimeLeft, Ports: Ports} //Everything except PortainerId first, to prevent issues when querying getTimeLeft, etc. while the instance is launching

	api_sql.AddInstance(InstanceId, userid, challid, InstanceTimeLeft, Ports)

	var PortainerId string

	ch := ds.Challenges[ds.ChallengeIDMap[challid]]
	if ch.DockerCompose {
		new_docker_compose := yaml.DockerComposeCopy(ch.DockerComposeFile, Ports)
		PortainerId = api_portainer.LaunchStack(ch.ChallengeName, new_docker_compose, discriminant)
	} else {
		PortainerId = api_portainer.LaunchContainer(ch.ChallengeName, ch.ImageName, ch.DockerCmds, ch.InternalPort, Ports[0], discriminant)
	}

	log.Debug("Portainer ID:", PortainerId)

	entry := ds.InstanceMap[InstanceId]
	entry.PortainerId = PortainerId
	ds.InstanceMap[InstanceId] = entry //Update PortainerId once it's available

	api_sql.SetInstancePortainerId(InstanceId, PortainerId)
}

func removeInstance(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil {
		panic(err)
	}
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}

	if ds.ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	InstanceId := ds.ActiveUserInstance[userid]
	if ds.InstanceMap[InstanceId].PortainerId == "" {
		http.Error(w, "The instance is still starting", http.StatusForbidden)
		return
	}

	go _removeInstance(InstanceId)
}

func _removeInstance(InstanceId int) { //Run Async
	var NewInstanceTimeLeft int64 = 0 //Force typecast

	entry := ds.InstanceMap[InstanceId]
	entry.InstanceTimeLeft = NewInstanceTimeLeft //Make sure that the instance will be killed in the next kill cycle
	ds.InstanceMap[InstanceId] = entry

	a, b := ds.InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(NewInstanceTimeLeft, InstanceId) //Replace

	//No need to update DB since it will be removed anyways...
}

func getTimeLeft(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil {
		panic(err)
	}
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}

	if ds.ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	InstanceId := ds.ActiveUserInstance[userid]

	fmt.Fprint(w, strconv.FormatInt((ds.InstanceMap[InstanceId].InstanceTimeLeft-time.Now().UnixNano())/1e9, 10))
}

func extendTimeLeft(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil {
		panic(err)
	}
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}

	if ds.ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	go _extendTimeLeft(userid)
}

func _extendTimeLeft(userid int) { //Run Async
	InstanceId := ds.ActiveUserInstance[userid]
	entry := ds.InstanceMap[InstanceId]

	NewInstanceTimeLeft := entry.InstanceTimeLeft + ds.DefaultNanosecondsPerInstance
	entry.InstanceTimeLeft = NewInstanceTimeLeft
	ds.InstanceMap[InstanceId] = entry

	a, b := ds.InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(NewInstanceTimeLeft, InstanceId) //Replace

	api_sql.UpdateInstanceTime(InstanceId, NewInstanceTimeLeft)
}