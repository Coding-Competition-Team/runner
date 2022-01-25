package main

import (

	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"
)

//
// Global Variables
//

var ActiveUserInstance map[int]int = make(map[int]int)                                               //UserId -> InstanceId
var InstanceMap map[int]InstanceData = make(map[int]InstanceData)                                    //InstanceId -> InstanceData
var InstanceQueue *treebidimap.Map = treebidimap.NewWith(utils.Int64Comparator, utils.IntComparator) //Unix (Nano) Timestamp of Instance Timeout -> InstanceId
var UsedPorts map[int]bool = make(map[int]bool)

var MaxInstanceCount int = 3
var NextInstanceId int = 1
var DefaultSecondsPerInstance int64 = 300
var DefaultNanosecondsPerInstance int64 = DefaultSecondsPerInstance * 1e9

var Challenges []Challenge
var ChallengeNamesMap map[string]int = make(map[string]int) //Challenge Name -> Challenges Array Index
var ChallengeIDMap map[int]int = make(map[int]int)          //Challenge ID -> Challenges Array Index

var CredentialsJsonFile string = "../../configs/Credentials/credentials.json"
var ChallDataFolder string = "../../configs/CTF Challenge Data"
var PS string = "/"

var MySQLIP string = ""
var MySQLUsername string = ""
var MySQLPassword string = ""

var PortainerURL string = ""
var PortainerUsername string = ""
var PortainerPassword string = ""
var PortainerJWT string = ""

//
// Worker Threads
// Source: https://bbengfort.github.io/2016/06/background-work-goroutines-timer/
//

// Worker will do its Action once every interval, making up for lost time that
// happened during the Action by only waiting the time left in the interval.
type Worker struct {
	Stopped         bool          // A flag determining the state of the worker
	ShutdownChannel chan string   // A channel to communicate to the routine
	Interval        time.Duration // The interval with which to run the Action
	period          time.Duration // The actual period of the wait
}

// NewWorker creates a new worker and instantiates all the data structures required.
func NewWorker(interval time.Duration) *Worker {
	return &Worker{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
	}
}

// Run starts the worker and listens for a shutdown call.
func (w *Worker) Run() {
	info("Worker Started")
	// Loop that runs forever
	for {
		select {
		case <-w.ShutdownChannel:
			w.ShutdownChannel <- "Down"
			return
		case <-time.After(w.period):
			// This breaks out of the select, not the for loop.
			break
		}

		started := time.Now()
		w.Action()
		finished := time.Now()

		duration := finished.Sub(started)
		w.period = w.Interval - duration
	}
}

// Shutdown is a graceful shutdown mechanism
func (w *Worker) Shutdown() {
	w.Stopped = true
	w.ShutdownChannel <- "Down"
	<-w.ShutdownChannel
	close(w.ShutdownChannel)
}

// Action defines what the worker does; override this.
// For now we'll just wait two seconds and print to simulate work.
func (w *Worker) Action() {
	info("Current Instances:")
	it := InstanceQueue.Iterator()
	current_timestamp := time.Now().UnixNano()

	for it.Next() { //Sorted by time
		timestamp, InstanceId := it.Key().(int64), it.Value().(int)
		if timestamp <= current_timestamp {
			db, err := sql.Open("mysql", MySQLUsername+":"+MySQLPassword+"@tcp("+MySQLIP+")/runner_db")
			if err != nil {
				panic(err)
			}
			defer db.Close()

			stmt, err := db.Prepare("DELETE FROM instances WHERE instance_id = ?")
			if err != nil {
				panic(err)
			}
			defer stmt.Close()

			_, err = stmt.Exec(InstanceId)
			if err != nil {
				panic(err)
			}

			PortainerId := InstanceMap[InstanceId].PortainerId
			if Challenges[ChallengeIDMap[InstanceMap[InstanceId].ChallengeId]].DockerCompose {
				deleteStack(PortainerId)
			} else {
				deleteContainer(PortainerId)
			}

			UserId := InstanceMap[InstanceId].UserId
			InstanceQueue.Remove(timestamp)
			for _, v := range InstanceMap[InstanceId].Ports {
				delete(UsedPorts, v)
			}
			delete(InstanceMap, InstanceId)
			delete(ActiveUserInstance, UserId)

			break //Only handle 1 instance a time to prevent tree iterator nonsense
		}
		info(timestamp, ":", InstanceId)
	}
}

//
// IO Stuff
//

func loadCredentials() {
	json_data, err := os.ReadFile(CredentialsJsonFile)
	if err != nil {
		panic(err)
	}

	var result map[string]map[string]string
	json.Unmarshal([]byte(json_data), &result)

	MySQLIP = result["mysql"]["ip"]
	MySQLUsername = result["mysql"]["username"]
	MySQLPassword = result["mysql"]["password"]

	PortainerURL = result["portainer"]["url"]
	PortainerUsername = result["portainer"]["username"]
	PortainerPassword = result["portainer"]["password"]
	PortainerJWT = getPortainerJWT()
}

func getFileNames(dir string) []string {
	file, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	lst, err := file.Readdirnames(0) //Read folders and files
	if err != nil {
		panic(err)
	}

	return lst
}

func doesFileExist(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func debug(s ...interface{}) {
	log.Println("[DEBUG]", s)
}

func info(s ...interface{}) {
	log.Println("[INFO]", s)
}

func warn(s ...interface{}) {
	log.Println("[WARN]", s)
}

//
// YAML API
//

func parseInternalPort(str string) string { //Returns the internal port
	return strings.Split(str, ":")[1]
}

func dockerComposeCopy(docker_compose string, ports []int) string {
	yml := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(docker_compose), &yml)
	if err != nil {
		panic(err)
	}

	ports_idx := 0
	for k1, v1 := range yml["services"].(map[interface{}]interface{}) {
		raw_port_mappings := v1.(map[interface{}]interface{})["ports"]
		if raw_port_mappings != nil { //There are ports
			raw_port_mappings := raw_port_mappings.([]interface{})
			new_port_mappings := make([]string, len(raw_port_mappings))
			for k2, v2 := range raw_port_mappings {
				new_port_mappings[k2] = strconv.Itoa(ports[ports_idx]) + ":" + parseInternalPort(v2.(string))
				ports_idx += 1
			}
			yml["services"].(map[interface{}]interface{})[k1].(map[interface{}]interface{})["ports"] = new_port_mappings //Override old port mappings
		}

		container_name := v1.(map[interface{}]interface{})["container_name"]
		if container_name != nil { //There is a container name
			delete(yml["services"].(map[interface{}]interface{})[k1].(map[interface{}]interface{}), "container_name") //Clear container name, let portainer substitute from stack name instead to prevent duplicate container names
		}
	}

	new_yml, err := yaml.Marshal(&yml)
	if err != nil {
		panic(err)
	}

	return string(new_yml)
}

func dockerComposePortCount(docker_compose string) int {
	yml := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(docker_compose), &yml)
	if err != nil {
		panic(err)
	}

	port_count := 0
	for _, v1 := range yml["services"].(map[interface{}]interface{}) {
		raw_port_mappings := v1.(map[interface{}]interface{})["ports"]
		if raw_port_mappings != nil { //There are ports
			port_count += len(raw_port_mappings.([]interface{}))
		}
	}

	return port_count
}

//
// MySQL API
//

func deserialize(str string, delim string) []string {
	return strings.Split(str, delim)
}

func deserializeNL(str string) []string {
	return strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")
}

func serialize(dat []string, delim string) string {
	str := ""
	for i, v := range dat {
		str += v
		if (i + 1) < len(dat) {
			str += delim
		}
	}
	return str
}

func serializeI(dat []int, delim string) string {
	str := ""
	for i, v := range dat {
		str += strconv.Itoa(v)
		if (i + 1) < len(dat) {
			str += delim
		}
	}
	return str
}

func loadChallenge(ctf_name string, challenge_name string) {
	path := ChallDataFolder + PS + ctf_name + PS + challenge_name

	docker_compose, err := doesFileExist(path + PS + "docker-compose.yml")
	if err != nil {
		panic(err)
	}

	if docker_compose {
		_docker_compose_file, err := os.ReadFile(path + PS + "docker-compose.yml")
		if err != nil {
			panic(err)
		}

		docker_compose_file := string(_docker_compose_file)

		Challenges = append(Challenges, Challenge{ChallengeId: -1, ChallengeName: challenge_name, DockerCompose: docker_compose, PortCount: dockerComposePortCount(docker_compose_file), DockerComposeFile: docker_compose_file})
	} else {
		internal_port, err := os.ReadFile(path + PS + "PORT.txt")
		if err != nil {
			panic(err)
		}

		image_name, err := os.ReadFile(path + PS + "IMAGE.txt")
		if err != nil {
			panic(err)
		}

		docker_cmds, err := os.ReadFile(path + PS + "CMDS.txt")
		if err != nil {
			panic(err)
		}

		Challenges = append(Challenges, Challenge{ChallengeId: -1, ChallengeName: challenge_name, DockerCompose: docker_compose, PortCount: 1, InternalPort: string(internal_port), ImageName: string(image_name), DockerCmds: deserializeNL(string(docker_cmds))})
	}

	ChallengeNamesMap[challenge_name] = len(Challenges) - 1 //Current index of most recently appended challenge
}

func loadCTF(ctf_name string) {
	path := ChallDataFolder + PS + ctf_name

	lst := getFileNames(path)
	for _, name := range lst {
		loadChallenge(ctf_name, name)
	}
}

func loadChallengeFiles() {
	lst := getFileNames(ChallDataFolder)
	for _, name := range lst {
		if name != ".gitignore" {
			loadCTF(name)
		}
	}
}

func syncChallenges() {
	db, err := sql.Open("mysql", MySQLUsername+":"+MySQLPassword+"@tcp("+MySQLIP+")/runner_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT challenge_id, challenge_name FROM challenges") //Get currently registered challenges in the DB
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var challenge_ids []int
	var challenge_names []string //Assumes no duplicate challenge names

	for rows.Next() {
		var challenge_id int
		var challenge_name string
		if err := rows.Scan(&challenge_id, &challenge_name); err != nil {
			panic(err)
		}

		challenge_ids = append(challenge_ids, challenge_id)
		challenge_names = append(challenge_names, challenge_name)
	}

	var new_challenge_names map[string]int = make(map[string]int) //TODO: Better way to do a deepcopy?
	jsonStr, err := json.Marshal(ChallengeNamesMap)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonStr, &new_challenge_names)
	if err != nil {
		panic(err)
	}

	var edit_challenge_ids []int
	var edit_challenge_names []string

	for i, name := range challenge_names {
		_, ok := new_challenge_names[name]
		if ok { //Challenge name already exists in DB
			id := challenge_ids[i]
			idx := ChallengeNamesMap[name]

			delete(new_challenge_names, name)
			edit_challenge_names = append(edit_challenge_names, name)
			edit_challenge_ids = append(edit_challenge_ids, id)

			Challenges[idx].ChallengeId = id //Replace with ChallengeId in DB
			ChallengeIDMap[id] = idx
		} else {
			warn("Challenge", name, "exists in DB but is not in use!")
		}
	}

	stmt1, err := db.Prepare("INSERT INTO challenges (challenge_name, docker_compose, port_count, internal_port, image_name, docker_cmds, docker_compose_file) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt1.Close()

	stmt1b, err := db.Prepare("SELECT challenge_id FROM challenges WHERE challenge_name = ?")
	if err != nil {
		panic(err)
	}
	defer stmt1b.Close()

	for k, v := range new_challenge_names { //Insert new challenges
		debug("New Challenge:", k, ",", v)

		ch := Challenges[v]
		_, err = stmt1.Exec(k, ch.DockerCompose, ch.PortCount, ch.InternalPort, ch.ImageName, serialize(ch.DockerCmds, "\n"), ch.DockerComposeFile)
		if err != nil {
			panic(err)
		}

		var challenge_id int
		if err := stmt1b.QueryRow(k).Scan(&challenge_id); err != nil {
			panic(err)
		}
		Challenges[v].ChallengeId = challenge_id //Get DB assigned challenge id
		ChallengeIDMap[challenge_id] = v
	}

	stmt2, err := db.Prepare("UPDATE challenges SET docker_compose = ?, port_count = ?, internal_port = ?, image_name = ?, docker_cmds = ?, docker_compose_file = ? WHERE challenge_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt2.Close()

	for i, name := range edit_challenge_names { //Edit pre-existing challenges
		debug("Edit Challenge:", i, ",", name)

		ch := Challenges[ChallengeNamesMap[name]]
		_, err = stmt2.Exec(ch.DockerCompose, ch.PortCount, ch.InternalPort, ch.ImageName, serialize(ch.DockerCmds, "\n"), ch.DockerComposeFile, edit_challenge_ids[i])
		if err != nil {
			panic(err)
		}
	}
}

func syncInstances() {
	db, err := sql.Open("mysql", MySQLUsername+":"+MySQLPassword+"@tcp("+MySQLIP+")/runner_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM instances") //Fully trust DB
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var instance_id int
		var usr_id int
		var challenge_id int
		var portainer_id string
		var instance_timeout int64
		var ports_used string
		if err := rows.Scan(&instance_id, &usr_id, &challenge_id, &portainer_id, &instance_timeout, &ports_used); err != nil {
			panic(err)
		}

		if (instance_id + 1) > NextInstanceId {
			NextInstanceId = instance_id + 1
		}
		ActiveUserInstance[usr_id] = instance_id
		InstanceQueue.Put(instance_timeout, instance_id)

		var ports []int
		deserialized_ports := deserialize(ports_used, ",")
		for _, v := range deserialized_ports {
			port, err := strconv.Atoi(v)
			if err != nil {
				panic(err)
			}
			ports = append(ports, port)
			UsedPorts[port] = true
		}
		InstanceMap[instance_id] = InstanceData{UserId: usr_id, ChallengeId: challenge_id, InstanceTimeLeft: instance_timeout, PortainerId: portainer_id, Ports: ports}
	}
}

func syncWithDB() {
	info("Starting DB Sync...")
	loadChallengeFiles()
	syncChallenges()
	debug("Challenge Data:", Challenges)
	debug("Challenge ID Map:", ChallengeIDMap)
	syncInstances()
	debug("Instance Map:", InstanceMap)
	info("DB Sync Complete!")
}

//
// Portainer API
//

func getPortainerJWT() string {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //TODO: Remove

	requestBody, err := json.Marshal(map[string]string{
		"Username": PortainerUsername,
		"Password": PortainerPassword,
	})
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(PortainerURL+"/api/auth", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil {
		panic(err)
	}

	return raw["jwt"]
}

func launchContainer(container_name string, image_name string, cmds []string, internal_port string, _external_port int, discriminant string) string {
	external_port := strconv.Itoa(_external_port)

	cmd := ""
	for i, s := range cmds {
		cmd += "\"" + s + "\""
		if (i + 1) < len(cmds) {
			cmd += ","
		}
	}

	tmp := "{\"Cmd\":[" + cmd + "],\"Image\":\"" + image_name + "\",\"ExposedPorts\":{\"" + internal_port + "/tcp\":{}},\"HostConfig\":{\"PortBindings\":{\"" + internal_port + "/tcp\":[{\"HostPort\":\"" + external_port + "\"}]}}}"
	debug("launchContainer Body:", tmp)

	requestBody := []byte(tmp)

	client := http.Client{}
	req, err := http.NewRequest("POST", PortainerURL+"/api/endpoints/2/docker/containers/create?name="+container_name+"_"+discriminant, bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
		"Content-Type":  []string{"application/json"},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	debug("launchContainer Response:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		panic(err)
	}
	id := raw["Id"].(string)

	startContainer(id)

	return id
}

func startContainer(id string) {
	requestBody := []byte("{}")

	client := http.Client{}
	req, err := http.NewRequest("POST", PortainerURL+"/api/endpoints/2/docker/containers/"+id+"/start", bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	info("startContainer", string(body))
}

func deleteContainer(id string) {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", PortainerURL+"/api/endpoints/2/docker/containers/"+id+"?force=true", nil)
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	info("deleteContainer", string(body))
}

func launchStack(stack_name string, docker_compose string, discriminant string) string {
	json_docker_compose, err := json.Marshal(docker_compose) //Make sure docker_compose is JSON Encoded
	if err != nil {
		panic(err)
	}

	tmp := "{\"name\":\"" + stack_name + "_" + discriminant + "\",\"stackFileContent\":" + string(json_docker_compose) + "}"
	debug("launchStack Body:", tmp)

	requestBody := []byte(tmp)

	client := http.Client{}
	req, err := http.NewRequest("POST", PortainerURL+"/api/stacks?type=2&method=string&endpointId=2", bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
		"Content-Type":  []string{"application/json"},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	debug("launchStack Response:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		panic(err)
	}
	id := int(raw["Id"].(float64)) //Cannot directly cast to string

	return strconv.Itoa(id)
}

func deleteStack(id string) {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", PortainerURL+"/api/stacks/"+id+"?endpointId=2", nil)
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	info("deleteStack", string(body))
}

func getRandomPort() int { //Returns an (unused) random port from [1024, 65536)
	for {
		port := rand.Intn(65536-1024) + 1024
		if !UsedPorts[port] {
			UsedPorts[port] = true
			return port
		}
	}
}

//
// Web Stuff
//

func handleRequests() {
	http.HandleFunc("/addInstance", addInstance)
	http.HandleFunc("/removeInstance", removeInstance)
	http.HandleFunc("/getTimeLeft", getTimeLeft)
	http.HandleFunc("/extendTimeLeft", extendTimeLeft)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

//fmt.Fprintf() - print to web

func validateUserid(userid int) bool {
	return true
}

func validateChallid(challid int) bool {
	_, ok := ChallengeIDMap[challid]
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

	if ActiveUserInstance[userid] > 0 {
		http.Error(w, "User is already running an instance", http.StatusForbidden)
		return
	}

	if len(InstanceMap) >= MaxInstanceCount { //Use >= instead of == just in case
		http.Error(w, "The max number of instances for the platform has already been reached, try again later", http.StatusForbidden)
		return
	}

	ch := Challenges[ChallengeIDMap[challid]]

	Ports := make([]int, ch.PortCount) //Cannot directly use [ch.PortCount]int
	for i := 0; i < ch.PortCount; i++ {
		Ports[i] = getRandomPort()
		fmt.Fprintln(w, strconv.Itoa(Ports[i]))
	}

	go _addInstance(userid, challid, Ports)
}

func _addInstance(userid int, challid int, Ports []int) { //Run Async
	InstanceId := NextInstanceId
	NextInstanceId++
	ActiveUserInstance[userid] = InstanceId
	InstanceTimeLeft := time.Now().UnixNano() + DefaultNanosecondsPerInstance
	InstanceQueue.Put(InstanceTimeLeft, InstanceId) //Use higher precision time to (hopefully) prevent duplicates
	discriminant := strconv.FormatInt(time.Now().Unix(), 10)
	InstanceMap[InstanceId] = InstanceData{UserId: userid, ChallengeId: challid, InstanceTimeLeft: InstanceTimeLeft, Ports: Ports} //Everything except PortainerId first, to prevent issues when querying getTimeLeft, etc. while the instance is launching

	db, err := sql.Open("mysql", MySQLUsername+":"+MySQLPassword+"@tcp("+MySQLIP+")/runner_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt1, err := db.Prepare("INSERT INTO instances (instance_id, usr_id, challenge_id, instance_timeout, ports_used) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt1.Close()

	serialized_ports := serializeI(Ports, ",")
	_, err = stmt1.Exec(InstanceId, userid, challid, InstanceTimeLeft, serialized_ports)
	if err != nil {
		panic(err)
	}

	var PortainerId string

	ch := Challenges[ChallengeIDMap[challid]]
	if ch.DockerCompose {
		new_docker_compose := dockerComposeCopy(ch.DockerComposeFile, Ports)
		PortainerId = launchStack(ch.ChallengeName, new_docker_compose, discriminant)
	} else {
		PortainerId = launchContainer(ch.ChallengeName, ch.ImageName, ch.DockerCmds, ch.InternalPort, Ports[0], discriminant)
	}

	debug("Portainer ID:", PortainerId)

	entry := InstanceMap[InstanceId]
	entry.PortainerId = PortainerId
	InstanceMap[InstanceId] = entry //Update PortainerId once it's available

	stmt2, err := db.Prepare("UPDATE instances SET portainer_id = ? WHERE instance_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt2.Close()

	_, err = stmt2.Exec(PortainerId, InstanceId)
	if err != nil {
		panic(err)
	}
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

	if ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	InstanceId := ActiveUserInstance[userid]
	if InstanceMap[InstanceId].PortainerId == "" {
		http.Error(w, "The instance is still starting", http.StatusForbidden)
		return
	}

	go _removeInstance(InstanceId)
}

func _removeInstance(InstanceId int) { //Run Async
	var NewInstanceTimeLeft int64 = 0 //Force typecast

	entry := InstanceMap[InstanceId]
	entry.InstanceTimeLeft = NewInstanceTimeLeft //Make sure that the instance will be killed in the next kill cycle
	InstanceMap[InstanceId] = entry

	a, b := InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	InstanceQueue.Remove(a)
	InstanceQueue.Put(NewInstanceTimeLeft, InstanceId) //Replace

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

	if ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	InstanceId := ActiveUserInstance[userid]

	fmt.Fprint(w, strconv.FormatInt((InstanceMap[InstanceId].InstanceTimeLeft-time.Now().UnixNano())/1e9, 10))
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

	if ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}

	go _extendTimeLeft(userid)
}

func _extendTimeLeft(userid int) { //Run Async
	InstanceId := ActiveUserInstance[userid]
	entry := InstanceMap[InstanceId]

	NewInstanceTimeLeft := entry.InstanceTimeLeft + DefaultNanosecondsPerInstance
	entry.InstanceTimeLeft = NewInstanceTimeLeft
	InstanceMap[InstanceId] = entry

	a, b := InstanceQueue.GetKey(InstanceId)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	InstanceQueue.Remove(a)
	InstanceQueue.Put(NewInstanceTimeLeft, InstanceId) //Replace

	db, err := sql.Open("mysql", MySQLUsername+":"+MySQLPassword+"@tcp("+MySQLIP+")/runner_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE instances SET instance_timeout = ? WHERE instance_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(NewInstanceTimeLeft, InstanceId)
	if err != nil {
		panic(err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	loadCredentials()

	UsedPorts[8000] = true //Portainer
	UsedPorts[9443] = true //Portainer
	UsedPorts[3306] = true //Runner DB
	UsedPorts[22] = true   //SSH

	syncWithDB()

	killWorker := NewWorker(10 * time.Second)
	go killWorker.Run()

	handleRequests()
}
