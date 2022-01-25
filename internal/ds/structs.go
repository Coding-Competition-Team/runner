package ds

type InstanceData struct {
	UserId           int
	ChallengeId      int
	InstanceTimeLeft int64 //Unix (Nano) Timestamp of Instance Timeout
	PortainerId      string
	Ports            []int
}

type Challenge struct {
	ChallengeId   int //Defaults to -1 (Unknown ChallengeId)
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