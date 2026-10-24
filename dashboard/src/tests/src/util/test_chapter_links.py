import os
import re

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.test_custom_manga_form import assert_no_exception
from src.util.util import chapter_link

DASHBOARD_RUNNER = os.path.join(os.path.dirname(__file__), "dashboard_runner.py")
MANAGE_RUNNER = os.path.join(os.path.dirname(__file__), "manage_mangas_runner.py")
MANGA_URL = "https://example.com/manga/one"


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def chapter_tags(at: AppTest) -> str:
    """The HTML of the chapter tags, which are the only links styled like this."""
    return "\n".join(m.value for m in at.markdown if "text-decoration: none" in m.value)


@pytest.mark.parametrize(
    "chapter, manga, want",
    [
        ({"URL": "https://example.com/c/1"}, {"URL": MANGA_URL}, "https://example.com/c/1"),
        ({"URL": ""}, {"URL": MANGA_URL}, MANGA_URL),
        ({"URL": ""}, {"URL": ""}, ""),
        ({}, {}, ""),
    ],
)
def test_chapter_link(chapter, manga, want):
    assert chapter_link(chapter, manga) == want


def custom_mangas_without_chapter_url(api) -> list[dict]:
    return [
        m for m in api.library
        if m["Source"] == "custom_manga"
        and m.get("LastReleasedChapter")
        and m["LastReleasedChapter"]["URL"] == ""
        and m["URL"].startswith("http")
    ]


@pytest.mark.parametrize("display_mode", ["Grid View", "List View"])
def test_chapters_without_url_link_to_the_manga(api, display_mode):
    api.dashboard_configs["display"]["displayMode"] = display_mode
    mangas = custom_mangas_without_chapter_url(api)
    assert mangas, "the fixture should have custom mangas tracked by the chapter name only"
    at = AppTest.from_file(DASHBOARD_RUNNER, default_timeout=60)
    at.session_state["status_filter"] = 0
    at.run()
    assert_no_exception(at)
    tags = chapter_tags(at)
    assert not re.search(r'href="\s*"', tags)
    for manga in mangas:
        assert f'href="{manga["URL"]}"' in tags, manga["Name"]


def test_chapter_and_name_without_any_url_are_text_in_manage_mangas(api):
    library = api.get_mangas()
    manga = dict(library[0])
    manga["ID"] = 10
    manga["URL"] = ""
    manga["LastReleasedChapter"] = {**manga["LastReleasedChapter"], "Chapter": "71", "URL": ""}
    linked = dict(library[1])
    linked["ID"] = 11
    linked["URL"] = MANGA_URL
    linked["LastReleasedChapter"] = {**linked["LastReleasedChapter"], "Chapter": "72", "URL": ""}

    at = AppTest.from_file(MANAGE_RUNNER, default_timeout=30)
    at.session_state["mangas"] = [manga, linked]
    at.run()
    assert_no_exception(at)
    html = "\n".join(m.value for m in at.markdown)
    assert not re.search(r'href="\s*"', html)
    assert "Ch. 71" in html
    assert f'href="{MANGA_URL}"' in chapter_tags(at)
