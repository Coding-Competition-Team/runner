package creds

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"runner/internal/ds"
	"runner/internal/log"
)

var PostgreSQLCreds ds.ThirdPartyCredentialsJson

var PortainerCreds ds.ThirdPartyCredentialsJson
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

	PostgreSQLCreds = result.Postgresql_Credentials
	PortainerCreds = result.Portainer_Credentials
	APIAuthorization = result.Api_Authorization

	PortainerJWT = getPortainerJWT()

	testSqlConnection()

	log.Info("Credentials Loaded!")
}

func getPortainerJWT() string {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //TODO: Remove

	requestBody, err := json.Marshal(map[string]string{
		"Username": PortainerCreds.Username,
		"Password": PortainerCreds.Password,
	})
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(PortainerCreds.Url+"/api/auth", "application/json", bytes.NewBuffer(requestBody))
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

func testSqlConnection() {
	for i := 1; i <= ds.Database_Max_Retry_Attempts; i++ {
		log.Info("Testing Database Connection... | Attempt No.", i)
		_, err := gorm.Open(GetSqlDataSource(), &gorm.Config{})
		if err == nil {
			log.Info("Database Connection Successful!")
			return
		} else {
			log.Warn("Database Connection Error:", err)
			if i == ds.Database_Max_Retry_Attempts { //No need to Sleep anymore since the last attempt was an error
				break
			}
			log.Info("Retrying Database Connection in", ds.Database_Error_Wait_Seconds, "seconds...")
			time.Sleep(time.Duration(ds.Database_Error_Wait_Seconds) * time.Second)
		}
	}
	panic("Unable to connect to database!")
}

func GetSqlDataSource() gorm.Dialector {
	return postgres.Open("host="+PostgreSQLCreds.Url+" user="+PostgreSQLCreds.Username+" password="+PostgreSQLCreds.Password+" dbname=runner_db")
}