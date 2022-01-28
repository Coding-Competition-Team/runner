package api_sql

import (
	"database/sql"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"runner/internal/ds"
	"runner/internal/log"
)

func syncChallenges() {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT challenge_id, challenge_name, docker_compose, port_count, internal_port, image_name, docker_cmds, docker_compose_file FROM challenges") //Get currently registered challenges in the DB
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var challenge_id int
		var challenge_name string
		var docker_compose bool
		var port_count int
		var internal_port string
		var image_name string
		var docker_cmds string
		var docker_compose_file string

		if err := rows.Scan(&challenge_id, &challenge_name, &docker_compose, &port_count, &internal_port, &image_name, &docker_cmds, &docker_compose_file); err != nil {
			panic(err)
		}

		ch := ds.Challenge{ChallengeId: challenge_id, ChallengeName: challenge_name, DockerCompose: docker_compose, PortCount: port_count, InternalPort: internal_port, ImageName: image_name, DockerCmds: DeserializeNL(docker_cmds), DockerComposeFile: docker_compose_file}
		ds.ChallengeMap[challenge_id] = ch
		ds.ChallengeNamesMap[challenge_name] = challenge_id
	}
}

func syncInstances() {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM instances") //Fully trust DB
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var instance_id int
		var usr_id int
		var challenge_id int
		var portainer_id string
		var instance_timeout int64
		var ports_used string
		if err := rows.Scan(&instance_id, &usr_id, &challenge_id, &portainer_id, &instance_timeout, &ports_used); err != nil {
			panic(err)
		}

		if (instance_id + 1) > ds.NextInstanceId {
			ds.NextInstanceId = instance_id + 1
		}
		ds.ActiveUserInstance[usr_id] = instance_id
		ds.InstanceQueue.Put(instance_timeout, instance_id)

		var ports []int
		deserialized_ports := Deserialize(ports_used, ",")
		for _, v := range deserialized_ports {
			port, err := strconv.Atoi(v)
			if err != nil {
				panic(err)
			}
			ports = append(ports, port)
			ds.UsedPorts[port] = true
		}
		ds.InstanceMap[instance_id] = ds.InstanceData{UserId: usr_id, ChallengeId: challenge_id, InstanceTimeLeft: instance_timeout, PortainerId: portainer_id, Ports: ports}
	}
}

func SyncWithDB() {
	log.Info("Starting DB Sync...")
	syncChallenges()
	log.Debug("Challenge Map:", ds.ChallengeMap)
	syncInstances()
	log.Debug("Instance Map:", ds.InstanceMap)
	log.Info("DB Sync Complete!")
}