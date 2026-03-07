from typing import Any
from urllib.parse import urljoin

import requests
from src.exceptions import APIException


class CustomMangaAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_custom_manga_url: str = urljoin(base_api_url, "/v1/custom_manga")
        self.acceptable_status_codes: tuple = (200,)

    def update_custom_manga_name(
        self,
        name: str,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/name"
        url = f"{self.base_custom_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&name={name}"

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga name",
                url,
                "PATCH",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def update_custom_manga_url(
        self,
        new_url: str,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/url"
        url = f"{self.base_custom_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&new_url={new_url}"

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga URL",
                url,
                "PATCH",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def update_custom_manga_last_released_chapter_selectors(
        self,
        last_released_chapter_name_selector: str,
        last_released_chapter_name_attribute: str,
        last_released_chapter_name_regex: str,
        last_released_chapter_name_get_first: str,
        last_released_chapter_url_selector: str,
        last_released_chapter_url_attribute: str,
        last_released_chapter_url_get_first: str,
        last_released_chapter_selector_use_browser: bool,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/last_released_chapter_selectors"
        url = f"{self.base_custom_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}"

        request_body: dict[str, Any] = {
            "use_browser": last_released_chapter_selector_use_browser,
        }
        if last_released_chapter_name_selector != "":
            request_body["name_selector"] = {
                "selector": last_released_chapter_name_selector,
                "attribute": last_released_chapter_name_attribute,
                "regex": last_released_chapter_name_regex,
                "get_first": last_released_chapter_name_get_first,
            }
        if last_released_chapter_url_selector != "":
            request_body["url_selector"] = {
                "selector": last_released_chapter_url_selector,
                "attribute": last_released_chapter_url_attribute,
                "get_first": last_released_chapter_url_get_first,
            }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga has more chapters property",
                url,
                "PATCH",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()
