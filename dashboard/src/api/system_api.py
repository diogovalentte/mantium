import requests
from src.exceptions import APIException


class DashboardAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_api_url = base_api_url
        self.acceptable_status_codes: tuple = (
            200,  # The acceptable status codes from the API requests
        )

    def check_health(self):
        """Check the health of the API."""
        url = self.base_api_url + "/v1/health"

        try:
            res = requests.get(url)
        except requests.exceptions.ConnectionError:
            raise Exception(
                "error while checking the health of the API at "
                + url
                + " (Connection Error)"
            )

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while checking the health of the API",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

    def check_for_updates(self):
        """Check for the last time some resource that should trigger an reload of the dashboard was updated.

        Returns:
            (str): Timestamp of the update.
        """
        url = self.base_api_url + "/v1/dashboard/last_update"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while checking for updates in the API",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

        return res.json()["message"]

    def get_dashboard_configs(self):
        """Get the dashboard configs from the API.

        Returns:
            (dict): The configs.
        """
        url = self.base_api_url + "/v1/dashboard/configs"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting the dashboarc configs from the API",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

        return res.json()["configs"]

    def update_dashboard_configs(
        self,
        configs: dict,
    ):
        """Update the dashboard configs.

        Args:
            configs (dict): The configs to update.
        """
        url = self.base_api_url + "/v1/dashboard/configs"

        res = requests.post(url, json=configs)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating the dashboard configs",
                url,
                "POST",
                {},
                res.status_code,
                res.text,
            )

    def get_last_background_error(self):
        """Get the last background error from the API.

        Returns:
            (dict): The error.
        """
        url = self.base_api_url + "/v1/dashboard/last_background_error"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting last background error",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def delete_last_background_error(self):
        """Delete the last background error from the API.

        Returns:
            (dict): The error.
        """
        url = self.base_api_url + "/v1/dashboard/last_background_error"

        res = requests.delete(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while deleting last background error",
                url,
                "DELETE",
                {},
                res.status_code,
                res.text,
            )

    def get_updated_message(self):
        """Get this version updated message.

        Returns:
            (str, str): Message and new version.
        """
        url = self.base_api_url + "/v1/dashboard/updated_message"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting the updated message from the API",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

        json_resp = res.json()

        return json_resp["message"], json_resp["version"]
