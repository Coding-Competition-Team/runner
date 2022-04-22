## Runner DB Setup

1. SSH into the server and copy 001_runner_db.sql into ~

2. Run the following command to create the runner_db container:
```
docker run --name runner_db -e POSTGRES_PASSWORD=<password> -p 5432:5432 -d postgres:latest
```

TODO

3. Run the following command to setup the runner_db (once it has finished initialization):
```
docker exec -i <docker-id> mysql -u root -p<password> < 001_runner_db.sql
```