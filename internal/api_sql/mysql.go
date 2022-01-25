package api_sql

import (
	"database/sql"
	"encoding/json"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
)

func DeleteInstance(InstanceId int) {
	db, err := sql.Open("mysql", creds.GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM instances WHERE instance_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(InstanceId)
	if err != nil {
		panic(err)
	}
}

func syncChallenges() {
	db, err := sql.Open("mysql", creds.GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT challenge_id, challenge_name FROM challenges") //Get currently registered challenges in the DB
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var challenge_ids []int
	var challenge_names []string //Assumes no duplicate challenge names

	for rows.Next() {
		var challenge_id int
		var challenge_name string
		if err := rows.Scan(&challenge_id, &challenge_name); err != nil {
			panic(err)
		}

		challenge_ids = append(challenge_ids, challenge_id)
		challenge_names = append(challenge_names, challenge_name)
	}

	var new_challenge_names map[string]int = make(map[string]int) //TODO: Better way to do a deepcopy?
	jsonStr, err := json.Marshal(ds.ChallengeNamesMap)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonStr, &new_challenge_names)
	if err != nil {
		panic(err)
	}

	var edit_challenge_ids []int
	var edit_challenge_names []string

	for i, name := range challenge_names {
		_, ok := new_challenge_names[name]
		if ok { //Challenge name already exists in DB
			id := challenge_ids[i]
			idx := ds.ChallengeNamesMap[name]

			delete(new_challenge_names, name)
			edit_challenge_names = append(edit_challenge_names, name)
			edit_challenge_ids = append(edit_challenge_ids, id)

			ds.Challenges[idx].ChallengeId = id //Replace with ChallengeId in DB
			ds.ChallengeIDMap[id] = idx
		} else {
			log.Warn("Challenge", name, "exists in DB but is not in use!")
		}
	}

	stmt1, err := db.Prepare("INSERT INTO challenges (challenge_name, docker_compose, port_count, internal_port, image_name, docker_cmds, docker_compose_file) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt1.Close()

	stmt1b, err := db.Prepare("SELECT challenge_id FROM challenges WHERE challenge_name = ?")
	if err != nil {
		panic(err)
	}
	defer stmt1b.Close()

	for k, v := range new_challenge_names { //Insert new challenges
		log.Debug("New Challenge:", k, ",", v)

		ch := ds.Challenges[v]
		_, err = stmt1.Exec(k, ch.DockerCompose, ch.PortCount, ch.InternalPort, ch.ImageName, Serialize(ch.DockerCmds, "\n"), ch.DockerComposeFile)
		if err != nil {
			panic(err)
		}

		var challenge_id int
		if err := stmt1b.QueryRow(k).Scan(&challenge_id); err != nil {
			panic(err)
		}
		ds.Challenges[v].ChallengeId = challenge_id //Get DB assigned challenge id
		ds.ChallengeIDMap[challenge_id] = v
	}

	stmt2, err := db.Prepare("UPDATE challenges SET docker_compose = ?, port_count = ?, internal_port = ?, image_name = ?, docker_cmds = ?, docker_compose_file = ? WHERE challenge_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt2.Close()

	for i, name := range edit_challenge_names { //Edit pre-existing challenges
		log.Debug("Edit Challenge:", i, ",", name)

		ch := ds.Challenges[ds.ChallengeNamesMap[name]]
		_, err = stmt2.Exec(ch.DockerCompose, ch.PortCount, ch.InternalPort, ch.ImageName, Serialize(ch.DockerCmds, "\n"), ch.DockerComposeFile, edit_challenge_ids[i])
		if err != nil {
			panic(err)
		}
	}
}

func syncInstances() {
	db, err := sql.Open("mysql", creds.GetSqlDataSource())
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
	loadChallengeFiles()
	syncChallenges()
	log.Debug("Challenge Data:", ds.Challenges)
	log.Debug("Challenge ID Map:", ds.ChallengeIDMap)
	syncInstances()
	log.Debug("Instance Map:", ds.InstanceMap)
	log.Info("DB Sync Complete!")
}