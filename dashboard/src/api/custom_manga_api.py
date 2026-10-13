import base64
from typing import Any
from urllib.parse import urlencode, urljoin

import requests
from src.api.timeouts import SOURCE_REQUEST_TIMEOUT
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
        url = f"{url}?{urlencode({'id': manga_id, 'url': manga_url, 'name': name})}"

        res = requests.patch(url, timeout=SOURCE_REQUEST_TIMEOUT)

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
        url = f"{url}?{urlencode({'id': manga_id, 'url': manga_url, 'new_url': new_url})}"

        res = requests.patch(url, timeout=SOURCE_REQUEST_TIMEOUT)

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

    def update_custom_manga(
        self,
        manga_id: int,
        name: str,
        manga_url: str,
        last_released_chapter_name_selector: str,
        last_released_chapter_name_attribute: str,
        last_released_chapter_name_regex: str,
        last_released_chapter_name_get_first: bool,
        last_released_chapter_url_selector: str,
        last_released_chapter_url_attribute: str,
        last_released_chapter_url_get_first: bool,
        last_released_chapter_selector_use_browser: bool,
        cover_img_url: str = "",
        cover_img: bytes = b"",
        use_mantium_default_cover_img: bool = False,
    ) -> dict[str, str]:
        """Updates all the custom manga fields at once. Nothing is saved if any of them is invalid."""
        url = f"{self.base_custom_manga_url}?{urlencode({'id': manga_id})}"

        request_body: dict[str, Any] = {
            "name": name,
            "url": manga_url,
            "use_browser": last_released_chapter_selector_use_browser,
            "cover_img_url": cover_img_url,
            "cover_img": base64.b64encode(cover_img).decode("utf-8") if cover_img else "",
            "use_mantium_default_cover_img": use_mantium_default_cover_img,
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

        res = requests.patch(url, json=request_body, timeout=SOURCE_REQUEST_TIMEOUT)

        if res.status_code not in self.acceptable_status_codes:
            request_body["cover_img"] = f"<{len(cover_img)} bytes>"
            raise APIException(
                "error while updating custom manga",
                url,
                "PATCH",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def test_custom_manga_selectors(
        self,
        manga_url: str,
        last_released_chapter_name_selector: str,
        last_released_chapter_name_attribute: str,
        last_released_chapter_name_regex: str,
        last_released_chapter_name_get_first: bool,
        last_released_chapter_url_selector: str,
        last_released_chapter_url_attribute: str,
        last_released_chapter_url_get_first: bool,
        last_released_chapter_selector_use_browser: bool,
    ) -> dict[str, Any]:
        """Gets the last released chapter using the selectors, without saving anything."""
        url = f"{self.base_custom_manga_url}/test_selectors"

        request_body: dict[str, Any] = {
            "url": manga_url,
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

        res = requests.post(url, json=request_body, timeout=SOURCE_REQUEST_TIMEOUT)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while testing custom manga selectors",
                url,
                "POST",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()["chapter"]

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
        url = f"{url}?{urlencode({'id': manga_id, 'url': manga_url})}"

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

        res = requests.patch(url, json=request_body, timeout=SOURCE_REQUEST_TIMEOUT)

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

    def update_custom_manga_cover_img(
        self,
        manga_id: int = 0,
        cover_img_url: str = "",
        cover_img: bytes = b"",
        use_mantium_default_img: bool = False,
    ) -> dict[str, str]:
        path = "/cover_img"
        url = f"{self.base_custom_manga_url}{path}"
        params: dict[str, Any] = {"id": manga_id}
        if cover_img_url:
            params["cover_img_url"] = cover_img_url
        if use_mantium_default_img:
            params["use_mantium_default_img"] = "true"
        url = f"{url}?{urlencode(params)}"

        res = requests.patch(url, files={"cover_img": cover_img}, timeout=SOURCE_REQUEST_TIMEOUT)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga cover image",
                url,
                "PATCH",
                {},
                res.status_code,
                res.text,
            )

        return res.json()
