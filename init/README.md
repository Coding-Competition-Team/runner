## Runner DB Setup

1. SSH into the server and copy 001_runner_db.sql into ~

2. Run the following command to create the runner_db container:
```
docker run --name runner_db -e POSTGRES_USER=root -e POSTGRES_PASSWORD=<password> -p 5432:5432 -d postgres:latest
```

3. Run the following command to create the runner_db database:
```
docker exec -it runner_db psql -c 'CREATE DATABASE runner_db;'
```

4. Run the following command to copy the sql schema into the container:

```
docker cp ./001_runner_db.sql runner_db:/
```

5. Run the following command to load the schema into the database:
```
docker exec -it runner_db psql -U root runner_db -f 001_runner_db.sql
```