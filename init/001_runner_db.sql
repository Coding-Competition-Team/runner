CREATE DATABASE runner_db;
USE runner_db;

CREATE TABLE challenges (
	challenge_id int unsigned NOT NULL AUTO_INCREMENT,
	challenge_name varchar(255) NOT NULL,
	docker_compose bool NOT NULL,
	port_count int NOT NULL,
	internal_port varchar(255),
	image_name varchar(255),
	docker_cmds varchar(255),
	docker_compose_file text,
	PRIMARY KEY (challenge_id)
);

CREATE TABLE instances (
	instance_id int unsigned NOT NULL AUTO_INCREMENT,
	usr_id int unsigned NOT NULL,
	challenge_id int unsigned NOT NULL,
	portainer_id varchar(64),
	instance_timeout bigint unsigned NOT NULL,
	ports_used varchar(255) NOT NULL,
	PRIMARY KEY (instance_id)
);