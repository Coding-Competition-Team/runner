package workers

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"runner/internal/api_portainer"
	"runner/internal/api_sql"
	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
	"runner/internal/yaml"
)

func HandleRequests() {
	r := gin.Default()
	
	r.GET("/addInstance", addInstance)
	r.GET("/removeInstance", removeInstance)
	r.GET("/removeInstance/admin", removeInstanceAdmin)
	r.GET("/getUserStatus", getUserStatus)
	r.GET("/extendTimeLeft", extendTimeLeft)
	r.GET("/addChallenge", addChallenge)
	r.GET("/removeChallenge", removeChallenge)
	r.GET("/getStatus", getStatus)

	r.Run(":" + strconv.Itoa(ds.RunnerPort))
}

//fmt.Fprintf() - print to web

func validateUserid(userid string) bool {
	return true
}

func validateChallid(challid string) bool {
	valid := api_sql.ValidChallenge(challid)
	if valid { //If challid exists in ChallengeMap, check if it is not unsafe to launch
		return !ds.ChallengeUnsafeToLaunch[challid]
	}
	return false //challid does not exist in ChallengeMap
}

func activeUserInstance(userid string) bool {
	instance := api_sql.GetActiveUserInstance(userid) //TODO: Optimize
	return instance.Usr_Id != ""
}

func addInstance(c *gin.Context) {
	log.Debug("Received /addInstance Request")

	userid, ok := c.GetQuery("userid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing userid"})
		return
	}
	if !validateUserid(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid userid"})
		return
	}

	challid, ok := c.GetQuery("challid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing challid"})
		return
	}
	if !validateChallid(challid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid challid"})
		return
	}

	if activeUserInstance(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "User is already running an instance"})
		return
	}

	if api_sql.GetInstanceCount() >= ds.MaxInstanceCount { //Use >= instead of == just in case
		c.JSON(http.StatusBadRequest, gin.H{"Error": "The max number of instances for the platform has already been reached, try again later"})
		return
	}

	ch := api_sql.GetChallenge(challid)

	var ports ds.PortsJson
	ports.Ports_Used = make([]int, ch.Port_Count) //Cannot directly use [ch.PortCount]int
	for i := 0; i < ch.Port_Count; i++ {
		ports.Ports_Used[i] = ds.GetRandomPort()
	}
	portainer_url := creds.GetBestPortainer()
	ports.Host = creds.ExtractHost(portainer_url)
	ports.Port_Types = api_sql.Deserialize(ch.Port_Types, ",")

	c.JSON(http.StatusOK, ports)

	go _addInstance(userid, challid, portainer_url, ports.Ports_Used)
}

func _addInstance(userid string, challid string, portainer_url string, Ports []int) { //Run Async
	log.Debug("Start /addInstance Request")
	InstanceId := ds.NextInstanceId
	ds.NextInstanceId++
	InstanceTimeout := time.Now().UnixNano() + ds.DefaultNanosecondsPerInstance
	ds.InstanceQueue.Put(InstanceTimeout, InstanceId) //Use higher precision time to (hopefully) prevent duplicates
	discriminant := strconv.FormatInt(time.Now().UnixNano(), 10)
	creds.IncrementPortainerQueue(portainer_url)

	instance := ds.Instance{Instance_Id: InstanceId, Usr_Id: userid, Challenge_Id: challid, Portainer_Url: portainer_url, Instance_Timeout: InstanceTimeout, Ports_Used: api_sql.SerializeI(Ports, ",")} //Everything except PortainerId first, to prevent issues when querying getTimeLeft, etc. while the instance is launching
	api_sql.AddInstance(instance)

	var PortainerId string

	ch := api_sql.GetChallenge(challid)
	if ch.Docker_Compose {
		new_docker_compose := yaml.DockerComposeCopy(ch.Docker_Compose_File, Ports)
		PortainerId = api_portainer.LaunchStack(portainer_url, ch.Challenge_Name, new_docker_compose, discriminant)
	} else {
		PortainerId = api_portainer.LaunchContainer(portainer_url, ch.Challenge_Name, ch.Image_Name, api_sql.DeserializeNL(ch.Docker_Cmds), ch.Internal_Port, Ports[0], discriminant)
	}

	log.Debug("Portainer ID:", PortainerId)

	instance.Portainer_Id = PortainerId
	api_sql.SetInstancePortainerId(InstanceId, PortainerId) //Update PortainerId once it's available

	log.Debug("Finish /addInstance Request")
}

func removeInstance(c *gin.Context) {
	log.Debug("Received /removeInstance Request")

	userid, ok := c.GetQuery("userid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing userid"})
		return
	}
	if !validateUserid(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid userid"})
		return
	}

	if !activeUserInstance(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "User does not have an instance"})
		return
	}

	instance := api_sql.GetActiveUserInstance(userid)
	if instance.Portainer_Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "The instance is still starting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"Success": true})

	go _removeInstance(userid)
}

func _removeInstance(userid string) { //Run Async
	log.Debug("Start /removeInstance Request")

	instance := api_sql.GetActiveUserInstance(userid)
	instance.Instance_Timeout = int64(0) //Make sure that the instance will be killed in the next kill cycle
	api_sql.UpdateInstance(instance)

	a, b := ds.InstanceQueue.GetKey(instance.Instance_Id)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(int64(0), instance.Instance_Id) //Replace

	log.Debug("Finish /removeInstance Request")
}

func removeInstanceAdmin(c *gin.Context) {
	log.Debug("Received /removeInstance/admin Request")

	userid, ok := c.GetQuery("userid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing userid"})
		return
	}
	if !validateUserid(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid userid"})
		return
	}

	if !activeUserInstance(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "User does not have an instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"Success": true})

	go _removeInstanceAdmin(userid)
}

func _removeInstanceAdmin(userid string) { //Run Async
	log.Debug("Start /removeInstance/admin Request")

	instance := api_sql.GetActiveUserInstance(userid)

	//Essentially makes the Runner forget that the user is running an instance (bypassing the "User does not have an instance" error)
	//Note that in reality, the instance spinned up by the user is still running or being created (and will only be automatically deleted when the instance expires)
	instance.Usr_Id = "" //Make sure the instance is no longer tied to any user id
	api_sql.UpdateInstance(instance)

	log.Debug("Finish /removeInstance/admin Request")
}

func getUserStatus(c *gin.Context) {
	log.Debug("Received /getUserStatus Request")

	userid, ok := c.GetQuery("userid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing userid"})
		return
	}
	if !validateUserid(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid userid"})
		return
	}

	if !activeUserInstance(userid) {
		c.JSON(http.StatusOK, ds.UserStatus{Running_Instance: false})
		return
	}

	log.Debug("Start /getUserStatus Request")

	instance := api_sql.GetActiveUserInstance(userid)

	c.JSON(http.StatusOK, ds.UserStatus{Running_Instance: true, Challenge_Id: instance.Challenge_Id, Time_Left: int((instance.Instance_Timeout-time.Now().UnixNano())/1e9), Host: creds.ExtractHost(instance.Portainer_Url), Ports_Used: api_sql.DeserializeI(instance.Ports_Used), Port_Types: api_sql.Deserialize(api_sql.GetChallenge(instance.Challenge_Id).Port_Types, ",")})

	log.Debug("Finish /getUserStatus Request")
}

func extendTimeLeft(c *gin.Context) {
	log.Debug("Received /extendTimeLeft Request")

	userid, ok := c.GetQuery("userid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing userid"})
		return
	}
	if !validateUserid(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid userid"})
		return
	}

	if !activeUserInstance(userid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "User does not have an instance"})
		return
	}

	instance := api_sql.GetActiveUserInstance(userid)

	if (instance.Instance_Timeout-time.Now().UnixNano())/1e9 > ds.MaxSecondsLeftBeforeExtendAllowed {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "User needs to wait until instance expires in " + strconv.FormatInt(ds.MaxSecondsLeftBeforeExtendAllowed, 10) + " seconds"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"Success": true})

	go _extendTimeLeft(userid)
}

func _extendTimeLeft(userid string) { //Run Async
	log.Debug("Start /extendTimeLeft Request")
	instance := api_sql.GetActiveUserInstance(userid)
	NewInstanceTimeout := time.Now().UnixNano() + ds.DefaultNanosecondsPerInstance

	a, b := ds.InstanceQueue.GetKey(instance.Instance_Id)
	if !b {
		panic("InstanceId is missing in InstanceQueue!")
	}
	ds.InstanceQueue.Remove(a)
	ds.InstanceQueue.Put(NewInstanceTimeout, instance.Instance_Id) //Replace

	api_sql.UpdateInstanceTime(instance.Instance_Id, NewInstanceTimeout)
	log.Debug("Finish /extendTimeLeft Request")
}

func addChallenge(c *gin.Context) {
	log.Debug("Received /addChallenge Request")

	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Authorization missing"})
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid authorization"})
		return
	}

	var raw_challenge_data ds.Challenge
	if err := c.BindJSON(&raw_challenge_data); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if raw_challenge_data.Challenge_Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing challenge_name"})
		return
	}
	if raw_challenge_data.Port_Types == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing port_types"})
		return
	}
	deserialized_port_types := api_sql.Deserialize(raw_challenge_data.Port_Types, ",")
	for _, port_type := range deserialized_port_types {
		if port_type != "nc" && port_type != "ssh" && port_type != "http" {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid port_type " + port_type})
			return
		}
	}

	if raw_challenge_data.Docker_Compose {
		if raw_challenge_data.Docker_Compose_File == "" {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing docker_compose_file"})
			return
		}
		_docker_compose_file, err := base64.StdEncoding.DecodeString(raw_challenge_data.Docker_Compose_File)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid base64 encoding for docker_compose_file"})
			return
		}
		docker_compose_file := string(_docker_compose_file)
		port_count := yaml.DockerComposePortCount(docker_compose_file)
		if port_count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "docker_compose_file does not have any ports exposed"})
			return
		}
		if len(deserialized_port_types) != port_count {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Number of ports exposed in docker_compose_file does not match the number of port types specified"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"Success": true})

		go _addChallengeDockerCompose(raw_challenge_data.Challenge_Name, raw_challenge_data.Port_Types, docker_compose_file)
	} else {
		if raw_challenge_data.Internal_Port == "" {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing internal_port"})
			return
		}
		if raw_challenge_data.Image_Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing image_name"})
			return
		}

		var docker_cmds []byte
		if raw_challenge_data.Docker_Cmds != "" { //docker_cmds is optional
			var err error
			docker_cmds, err = base64.StdEncoding.DecodeString(raw_challenge_data.Docker_Cmds)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid base64 encoding for docker_cmds"})
				return
			}
		}

		if len(deserialized_port_types) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Number of port types specified is not 1"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"Success": true})

		go _addChallengeNonDockerCompose(raw_challenge_data.Challenge_Name, raw_challenge_data.Port_Types, raw_challenge_data.Internal_Port, raw_challenge_data.Image_Name, string(docker_cmds))
	}
}

func _addChallengeDockerCompose(challenge_name string, port_types string, docker_compose_file string) { //Run Async
	log.Debug("Start /addChallenge Request (Docker Compose)")
	port_count := yaml.DockerComposePortCount(docker_compose_file)
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, true, port_count)
	ch := ds.Challenge{Challenge_Id: challenge_id, Challenge_Name: challenge_name, Port_Types: port_types, Docker_Compose: true, Port_Count: port_count, Docker_Compose_File: docker_compose_file}
	api_sql.UpdateChallenge(ch)

	log.Debug("Finish /addChallenge Request (Docker Compose)")
}

func _addChallengeNonDockerCompose(challenge_name string, port_types string, internal_port string, image_name string, docker_cmds string) { //Run Async
	log.Debug("Start /addChallenge Request (Non Docker Compose)")
	challenge_id := api_sql.GetOrCreateChallengeId(challenge_name, false, 1)
	ch := ds.Challenge{Challenge_Id: challenge_id, Challenge_Name: challenge_name, Port_Types: port_types, Docker_Compose: false, Port_Count: 1, Internal_Port: internal_port, Image_Name: image_name, Docker_Cmds: docker_cmds}
	api_sql.UpdateChallenge(ch)

	log.Debug("Finish /addChallenge Request (Non Docker Compose)")
}

func removeChallenge(c *gin.Context) {
	log.Debug("Received /removeChallenge Request")

	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Authorization missing"})
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid authorization"})
		return
	}

	challid, ok := c.GetQuery("challid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Missing challid"})
		return
	}
	if !validateChallid(challid) {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid challid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"Success": true})

	go _removeChallenge(challid)
}

func _removeChallenge(challid string) { //Run Async
	log.Debug("Start /removeChallenge Request")

	ds.ChallengeUnsafeToLaunch[challid] = true; //Mark challenge as unsafe to launch

	for _, instance := range api_sql.GetInstances() {
		if instance.Challenge_Id == challid {
			go _removeInstance(instance.Usr_Id) //Make sure that all instances running this challenge are killed
		}
	}

	ClearInstanceQueue() //Manually trigger ClearInstanceQueue() rather than waiting for the Kill Worker

	api_sql.DeleteChallenge(challid)

	delete(ds.ChallengeUnsafeToLaunch, challid)

	log.Debug("Finish /removeChallenge Request")
}

func getStatus(c *gin.Context) {
	log.Debug("Received /getStatus Request")

	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Authorization missing"})
		return
	} else if auth != creds.APIAuthorization { //TODO: Make this comparison secure
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid authorization"})
		return
	}

	log.Debug("Start /getStatus Request")

	c.JSON(http.StatusOK, ds.RunnerStatus{Current_Instance_Count: api_sql.GetInstanceCount(), Max_Instance_Count: ds.MaxInstanceCount, Instances: api_sql.GetInstances(), Challenges: api_sql.GetChallenges()})

	log.Debug("Finish /getStatus Request")
}