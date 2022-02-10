package ds

type InstanceData struct {
	UserId           int
	ChallengeId      string
	InstanceTimeLeft int64 //Unix (Nano) Timestamp of Instance Timeout
	PortainerId      string
	Ports            []int
}

type Challenge struct {
	ChallengeId   string //Defaults to "" (Unknown ChallengeId)
	ChallengeName string
	DockerCompose bool
	PortCount     int

	//For DockerCompose = false:
	InternalPort string
	ImageName    string
	DockerCmds   []string

	//For DockerCompose = true:
	DockerComposeFile string
}