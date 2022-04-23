package creds

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"runner/internal/ds"
	"runner/internal/log"
)

var MySQLIP string = ""
var MySQLUsername string = ""
var MySQLPassword string = ""

var PortainerURL string = ""
var PortainerUsername string = ""
var PortainerPassword string = ""
var PortainerJWT string = ""

var APIAuthorization string = ""

func LoadCredentials() {
	log.Info("Loading Credentials...")
	json_data, err := os.ReadFile(ds.CredentialsJsonFile)
	if err != nil {
		panic(err)
	}

	var result map[string]map[string]string
	json.Unmarshal(json_data, &result)

	MySQLIP = result["mysql"]["ip"]
	MySQLUsername = result["mysql"]["username"]
	MySQLPassword = result["mysql"]["password"]

	PortainerURL = result["portainer"]["url"]
	PortainerUsername = result["portainer"]["username"]
	PortainerPassword = result["portainer"]["password"]

	APIAuthorization = result["api"]["authorization"]

	PortainerJWT = getPortainerJWT()
	log.Info("Credentials Loaded!")
}

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