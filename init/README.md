## Runner DB Setup

1. SSH into the server and copy 001_runner_db.sql into ~

2. Run the following command to create the runner_db container:
```
docker run --name runner_db -e MYSQL_ROOT_PASSWORD=<password> -d mysql:latest
```

3. Run the following command to initialize the runner_db (once it has finished initialization):
```
docker exec -i <docker-id> mysql -u root -p<password> < 001_runner_db.sql
```

4. Use Portainer's web panel to edit the runner_db container, and expose the port mapping 3306:3306 (so that the runner can access the DB)
