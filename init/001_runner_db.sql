CREATE TABLE challenges (
	challenge_id varchar(64) PRIMARY KEY,
	challenge_name varchar(255) NOT NULL,
	docker_compose bool NOT NULL,
	port_count int NOT NULL,
	internal_port varchar(255),
	image_name varchar(255),
	docker_cmds varchar(255),
	docker_compose_file text
);

CREATE TABLE instances (
	instance_id SERIAL PRIMARY KEY,
	usr_id varchar(64) NOT NULL,
	challenge_id varchar(64) NOT NULL,
	portainer_id varchar(64) NOT NULL DEFAULT '',
	instance_timeout bigint NOT NULL,
	ports_used varchar(255) NOT NULL
);