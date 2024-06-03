from datetime import datetime
from typing import Any
from urllib.parse import urljoin

import requests
from src.exceptions import APIException


class MangaAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_api_url: str = (
            base_api_url  # The base URL of the API, e.g. http://localhost:8080
        )
        self.base_url: str = urljoin(
            self.base_api_url, "/v1/manga"
        )  # The base URL to be used by this client
        self.acceptable_status_codes: tuple = (
            200,  # The acceptable status codes from the API requests
        )

    def add_manga(
        self,
        manga_url: str,
        manga_status: int,
        last_read_chapter: str,
        last_read_chapter_url: str,
    ) -> dict[str, str]:
        url = self.base_url

        request_body = {
            "url": manga_url,
            "status": manga_status,
            "last_read_chapter": last_read_chapter,
            "last_read_chapter_url": last_read_chapter_url,
            "manga_has_no_chapters": False,
        }

        if last_read_chapter == "":
            del request_body["last_read_chapter"]
        if last_read_chapter_url == "":
            del request_body["last_read_chapter_url"]
        if last_read_chapter == "" and last_read_chapter_url == "":
            request_body["manga_has_no_chapters"] = True

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
        url = self.base_url
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
            manga["LastReleasedChapter"]["UpdatedAt"] = self.get_updated_at_datetime(
                manga["LastReleasedChapter"]["UpdatedAt"]
            )
        else:
            manga["LastReleasedChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime(1970, 1, 1),
                "URL": manga["URL"],
            }
        if manga["LastReadChapter"] is not None:
            manga["LastReadChapter"]["UpdatedAt"] = self.get_updated_at_datetime(
                manga["LastReadChapter"]["UpdatedAt"]
            )
        else:
            manga["LastReadChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime(1970, 1, 1),
                "URL": manga["URL"],
            }

        return manga

    def get_mangas(self) -> list[dict[str, Any]]:
        url = self.base_url
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
                manga["LastReleasedChapter"][
                    "UpdatedAt"
                ] = self.get_updated_at_datetime(
                    manga["LastReleasedChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReleasedChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime(1970, 1, 1),
                    "URL": manga["URL"],
                }
            if manga["LastReadChapter"] is not None:
                manga["LastReadChapter"]["UpdatedAt"] = self.get_updated_at_datetime(
                    manga["LastReadChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReadChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime(1970, 1, 1),
                    "URL": manga["URL"],
                }

        return mangas

    def update_manga_status(
        self,
        status: int,
        manga_id: int = 0,
        manga_url: str = "",
    ) -> dict[str, str]:
        path = "/status"
        url = f"{self.base_url}{path}"
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

    def update_manga_last_read_chapter(
        self,
        manga_id: int,
        manga_url: str,
        chapter: str = "",
        chapter_url: str = "",
    ) -> dict[str, str]:
        path = "/last_read_chapter"
        url = f"{self.base_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}"

        request_body = {
            "chapter": chapter,
            "chapter_url": chapter_url,
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
        manga_id: int,
        manga_url: str,
        cover_img_url: str = "",
        cover_img: bytes = b"",
        get_cover_img_from_source: bool = False,
    ) -> dict[str, str]:
        path = "/cover_img"
        url = f"{self.base_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}{'&cover_img_url=%s' % cover_img_url if cover_img_url else ''}{f'&get_cover_img_from_source={str(get_cover_img_from_source).lower()}' if get_cover_img_from_source else ''}"

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

    def delete_manga(self, manga_id: int = 0, manga_url: str = "") -> dict[str, str]:
        url = self.base_url
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

    def get_manga_chapters(self, manga_id: int = 0, manga_url: str = "") -> list[dict]:
        path = "/chapters"
        url = f"{self.base_url}{path}"
        url = f"{url}?id={manga_id}&url={manga_url}"

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

    def get_updated_at_datetime(self, updated_at: str) -> datetime:
        updated_at = self.remove_nano_from_datetime(updated_at)
        return datetime.strptime(updated_at, "%Y-%m-%dT%H:%M:%SZ")

    def remove_nano_from_datetime(self, datetime_string: str):
        if len(datetime_string) > 19:
            return datetime_string[:19] + "Z"
        else:
            return datetime_string

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
