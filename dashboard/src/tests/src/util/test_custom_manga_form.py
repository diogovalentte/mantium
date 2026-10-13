import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.fake_api import RSS_PRESET, SITE_PRESET, detection, user_error

RUNNER = os.path.join(os.path.dirname(__file__), "custom_manga_form_runner.py")
P = "add_custom_manga_form_"
MM = "add_custom_manga_to_multimanga_form_"
FEED_URL = "https://feeds.example.org/series/1/rss"
SITE_URL = "https://example.com/manga/one"


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def run_app(**session_state) -> AppTest:
    at = AppTest.from_file(RUNNER, default_timeout=30)
    for key, value in session_state.items():
        at.session_state[key] = value
    return at.run()


def labels_in_order(at: AppTest) -> list[str]:
    labels = []

    def walk(node):
        label = getattr(node, "label", None)
        if label and type(node).__name__ in ("TextInput", "Selectbox", "Expander", "Toggle", "Button", "FileUploader", "Checkbox", "Popover"):
            labels.append(label)
        for child in getattr(node, "children", {}).values():
            walk(child)

    walk(at.main)
    return labels


def texts(at: AppTest, kind: str) -> list[str]:
    return [str(e.value) for e in getattr(at, kind)]


def expanded(at: AppTest, prefix: str = P) -> bool:
    """Whether the Last Released Chapter expander is open."""
    return bool(at.session_state[prefix + "last_released_chapter_expander"])


def field(at: AppTest, name: str, prefix: str = P):
    return at.text_input(key=prefix + "last_released_chapter_" + name).value


def assert_no_exception(at: AppTest):
    assert not at.exception, [e.value for e in at.exception]


def feed_detected(api, chapter: str | None = "88"):
    api.detections[FEED_URL] = {"domain_presets": [], "detections": [detection(RSS_PRESET, chapter)]}


def test_layout_order(api):
    at = run_app()
    assert_no_exception(at)
    labels = labels_in_order(at)
    order = ["Manga Name", "Manga URL", "Status", "Cover Image", "Last Read Chapter", "Last Released Chapter", "Add"]
    seen = [l for l in labels if l in order]
    # "Last Read Chapter" is both the expander and the field inside it
    assert [l for i, l in enumerate(seen) if i == 0 or l != seen[i - 1]] == order
    assert not expanded(at)


def test_url_with_working_preset_opens_the_selectors_and_fills_them(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert expanded(at)
    assert at.selectbox(key=P + "preset").value == RSS_PRESET["id"]
    assert field(at, "name_selector") == "xpath://item/title"
    assert field(at, "name_regex") == r"(\d+)"
    assert field(at, "url_selector") == "xpath://item/link"
    assert at.checkbox(key=P + "last_released_chapter_name_get_first").value is True
    assert any('Preset "RSS feed" selected' in t for t in texts(at, "success"))
    assert any("Presets that work" in m.value for m in at.markdown)
    # Only one detection per URL change, not one per rerun
    at.run()
    assert len(api.called("detect_custom_manga_presets")) == 1


def test_domain_preset_wins_over_working_preset(api):
    api.detections[SITE_URL] = {"domain_presets": [SITE_PRESET], "detections": [detection(RSS_PRESET, "5"), detection(SITE_PRESET, "6")]}
    at = run_app()
    at.text_input(key=P + "url").input(SITE_URL).run()
    assert_no_exception(at)
    assert at.selectbox(key=P + "preset").value == SITE_PRESET["id"]
    assert at.checkbox(key=P + "last_released_chapter_use_browser").value is True
    # The preset has no URL selector, so the URL fields are cleared
    assert field(at, "url_selector") == ""


def test_url_change_does_not_overwrite_filled_selectors(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "last_released_chapter_name_selector").input("css:h1").run()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert field(at, "name_selector") == "css:h1"
    assert at.selectbox(key=P + "preset").value is None
    assert any("Presets that work" in m.value for m in at.markdown)


def test_no_preset_works_keeps_the_selectors_closed(api):
    feed_detected(api, chapter=None)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert not expanded(at)
    assert "No preset works for this page" in texts(at, "info")


def test_detection_error_is_shown(api):
    api.detections[FEED_URL] = user_error("Couldn't open the manga URL: Not Found")
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert not expanded(at)
    assert any("Not Found" in t for t in texts(at, "warning"))


def test_invalid_or_empty_url_does_not_call_detection(api):
    at = run_app()
    at.text_input(key=P + "url").input("not a url").run()
    at.text_input(key=P + "url").input("").run()
    assert_no_exception(at)
    assert api.called("detect_custom_manga_presets") == []


def test_picking_presets_by_hand(api):
    at = run_app()
    at.selectbox(key=P + "preset").set_value(SITE_PRESET["id"]).run()
    assert_no_exception(at)
    assert field(at, "name_selector") == "css:a.chapter"
    at.selectbox(key=P + "preset").set_value(RSS_PRESET["id"]).run()
    assert field(at, "name_selector") == "xpath://item/title"
    assert at.checkbox(key=P + "last_released_chapter_use_browser").value is False
    # Going back to None keeps what was filled
    at.selectbox(key=P + "preset").set_value(None).run()
    assert_no_exception(at)
    assert field(at, "name_selector") == "xpath://item/title"
    at.button(key=P + "clear_selectors_button").click().run()
    assert_no_exception(at)
    assert field(at, "name_selector") == ""
    assert field(at, "url_selector") == ""
    assert at.checkbox(key=P + "last_released_chapter_name_get_first").value is False


def test_detect_button_runs_detection_again(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    at.button(key=P + "detect_presets_button").click().run()
    assert_no_exception(at)
    assert len(api.called("detect_custom_manga_presets")) == 2


def test_save_preset(api):
    at = run_app()
    at.text_input(key=P + "url").input(SITE_URL).run()
    at.text_input(key=P + "last_released_chapter_name_selector").input("css:li a").run()
    at.text_input(key=P + "last_released_chapter_name_regex").input(r"(\d+)").run()
    assert at.text_input(key=P + "new_preset_domains").value == "example.com"
    at.text_input(key=P + "new_preset_name").input("Example").run()
    at.button(key=P + "save_preset").click().run()
    assert_no_exception(at)
    (preset,), _ = api.called("add_custom_manga_preset")[0]
    assert preset["name"] == "Example"
    assert preset["domains"] == ["example.com"]
    assert preset["name_selector"]["selector"] == "css:li a"
    assert "url_selector" not in preset
    assert any('Preset "Example" saved' in t for t in texts(at, "success"))
    assert at.selectbox(key=P + "preset").value == 3


def test_save_preset_error(api):
    api.add_preset_error = user_error('A preset named "RSS feed" already exists')
    at = run_app()
    at.text_input(key=P + "new_preset_name").input("RSS feed").run()
    at.button(key=P + "save_preset").click().run()
    assert_no_exception(at)
    assert any("already exists" in t for t in texts(at, "warning"))


def test_delete_user_preset(api):
    at = run_app()
    at.selectbox(key=P + "preset").set_value(SITE_PRESET["id"]).run()
    at.button(key=P + "delete_preset").click().run()
    assert_no_exception(at)
    assert api.called("delete_custom_manga_preset") == [((SITE_PRESET["id"],), {})]
    assert at.selectbox(key=P + "preset").value is None
    assert SITE_PRESET["id"] not in at.selectbox(key=P + "preset").options
    # Builtin presets have no delete button
    at.selectbox(key=P + "preset").set_value(RSS_PRESET["id"]).run()
    assert not [b for b in at.button if b.key == P + "delete_preset"]


def test_reopening_the_form_clears_messages_and_selectors(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert expanded(at)
    at.session_state["show_form"] = False
    at.run()
    at.session_state["show_form"] = True
    at.run()
    assert_no_exception(at)
    assert at.text_input(key=P + "url").value == ""
    assert not expanded(at)
    assert not any("Presets that work" in m.value for m in at.markdown)
    assert not texts(at, "success")
    assert field(at, "name_selector") == ""


def test_test_selectors_uses_current_values(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    at.button(key=P + "test_selectors").click().run()
    assert_no_exception(at)
    (args, _), = api.called("test_custom_manga_selectors")
    assert args[0] == FEED_URL
    assert args[1] == "xpath://item/title"
    assert any("The selectors work" in t for t in texts(at, "success"))


def add_args(api) -> tuple:
    (args, _), = api.called("add_multimanga")
    return args


def test_add_without_tracking_sends_no_selectors(api):
    at = run_app()
    at.text_input(key=P + "name").input("Manga").run()
    at.text_input(key=P + "url").input(SITE_URL).run()
    at.button(key=P + "add").click().run()
    args = add_args(api)
    assert args[0] == "Manga" and args[1] == SITE_URL
    assert args[9:16] == ("", "", "", False, "", "", False)
    assert args[16] is False


def test_add_with_tracking_sends_selectors(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "name").input("Manga").run()
    at.text_input(key=P + "url").input(FEED_URL).run()
    at.button(key=P + "add").click().run()
    args = add_args(api)
    assert args[9:13] == ("xpath://item/title", "", r"(\d+)", True)
    assert args[13:16] == ("xpath://item/link", "", True)


def test_clearing_selectors_after_detection_sends_no_selectors(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "name").input("Manga").run()
    at.text_input(key=P + "url").input(FEED_URL).run()
    at.button(key=P + "clear_selectors_button").click().run()
    assert at.selectbox(key=P + "preset").value is None
    at.button(key=P + "add").click().run()
    assert add_args(api)[9:16] == ("", "", "", False, "", "", False)


def test_closing_the_selectors_keeps_them(api):
    feed_detected(api)
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    at.text_input(key=P + "last_released_chapter_name_regex").input(r"c\.(\d+)").run()
    at.session_state[P + "last_released_chapter_expander"] = False
    at.run()
    assert_no_exception(at)
    assert not expanded(at)
    assert field(at, "name_selector") == "xpath://item/title"
    assert field(at, "name_regex") == r"c\.(\d+)"
    at.text_input(key=P + "name").input("Manga").run()
    at.button(key=P + "add").click().run()
    assert add_args(api)[9:12] == ("xpath://item/title", "", r"c\.(\d+)")


def test_detected_preset_that_no_longer_exists_is_ignored(api):
    ghost = dict(RSS_PRESET, id=99, name="Deleted meanwhile")
    api.detections[FEED_URL] = {"domain_presets": [], "detections": [detection(ghost, "1")]}
    at = run_app()
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert not expanded(at)
    assert field(at, "name_selector") == ""


def test_add_requires_name(api):
    at = run_app()
    at.button(key=P + "add").click().run()
    assert "Provide a manga name" in texts(at, "warning")
    assert api.called("add_multimanga") == []


def test_add_to_multimanga_form(api):
    feed_detected(api)
    at = run_app(multimanga_id=7)
    labels = labels_in_order(at)
    assert "Status" not in labels and "Last Read Chapter" not in labels
    at.text_input(key=MM + "name").input("Other source").run()
    at.text_input(key=MM + "url").input(FEED_URL).run()
    assert expanded(at, MM)
    at.button(key=MM + "add").click().run()
    assert_no_exception(at)
    (args, _), = api.called("add_manga_to_multimanga")
    assert args[0] == 7 and args[1] == "Other source"
    assert args[6] == "xpath://item/title"


def test_both_forms_in_one_session_dont_share_state(api):
    feed_detected(api)
    at = run_app(show_second_form=True)
    assert_no_exception(at)
    at.text_input(key=P + "url").input(FEED_URL).run()
    assert_no_exception(at)
    assert expanded(at, P)
    assert not expanded(at, MM)
    assert at.text_input(key=MM + "url").value == ""


def test_url_inputs_are_validated_in_the_browser(api):
    import re

    from src.util.util import URL_INPUT_VALIDATION

    at = run_app()
    for key in ("url", "cover_img_url", "last_read_chapter_url"):
        widget = at.text_input(key=P + key)
        assert widget.proto.type == widget.proto.URL, key
        assert widget.proto.validate_regex == URL_INPUT_VALIDATION[0], key
        assert widget.proto.validate_message == URL_INPUT_VALIDATION[1], key

    pattern = re.compile(URL_INPUT_VALIDATION[0])
    for valid in ("https://example.com", "http://a.b/c?d=1#e", "https://x.org/feed.xml"):
        assert pattern.match(valid), valid
    for invalid in ("example.com", "ftp://example.com", "https://", "https:// space.com", "javascript:alert(1)"):
        assert not pattern.match(invalid), invalid


def test_name_is_required(api):
    at = run_app()
    assert at.text_input(key=P + "name").proto.required
