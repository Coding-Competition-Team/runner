package api_sql

import (
	"gorm.io/gorm"

	"runner/internal/creds"
	"runner/internal/ds"
)

//instance needs Instance_Id, Usr_Id, Challenge_Id, Instance_Timeout, Ports_Used
func AddInstance(Instance ds.Instance) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.Create(&Instance)
}

func DeleteInstance(Instance_Id int) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.Delete(&ds.Instance{}, Instance_Id)
}

func SetInstancePortainerId(Instance_Id int, Portainer_Id string) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.Model(&ds.Instance{}).Where("instance_id = ?", Instance_Id).Update("portainer_id", Portainer_Id)
}

func UpdateInstanceTime(Instance_Id int, New_Instance_Timeout int64) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.Model(&ds.Instance{}).Where("instance_id = ?", Instance_Id).Update("instance_timeout", New_Instance_Timeout)
}

func GetOrCreateChallengeId(Challenge_Name string, Docker_Compose bool, Port_Count int) string {
	challenge_id := getChallengeId(Challenge_Name)

	if challenge_id != "" {
		return challenge_id
	}

	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{}) //If control reaches here, the challenge does not exist in the DB
	if err != nil {
		panic(err)
	}

	challenge := ds.Challenge{Challenge_Id: ds.GenerateChallengeId(Challenge_Name), Challenge_Name: Challenge_Name, Docker_Compose: Docker_Compose, Port_Count: Port_Count}
	db.Create(&challenge)

	return getChallengeId(Challenge_Name)
}

func getChallengeId(challenge_name string) string {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	ch := ds.Challenge{}
	result := db.Select("challenge_id").Where("challenge_name = ?", challenge_name).Find(&ch)

	err = result.Error
	if err == gorm.ErrRecordNotFound {
		return ""
	} else if err != nil {
		panic(err)
	}

	return ch.Challenge_Id //Assume there are no duplicate challenge names
}

func UpdateChallenge(ch ds.Challenge) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	if db.Model(&ch).Where("challenge_id = ?", ch.Challenge_Id).Updates(&ch).RowsAffected == 0 {
		panic("Updating challenge that does not exist")
	}
}

func DeleteChallenge(challid string) {
	db, err := gorm.Open(creds.GetSqlDataSource(), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.Delete(&ds.Challenge{}, ds.Challenge{Challenge_Id: challid}) //For some reason, db.Delete(&ds.Challenge{}, challid) does not seem to work
}