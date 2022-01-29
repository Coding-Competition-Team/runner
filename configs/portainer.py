import os
import subprocess

import requests
import yaml

CTFs = []
challenges = []
runner_endpoint = 'http://localhost'
runner_pw = os.getenv('API_AUTH')
if runner_pw == '':
    print('Runner authentication token has not been set! (API_AUTH=) Please set it before running the script.')

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
    try:
        with open(os.path.join(dir, 'docker-compose.yml')) as file:
            chall_data = yaml.safe_load(file)
    except:
        print(f'{challenge.name} has no docker-compose')
        continue

    # Get port
    for key in chall_data['services'].keys():
        challenge.port = chall_data['services'][key]['ports'][0][5:]
        # TODO: add support for multiple services
        break

    # Build image 
    docker = subprocess.run([f'docker build --tag {challenge.CTF}_{challenge.name} {dir}'], shell=True, capture_output=True, env=env)
    if docker.stderr != b'':
        print(f'An error occurred while building the image. {bytes.decode(docker.stderr, "utf-8")}')
        continue
    challenge.image = f'{challenge.CTF}_{challenge.name}'

    # Send POST to runner
    headers = { 
            'Authorization': runner_pw
    }
    payload = {
            'challenge_name': challenge.name,
            'docker_compose': 'False',
            'internal_port': challenge.port,
            'image_name': challenge.image
    }
    requests.post(f'{runner_endpoint}/addChallenge', headers=headers, json=payload)
    print(f'{challenge.name} in {challenge.CTF} has been deployed successfully as {challenge.image}')
