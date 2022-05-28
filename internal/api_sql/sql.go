package api_sql

import (
	"gorm.io/gorm"

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

func GetOrCreateChallengeId(Challenge_Name string, Docker_Compose bool, Port_Count int) string {
	challenge_id := getChallengeId(Challenge_Name)

	if challenge_id != "" {
		return challenge_id
	}

	//If control reaches here, the challenge does not exist in the DB
	challenge := ds.Challenge{Challenge_Id: ds.GenerateChallengeId(Challenge_Name), Challenge_Name: Challenge_Name, Docker_Compose: Docker_Compose, Port_Count: Port_Count}
	DB.Create(&challenge)

	return getChallengeId(Challenge_Name)
}

func getChallengeId(challenge_name string) string {
	ch := ds.Challenge{}
	result := DB.Select("challenge_id").Where("challenge_name = ?", challenge_name).Find(&ch)

	err := result.Error
	if err == gorm.ErrRecordNotFound {
		return ""
	} else if err != nil {
		panic(err)
	}

	return ch.Challenge_Id //Assume there are no duplicate challenge names
}

func UpdateChallenge(ch ds.Challenge) {
	if DB.Model(&ch).Where("challenge_id = ?", ch.Challenge_Id).Updates(&ch).RowsAffected == 0 {
		panic("Updating challenge that does not exist")
	}
}

func DeleteChallenge(challid string) {
	DB.Delete(&ds.Challenge{}, ds.Challenge{Challenge_Id: challid}) //For some reason, db.Delete(&ds.Challenge{}, challid) does not seem to work
}