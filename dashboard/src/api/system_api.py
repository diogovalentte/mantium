import requests
from src.exceptions import APIException


class DashboardAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_api_url = base_api_url
        self.acceptable_status_codes: tuple = (
            200,  # The acceptable status codes from the API requests
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
                "error while adding manga",
                url,
                "POST",
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
                "error while adding manga",
                url,
                "POST",
                res.status_code,
                res.text,
            )

        return res.json()["configs"]

    def update_dashboard_configs_columns(self, columns: int):
        """Update the columns in the dashboard configs.

        Args:
            columns (int): New columns number.
        """
        url = (
            self.base_api_url + "/v1/dashboard/configs/columns?columns=" + str(columns)
        )

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while adding manga",
                url,
                "POST",
                res.status_code,
                res.text,
            )
