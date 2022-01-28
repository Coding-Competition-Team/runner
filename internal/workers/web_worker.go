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
	http.HandleFunc("/getTimeLeft", getTimeLeft)
	http.HandleFunc("/extendTimeLeft", extendTimeLeft)
	http.HandleFunc("/addChallenge", addChallenge)
	http_log.Fatal(http.ListenAndServe(":" + strconv.Itoa(ds.RunnerPort), nil))
}

//fmt.Fprintf() - print to web

func validateUserid(userid int) bool {
	return true
}

func validateChallid(challid int) bool {
	_, ok := ds.ChallengeMap[challid]
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

	ch := ds.ChallengeMap[challid]

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

	ch := ds.ChallengeMap[challid]
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

func addChallenge(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, "Authorization missing", http.StatusBadRequest)
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		http.Error(w, "Invalid authorization", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}

	var result map[string]string
	json.Unmarshal(body, &result)

	challenge_name, ok := result["challenge_name"]
	if !ok {
		http.Error(w, "Missing challenge_name", http.StatusBadRequest)
		return
	}
	_docker_compose, ok := result["docker_compose"]
	if !ok {
		http.Error(w, "Missing docker_compose", http.StatusBadRequest)
		return
	}
	docker_compose, err := strconv.ParseBool(_docker_compose)
	if err != nil {
		http.Error(w, "Invalid docker_compose", http.StatusBadRequest)
		return
	}

	if docker_compose {
		_docker_compose_file, ok := result["docker_compose_file"]
		if !ok {
			http.Error(w, "Missing docker_compose_file", http.StatusBadRequest)
			return
		}
		docker_compose_file, err := base64.StdEncoding.DecodeString(_docker_compose_file)
		if err != nil {
			http.Error(w, "Invalid base64 encoding for docker_compose_file", http.StatusBadRequest)
			return
		}

		go _addChallengeDockerCompose(challenge_name, string(docker_compose_file))
	} else {
		internal_port, ok := result["internal_port"]
		if !ok {
			http.Error(w, "Missing internal_port", http.StatusBadRequest)
			return
		}
		image_name, ok := result["image_name"]
		if !ok {
			http.Error(w, "Missing image_name", http.StatusBadRequest)
			return
		}

		_docker_cmds, ok := result["docker_cmds"]
		var docker_cmds []byte
		if ok { //docker_cmds is optional
			docker_cmds, err = base64.StdEncoding.DecodeString(_docker_cmds)
			if err != nil {
				http.Error(w, "Invalid base64 encoding for docker_cmds", http.StatusBadRequest)
				return
			}
		}

		go _addChallengeNonDockerCompose(challenge_name, internal_port, image_name, string(docker_cmds))
	}
}

func _addChallengeDockerCompose(challenge_name string, docker_compose_file string) { //Run Async
	port_count := yaml.DockerComposePortCount(docker_compose_file)
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, true, port_count)
	ch := ds.Challenge{ChallengeId: challenge_id, ChallengeName: challenge_name, DockerCompose: true, PortCount: port_count, DockerComposeFile: docker_compose_file}
	api_sql.UpdateChallenge(ch)

	ds.ChallengeMap[challenge_id] = ch
	ds.ChallengeNamesMap[challenge_name] = challenge_id
}

func _addChallengeNonDockerCompose(challenge_name string, internal_port string, image_name string, docker_cmds string) { //Run Async
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, false, 1)
	ch := ds.Challenge{ChallengeId: challenge_id, ChallengeName: challenge_name, DockerCompose: false, PortCount: 1, InternalPort: internal_port, ImageName: image_name, DockerCmds: api_sql.DeserializeNL(docker_cmds)}
	api_sql.UpdateChallenge(ch)

	ds.ChallengeMap[challenge_id] = ch
	ds.ChallengeNamesMap[challenge_name] = challenge_id
}