package api_sql

import (
	"runner/internal/ds"
)

//instance needs Instance_Id, Usr_Id, Challenge_Id, Instance_Timeout, Ports_Used
func AddInstance(Instance ds.Instance) {
	DB.Create(&Instance)
}

func DeleteInstance(Instance_Id int) {
	DB.Delete(&ds.Instance{}, Instance_Id)
}

func SetInstancePortainerId(Instance_Id int, Portainer_Id string) {
	DB.Model(&ds.Instance{}).Where("instance_id = ?", Instance_Id).Update("portainer_id", Portainer_Id)
}

func UpdateInstanceTime(Instance_Id int, New_Instance_Timeout int64) {
	DB.Model(&ds.Instance{}).Where("instance_id = ?", Instance_Id).Update("instance_timeout", New_Instance_Timeout)
}