from datetime import datetime
from typing import Any
from urllib.parse import urljoin

import requests
import src.util.defaults as defaults
from src.exceptions import APIException
from src.util.util import get_updated_at_datetime


class MangaAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_manga_url: str = urljoin(base_api_url, "/v1/manga")
        self.acceptable_status_codes: tuple = (200,)

    def add_manga(
        self,
        manga_url: str,
        manga_status: int,
        manga_internal_id: str,
        last_read_chapter: str,
        last_read_chapter_url: str,
        last_read_chapter_internal_id: str,
    ) -> dict[str, str]:
        url = self.base_manga_url

        request_body = {
            "url": manga_url,
            "status": manga_status,
            "manga_internal_id": manga_internal_id,
            "last_read_chapter": last_read_chapter,
            "last_read_chapter_url": last_read_chapter_url,
            "last_read_chapter_internal_id": last_read_chapter_internal_id,
        }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while adding manga",
                url,
                "POST",
                res.status_code,
                res.text,
            )

        return res.json()

    def get_manga(self, manga_id: int = 0, manga_url: str = "") -> dict[str, Any]:
        url = self.base_manga_url
        url = f"{url}?id={manga_id}&url={manga_url}"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting manga",
                url,
                "GET",
                res.status_code,
                res.text,
            )

        manga = res.json().get("manga")
        manga["CoverImg"] = bytes(manga["CoverImg"], "utf-8")

        if manga["LastReleasedChapter"] is not None:
            manga["LastReleasedChapter"]["UpdatedAt"] = get_updated_at_datetime(
                manga["LastReleasedChapter"]["UpdatedAt"]
            )
        else:
            manga["LastReleasedChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime(1970, 1, 1),
                "URL": manga["URL"]
                if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE
                else "",
            }
        if manga["LastReadChapter"] is not None:
            manga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                manga["LastReadChapter"]["UpdatedAt"]
            )
        else:
            manga["LastReadChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime(1970, 1, 1),
                "URL": manga["URL"]
                if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE
                else "",
            }

        return manga

    def get_mangas(self) -> list[dict[str, Any]]:
        url = self.base_manga_url
        url = f"{url}s"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting mangas",
                url,
                "GET",
                res.status_code,
                res.text,
            )

        mangas = res.json().get("mangas")
        if mangas is None:
            return []
        for manga in mangas:
            manga["CoverImg"] = bytes(manga["CoverImg"], "utf-8")

            if manga["LastReleasedChapter"] is not None:
                manga["LastReleasedChapter"]["UpdatedAt"] = get_updated_at_datetime(
                    manga["LastReleasedChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReleasedChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime(1970, 1, 1),
                    "URL": manga["URL"]
                    if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE
                    else "",
                }
            if manga["LastReadChapter"] is not None:
                manga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                    manga["LastReadChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReadChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime(1970, 1, 1),
                    "URL": manga["URL"]
                    if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE
                    else "",
                }

        return mangas

    def update_manga_status(
        self,
        status: int,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/status"
        url = f"{self.base_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}"

        request_body = {
            "status": status,
        }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating manga status",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()

    def update_manga_name(
        self,
        name: str,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/name"
        url = f"{self.base_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&name={name}"

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating manga name",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()

    def update_manga_url(
        self,
        new_url: str,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/url"
        url = f"{self.base_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&new_url={new_url}"

        res = requests.patch(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating manga URL",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()

    def update_manga_last_read_chapter(
        self,
        manga_id: int = 0,
        manga_url: str = "",
        manga_internal_id: str = "",
        chapter: str = "",
        chapter_url: str = "",
        chapter_internal_id: str = "",
    ) -> dict[str, str]:
        path = "/last_read_chapter"
        url = f"{self.base_manga_url}{path}"
        url = (
            f"{url}?id={manga_id}&url={manga_url}&manga_internal_id={manga_internal_id}"
        )

        request_body = {
            "chapter": chapter,
            "chapter_url": chapter_url,
            "chapter_internal_id": chapter_internal_id,
        }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating manga last read chapter",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()

    def update_manga_cover_img(
        self,
        manga_id: int = 0,
        manga_url: str = "",
        manga_internal_id: str = "",
        cover_img_url: str = "",
        cover_img: bytes = b"",
        get_cover_img_from_source: bool = False,
        use_mantium_default_img: bool = False,
    ) -> dict[str, str]:
        path = "/cover_img"
        url = f"{self.base_manga_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}&manga_internal_id={manga_internal_id}&{'&cover_img_url=%s' % cover_img_url if cover_img_url else ''}{f'&get_cover_img_from_source={str(get_cover_img_from_source).lower()}' if get_cover_img_from_source else ''}{'&use_mantium_default_img=%s' % str(use_mantium_default_img).lower() if use_mantium_default_img else ''}"

        res = requests.patch(url, files={"cover_img": cover_img})

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating manga cover image",
                url,
                "PATCH",
                res.status_code,
                res.text,
            )

        return res.json()

    def turn_manga_into_multimanga(self, manga_id: int = 0) -> dict[str, str]:
        url = self.base_manga_url + "/turn_into_multimanga"
        url = f"{url}?id={manga_id}"

        res = requests.post(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while turning manga into multimanga",
                url,
                "POST",
                res.status_code,
                res.text,
            )

        return res.json()

    def delete_manga(self, manga_id: int = 0, manga_url: str = "") -> dict[str, str]:
        url = self.base_manga_url
        url = f"{url}?id={manga_id}&url={manga_url}"

        res = requests.delete(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while deleting manga",
                url,
                "DELETE",
                res.status_code,
                res.text,
            )

        return res.json()

    def get_manga_chapters(
        self, manga_id: int = 0, manga_url: str = "", manga_internal_id: str = ""
    ) -> list[dict]:
        path = "/chapters"
        url = f"{self.base_manga_url}{path}"
        url = (
            f"{url}?id={manga_id}&url={manga_url}&manga_internal_id={manga_internal_id}"
        )

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting manga chapters",
                url,
                "GET",
                res.status_code,
                res.text,
            )

        chapters = res.json().get("chapters")
        if chapters is None:
            return []
        return chapters

    def search_mangas(
        self, term: str, limit: int, source_site_url: str
    ) -> dict[str, str]:
        url = self.base_manga_url + "s/search"

        request_body = {
            "q": term,
            "limit": limit,
            "source_url": source_site_url,
        }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while searching manga",
                url,
                "POST",
                res.status_code,
                res.text,
            )

        return res.json()["mangas"]

    def sort_mangas(
        self, mangas: list[dict[str, Any]], sort_option: str, reverse: bool = False
    ) -> list[dict[str, Any]]:
        def unread_sorting(manga: dict[str, Any]) -> tuple[int, Any]:
            """Sort mangas by unread chapters.
            Define two priority groups:
                0 = Unread chapters, sorted by last chapter release date (desc).
                1 = Read chapters, sorted by last chapter release date (desc).
            Unread chapters are prioritized over read chapters."""
            if manga["LastReadChapter"] is not None:
                if (
                    manga["LastReadChapter"]["Chapter"]
                    != manga["LastReleasedChapter"]["Chapter"]
                ):
                    return (0, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())
                else:
                    return (1, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())
            else:
                return (0, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())

        def chapters_released_sorting(manga: dict[str, Any]) -> float:
            chapter = manga["LastReleasedChapter"]["Chapter"]
            if chapter.isdigit():
                return float(chapter)
            else:
                # Assign a very large number as a placeholder for non-numeric chapter names
                # This ensures that they appear after numeric chapters
                return -float("inf")

        if sort_option == "Unread":
            mangas.sort(key=unread_sorting, reverse=reverse)
        elif sort_option == "Last Read":
            mangas.sort(
                key=lambda manga: manga["LastReadChapter"]["UpdatedAt"]
                if manga["LastReadChapter"] is not None
                else datetime.min,
                reverse=not reverse,
            )
        elif sort_option == "Released Chapter Date":
            mangas.sort(
                key=lambda manga: manga["LastReleasedChapter"]["UpdatedAt"],
                reverse=not reverse,
            )
        elif sort_option == "Name":
            mangas.sort(key=lambda manga: manga["Name"], reverse=reverse)
        elif sort_option == "Chapters Released":
            mangas.sort(
                key=chapters_released_sorting,
                reverse=not reverse,
            )

        return mangas
