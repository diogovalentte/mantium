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
        mangas_has_more_chapters: bool,
        cover_img_url: str,
        cover_img: bytes,
        next_chapter_chapter: str,
        next_chapter_url: str,
    ) -> dict[str, str]:
        url = self.base_custom_manga_url

        request_body = {
            "name": manga_name,
            "url": manga_url,
            "status": manga_status,
            "mangas_has_more_chapters": mangas_has_more_chapters,
            "cover_img_url": cover_img_url,
            "cover_img": cover_img,
        }

        if next_chapter_chapter:
            request_body["next_chapter"] = {
                "chapter": next_chapter_chapter,
                "url": next_chapter_url,
            }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while adding custom manga",
                url,
                "POST",
                res.status_code,
                res.text,
            )

        return res.json()

    def update_custom_manga_has_more_chapters(
        self,
        has_more_chapters: bool,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/has_more_chapters"
        url = f"{self.base_custom_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&has_more_chapters={str(has_more_chapters).lower()}"

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating custom manga has more chapters property",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()
