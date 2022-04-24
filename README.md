# Runner 
Runner is meant to simplify deployment of isolated challenges during a CTF. Runner utlises Portainer to deploy challenges and allows the deployment of isolated challenges. Isolated challenges are challegnes unique to each user. 

Features:
- Deploy isolated challenges
- Time limit for deployed challenges
- Supports multiple Portainer servers (in future)

## Config and Credentials
More details are provided in /config

## Quickstart
1. Configure `/config/config.json` and `/config/credentials.json`
2. `docker-compose up -d`

That's it really. 

## API Reference

 * `addInstance`
   * Adds an additional Instance for a specific user and challenge. 
   * `/addInstance?userid=XXXX&challid=XXXX`
   * `userid` must be a valid userid
   * `challid` is the SHA256 hash of the challenge name, and must be a valid challid within the database (i.e to say, the challengeID has been mapped to an image/stack name)
   * Errors: 
     * Missing/Invalid `userID`
     * Mising/Invalid `challID`
     * User has already deployed a challenge (only one challenge per user at any one time)
     * Max number of instances has been reached

  * `removeInstance`
    * Removes an Instance for a specific user
    * `/removeInstance?userid=XXXX`
    * `userid` must be a valid userid
    * Errors:
      * Missing/Invalid `userID`
      * User does not have an instance running
      * User's Instance is still starting

* `getTimeLeft`
  * gets time left for a a specific user's instance
  * `/getTimeLeft?userid=XXXX`
  * `userid` must be a valid userid
    * Errors:
      * Missing/Invalid `userID`
      * User does not have an instance running
* `extendTimeLeft`
  * Extends the time left for a specific user's instance
  * `extendTimeLeft?userid=XXXX`
  * `userid` must be a valid userid
    * Errors:
      * Missing/Invalid `userID`
      * User does not have an instance running
* `addChallenge`
  * Maps a challenge name to its Portainer Image/Stack as well as it's docker-compose file.
  * Requires authorisation header!
  * Data is sent as a JSON
  
    ```
    {
            'challenge_name': , 
            'docker_compose': ,
            'internal_port': ,
            'image_name': ,
            'docker_compose_file' : ,
    }
    ```
    * `challenge_name`: any valid challenge name in LOWERCASE
    * `docker_compose`: Either `'True'` or `'False'`
    * `internal_port`: Dockerfile Exposed Port
    * `image_name`: Image name of built Docker image
    * `docker_compose_file`: Docker Compose file that is **compatible with Portainer Stacks** and **base64 encoded**
  * `challenge_name`, `docker_compose=` are compulsory
  * For Dockerfiles, `internal_port` and `image_name` are also required.
  * For Docker Compose, `docker_compose_file` is also required.
  * Errors:
    * JSON Invalid
    * Missing/Invalid `docker_compose` value
    * Missing/Invalid `docker_compose_file`
    * Missing `internal_port`
    * Missing `image_name`
