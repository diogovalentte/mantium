import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.fake_api import user_error
from src.tests.src.util.test_custom_manga_form import assert_no_exception, popover_texts, texts

RUNNER = os.path.join(os.path.dirname(__file__), "manage_mangas_runner.py")
IGNORE = "Ignore Last Chapter"
STOP_IGNORING = "Stop Ignoring Chapter"


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def mangas(api, count: int = 2) -> list[dict]:
    library = api.get_mangas()
    result = []
    for i in range(count):
        manga = dict(library[i])
        manga["ID"] = 10 + i
        manga["LastReleasedChapter"] = {**manga["LastReleasedChapter"], "Chapter": str(71 + i), "URL": f"https://s{i}/ch"}
        result.append(manga)
    return result


def run_app(mangas: list[dict]) -> AppTest:
    at = AppTest.from_file(RUNNER, default_timeout=30)
    at.session_state["mangas"] = mangas
    return at.run()


def popover_labels(at: AppTest) -> list[str]:
    labels = []

    def walk(node):
        if getattr(node, "type", None) == "popover":
            labels.append(node.proto.popover.label)
        for child in getattr(node, "children", {}).values():
            walk(child)

    walk(at.main)
    return labels


def test_each_manga_can_ignore_its_last_chapter(api):
    at = run_app(mangas(api))
    assert_no_exception(at)
    assert popover_labels(at).count(IGNORE) == 2
    assert any("Ignore chapter 71 from" in t for t in popover_texts(at, IGNORE, "warning"))
    at.button(key="update_multimanga_mangas_ignore_chapter_11").click().run()
    assert_no_exception(at)
    assert api.called("ignore_multimanga_manga_chapter") == [((1, 11), {})]
    assert api.called("get_multimanga") == [((1,), {})]
    assert at.session_state["update_manga_success_message"] == "Chapter ignored successfully"


def test_only_manga_cant_ignore_its_last_chapter(api):
    at = run_app(mangas(api, 1))
    assert_no_exception(at)
    assert IGNORE not in popover_labels(at)


def test_ignored_chapter_is_tagged_and_can_stop_being_ignored(api):
    ms = mangas(api)
    ms[1]["IgnoredChapter"] = {"Chapter": "72", "URL": "https://s1/ch"}
    at = run_app(ms)
    assert_no_exception(at)
    assert any("Ch. 72 ignored" in m.value for m in at.markdown)
    assert popover_labels(at).count(IGNORE) == 1
    assert STOP_IGNORING in popover_labels(at)
    at.button(key="update_multimanga_mangas_stop_ignoring_chapter_11").click().run()
    assert_no_exception(at)
    assert api.called("stop_ignoring_multimanga_manga_chapter") == [((1, 11), {})]
    assert at.session_state["update_manga_success_message"] == "Chapter is no longer ignored"


def test_ignore_user_error_is_shown(api):
    message = "Can't ignore this chapter: no other manga in the multimanga has a last released chapter that isn't ignored"
    api.errors["ignore_multimanga_manga_chapter"] = user_error(message)
    at = run_app(mangas(api))
    at.button(key="update_multimanga_mangas_ignore_chapter_10").click().run()
    assert_no_exception(at)
    assert message in texts(at, "warning")
    assert "update_manga_success_message" not in at.session_state
