from datetime import datetime, timezone
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

    def get_mangas(self) -> list[dict[str, Any]]:
        url = self.base_manga_url
        url = f"{url}s"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting mangas",
                url,
                "GET",
                {},
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
                    "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                    "URL": manga["URL"] if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE else "",
                }
            if manga["LastReadChapter"] is not None:
                manga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                    manga["LastReadChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReadChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                    "URL": manga["URL"] if manga["Source"] != defaults.CUSTOM_MANGA_SOURCE else "",
                    "FromSourceSite": False,
                }

            if manga["LastReleasedChapterNameSelector"] is None:
                manga["LastReleasedChapterNameSelector"] = {
                    "Selector": "",
                    "Attribute": "",
                    "Regex": "",
                    "GetFirst": False,
                }
            if manga["LastReleasedChapterURLSelector"] is None:
                manga["LastReleasedChapterURLSelector"] = {
                    "Selector": "",
                    "Attribute": "",
                    "GetFirst": False,
                }

        return mangas

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
                {},
                res.status_code,
                res.text,
            )

        chapters = res.json().get("chapters")
        if chapters is None:
            return []
        return chapters

    def search_mangas(self, term: str, limit: int, source: str) -> dict[str, str]:
        url = self.base_manga_url + "s/search"

        request_body = {
            "q": term,
            "limit": limit,
            "source": source,
        }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while searching manga",
                url,
                "POST",
                request_body,
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

            # If changing this logic, also change the is_unread_chapter function in util.py
            last_read_chapter = manga["LastReadChapter"]["Chapter"]
            last_released_chapter = manga["LastReleasedChapter"]["Chapter"]

            if last_read_chapter == last_released_chapter:
                return (1, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())
            elif last_released_chapter == "":
                return (1, -manga["LastReadChapter"]["UpdatedAt"].timestamp())

            try:
                last_read_chapter_number = float(last_read_chapter)
                last_released_chapter_number = float(last_released_chapter)
            except ValueError:
                return (0, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())

            if last_read_chapter_number < last_released_chapter_number:
                return (0, -manga["LastReleasedChapter"]["UpdatedAt"].timestamp())

            return (1, -manga["LastReadChapter"]["UpdatedAt"].timestamp())

        def chapters_released_sorting(manga: dict[str, Any]) -> float:
            chapter = manga["LastReleasedChapter"]["Chapter"]
            try:
                return float(chapter)
            except ValueError:
                return -float("inf")

        if sort_option == "Unread":
            mangas.sort(key=unread_sorting, reverse=reverse)
        elif sort_option == "Last Read":
            mangas.sort(
                key=lambda manga: (
                    manga["LastReadChapter"]["UpdatedAt"]
                    if manga["LastReadChapter"] is not None
                    else datetime.min.replace(tzinfo=timezone.utc)
                ),
                reverse=not reverse,
            )
        elif sort_option == "Released Chapter Date":
            mangas.sort(
                key=lambda manga: manga["LastReleasedChapter"]["UpdatedAt"],
                reverse=not reverse,
            )
        elif sort_option == "Name":
            mangas.sort(key=lambda manga: str(manga["Name"]).lower(), reverse=reverse)
        elif sort_option == "Chapters Released":
            mangas.sort(
                key=chapters_released_sorting,
                reverse=not reverse,
            )

        return mangas
