# Runner 
Runner is meant to simply deployment of isolated chalenges during a CTF. Runner utlises Portainer to deploy challenges and allows the deployment of isolated challenges. Isolated challenges are challegnes unique to each user. 

Features:
- Deploy isolated challenges
- Time limit for deployed challenges
- Supports multiple Portainer servers (in future)
- 
## Config and Credentials
More details are provided in /config

## Quickstart
1. Configure `/config/cnfig.json` and `/config/credentials.json`
2. `docker-compose up -d`

That's it really. 