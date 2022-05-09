package workers

import (
	"time"

	"runner/internal/creds"
	"runner/internal/ds"
	"runner/internal/log"
)

//Note:
//When refreshing to get a new Portainer JWT Token, the previous tokens are not invalidated.
//This means that there are no issues with running this asynchronously, as prior tokens are still valid, and can be used until their expiry.
func JWTRefreshWorker() {
	tick := time.Tick(time.Duration(ds.PortainerJWTSecondsPerRefresh) * time.Second)
	for range tick {
		current_timestamp := time.Now().UnixNano()
		log.Info("JWT Refresh Worker", current_timestamp)

		for _, credentials := range creds.PortainerCreds {
			creds.PortainerJWT[credentials.Url] = creds.GetPortainerJWT(credentials)
		}
	}
}