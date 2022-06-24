package api_sql

import (
	"runner/internal/ds"
)

func GetInstance(instance_id int) ds.Instance {
	var instance ds.Instance
	DB.Where("instance_id = ?", instance_id).First(&instance)
	return instance
}

func GetInstances() []ds.Instance {
	instances := []ds.Instance{}
	DB.Find(&instances)
	return instances
}

func GetInstanceCount() int64 {
	var count int64
	DB.Model(&ds.Instance{}).Count(&count)
	return count
}

//instance needs Instance_Id, Usr_Id, Challenge_Id, Instance_Timeout, Ports_Used
func AddInstance(Instance ds.Instance) {
	DB.Create(&Instance)
}

func UpdateInstance(instance ds.Instance) {
	if DB.Model(&instance).Where("instance_id = ?", instance.Instance_Id).Updates(&instance).RowsAffected == 0 {
		panic("Updating instance that does not exist")
	}
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