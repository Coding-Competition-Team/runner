package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"crypto/tls"
)

import (
	"github.com/emirpasic/gods/maps/treebidimap"
	"github.com/emirpasic/gods/utils"
)

type InstanceData struct {
	UserId int
	InstanceTimeLeft int64 //Unix Timestamp of Instance Timeout
	DockerId string
}

//
// Global Variables
//

var ActiveUserInstance map[int]int //UserId -> InstanceId
var InstanceMap map[int]InstanceData //InstanceId -> InstanceData
var InstanceQueue *treebidimap.Map //Unix (Micro) Timestamp of Instance Timeout -> InstanceId

var NextInstanceId int
var DefaultSecondsPerInstance int64
var DefaultMicrosecondsPerInstance int64

var PortainerURL string
var PortainerUsername string
var PortainerPassword string
var PortainerJWT string

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
	log.Println("Worker Started")
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
	log.Println("Current Instances:")
	it := InstanceQueue.Iterator()
	for it.Next() { //Sorted by time
		timestamp, InstanceId := it.Key().(int64), it.Value().(int)
		if timestamp <= time.Now().UnixMicro() {
			DockerId := InstanceMap[InstanceId].DockerId
			deleteContainer(DockerId)
			UserId := InstanceMap[InstanceId].UserId
			InstanceQueue.Remove(timestamp)
			delete(InstanceMap, InstanceId)
			delete(ActiveUserInstance, UserId)
			break //Only handle 1 instance a time to prevent tree iterator nonsense
		}
		log.Println(timestamp)
		log.Println(InstanceId)
	}
}

//
// Portainer API
//

func getPortainerJWT() {
	requestBody, err := json.Marshal(map[string]string {
		"Username": PortainerUsername,
		"Password": PortainerPassword,
	})
	if err != nil { panic(err) }
	
	resp, err := http.Post(PortainerURL + "/api/auth", "application/json", bytes.NewBuffer(requestBody))
	if err != nil { panic(err) }
	
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	
	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil { panic(err) }
	
	PortainerJWT = raw["jwt"]
}

func getEndpoints() {
	client := http.Client{}
	req, err := http.NewRequest("GET", PortainerURL + "/api/endpoints", nil)
	if err != nil { panic(err) }

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil { panic(err) }
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	
	log.Println(string(body))
}

func launchContainer(container_name string, image_name string, cmds []string, _internal_port int, _external_port int) string {
	internal_port := strconv.Itoa(_internal_port)
	external_port := strconv.Itoa(_external_port)
	
	cmd := ""
	for i, s := range cmds {
		cmd += "\"" + s + "\""
		if (i+1) < len(cmds) {
			cmd += ","
		}
	}
	
	tmp := "{\"Cmd\":[" + cmd + "],\"Image\":\"" + image_name + "\",\"ExposedPorts\":{\"" + internal_port + "/tcp\":{}},\"HostConfig\":{\"PortBindings\":{\"" + internal_port + "/tcp\":[{\"HostPort\":\"" + external_port + "\"}]}}}"
	log.Println(tmp)
	
	requestBody := []byte(tmp)

	client := http.Client{}
	req, err := http.NewRequest("POST", PortainerURL + "/api/endpoints/2/docker/containers/create?name=" + container_name + "_" + external_port, bytes.NewBuffer(requestBody))
	if err != nil { panic(err) }

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
		"Content-Type": []string{"application/json"},
	}

	resp, err := client.Do(req)
	if err != nil { panic(err) }
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	log.Println(string(body))
	
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil { panic(err) }
	id := raw["Id"].(string)
	
	startContainer(id)
	
	return id
}

func containersList(){
	client := http.Client{}
	req, err := http.NewRequest("GET", PortainerURL + "/api/endpoints/2/docker/containers/json", nil)
	if err != nil { panic(err) }

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil { panic(err) }
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	
	log.Println(string(body))
}

func startContainer(id string) {
	requestBody := []byte("{}")

	client := http.Client{}
	req, err := http.NewRequest("POST", PortainerURL + "/api/endpoints/2/docker/containers/" + id + "/start", bytes.NewBuffer(requestBody))
	if err != nil { panic(err) }

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil { panic(err) }
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	
	log.Println(string(body))
}

func deleteContainer(id string) {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", PortainerURL + "/api/endpoints/2/docker/containers/" + id + "?force=true", nil)
	if err != nil { panic(err) }

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + PortainerJWT},
	}

	resp, err := client.Do(req)
	if err != nil { panic(err) }
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }
	
	log.Println(string(body))
}

var usedPorts map[int]bool

func getRandomPort() int { //Returns an (unused) random port from [1024, 65536)
	for {
		port := rand.Intn(65536 - 1024) + 1024
		if !usedPorts[port] {
			usedPorts[port] = true
			return port
		}
	}
}

//
// Web Stuff
//

func handleRequests() {
	http.HandleFunc("/addInstance", addInstance)
	http.HandleFunc("/getTimeLeft", getTimeLeft)
	http.HandleFunc("/extendTimeLeft", extendTimeLeft)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

//fmt.Println() - console
//fmt.Fprintf() - print to web

func validateUserid(userid int) bool {
	return true
}

func validateChallid(challid int) bool {
	return true
}

func addInstance(w http.ResponseWriter, r *http.Request){
	params := r.URL.Query()
	
	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil { panic(err) }
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
	if err != nil { panic(err) }
	if !validateChallid(challid) {
		http.Error(w, "Invalid challid", http.StatusForbidden)
		return
	}
	
	if ActiveUserInstance[userid] > 0 {
		http.Error(w, "User is already running an instance", http.StatusForbidden)
		return
	}
	
	InstanceId := NextInstanceId	
	NextInstanceId++
	
	ActiveUserInstance[userid] = InstanceId
	InstanceQueue.Put(time.Now().UnixMicro() + DefaultMicrosecondsPerInstance, InstanceId) //Use higher precision time to (hopefully) prevent duplicates
	external_port := getRandomPort()
	DockerId := launchContainer("nginx", "nginx:latest", []string{"nginx", "-g", "daemon off;"}, 80, external_port)
	InstanceMap[InstanceId] = InstanceData{UserId: userid, InstanceTimeLeft: time.Now().Unix() + DefaultSecondsPerInstance, DockerId: DockerId}
	
	fmt.Fprintf(w, strconv.Itoa(external_port))
}

func getTimeLeft(w http.ResponseWriter, r *http.Request){
	params := r.URL.Query()
	
	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil { panic(err) }
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}
	
	if ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}
	
	InstanceId := ActiveUserInstance[userid]
	
	fmt.Fprintf(w, strconv.FormatInt(InstanceMap[InstanceId].InstanceTimeLeft - time.Now().Unix(), 10))
}

func extendTimeLeft(w http.ResponseWriter, r *http.Request){
	params := r.URL.Query()
	
	_userid := params["userid"]
	if len(_userid) == 0 {
		http.Error(w, "Missing userid", http.StatusForbidden)
		return
	}
	userid, err := strconv.Atoi(_userid[0])
	if err != nil { panic(err) }
	if !validateUserid(userid) {
		http.Error(w, "Invalid userid", http.StatusForbidden)
		return
	}
	
	if ActiveUserInstance[userid] <= 0 {
		http.Error(w, "User does not have an instance", http.StatusForbidden)
		return
	}
	
	InstanceId := ActiveUserInstance[userid]
	if entry, ok := InstanceMap[InstanceId]; !ok { panic(err) } else {
		entry.InstanceTimeLeft += DefaultSecondsPerInstance
		InstanceMap[InstanceId] = entry
	}
	
	a, b := InstanceQueue.GetKey(InstanceId)
	if b == false {
		panic("InstanceId missing")
	}
	InstanceQueue.Remove(a)
	InstanceQueue.Put(a.(int64) + DefaultMicrosecondsPerInstance, InstanceId) //Replace
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ActiveUserInstance = make(map[int]int)
	InstanceMap = make(map[int]InstanceData)
	NextInstanceId = 1
	DefaultSecondsPerInstance = 60
	DefaultMicrosecondsPerInstance = DefaultSecondsPerInstance*1000000
	InstanceQueue = treebidimap.NewWith(utils.Int64Comparator, utils.IntComparator)
	
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //todo remove
	
	PortainerURL = ""
	PortainerUsername = ""
	PortainerPassword = ""
	usedPorts = make(map[int]bool)
	getPortainerJWT()
	getEndpoints()
	containersList()
	
	killWorker := NewWorker(10 * time.Second)
	go killWorker.Run()
	
	handleRequests()
}
