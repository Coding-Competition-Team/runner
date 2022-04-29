package ds

import (
	"encoding/json"
	"os"

	"runner/internal/log"
)

func LoadConfig() {
	log.Info("Loading Config...")
	json_data, err := os.ReadFile(ConfigFolderPath+PS+ConfigFileName)
	if err != nil {
		panic(err)
	}

	var result ConfigJson
	json.Unmarshal(json_data, &result)

	RunnerPort = result.Runner_Port
	MaxInstanceCount = result.Max_Instance_Count
	DefaultSecondsPerInstance = result.Default_Seconds_Per_Instance
	DefaultNanosecondsPerInstance = DefaultSecondsPerInstance * 1e9
	UsedPorts[RunnerPort] = true //Runner
	for _, port := range result.Reserved_Ports {
		UsedPorts[port] = true
	}
	Database_Max_Retry_Attempts = result.Database_Max_Retry_Attempts
	Database_Error_Wait_Seconds = result.Database_Error_Wait_Seconds
	log.Info("Config Loaded!")
}