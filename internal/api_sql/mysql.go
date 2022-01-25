package api_sql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
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