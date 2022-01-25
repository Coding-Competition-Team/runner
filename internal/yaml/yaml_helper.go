package yaml

import (
	"strconv"
	"strings"
	
	"gopkg.in/yaml.v2"
)

func parseInternalPort(str string) string { //Returns the internal port
	return strings.Split(str, ":")[1]
}

func DockerComposeCopy(docker_compose string, ports []int) string {
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

func DockerComposePortCount(docker_compose string) int {
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