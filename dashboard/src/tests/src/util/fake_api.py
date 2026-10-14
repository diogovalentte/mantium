"""A fake API client for AppTest tests of dashboard forms. Tests configure the
module-level FAKE before running the app, and inspect FAKE.calls afterwards."""

import copy
import json
import os
from typing import Any
from unittest import mock

from src.exceptions import APIException


def user_error(message: str) -> APIException:
    """An API error that has a message meant to be shown to the user."""
    import json

    return APIException(
        "error", "http://fake", "POST", {}, 400, json.dumps({"message": message, "user_message": message})
    )


RSS_PRESET = {
    "id": 1,
    "name": "RSS feed",
    "description": "",
    "domains": [],
    "builtin": True,
    "use_browser": False,
    "name_selector": {"Selector": "xpath://item/title", "Attribute": "", "Regex": r"(\d+)", "GetFirst": True},
    "url_selector": {"Selector": "xpath://item/link", "Attribute": "", "Regex": "", "GetFirst": True},
}

SITE_PRESET = {
    "id": 2,
    "name": "My site",
    "description": "",
    "domains": ["example.com"],
    "builtin": False,
    "use_browser": True,
    "name_selector": {"Selector": "css:a.chapter", "Attribute": "", "Regex": r"Chapter (\d+)", "GetFirst": False},
    "url_selector": None,
}


CHAPTERS = [
    {"Chapter": "11", "URL": "https://mangadex.org/chapter/11", "InternalID": "c11", "UpdatedAt": "0001-01-01T00:00:00Z"},
    {"Chapter": "10", "URL": "https://mangadex.org/chapter/10", "InternalID": "c10", "UpdatedAt": "0001-01-01T00:00:00Z"},
]

_MANGA = {
    "ID": 5,
    "Name": "One Piece",
    "Source": "mangadex.org",
    "URL": "https://mangadex.org/title/one-piece",
    "InternalID": "op",
}

MULTIMANGA = {
    "ID": 1,
    "Status": 1,
    "CoverImgFixed": False,
    "CurrentManga": _MANGA,
    "Mangas": [_MANGA],
    "LastReadChapter": {"Chapter": "", "URL": "", "FromSourceSite": False},
}


class FakeAPI:
    def __init__(self) -> None:
        self.calls: list[tuple[str, tuple, dict]] = []
        self.presets: list[dict[str, Any]] = [copy.deepcopy(RSS_PRESET), copy.deepcopy(SITE_PRESET)]
        # URL -> detect response, or an exception to raise
        self.detections: dict[str, Any] = {}
        self.add_preset_error: Exception | None = None
        self.presets_error: Exception | None = None
        self.test_selectors_result: Any = {"Chapter": "10", "URL": "https://example.com/c/10"}
        self.updated_manga: dict[str, Any] | None = None
        with open(os.path.join(os.path.dirname(__file__), "library_fixture.json")) as f:
            fixture = json.load(f)
        # Library and configs read from a running API with the demo mangas, covers replaced by a 1x1 PNG
        self.library: list[dict[str, Any]] = fixture["mangas"]
        self.dashboard_configs: dict[str, Any] = fixture["configs"]
        self.background_error = {"message": "", "time": "0001-01-01 00:00:00"}
        self.last_update = "1"
        # Method name -> exception the method raises after being recorded
        self.errors: dict[str, Exception] = {}
        self.multimanga: dict[str, Any] = copy.deepcopy(MULTIMANGA)

    def _record(self, name: str, *args, **kwargs):
        self.calls.append((name, args, kwargs))
        if name in self.errors:
            raise self.errors[name]

    def called(self, name: str) -> list[tuple[tuple, dict]]:
        return [(a, k) for n, a, k in self.calls if n == name]

    def get_custom_manga_presets(self):
        self._record("get_custom_manga_presets")
        if self.presets_error is not None:
            raise self.presets_error
        return copy.deepcopy(self.presets)

    def detect_custom_manga_presets(self, manga_url: str):
        self._record("detect_custom_manga_presets", manga_url)
        result = self.detections.get(manga_url, {"domain_presets": [], "detections": []})
        if isinstance(result, Exception):
            raise result
        return copy.deepcopy(result)

    def add_custom_manga_preset(self, preset: dict[str, Any]):
        self._record("add_custom_manga_preset", preset)
        if self.add_preset_error is not None:
            raise self.add_preset_error
        saved = {
            "id": max(p["id"] for p in self.presets) + 1,
            "description": "",
            "builtin": False,
            "domains": preset.get("domains", []),
            "use_browser": preset.get("use_browser", False),
            "name": preset["name"],
            "name_selector": _api_selector(preset.get("name_selector")),
            "url_selector": _api_selector(preset.get("url_selector")),
        }
        self.presets.append(saved)
        return copy.deepcopy(saved)

    def delete_custom_manga_preset(self, preset_id: int):
        self._record("delete_custom_manga_preset", preset_id)
        self.presets = [p for p in self.presets if p["id"] != preset_id]
        return {"message": "ok"}

    def test_custom_manga_selectors(self, *args):
        self._record("test_custom_manga_selectors", *args)
        if isinstance(self.test_selectors_result, Exception):
            raise self.test_selectors_result
        return self.test_selectors_result

    def add_multimanga(self, *args):
        self._record("add_multimanga", *args)
        return {"message": "ok"}

    def add_manga_to_multimanga(self, *args):
        self._record("add_manga_to_multimanga", *args)
        return {"message": "ok"}

    def update_custom_manga(self, *args, **kwargs):
        self._record("update_custom_manga", *args, **kwargs)
        return {"message": "ok"}

    def get_multimanga(self, multimanga_id: int):
        self._record("get_multimanga", multimanga_id)
        return copy.deepcopy({**self.multimanga, "ID": multimanga_id})

    def choose_current_manga(self, multimanga_id: int, exclude_manga_ids=None):
        self._record("choose_current_manga", multimanga_id, exclude_manga_ids)
        return copy.deepcopy(self.multimanga["CurrentManga"])

    def get_cached_manga_chapters(self, *args):
        self._record("get_cached_manga_chapters", *args)
        return copy.deepcopy(CHAPTERS)

    def update_multimanga_status(self, *args):
        self._record("update_multimanga_status", *args)
        return {"message": "ok"}

    def update_multimanga_cover_img(self, *args, **kwargs):
        self._record("update_multimanga_cover_img", *args, **kwargs)
        return {"message": "ok"}

    def delete_multimanga(self, *args):
        self._record("delete_multimanga", *args)
        return {"message": "ok"}


    # Main page

    def check_health(self):
        return None

    def get_updated_message(self):
        return "", ""

    def check_for_updates(self):
        return self.last_update

    def get_dashboard_configs(self):
        return copy.deepcopy(self.dashboard_configs)

    def get_last_background_error(self):
        return copy.deepcopy(self.background_error)

    def delete_last_background_error(self):
        self._record("delete_last_background_error")

    def get_mangas(self):
        """Runs the real client's parsing on the fixture, so the page gets the same types."""
        from src.api.manga_api import MangaAPIClient

        self._record("get_mangas")
        response = mock.Mock(status_code=200)
        response.json.return_value = {"mangas": copy.deepcopy(self.library)}
        with mock.patch("src.api.manga_api.requests.get", return_value=response):
            return MangaAPIClient("http://fake").get_mangas()

    def sort_mangas(self, mangas, sort_option, reverse=False):
        from src.api.manga_api import MangaAPIClient

        return MangaAPIClient("http://fake").sort_mangas(mangas, sort_option, reverse)

    def update_multimanga_last_read_chapter(self, *args):
        self._record("update_multimanga_last_read_chapter", *args)
        return {"message": "ok"}


def _api_selector(selector: dict[str, Any] | None):
    if not selector:
        return None
    return {
        "Selector": selector["selector"],
        "Attribute": selector.get("attribute", ""),
        "Regex": selector.get("regex", ""),
        "GetFirst": selector.get("get_first", False),
    }


def detection(preset: dict[str, Any], chapter: str | None, url: str = "") -> dict[str, Any]:
    return {
        "preset": copy.deepcopy(preset),
        "chapter": None if chapter is None else {"Chapter": chapter, "URL": url},
        "user_message": "" if chapter is not None else "Chapter name selector didn't match any element on the page",
    }


FAKE = FakeAPI()
