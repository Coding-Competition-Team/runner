import base64
import logging
import os
import subprocess

import regex
import requests
import yaml



# -----------------------------------------
# --- config vars (change as necessary) --- 
# -----------------------------------------
DEBUG = True
runner_endpoint = 'http://localhost'
runner_pw = os.getenv('API_AUTH', 'foobar')



# --- logging config ---
if DEBUG:
    logging.basicConfig(encoding='utf-8', level=logging.DEBUG)
else:
    logging.basicConfig(encoding='utf-8', level=logging.INFO)

# --- global vars --- 
CTFs = []
challenges = []

# --- build docker-compose.yml and send data to runner endpoint --- 
def docker_compose(challenge, chall_data, dir, env):
    # Reject composes with local volumes
    for service in chall_data['services']:
        try:
            defined_volumes = chall_data.get('volumes')
            for volume in chall_data['services'][service]['volumes']:
                if defined_volumes == None or volume not in defined_volumes:
                    logging.warning(f'Service {service} of {challenge} could not be built, volume {volume} is not defined in docker-compose top level. Likely locally mounted (not supported)?')
                    return
        except:
            continue

    # Build docker-compose
    logging.info(f'Building {challenge}...')
    docker = subprocess.run([f'docker-compose', 'build'], cwd=dir, stdout=subprocess.PIPE, env=env) 
    try:
        for stdout_line in iter(docker.stdout.readline, ""):
            print('| ' + stdout_line)
    except AttributeError:
        pass
    if docker.returncode != 0:
        logging.warning(f'An error occured while building {challenge}. {bytes.decode(docker.stderr, "utf-8")}')
        return
    else:
        logging.info(f'{challenge} built successfully')
    
    # Create new docker-compose
    new_compose = chall_data
    for service, data in chall_data['services'].items():
        new_compose['services'][service].pop('build', 0)
        new_compose['services'][service]['image'] = f'{challenge}_{service}'
    new_compose = bytes(yaml.dump(new_compose), 'utf-8')

    # Send data to runner
    headers = { 
            'Authorization': runner_pw
    }
    payload = {
            'challenge_name': challenge,
            'docker_compose': 'True',
            'docker_compose_file': bytes.decode(base64.b64encode(new_compose)),
    }
    if DEBUG != True:
        r = requests.post(f'{runner_endpoint}/addChallenge', headers=headers, json=payload)
        if r.status_code != 200:
            logging.warning(f'Runner down? {r.content}')
    else:
        logging.debug(f'{headers},{payload}')
    logging.info(f'{challenge} deployed successfully')
    return


# --- main --- 
def main():
    # TODO: 1 thread per challenge to speed up composing
    # Scan for challenges
    with os.scandir() as scan:
        for i in scan:
            if i.is_dir():
                challenges.append(i.name)

    env = os.environ.copy()

    # Save challenge data and build image
    for challenge in challenges:
        # Cache challenge directory
        dir = os.path.join('.', challenge)

        # Check for docker-compose.yml 
        try:
            with open(os.path.join(dir, 'docker-compose.yml')) as file:
                chall_data = yaml.safe_load(file)
        except:
            logging.warning(f'{challenge} has no docker-compose, skipping')
            continue
        else:
            result = docker_compose(challenge, chall_data, dir, env)

if __name__ == '__main__':
    main()
