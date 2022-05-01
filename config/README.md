## Config

The ``config.json`` config file has already been provided. Feel free to amend the values as needed.

For ``Reserved_Ports``,
- 8000 -> Portainer
- 9443 -> Portainer
- 5432 -> PostgreSQL
- 22 -> SSH

Note that the ``Runner_Port`` is automatically reserved.

## Credentials

In order for the runner to interface with the PostgreSQL DB and Portainer, credentials (IP/URL addresses, usernames, passwords, etc.) need to be provided.

A sample JSON credentials file has been provided in ``credentials.example.json``, which should be copied to a file named ``credentials.json`` with the fields filled out.

Multiple Portainer credentials may be supplied, as the runner supports the use of multiple Portainer instances.