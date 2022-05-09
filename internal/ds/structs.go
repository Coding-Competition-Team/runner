package ds

import (
	"encoding/json"
)

type ConfigJson struct {
	Runner_Port                            int
	Max_Instance_Count                     int
	Portainer_JWT_Seconds_Per_Refresh      int
	Default_Seconds_Per_Instance           int64
	Max_Seconds_Left_Before_Extend_Allowed int64
	Reserved_Ports                         []int
	Database_Max_Retry_Attempts            int
	Database_Error_Wait_Seconds            int
	Portainer_Balance_Strategy             string
}

type ThirdPartyCredentialsJson struct {
	Url      string
	Username string
	Password string
}

type CredentialsJson struct {
	Postgresql_Credentials ThirdPartyCredentialsJson
	Portainer_Credentials  []ThirdPartyCredentialsJson
	Api_Authorization      string
}

type Error struct {
	Error string
}

type Success struct {
	Success bool
}

type PortsJson struct {
	Host       string
	Ports_Used []int
}

type UserStatus struct {
	Running_Instance bool
	Challenge_Id     string
	Time_Left        int
	Host             string
	Ports_Used       string
}

type Instance struct {
	Instance_Id      int
	Usr_Id           string
	Challenge_Id     string
	Portainer_Url    string
	Portainer_Id     string
	Instance_Timeout int64 //Unix (Nano) Timestamp of Instance Timeout
	Ports_Used       string
}

type Challenge struct {
	Challenge_Id   string //Defaults to "" (Unknown ChallengeId)
	Challenge_Name string
	Docker_Compose bool
	Port_Count     int

	//For DockerCompose = false:
	Internal_Port string
	Image_Name    string
	Docker_Cmds   string

	//For DockerCompose = true:
	Docker_Compose_File string
}

func (error Error) ToString() string {
	errorJson, err := json.Marshal(error) //No need to pretty print
    if err != nil {
        panic(err)
    }
	return string(errorJson)
}

func (success Success) ToString() string {
	successJson, err := json.Marshal(success) //No need to pretty print
    if err != nil {
        panic(err)
    }
	return string(successJson)
}

func (portsJson PortsJson) ToString() string {
	statusJson, err := json.Marshal(portsJson) //No need to pretty print
    if err != nil {
        panic(err)
    }
	return string(statusJson)
}

func (status UserStatus) ToString() string {
	statusJson, err := json.Marshal(status) //No need to pretty print
    if err != nil {
        panic(err)
    }
	return string(statusJson)
}

func (instance Instance) ToString() string {
	instanceJson, err := json.MarshalIndent(instance, "", "  ") //Pretty print
    if err != nil {
        panic(err)
    }
	return string(instanceJson)
}

func (challenge Challenge) ToString() string {
	challengeJson, err := json.MarshalIndent(challenge, "", "  ") //Pretty print
    if err != nil {
        panic(err)
    }
	return string(challengeJson)
}