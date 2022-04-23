package ds

type ConfigJson struct {
	Runner_Port                  int
	Max_Instance_Count           int
	Default_Seconds_Per_Instance int64
	Reserved_Ports               []int
}

type Instance struct {
	Instance_Id      int
	Usr_Id           int
	Challenge_Id     string
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