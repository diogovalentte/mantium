import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.fake_api import user_error
from src.tests.src.util.test_custom_manga_form import assert_no_exception, popover_texts, texts

RUNNER = os.path.join(os.path.dirname(__file__), "update_multimanga_form_runner.py")
P = "update_multimanga_form_"
S = "_1"
COVER_CONFLICT = "To update the cover image, provide either an URL"


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def run_app() -> AppTest:
    return AppTest.from_file(RUNNER, default_timeout=30).run()


def test_chapters_fetched_once_while_the_dialog_is_open(api):
    at = run_app()
    at.selectbox(key=P + "status" + S).set_value(2).run()
    at.checkbox(key=P + "delete_last_read_chapter" + S).check().run()
    assert_no_exception(at)
    assert len(api.called("get_cached_manga_chapters")) == 1
    assert len(at.selectbox(key=P + "chapter_from_chapters_list" + S).options) == 2
    # Opening the dialog again fetches them again
    at.session_state["open_dialog"] = True
    at.run()
    assert len(api.called("get_cached_manga_chapters")) == 2


def test_reset_last_read_chapter_disables_the_chapter_fields(api):
    at = run_app()
    keys = ("chapter_from_chapters_list", "chapter_manual_chapter", "chapter_manual_url")
    assert not at.selectbox(key=P + keys[0] + S).disabled
    at.checkbox(key=P + "delete_last_read_chapter" + S).check().run()
    assert at.selectbox(key=P + keys[0] + S).disabled
    assert at.text_input(key=P + keys[1] + S).disabled
    assert at.text_input(key=P + keys[2] + S).disabled


def test_cover_conflict_is_shown_live_and_blocks_the_update(api):
    at = run_app()
    at.text_input(key=P + "cover_img_url" + S).input("https://example.com/c.png").run()
    assert not [t for t in texts(at, "warning") if COVER_CONFLICT in t]
    at.checkbox(key=P + "use_current_manga_cover_img" + S).check().run()
    assert [t for t in texts(at, "warning") if COVER_CONFLICT in t]
    at.button(key=P + "update" + S).click().run()
    assert_no_exception(at)
    assert api.called("update_multimanga_cover_img") == []
    assert api.called("update_multimanga_status") == []


def test_update_runs_in_place_and_closes_on_success(api):
    at = run_app()
    at.selectbox(key=P + "status" + S).set_value(2).run()
    at.button(key=P + "update" + S).click().run()
    assert_no_exception(at)
    assert api.called("update_multimanga_status") == [((2, 1), {})]
    assert at.session_state["update_manga_success_message"] == "Multimanga updated successfully"


def test_partial_update_error_names_the_failed_and_the_updated_parts(api):
    api.errors["update_multimanga_last_read_chapter"] = user_error("boom")
    at = run_app()
    at.selectbox(key=P + "status" + S).set_value(2).run()
    at.text_input(key=P + "chapter_manual_chapter" + S).input("12").run()
    at.button(key=P + "update" + S).click().run()
    assert_no_exception(at)
    (error,) = texts(at, "error")
    assert error.startswith("Error while updating the last read chapter")
    assert "Already updated before the error: status" in error
    # The form is still there to try again
    assert [b for b in at.button if b.key == P + "update" + S]
    assert "update_manga_success_message" not in at.session_state


def test_delete_asks_for_confirmation(api):
    at = run_app()
    assert "One Piece" in " ".join(popover_texts(at, "Delete", "warning"))
    assert api.called("delete_multimanga") == []
    at.button(key=P + "confirm_delete" + S).click().run()
    assert_no_exception(at)
    assert api.called("delete_multimanga") == [((1,), {})]
    assert at.session_state["update_manga_success_message"] == "Multimanga deleted successfully"


def test_delete_error_stays_in_the_confirmation(api):
    api.errors["delete_multimanga"] = user_error("boom")
    at = run_app()
    at.button(key=P + "confirm_delete" + S).click().run()
    assert_no_exception(at)
    assert any("Error while deleting multimanga" in t for t in popover_texts(at, "Delete", "error"))


def test_custom_only_multimanga_doesnt_show_the_chapters_of_the_previous_one(api):
    at = run_app()
    assert len(at.selectbox(key=P + "chapter_from_chapters_list" + S).options) == 2
    custom = {**api.multimanga["CurrentManga"], "ID": 9, "Source": "custom_manga"}
    api.multimanga.update(CurrentManga=custom, Mangas=[custom])
    at.session_state["open_dialog"] = True
    at.run()
    assert_no_exception(at)
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).options == []
