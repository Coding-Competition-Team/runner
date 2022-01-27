import os
import subprocess

import docker
import yaml

CTFs = []
challenges = []

class Challenge():
    def __init__(self, CTF, name):
        self.CTF = CTF
        self.name = name

# Scan for CTFs
with os.scandir() as scan:
    for i in scan:
        if i.is_dir(): 
            CTFs.append(i.name)

# Scan for challenges in CTF
for CTF in CTFs:
    with os.scandir('./'+CTF) as scan:
        for i in scan:
            if i.is_dir():
                challenges.append(Challenge(CTF, i.name))

env = os.environ.copy()
env['DOCKER_BUILDKIT'] = '1'


# Save challenge data and build image
for challenge in challenges:
    # Cache challenge directory
    dir = os.path.join('.', challenge.CTF, challenge.name)

    # Parse docker-compose.yml 
    with open(os.path.join(dir, 'docker-compose.yml')) as file:
        chall_data = yaml.safe_load(file)

    # Get port
    for key in chall_data['services'].keys():
        challenge.service = key
        challenge.port = chall_data['services'][key]['ports'][0][:4]
        break

    # Build image 
    docker_compose = subprocess.run(['docker-compose', '-f', os.path.join(dir, 'docker-compose.yml'), 'build'], shell=True, capture_output=True, env=env)
    challenge.image = challenge.name + challenge.service

    # Write to files
    with open(os.path.join(dir, 'PORT.txt'), 'w') as file:
        file.write(challenge.port)
    with open(os.path.join(dir, 'IMAGE.txt'), 'w') as file:
        file.write(challenge.image)
