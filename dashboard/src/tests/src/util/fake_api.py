"""A fake API client for AppTest tests of dashboard forms. Tests configure the
module-level FAKE before running the app, and inspect FAKE.calls afterwards."""

import copy
from typing import Any

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

    def _record(self, name: str, *args, **kwargs):
        self.calls.append((name, args, kwargs))

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
