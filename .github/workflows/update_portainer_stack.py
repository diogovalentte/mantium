import sys

import requests


def get_auth_token(portainer_url, portainer_username, portainer_password):
    url = f"{portainer_url}/api/auth"
    headers = {
        "Content-Type": "application/json",
    }
    data = {
        "username": portainer_username,
        "password": portainer_password,
    }
    response = requests.post(url, headers=headers, json=data)
    response.raise_for_status()
    return response.json()["jwt"]


def get_stack(portainer_url, auth_token, portainer_stack_name):
    url = f"{portainer_url}/api/stacks"
    headers = {
        "Authorization": f"Bearer {auth_token}",
    }
    response = requests.get(url, headers=headers)
    response.raise_for_status()
    stacks = response.json()
    stack_return = None
    for stack in stacks:
        if stack["Name"] == portainer_stack_name:
            stack_return = stack
            break

    if stack_return is None:
        raise ValueError(f"Stack {portainer_stack_name} not found")

    return stack_return


def update_stack(portainer_url, auth_token, stack, github_username, github_token):
    stack_id = stack["Id"]
    stack_endpoint_id = stack["EndpointId"]
    stack_env = stack["Env"]

    url = f"{portainer_url}/api/stacks/{stack_id}/git/redeploy?endpointId={stack_endpoint_id}"
    body = {
        "env": stack_env,
        "prune": True,
        "pullImage": True,
        "repositoryAuthentication": True,
        "repositoryUsername": github_username,
        "repositoryPassword": github_token,
    }
    headers = {
        "Authorization": f"Bearer {auth_token}",
    }
    response = requests.put(url, headers=headers, json=body)
    response.raise_for_status()


def main():
    if len(sys.argv) != 7:
        print(
            "Usage: python main.py <portainer_url> <portainer_username> <portainer_password> <portainer_stack_name> <github_username> <github_token/password>"
        )
        sys.exit(1)
    portainer_url = sys.argv[1]
    portainer_username = sys.argv[2]
    portainer_password = sys.argv[3]
    portainer_stack_name = sys.argv[4]
    github_username = sys.argv[5]
    github_token = sys.argv[6]

    portainer_auth_token = get_auth_token(
        portainer_url, portainer_username, portainer_password
    )
    portainer_stack = get_stack(
        portainer_url, portainer_auth_token, portainer_stack_name
    )
    update_stack(
        portainer_url,
        portainer_auth_token,
        portainer_stack,
        github_username,
        github_token,
    )


if __name__ == "__main__":
    main()
