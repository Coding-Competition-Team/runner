package api_sql

import (
	"gorm.io/gorm"

	"runner/internal/ds"
)

func ValidRunnerChallenge(challid string) bool {
	return ValidStruct(ds.RunnerChallenge{}, "challenge_id", challid)
}

func GetRunnerChallenge(challid string) ds.RunnerChallenge {
	var challenge ds.RunnerChallenge
	DB.Where("challenge_id = ?", challid).First(&challenge)
	return challenge
}

func GetRunnerChallenges() []ds.RunnerChallenge {
	challenges := []ds.RunnerChallenge{}
	DB.Find(&challenges)
	return challenges
}

func GetOrCreateRunnerChallengeId(Challenge_Name string, Docker_Compose bool, Port_Count int) string {
	challenge_id := getRunnerChallengeId(Challenge_Name)

	if challenge_id != "" {
		return challenge_id
	}

	//If control reaches here, the challenge does not exist in the DB
	challenge := ds.RunnerChallenge{Challenge_Id: ds.GenerateChallengeId(Challenge_Name), Challenge_Name: Challenge_Name, Docker_Compose: Docker_Compose, Port_Count: Port_Count}
	DB.Create(&challenge)

	return getRunnerChallengeId(Challenge_Name)
}

func getRunnerChallengeId(challenge_name string) string {
	ch := ds.RunnerChallenge{}
	result := DB.Select("challenge_id").Where("challenge_name = ?", challenge_name).Find(&ch)

	err := result.Error
	if err == gorm.ErrRecordNotFound {
		return ""
	} else if err != nil {
		panic(err)
	}

	return ch.Challenge_Id //Assume there are no duplicate challenge names
}

func UpdateRunnerChallenge(ch ds.RunnerChallenge) {
	if DB.Model(&ch).Where("challenge_id = ?", ch.Challenge_Id).Updates(&ch).RowsAffected == 0 {
		panic("Updating challenge that does not exist")
	}
}

func DeleteRunnerChallenge(challid string) {
	DB.Delete(&ds.RunnerChallenge{}, ds.RunnerChallenge{Challenge_Id: challid}) //For some reason, db.Delete(&ds.Challenge{}, challid) does not seem to work
}