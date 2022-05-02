package workers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	http_log "log"
	"net/http"
	"strconv"
	"time"

	"runner/internal/api_portainer"
	"runner/internal/api_sql"
	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
	"runner/internal/yaml"
)

func HandleRequests() {
	http.HandleFunc("/addInstance", addInstance)
	http.HandleFunc("/removeInstance", removeInstance)
	http.HandleFunc("/getUserStatus", getUserStatus)
	http.HandleFunc("/extendTimeLeft", extendTimeLeft)
	http.HandleFunc("/addChallenge", addChallenge)
	http.HandleFunc("/removeChallenge", removeChallenge)
	http.HandleFunc("/getStatus", getStatus)
	http_log.Fatal(http.ListenAndServe(":" + strconv.Itoa(ds.RunnerPort), nil))
}

//fmt.Fprintf() - print to web

func validateUserid(userid string) bool {
	return true
}

func validateChallid(challid string) bool {
	_, ok := ds.ChallengeMap[challid]
	if ok { //If challid exists in ChallengeMap, check if it is not unsafe to launch
		return !ds.ChallengeUnsafeToLaunch[challid]
	}
	return false //challid does not exist in ChallengeMap
}

func activeUserInstance(userid string) bool {
	_, ok := ds.ActiveUserInstance[userid]
	return ok
}

func addInstance(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /addInstance Request")

	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, ds.Error{Error: "Missing userid"}.ToString(), http.StatusForbidden)
		return
	}
	userid := _userid[0]

	if !validateUserid(userid) {
		http.Error(w, ds.Error{Error: "Invalid userid"}.ToString(), http.StatusForbidden)
		return
	}

	_challid := params["challid"]
	if len(_challid) == 0 {
		http.Error(w, ds.Error{Error: "Missing challid"}.ToString(), http.StatusForbidden)
		return
	}
	challid := _challid[0]
	if !validateChallid(challid) {
		http.Error(w, ds.Error{Error: "Invalid challid"}.ToString(), http.StatusForbidden)
		return
	}

	if activeUserInstance(userid) {
		http.Error(w, ds.Error{Error: "User is already running an instance"}.ToString(), http.StatusForbidden)
		return
	}

	if len(ds.InstanceMap) >= ds.MaxInstanceCount { //Use >= instead of == just in case
		http.Error(w, ds.Error{Error: "The max number of instances for the platform has already been reached, try again later"}.ToString(), http.StatusForbidden)
		return
	}

	ch := ds.ChallengeMap[challid]

	var ports ds.PortsJson
	ports.Ports_Used = make([]int, ch.Port_Count) //Cannot directly use [ch.PortCount]int
	for i := 0; i < ch.Port_Count; i++ {
		ports.Ports_Used[i] = ds.GetRandomPort()
	}

	fmt.Fprint(w, ports.ToString())

	go _addInstance(userid, challid, ports.Ports_Used)
}

func _addInstance(userid string, challid string, Ports []int) { //Run Async
	log.Debug("Start /addInstance Request")
	InstanceId := ds.NextInstanceId
	ds.NextInstanceId++
	ds.ActiveUserInstance[userid] = InstanceId
	InstanceTimeout := time.Now().UnixNano() + ds.DefaultNanosecondsPerInstance
	ds.InstanceQueue.Put(InstanceTimeout, InstanceId) //Use higher precision time to (hopefully) prevent duplicates
	discriminant := strconv.FormatInt(time.Now().UnixNano(), 10)
	portainer_url := creds.GetBestPortainer()
	if ds.PortainerBalanceStrategy == "DISTRIBUTE" {
		creds.RemovePortainerQueue(creds.PortainerInstanceCounts[portainer_url], portainer_url)
		creds.PortainerInstanceCounts[portainer_url] += 1
		creds.AddPortainerQueue(creds.PortainerInstanceCounts[portainer_url], portainer_url)
	}

	instance := ds.Instance{Instance_Id: InstanceId, Usr_Id: userid, Challenge_Id: challid, Portainer_Url: portainer_url, Instance_Timeout: InstanceTimeout, Ports_Used: api_sql.SerializeI(Ports, ",")} //Everything except PortainerId first, to prevent issues when querying getTimeLeft, etc. while the instance is launching
	ds.InstanceMap[InstanceId] = instance

	api_sql.AddInstance(instance)

	var PortainerId string

	ch := ds.ChallengeMap[challid]
	if ch.Docker_Compose {
		new_docker_compose := yaml.DockerComposeCopy(ch.Docker_Compose_File, Ports)
		PortainerId = api_portainer.LaunchStack(portainer_url, ch.Challenge_Name, new_docker_compose, discriminant)
	} else {
		PortainerId = api_portainer.LaunchContainer(portainer_url, ch.Challenge_Name, ch.Image_Name, api_sql.DeserializeNL(ch.Docker_Cmds), ch.Internal_Port, Ports[0], discriminant)
	}

	log.Debug("Portainer ID:", PortainerId)

	entry := ds.InstanceMap[InstanceId]
	entry.Portainer_Id = PortainerId
	ds.InstanceMap[InstanceId] = entry //Update PortainerId once it's available

	api_sql.SetInstancePortainerId(InstanceId, PortainerId)
	log.Debug("Finish /addInstance Request")
}

func removeInstance(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /removeInstance Request")

	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, ds.Error{Error: "Missing userid"}.ToString(), http.StatusForbidden)
		return
	}
	userid := _userid[0]

	if !validateUserid(userid) {
		http.Error(w, ds.Error{Error: "Invalid userid"}.ToString(), http.StatusForbidden)
		return
	}

	if !activeUserInstance(userid) {
		http.Error(w, ds.Error{Error: "User does not have an instance"}.ToString(), http.StatusForbidden)
		return
	}

	InstanceId := ds.ActiveUserInstance[userid]
	if ds.InstanceMap[InstanceId].Portainer_Id == "" {
		http.Error(w, ds.Error{Error: "The instance is still starting"}.ToString(), http.StatusForbidden)
		return
	}

	fmt.Fprint(w, ds.Success{Success: true}.ToString())

	go _removeInstance(InstanceId)
}

func _removeInstance(InstanceId int) { //Run Async
	log.Debug("Start /removeInstance Request")
	var NewInstanceTimeout int64 = 0 //Force typecast

	entry := ds.InstanceMap[InstanceId]

	portainer_url := entry.Portainer_Url
	if ds.PortainerBalanceStrategy == "DISTRIBUTE" {
		creds.RemovePortainerQueue(creds.PortainerInstanceCounts[portainer_url], portainer_url)
		creds.PortainerInstanceCounts[portainer_url] -= 1
		creds.AddPortainerQueue(creds.PortainerInstanceCounts[portainer_url], portainer_url)
	}

	entry.Instance_Timeout = NewInstanceTimeout //Make sure that the instance will be killed in the next kill cycle
	ds.InstanceMap[InstanceId] = entry

	a, b := ds.InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(NewInstanceTimeout, InstanceId) //Replace

	//No need to update DB since it will be removed anyways...
	log.Debug("Finish /removeInstance Request")
}

func getUserStatus(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /getUserStatus Request")

	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, ds.Error{Error: "Missing userid"}.ToString(), http.StatusForbidden)
		return
	}
	userid := _userid[0]

	if !validateUserid(userid) {
		http.Error(w, ds.Error{Error: "Invalid userid"}.ToString(), http.StatusForbidden)
		return
	}

	if !activeUserInstance(userid) {
		fmt.Fprint(w, ds.UserStatus{Running_Instance: false}.ToString())
		return
	}

	log.Debug("Start /getUserStatus Request")

	InstanceId := ds.ActiveUserInstance[userid]

	fmt.Fprint(w, ds.UserStatus{Running_Instance: true, Challenge_Name: ds.ChallengeMap[ds.InstanceMap[InstanceId].Challenge_Id].Challenge_Name, Time_Left: int((ds.InstanceMap[InstanceId].Instance_Timeout-time.Now().UnixNano())/1e9), IP_Address: ds.InstanceMap[InstanceId].Portainer_Url, Ports_Used: ds.InstanceMap[InstanceId].Ports_Used}.ToString())

	log.Debug("Finish /getUserStatus Request")
}

func extendTimeLeft(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /extendTimeLeft Request")

	params := r.URL.Query()

	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, ds.Error{Error: "Missing userid"}.ToString(), http.StatusForbidden)
		return
	}
	userid := _userid[0]

	if !validateUserid(userid) {
		http.Error(w, ds.Error{Error: "Invalid userid"}.ToString(), http.StatusForbidden)
		return
	}

	if !activeUserInstance(userid) {
		http.Error(w, ds.Error{Error: "User does not have an instance"}.ToString(), http.StatusForbidden)
		return
	}

	InstanceId := ds.ActiveUserInstance[userid]

	if (ds.InstanceMap[InstanceId].Instance_Timeout-time.Now().UnixNano())/1e9 > ds.MaxSecondsLeftBeforeExtendAllowed {
		http.Error(w, ds.Error{Error: "User needs to wait until instance expires in " + strconv.FormatInt(ds.MaxSecondsLeftBeforeExtendAllowed, 10) + " seconds"}.ToString(), http.StatusForbidden)
		return
	}

	fmt.Fprint(w, ds.Success{Success: true}.ToString())

	go _extendTimeLeft(userid)
}

func _extendTimeLeft(userid string) { //Run Async
	log.Debug("Start /extendTimeLeft Request")
	InstanceId := ds.ActiveUserInstance[userid]
	entry := ds.InstanceMap[InstanceId]

	NewInstanceTimeout := time.Now().UnixNano() + ds.DefaultNanosecondsPerInstance
	entry.Instance_Timeout = NewInstanceTimeout
	ds.InstanceMap[InstanceId] = entry

	a, b := ds.InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(NewInstanceTimeout, InstanceId) //Replace

	api_sql.UpdateInstanceTime(InstanceId, NewInstanceTimeout)
	log.Debug("Finish /extendTimeLeft Request")
}

func addChallenge(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /addChallenge Request")

	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, ds.Error{Error: "Authorization missing"}.ToString(), http.StatusBadRequest)
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		http.Error(w, ds.Error{Error: "Invalid authorization"}.ToString(), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ds.Error{Error: "Cannot read body"}.ToString(), http.StatusBadRequest)
		return
	}

	var result map[string]string
	json.Unmarshal(body, &result)

	challenge_name, ok := result["challenge_name"]
	if !ok {
		http.Error(w, ds.Error{Error: "Missing challenge_name"}.ToString(), http.StatusBadRequest)
		return
	}
	_docker_compose, ok := result["docker_compose"]
	if !ok {
		http.Error(w, ds.Error{Error: "Missing docker_compose"}.ToString(), http.StatusBadRequest)
		return
	}
	docker_compose, err := strconv.ParseBool(_docker_compose)
	if err != nil {
		http.Error(w, ds.Error{Error: "Invalid docker_compose"}.ToString(), http.StatusBadRequest)
		return
	}

	if docker_compose {
		_docker_compose_file, ok := result["docker_compose_file"]
		if !ok {
			http.Error(w, ds.Error{Error: "Missing docker_compose_file"}.ToString(), http.StatusBadRequest)
			return
		}
		docker_compose_file, err := base64.StdEncoding.DecodeString(_docker_compose_file)
		if err != nil {
			http.Error(w, ds.Error{Error: "Invalid base64 encoding for docker_compose_file"}.ToString(), http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, ds.Success{Success: true}.ToString())

		go _addChallengeDockerCompose(challenge_name, string(docker_compose_file))
	} else {
		internal_port, ok := result["internal_port"]
		if !ok {
			http.Error(w, ds.Error{Error: "Missing internal_port"}.ToString(), http.StatusBadRequest)
			return
		}
		image_name, ok := result["image_name"]
		if !ok {
			http.Error(w, ds.Error{Error: "Missing image_name"}.ToString(), http.StatusBadRequest)
			return
		}

		_docker_cmds, ok := result["docker_cmds"]
		var docker_cmds []byte
		if ok { //docker_cmds is optional
			docker_cmds, err = base64.StdEncoding.DecodeString(_docker_cmds)
			if err != nil {
				http.Error(w, ds.Error{Error: "Invalid base64 encoding for docker_cmds"}.ToString(), http.StatusBadRequest)
				return
			}
		}

		fmt.Fprint(w, ds.Success{Success: true}.ToString())

		go _addChallengeNonDockerCompose(challenge_name, internal_port, image_name, string(docker_cmds))
	}
}

func _addChallengeDockerCompose(challenge_name string, docker_compose_file string) { //Run Async
	log.Debug("Start /addChallenge Request (Docker Compose)")
	port_count := yaml.DockerComposePortCount(docker_compose_file)
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, true, port_count)
	ch := ds.Challenge{Challenge_Id: challenge_id, Challenge_Name: challenge_name, Docker_Compose: true, Port_Count: port_count, Docker_Compose_File: docker_compose_file}
	api_sql.UpdateChallenge(ch)

	ds.ChallengeMap[challenge_id] = ch
	log.Debug("Finish /addChallenge Request (Docker Compose)")
}

func _addChallengeNonDockerCompose(challenge_name string, internal_port string, image_name string, docker_cmds string) { //Run Async
	log.Debug("Start /addChallenge Request (Non Docker Compose)")
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, false, 1)
	ch := ds.Challenge{Challenge_Id: challenge_id, Challenge_Name: challenge_name, Docker_Compose: false, Port_Count: 1, Internal_Port: internal_port, Image_Name: image_name, Docker_Cmds: docker_cmds}
	api_sql.UpdateChallenge(ch)

	ds.ChallengeMap[challenge_id] = ch
	log.Debug("Finish /addChallenge Request (Non Docker Compose)")
}

func removeChallenge(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /removeChallenge Request")

	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, ds.Error{Error: "Authorization missing"}.ToString(), http.StatusBadRequest)
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		http.Error(w, ds.Error{Error: "Invalid authorization"}.ToString(), http.StatusBadRequest)
		return
	}

	params := r.URL.Query()

	_challid := params["challid"]
	if len(_challid) == 0 {
		http.Error(w, ds.Error{Error: "Missing challid"}.ToString(), http.StatusForbidden)
		return
	}
	challid := _challid[0]
	if !validateChallid(challid) {
		http.Error(w, ds.Error{Error: "Invalid challid"}.ToString(), http.StatusForbidden)
		return
	}

	fmt.Fprint(w, ds.Success{Success: true}.ToString())

	go _removeChallenge(challid)
}

func _removeChallenge(challid string) { //Run Async
	log.Debug("Start /removeChallenge Request")

	ds.ChallengeUnsafeToLaunch[challid] = true; //Mark challenge as unsafe to launch

	for _, instance := range ds.InstanceMap {
		if instance.Challenge_Id == challid {
			go _removeInstance(instance.Instance_Id) //Make sure that all instances running this challenge are killed
		}
	}

	ClearInstanceQueue() //Manually trigger ClearInstanceQueue() rather than waiting for the Kill Worker

	api_sql.DeleteChallenge(challid)

	delete(ds.ChallengeMap, challid)
	delete(ds.ChallengeUnsafeToLaunch, challid)

	log.Debug("Finish /removeChallenge Request")
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received /getStatus Request")

	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, ds.Error{Error: "Authorization missing"}.ToString(), http.StatusBadRequest)
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		http.Error(w, ds.Error{Error: "Invalid authorization"}.ToString(), http.StatusBadRequest)
		return
	}

	log.Debug("Start /getStatus Request")

	fmt.Fprintln(w, "Instance Count:", len(ds.InstanceMap), "/", ds.MaxInstanceCount)
	fmt.Fprintln(w, "Current Instances:")
	for _, instance := range ds.InstanceMap {
		fmt.Fprintln(w, instance.ToString())
	}
	fmt.Fprintln(w, "Current Challenges:")
	for _, challenge := range ds.ChallengeMap {
		fmt.Fprintln(w, challenge.ToString())
	}

	log.Debug("Finish /getStatus Request")
}