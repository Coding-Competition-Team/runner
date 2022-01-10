CREATE DATABASE runner_db;
USE runner_db;

CREATE TABLE challenges (
	challenge_id int unsigned NOT NULL AUTO_INCREMENT,
	challenge_name varchar(255) NOT NULL,
	docker_type enum('single', 'multi') NOT NULL,
	port_mappings varchar(255) NOT NULL,
	image_name varchar(255),
	docker_cmds varchar(255),
	docker_compose text,
	PRIMARY KEY (challenge_id)
);

CREATE TABLE instances (
	instance_id int unsigned NOT NULL AUTO_INCREMENT,
	usr_id int unsigned NOT NULL,
	docker_id varchar(64) NOT NULL,
	instance_timeout bigint unsigned NOT NULL,
	ports_used varchar(255) NOT NULL,
	PRIMARY KEY (instance_id)
);