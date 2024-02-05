package api_portainer

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"runner/internal/creds"
	"runner/internal/log"
)

func LaunchContainer(portainer_url string, container_name string, image_name string, cmds []string, internal_port string, _external_port int, discriminant string) string {
	external_port := strconv.Itoa(_external_port)

    // wtf is this
	cmd := ""
	for i, s := range cmds {
		cmd += "\"" + s + "\""
		if (i + 1) < len(cmds) {
			cmd += ","
		}
	}

	tmp := "{\"Cmd\":[" + cmd + "],\"Image\":\"" + image_name + "\",\"ExposedPorts\":{\"" + internal_port + "/tcp\":{}},\"HostConfig\":{\"PortBindings\":{\"" + internal_port + "/tcp\":[{\"HostPort\":\"" + external_port + "\"}]}}}"
	log.Debug("launchContainer Body:", tmp)

	requestBody := []byte(tmp)

	client := http.Client{}
	req, err := http.NewRequest("POST", portainer_url+"/api/endpoints/2/docker/containers/create?name="+container_name+"_"+discriminant, bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + creds.PortainerJWT[portainer_url]},
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
	log.Debug("launchContainer Response:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		panic(err)
	}
	id := raw["Id"].(string)

	startContainer(portainer_url, id)

	return id
}

func startContainer(portainer_url string, id string) {
	requestBody := []byte("{}")

	client := http.Client{}
	req, err := http.NewRequest("POST", portainer_url+"/api/endpoints/2/docker/containers/"+id+"/start", bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + creds.PortainerJWT[portainer_url]},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	log.Info("startContainer", string(body))
}

func DeleteContainer(portainer_url string, id string) {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", portainer_url+"/api/endpoints/2/docker/containers/"+id+"?force=true", nil)
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + creds.PortainerJWT[portainer_url]},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	log.Info("deleteContainer", string(body))
}

func LaunchStack(portainer_url string, stack_name string, docker_compose string, discriminant string) string {
	json_docker_compose, err := json.Marshal(docker_compose) //Make sure docker_compose is JSON Encoded
	if err != nil {
		panic(err)
	}

    // bruh
	tmp := "{\"name\":\"" + stack_name + "_" + discriminant + "\",\"stackFileContent\":" + string(json_docker_compose) + "}"
    
    // tmp := map[string]interface{}{
    //     "name": stack_name + "_" + discriminant,
    //     "stackFileContent": string(json_docker_compose),
    // }
    log.Debug(tmp)
    // reqJson, err := json.Marshal(tmp)
	log.Debug("launchStack Body:", tmp)

	requestBody := []byte(tmp)

	client := http.Client{}
	// req, err := http.NewRequest("POST", portainer_url+"/api/stacks?type=2&method=string&endpointId=2", bytes.NewBuffer(reqJson))
	req, err := http.NewRequest("POST", portainer_url+"/api/stacks/create/standalone/string?endpointId=2", bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + creds.PortainerJWT[portainer_url]},
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
	log.Debug("launchStack Response:", string(body))

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		panic(err)
	}
	id := int(raw["Id"].(float64)) //Cannot directly cast to string

	return strconv.Itoa(id)
}

func DeleteStack(portainer_url string, id string) {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", portainer_url+"/api/stacks/"+id+"?endpointId=2", nil)
	if err != nil {
        print(portainer_url+"/api/stacks/"+id+"?endpointId=2")
		panic(err)
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + creds.PortainerJWT[portainer_url]},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	log.Info("deleteStack", string(body))
}
