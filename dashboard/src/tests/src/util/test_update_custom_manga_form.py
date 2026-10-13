import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.fake_api import RSS_PRESET, SITE_PRESET, detection
from src.tests.src.util.test_custom_manga_form import assert_no_exception, expanded, field, labels_in_order, texts

RUNNER = os.path.join(os.path.dirname(__file__), "update_custom_manga_form_runner.py")
P = "update_custom_manga_form_"
FEED_URL = "https://feeds.example.org/series/1/rss"


def manga(**changes):
    m = {
        "ID": 42,
        "Name": "Tracked",
        "URL": "https://example.com/manga/tracked",
        "LastReleasedChapterNameSelector": {"Selector": "css:a.ch", "Attribute": "", "Regex": r"(\d+)", "GetFirst": True},
        "LastReleasedChapterURLSelector": {"Selector": "css:a.ch", "Attribute": "href", "Regex": "", "GetFirst": True},
        "LastReleasedChapterSelectorUseBrowser": True,
    }
    m.update(changes)
    return m


UNTRACKED = dict(
    Name="Untracked",
    URL="https://example.com/manga/untracked",
    LastReleasedChapterNameSelector={"Selector": "", "Attribute": "", "Regex": "", "GetFirst": False},
    LastReleasedChapterURLSelector={"Selector": "", "Attribute": "", "Regex": "", "GetFirst": False},
    LastReleasedChapterSelectorUseBrowser=False,
)


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def run_app(m) -> AppTest:
    at = AppTest.from_file(RUNNER, default_timeout=30)
    at.session_state["manga"] = m
    return at.run()


def update_args(api) -> tuple:
    (args, _), = api.called("update_custom_manga")
    return args


def test_prefilled_from_tracked_manga(api):
    at = run_app(manga())
    assert_no_exception(at)
    labels = labels_in_order(at)
    assert labels.index("Cover Image") < labels.index("Last Released Chapter") < labels.index("Update")
    # The selectors of a manga that already tracks new chapters start open
    assert expanded(at, P)
    assert at.text_input(key=P + "name").value == "Tracked"
    assert at.text_input(key=P + "url").value == "https://example.com/manga/tracked"
    assert field(at, "name_selector", P) == "css:a.ch"
    assert field(at, "url_attribute", P) == "href"
    assert at.checkbox(key=P + "last_released_chapter_use_browser").value is True
    # No warning about value= and session_state used together
    assert not texts(at, "warning")


def test_untracked_manga_starts_with_the_selectors_closed(api):
    at = run_app(manga(**UNTRACKED))
    assert_no_exception(at)
    assert not expanded(at, P)


def test_update_without_changes_sends_current_values(api):
    at = run_app(manga())
    at.button(key=P + "update").click().run()
    assert_no_exception(at)
    args = update_args(api)
    assert args[:3] == (42, "Tracked", "https://example.com/manga/tracked")
    assert args[3:11] == ("css:a.ch", "", r"(\d+)", True, "css:a.ch", "href", True, True)


def test_clear_removes_the_selectors(api):
    at = run_app(manga())
    at.button(key=P + "clear_selectors_button").click().run()
    at.button(key=P + "update").click().run()
    assert_no_exception(at)
    assert update_args(api)[3:11] == ("", "", "", False, "", "", False, False)


def test_url_change_keeps_existing_selectors(api):
    api.detections[FEED_URL] = {"domain_presets": [], "detections": [detection(RSS_PRESET, "3")]}
    at = run_app(manga())
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert field(at, "name_selector", P) == "css:a.ch"
    assert any("Presets that work" in m.value for m in at.markdown)
    at.button(key=P + "update").click().run()
    assert update_args(api)[2:4] == (FEED_URL, "css:a.ch")


def test_url_change_on_untracked_manga_applies_preset(api):
    api.detections[FEED_URL] = {"domain_presets": [], "detections": [detection(RSS_PRESET, "3")]}
    at = run_app(manga(**UNTRACKED))
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert expanded(at, P)
    assert field(at, "name_selector", P) == "xpath://item/title"
    assert not texts(at, "warning")


def test_picking_a_preset_replaces_prefilled_selectors(api):
    at = run_app(manga())
    at.selectbox(key=P + "preset").set_value(SITE_PRESET["id"]).run()
    assert_no_exception(at)
    assert field(at, "name_selector", P) == "css:a.chapter"
    assert field(at, "url_selector", P) == ""
    assert not texts(at, "warning")


def test_reopening_for_another_manga_loads_its_values(api):
    at = run_app(manga())
    at.text_input(key=P + "name").input("Edited but not saved").run()
    at.session_state["manga"] = None
    at.run()
    at.session_state["manga"] = manga(ID=7, **UNTRACKED)
    at.run()
    assert_no_exception(at)
    assert at.text_input(key=P + "name").value == "Untracked"
    assert not expanded(at, P)
    at.button(key=P + "update").click().run()
    assert update_args(api)[0] == 7


def test_update_requires_name(api):
    at = run_app(manga())
    at.text_input(key=P + "name").input("").run()
    at.button(key=P + "update").click().run()
    assert "Provide a manga name" in texts(at, "warning")
    assert api.called("update_custom_manga") == []


def test_closing_the_selectors_keeps_them(api):
    at = run_app(manga())
    at.text_input(key=P + "last_released_chapter_name_regex").input(r"Ch (\d+)").run()
    at.session_state[P + "last_released_chapter_expander"] = True
    at.run()
    at.session_state[P + "last_released_chapter_expander"] = False
    at.run()
    assert_no_exception(at)
    assert field(at, "name_selector", P) == "css:a.ch"
    assert at.checkbox(key=P + "last_released_chapter_use_browser").value is True
    at.button(key=P + "update").click().run()
    assert update_args(api)[3:6] == ("css:a.ch", "", r"Ch (\d+)")


def test_clear_works_when_the_presets_fail_to_load(api):
    api.presets_error = fake_api.user_error("API down")
    at = run_app(manga())
    assert_no_exception(at)
    assert any("Couldn't load the presets" in t for t in texts(at, "warning"))
    at.button(key=P + "clear_selectors_button").click().run()
    at.button(key=P + "update").click().run()
    assert_no_exception(at)
    assert update_args(api)[3:11] == ("", "", "", False, "", "", False, False)


def test_url_inputs_are_validated_in_the_browser(api):
    at = run_app(manga())
    for key in ("url", "cover_img_url"):
        widget = at.text_input(key=P + key)
        assert widget.proto.type == widget.proto.URL, key
        assert widget.proto.validate_regex, key
