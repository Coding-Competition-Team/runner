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

var PostgreSQLUrl string
var PostgreSQLUsername string
var PostgreSQLPassword string

var PortainerURL string
var PortainerUsername string
var PortainerPassword string
var PortainerJWT string

var APIAuthorization string

func LoadCredentials() {
	log.Info("Loading Credentials...")
	json_data, err := os.ReadFile(ds.ConfigFolderPath+ds.PS+ds.CredentialsFileName)
	if err != nil {
		panic(err)
	}

	var result ds.CredentialsJson
	json.Unmarshal(json_data, &result)

	PostgreSQLUrl = result.Postgresql_Credentials.Url
	PostgreSQLUsername = result.Postgresql_Credentials.Username
	PostgreSQLPassword = result.Postgresql_Credentials.Password

	PortainerURL = result.Portainer_Credentials.Url
	PortainerUsername = result.Portainer_Credentials.Username
	PortainerPassword = result.Portainer_Credentials.Password

	APIAuthorization = result.Api_Authorization

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

	if raw["jwt"] == "" {
		panic("Invalid Portainer credentials")
	}

	return raw["jwt"]
}