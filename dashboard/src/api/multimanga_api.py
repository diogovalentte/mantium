from datetime import datetime, timezone
from typing import Any
from urllib.parse import urljoin

import requests
import src.util.defaults as defaults
from src.exceptions import APIException
from src.util.util import get_updated_at_datetime


class MultiMangaAPIClient:
    def __init__(self, base_api_url: str) -> None:
        self.base_multimanga_url: str = urljoin(base_api_url, "/v1/multimanga")
        self.acceptable_status_codes: tuple = (200,)

    def add_multimanga(
        self,
        manga_url: str,
        manga_status: int,
        manga_internal_id: str,
        last_read_chapter: str,
        last_read_chapter_url: str,
        last_read_chapter_internal_id: str,
    ) -> dict[str, str]:
        url = self.base_multimanga_url

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
                "error while adding multimanga",
                url,
                "POST",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def get_multimanga(self, multimanga_id: int) -> dict[str, Any]:
        url = self.base_multimanga_url
        url = f"{url}?id={multimanga_id}"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting multimanga",
                url,
                "GET",
                {},
                res.status_code,
                res.text,
            )

        multimanga = res.json().get("multimanga")
        if multimanga["CoverImg"] is not None:
            multimanga["CoverImg"] = bytes(multimanga["CoverImg"], "utf-8")

        multimanga["CurrentManga"]["CoverImg"] = bytes(
            multimanga["CurrentManga"]["CoverImg"], "utf-8"
        )
        if multimanga["CurrentManga"]["LastReleasedChapter"] is not None:
            multimanga["CurrentManga"]["LastReleasedChapter"][
                "UpdatedAt"
            ] = get_updated_at_datetime(
                multimanga["CurrentManga"]["LastReleasedChapter"]["UpdatedAt"]
            )
        else:
            multimanga["CurrentManga"]["LastReleasedChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                "URL": multimanga["CurrentManga"]["URL"],
            }
        if multimanga["CurrentManga"]["LastReadChapter"] is not None:
            multimanga["CurrentManga"]["LastReadChapter"][
                "UpdatedAt"
            ] = get_updated_at_datetime(
                multimanga["CurrentManga"]["LastReadChapter"]["UpdatedAt"]
            )
        else:
            multimanga["CurrentManga"]["LastReadChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                "URL": multimanga["CurrentManga"]["URL"],
            }
        if multimanga["LastReadChapter"] is not None:
            multimanga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                multimanga["LastReadChapter"]["UpdatedAt"]
            )
        else:
            multimanga["LastReadChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                "URL": multimanga["CurrentManga"]["URL"],
            }

        for manga in multimanga["Mangas"]:
            manga["CoverImg"] = bytes(manga["CoverImg"], "utf-8")
            if manga["LastReleasedChapter"] is not None:
                manga["LastReleasedChapter"]["UpdatedAt"] = get_updated_at_datetime(
                    manga["LastReleasedChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReleasedChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                    "URL": manga["URL"],
                }
            if manga["LastReadChapter"] is not None:
                manga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                    manga["LastReadChapter"]["UpdatedAt"]
                )
            else:
                manga["LastReadChapter"] = {
                    "Chapter": "",
                    "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                    "URL": manga["URL"],
                }

        return multimanga

    def choose_current_manga(
        self, multimanga_id: int, exclude_manga_ids: list[int] = []
    ) -> dict[str, Any]:
        url = self.base_multimanga_url
        url = f"{url}/choose_current_manga?id={multimanga_id}"

        if exclude_manga_ids:
            url = f"{url}&exclude_manga_ids={','.join(map(str, exclude_manga_ids))}"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting choose current manga from multimanga",
                url,
                "GET",
                {},
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
                "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                "URL": manga["URL"],
            }
        if manga["LastReadChapter"] is not None:
            manga["LastReadChapter"]["UpdatedAt"] = get_updated_at_datetime(
                manga["LastReadChapter"]["UpdatedAt"]
            )
        else:
            manga["LastReadChapter"] = {
                "Chapter": "",
                "UpdatedAt": datetime.min.replace(tzinfo=timezone.utc),
                "URL": manga["URL"],
            }

        return manga

    def add_manga_to_multimanga(
        self,
        id: int,
        manga_url: str,
        manga_internal_id: str,
    ) -> dict[str, str]:
        url = self.base_multimanga_url + "/manga?id=" + str(id)

        request_body = {
            "manga_url": manga_url,
            "manga_internal_id": manga_internal_id,
        }

        res = requests.post(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while adding manga to multimanga",
                url,
                "POST",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def remove_manga_from_multimanga(self, id: int, manga_id: int) -> dict[str, str]:
        url = self.base_multimanga_url + "/manga"
        url = f"{url}?id={id}&manga_id={manga_id}"

        res = requests.delete(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while removing manga from multimanga",
                url,
                "DELETE",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def update_multimanga_status(
        self,
        status: int,
        multimanga_id: int,
    ) -> dict[str, str]:
        path = "/status"
        url = f"{self.base_multimanga_url}{path}"
        url = f"{url}?id={multimanga_id}"

        request_body = {
            "status": status,
        }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating multimanga status",
                url,
                "PATCH",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def update_multimanga_last_read_chapter(
        self,
        multimanga_id: int,
        manga_id: int,
        chapter: str = "",
        chapter_url: str = "",
        chapter_internal_id: str = "",
    ) -> dict[str, str]:
        path = "/last_read_chapter"
        url = f"{self.base_multimanga_url}{path}"
        url = f"{url}?id={multimanga_id}&manga_id={manga_id}"

        request_body = {
            "chapter": chapter,
            "chapter_url": chapter_url,
            "chapter_internal_id": chapter_internal_id,
        }

        res = requests.patch(url, json=request_body)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating multimanga last read chapter",
                url,
                "PATCH",
                request_body,
                res.status_code,
                res.text,
            )

        return res.json()

    def update_multimanga_cover_img(
        self,
        id: int,
        cover_img_url: str = "",
        cover_img: bytes = b"",
        use_current_manga_cover_img: bool = False,
    ) -> dict[str, str]:
        path = "/cover_img"
        url = f"{self.base_multimanga_url}{path}"
        url = f"{url}?id={id}{'&cover_img_url=%s' % cover_img_url if cover_img_url else ''}{f'&use_current_manga_cover_img={str(use_current_manga_cover_img).lower()}' if use_current_manga_cover_img else ''}"

        res = requests.patch(url, files={"cover_img": cover_img})

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while updating multimanga cover image",
                url,
                "PATCH",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def delete_multimanga(self, multimanga_id: int) -> dict[str, str]:
        url = self.base_multimanga_url
        url = f"{url}?id={multimanga_id}"

        res = requests.delete(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while deleting multimanga",
                url,
                "DELETE",
                {},
                res.status_code,
                res.text,
            )

        return res.json()

    def get_multimanga_chapters(self, multimanga_id: int, manga_id: int) -> list[dict]:
        path = "/chapters"
        url = f"{self.base_multimanga_url}{path}"
        url = f"{url}?id={multimanga_id}&manga_id={manga_id}"

        res = requests.get(url)

        if res.status_code not in self.acceptable_status_codes:
            raise APIException(
                "error while getting multimanga chapters",
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
