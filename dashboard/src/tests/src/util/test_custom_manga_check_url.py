import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.test_custom_manga_form import assert_no_exception, texts
from src.tests.src.util.test_update_custom_manga_form import manga

ADD_RUNNER = os.path.join(os.path.dirname(__file__), "custom_manga_form_runner.py")
UPDATE_RUNNER = os.path.join(os.path.dirname(__file__), "update_custom_manga_form_runner.py")
ADD = "add_custom_manga_form_"
MM = "add_custom_manga_to_multimanga_form_"
UPDATE = "update_custom_manga_form_"
MANGA_URL = "https://example.com/manga/one"
FEED_URL = "https://feeds.example.org/series/1/rss"


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def run_add(**session_state) -> AppTest:
    at = AppTest.from_file(ADD_RUNNER, default_timeout=30)
    for key, value in session_state.items():
        at.session_state[key] = value
    return at.run()


def run_update(m) -> AppTest:
    at = AppTest.from_file(UPDATE_RUNNER, default_timeout=30)
    at.session_state["manga"] = m
    return at.run()


def test_add_sends_the_check_url(api):
    at = run_add()
    at.text_input(key=ADD + "name").input("Manga").run()
    at.text_input(key=ADD + "url").input(MANGA_URL).run()
    at.text_input(key=ADD + "check_url").input(FEED_URL).run()
    at.button(key=ADD + "add").click().run()
    assert_no_exception(at)
    (args, kwargs), = api.called("add_multimanga")
    assert args[1] == MANGA_URL
    assert kwargs == {"check_url": FEED_URL}


def test_add_to_multimanga_sends_the_check_url(api):
    at = run_add(multimanga_id=7)
    at.text_input(key=MM + "name").input("Other source").run()
    at.text_input(key=MM + "check_url").input(FEED_URL).run()
    at.button(key=MM + "add").click().run()
    assert_no_exception(at)
    (_, kwargs), = api.called("add_manga_to_multimanga")
    assert kwargs == {"check_url": FEED_URL}


def test_invalid_check_url_is_reported_on_add(api):
    at = run_add()
    at.text_input(key=ADD + "name").input("Manga").run()
    at.session_state[ADD + "check_url"] = "not a url"
    at.run()
    at.button(key=ADD + "add").click().run()
    assert_no_exception(at)
    assert api.called("add_multimanga") == []
    assert any("check URL" in t for t in texts(at, "warning"))


def test_presets_are_detected_on_the_check_url(api):
    at = run_add()
    at.text_input(key=ADD + "url").input(MANGA_URL).run()
    at.text_input(key=ADD + "check_url").input(FEED_URL).run()
    assert_no_exception(at)
    urls = [args[0] for args, _ in api.called("detect_custom_manga_presets")]
    assert urls == [MANGA_URL, FEED_URL]
    assert at.text_input(key=ADD + "new_preset_domains").value == "feeds.example.org"


def test_test_selectors_with_only_the_check_url(api):
    at = run_add()
    at.text_input(key=ADD + "check_url").input(FEED_URL).run()
    at.text_input(key=ADD + "last_released_chapter_name_selector").input("xpath://item/title").run()
    at.button(key=ADD + "test_selectors").click().run()
    assert_no_exception(at)
    (args, kwargs), = api.called("test_custom_manga_selectors")
    assert args[0] == "" and kwargs == {"check_url": FEED_URL}


def test_test_selectors_without_any_url(api):
    at = run_add()
    at.text_input(key=ADD + "last_released_chapter_name_selector").input("css:a").run()
    at.button(key=ADD + "test_selectors").click().run()
    assert_no_exception(at)
    assert api.called("test_custom_manga_selectors") == []
    assert "Provide the manga URL or the check URL to test the selectors" in texts(at, "warning")


def test_update_form_is_prefilled_and_sends_the_check_url(api):
    at = run_update(manga(CheckURL=FEED_URL))
    assert_no_exception(at)
    assert at.text_input(key=UPDATE + "check_url").value == FEED_URL
    assert at.text_input(key=UPDATE + "new_preset_domains").value == "feeds.example.org"
    at.text_input(key=UPDATE + "check_url").input(FEED_URL + "?page=2").run()
    at.button(key=UPDATE + "update").click().run()
    assert_no_exception(at)
    (_, kwargs), = api.called("update_custom_manga")
    assert kwargs == {"check_url": FEED_URL + "?page=2"}


def test_update_of_a_manga_without_check_url(api):
    at = run_update(manga())
    at.button(key=UPDATE + "update").click().run()
    assert_no_exception(at)
    (_, kwargs), = api.called("update_custom_manga")
    assert kwargs == {"check_url": ""}
