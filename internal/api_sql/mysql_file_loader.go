package api_sql

import (
	"errors"
	"os"

	"runner/internal/ds"
	"runner/internal/yaml"
)

func getFileNames(dir string) []string {
	file, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	lst, err := file.Readdirnames(0) //Read folders and files
	if err != nil {
		panic(err)
	}

	return lst
}

func doesFileExist(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func loadChallenge(ctf_name string, challenge_name string) {
	path := ds.ChallDataFolder + ds.PS + ctf_name + ds.PS + challenge_name

	docker_compose, err := doesFileExist(path + ds.PS + "docker-compose.yml")
	if err != nil {
		panic(err)
	}

	var ch ds.Challenge

	if docker_compose {
		_docker_compose_file, err := os.ReadFile(path + ds.PS + "docker-compose.yml")
		if err != nil {
			panic(err)
		}

		docker_compose_file := string(_docker_compose_file)

		ch = ds.Challenge{ChallengeId: -1, ChallengeName: challenge_name, DockerCompose: docker_compose, PortCount: yaml.DockerComposePortCount(docker_compose_file), DockerComposeFile: docker_compose_file}
	} else {
		internal_port, err := os.ReadFile(path + ds.PS + "PORT.txt")
		if err != nil {
			panic(err)
		}

		image_name, err := os.ReadFile(path + ds.PS + "IMAGE.txt")
		if err != nil {
			panic(err)
		}

		docker_cmds, _ := os.ReadFile(path + ds.PS + "CMDS.txt") //If CMDS.txt does not exist, it's fine

		ch = ds.Challenge{ChallengeId: -1, ChallengeName: challenge_name, DockerCompose: docker_compose, PortCount: 1, InternalPort: string(internal_port), ImageName: string(image_name), DockerCmds: DeserializeNL(string(docker_cmds))}
	}

	ds.Challenges = append(ds.Challenges, ch)
	ds.ChallengeNamesMap[challenge_name] = len(ds.Challenges) - 1 //Current index of most recently appended challenge
}

func loadCTF(ctf_name string) {
	path := ds.ChallDataFolder + ds.PS + ctf_name

	lst := getFileNames(path)
	for _, name := range lst {
		loadChallenge(ctf_name, name)
	}
}

func loadChallengeFiles() {
	lst := getFileNames(ds.ChallDataFolder)
	for _, name := range lst {
		if name != ".gitignore" {
			loadCTF(name)
		}
	}
}