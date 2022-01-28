package api_sql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"runner/internal/ds"
)

func AddInstance(InstanceId int, userid int, challid int, InstanceTimeLeft int64, Ports []int) {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO instances (instance_id, usr_id, challenge_id, instance_timeout, ports_used) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	serialized_ports := SerializeI(Ports, ",")
	_, err = stmt.Exec(InstanceId, userid, challid, InstanceTimeLeft, serialized_ports)
	if err != nil {
		panic(err)
	}
}

func DeleteInstance(InstanceId int) {
	db, err := sql.Open("mysql", GetSqlDataSource())
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

func SetInstancePortainerId(InstanceId int, PortainerId string) {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE instances SET portainer_id = ? WHERE instance_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(PortainerId, InstanceId)
	if err != nil {
		panic(err)
	}
}

func UpdateInstanceTime(InstanceId int, NewInstanceTimeLeft int64) {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE instances SET instance_timeout = ? WHERE instance_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(NewInstanceTimeLeft, InstanceId)
	if err != nil {
		panic(err)
	}
}

func GetOrCreateChallengeId(challenge_name string, docker_compose bool, port_count int) int {
	challenge_id := getChallengeId(challenge_name)

	if challenge_id != -1 {
		return challenge_id
	}

	db, err := sql.Open("mysql", GetSqlDataSource()) //If control reaches here, the challenge does not exist in the DB
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO challenges (challenge_name, docker_compose, port_count) VALUES (?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(challenge_name, docker_compose, port_count)
	if err != nil {
		panic(err)
	}

	return getChallengeId(challenge_name)
}

func getChallengeId(challenge_name string) int {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT challenge_id FROM challenges WHERE challenge_name = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	var challenge_id int
	err = stmt.QueryRow(challenge_name).Scan(&challenge_id)
	if err == sql.ErrNoRows {
		return -1
	} else if err != nil {
		panic(err)
	}
	
	return challenge_id //Assume there are no duplicate challenge names
}

func UpdateChallenge(ch ds.Challenge) {
	db, err := sql.Open("mysql", GetSqlDataSource())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE challenges SET docker_compose = ?, port_count = ?, internal_port = ?, image_name = ?, docker_cmds = ?, docker_compose_file = ? WHERE challenge_id = ?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(ch.DockerCompose, ch.PortCount, ch.InternalPort, ch.ImageName, Serialize(ch.DockerCmds, "\n"), ch.DockerComposeFile, ch.ChallengeId)
	if err != nil {
		panic(err)
	}
}