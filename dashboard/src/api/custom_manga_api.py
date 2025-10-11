import base64
from urllib.parse import urljoin

import requests
from src.exceptions import APIException


class CustomMangaAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_custom_manga_url: str = urljoin(base_api_url, "/v1/custom_manga")
        self.acceptable_status_codes: tuple = (200,)

    def add_custom_manga(
        self,
        manga_name: str,
        manga_url: str,
        manga_status: int,
        cover_img_url: str,
        cover_img: bytes,
        last_read_chapter: str,
        last_read_chapter_url: str,
        last_released_chapter_name_selector: str,
        last_released_chapter_name_attribute: str,
        last_released_chapter_name_regex: str,
        last_released_chapter_url_selector: str,
        last_released_chapter_url_attribute: str,
    ) -> dict[str, str]:
        url = self.base_custom_manga_url

        request_body = {
            "name": manga_name,
            "url": manga_url,
            "status": manga_status,
            "cover_img_url": cover_img_url,
            "cover_img": base64.b64encode(cover_img).decode("utf-8") if cover_img else "",
        }

        if last_read_chapter:
            request_body["last_read_chapter"] = {
                "chapter": last_read_chapter,
                "url": last_read_chapter_url,
            }
        if last_released_chapter_name_selector != "":
            request_body["last_released_chapter_name_selector"] = {
                "selector": last_released_chapter_name_selector,
                "attribute": last_released_chapter_name_attribute,
                "regex": last_released_chapter_name_regex,
            }
        if last_released_chapter_url_selector != "":
            request_body["last_released_chapter_url_selector"] = {
                "selector": last_released_chapter_url_selector,
                "attribute": last_released_chapter_url_attribute,
            }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while adding custom manga",
                url,
                "POST",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

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

    def update_custom_manga_last_read_chapter(
        self,
        manga_id: int = 0,
        manga_url: str = "",
        chapter: str = "",
        chapter_url: str = "",
        set_to_last_released_chapter: bool = False,
    ) -> dict[str, str]:
        path = "/last_read_chapter"
        url = f"{self.base_custom_manga_url}{path}"
        url = (
            f"{url}?id={manga_id}&url={manga_url}"
        )

        request_body = {}
        if not set_to_last_released_chapter:
            request_body["chapter"] = {
                "chapter": chapter,
                "url": chapter_url,
            }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga last read chapter",
                url,
                "PATCH",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def update_custom_manga_last_released_chapter_selectors(
        self,
        last_released_chapter_name_selector: str,
        last_released_chapter_name_attribute: str,
        last_released_chapter_name_regex: str,
        last_released_chapter_url_selector: str,
        last_released_chapter_url_attribute: str,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/last_released_chapter_selectors"
        url = f"{self.base_custom_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}"

        request_body = {}
        if last_released_chapter_name_selector != "":
            request_body["name_selector"] = {
                "selector": last_released_chapter_name_selector,
                "attribute": last_released_chapter_name_attribute,
                "regex": last_released_chapter_name_regex,
            }
        if last_released_chapter_url_selector != "":
            request_body["url_selector"] = {
                "selector": last_released_chapter_url_selector,
                "attribute": last_released_chapter_url_attribute,
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
