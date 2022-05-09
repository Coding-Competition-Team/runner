package ds

import (
	"encoding/json"
	"os"

	"runner/internal/log"
)

func validatePortainerBalanceStrategy(strategy string) bool {
	for _, validStrategy := range PortainerBalanceStrategies {
		if strategy == validStrategy {
			return true
		}
	}
	return false
}

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
	PortainerJWTSecondsPerRefresh = result.Portainer_JWT_Seconds_Per_Refresh
	DefaultSecondsPerInstance = result.Default_Seconds_Per_Instance
	DefaultNanosecondsPerInstance = DefaultSecondsPerInstance * 1e9
	UsedPorts[RunnerPort] = true //Runner
	for _, port := range result.Reserved_Ports {
		UsedPorts[port] = true
	}
	Database_Max_Retry_Attempts = result.Database_Max_Retry_Attempts
	Database_Error_Wait_Seconds = result.Database_Error_Wait_Seconds
	if !validatePortainerBalanceStrategy(result.Portainer_Balance_Strategy){
		panic("Please specify a valid Portainer Balance Strategy")
	}
	PortainerBalanceStrategy = result.Portainer_Balance_Strategy
	log.Info("Config Loaded!")
}