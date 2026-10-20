import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.fake_api import user_error
from src.tests.src.util.test_custom_manga_form import assert_no_exception, popover_texts, texts
from src.util.update_manga import chapters_list_warning

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


def chapter_warnings(at: AppTest) -> list[str]:
    """The warnings of the form, without the one of the Delete confirmation."""
    return [t for t in texts(at, "warning") if not t.startswith("Delete ")]


def custom_manga(name_selector: str = "css:a.chapter") -> dict:
    return {
        "ID": 9,
        "Name": "My custom",
        "Source": "custom_manga",
        "URL": "https://example.com/my-custom",
        "InternalID": "",
        "LastReleasedChapterNameSelector": {"Selector": name_selector, "Attribute": "", "Regex": "", "GetFirst": True} if name_selector else None,
        "LastReleasedChapterURLSelector": None,
    }


def test_custom_current_manga_lists_its_chapters(api):
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom, fake_api._MANGA])
    at = run_app()
    assert_no_exception(at)
    options = at.selectbox(key=P + "chapter_from_chapters_list" + S).options
    assert [o.split(" ")[1] for o in options] == ["3", "2", "1"]
    assert api.called("get_manga_chapters") == [((9, "https://example.com/my-custom", ""), {})]
    assert api.called("get_cached_manga_chapters") == []
    assert chapter_warnings(at) == []

    at.selectbox(key=P + "chapter_from_chapters_list" + S).set_value(fake_api.CUSTOM_CHAPTERS[1]).run()
    at.button(key=P + "update" + S).click().run()
    assert_no_exception(at)
    assert api.called("update_multimanga_last_read_chapter") == [((1, 9, "2", "https://example.com/c/2", "", False, True), {})]


def test_custom_last_read_chapter_from_the_list_is_selected(api):
    custom = custom_manga()
    api.multimanga.update(
        CurrentManga=custom,
        Mangas=[custom],
        LastReadChapter={"Chapter": "2", "URL": "https://example.com/c/2", "FromSourceSite": True},
    )
    at = run_app()
    assert_no_exception(at)
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).value["Chapter"] == "2"
    assert at.text_input(key=P + "chapter_manual_chapter" + S).value == ""


def test_custom_chapters_error_falls_back_to_another_manga(api):
    api.errors["get_manga_chapters"] = user_error("Chapter name selector didn't match any element on the page")
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom, fake_api._MANGA])
    at = run_app()
    assert_no_exception(at)
    assert len(at.selectbox(key=P + "chapter_from_chapters_list" + S).options) == 2
    (warning,) = chapter_warnings(at)
    assert "Chapter name selector didn't match any element on the page" in warning
    assert "Chapters list fetched from the manga of source" in warning
    # The chapters came from the other manga, so a selected one is saved from it
    assert at.session_state["update_multimanga_chapter_options_from_manga_id"] == 5


def test_custom_only_multimanga_chapters_error_is_shown(api):
    api.errors["get_manga_chapters"] = user_error("Chapter name selector didn't match any element on the page")
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom])
    at = run_app()
    assert_no_exception(at)
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).options == []
    (warning,) = chapter_warnings(at)
    assert warning == "Could not get the chapters list from the current custom manga: Chapter name selector didn't match any element on the page."


def test_custom_manga_without_name_selector_doesnt_list_chapters(api):
    custom = custom_manga(name_selector="")
    api.multimanga.update(CurrentManga=custom, Mangas=[custom])
    at = run_app()
    assert_no_exception(at)
    assert api.called("get_manga_chapters") == []
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).options == []
    assert chapter_warnings(at) == []


def other_manga(manga_id: int) -> dict:
    return {**fake_api._MANGA, "ID": manga_id, "URL": f"https://mangadex.org/title/{manga_id}"}


def test_custom_chapters_error_skips_a_manga_without_chapters(api):
    api.errors["get_manga_chapters"] = user_error("Chapter name selector didn't match any element on the page")
    api.chapters_by_manga_id = {5: []}
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom, fake_api._MANGA, other_manga(6)])
    at = run_app()
    assert_no_exception(at)
    assert len(at.selectbox(key=P + "chapter_from_chapters_list" + S).options) == 2
    assert at.session_state["update_multimanga_chapter_options_from_manga_id"] == 6
    (warning,) = chapter_warnings(at)
    assert warning.startswith("Could not get the chapters list from the current custom manga: Chapter name selector didn't match any element on the page.")
    assert "Chapters list fetched from the manga of source" in warning


def test_custom_chapters_error_and_no_other_manga_has_chapters(api):
    api.errors["get_manga_chapters"] = user_error("Chapter name selector didn't match any element on the page")
    api.chapters_by_manga_id = {5: [], 6: user_error("down")}
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom, fake_api._MANGA, other_manga(6)])
    at = run_app()
    assert_no_exception(at)
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).options == []
    assert chapter_warnings(at) == [
        "Could not get the chapters list from the current custom manga: Chapter name selector didn't match any element on the page."
        " The other mangas failed or have no released chapters either."
    ]


def test_current_manga_without_chapters_uses_another_manga(api):
    api.chapters_by_manga_id = {5: []}
    api.multimanga.update(Mangas=[fake_api._MANGA, other_manga(6)])
    at = run_app()
    assert_no_exception(at)
    assert len(at.selectbox(key=P + "chapter_from_chapters_list" + S).options) == 2
    (warning,) = chapter_warnings(at)
    assert warning.startswith("The current manga has no released chapters. Chapters list fetched from the manga of source")


def test_only_manga_without_chapters(api):
    api.chapters_by_manga_id = {5: []}
    at = run_app()
    assert_no_exception(at)
    assert at.selectbox(key=P + "chapter_from_chapters_list" + S).options == []
    assert chapter_warnings(at) == ["No released chapters found."]


def attempt(manga_id: int, result: str, custom: bool = False, error: str | None = None) -> dict:
    return {"manga_id": manga_id, "custom": custom, "source_name": f"Source {manga_id}", "result": result, "error": error}


@pytest.mark.parametrize(
    "attempts, from_manga_id, want",
    [
        ([], None, None),
        ([attempt(1, "ok")], 1, None),
        # The current manga can't list chapters (custom without name selector)
        ([attempt(2, "ok")], 2, None),
        ([attempt(2, "empty")], None, "No released chapters found."),
        ([attempt(2, "error")], None, "Could not get the chapters list from the multimanga mangas. Maybe the source sites are down or the manga URLs changed in the source sites."),
        ([attempt(2, "error"), attempt(3, "empty")], None, "Could not get the chapters list from the multimanga mangas: they failed or have no released chapters."),
        # Only the current manga
        ([attempt(1, "empty")], None, "No released chapters found."),
        ([attempt(1, "error")], None, "Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site."),
        ([attempt(1, "error", True, "Bad selector")], None, "Could not get the chapters list from the current custom manga: Bad selector."),
        # The current manga and others
        ([attempt(1, "error"), attempt(2, "ok")], 2, "Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site. Chapters list fetched from the manga of source Source 2."),
        ([attempt(1, "empty"), attempt(2, "empty"), attempt(3, "ok")], 3, "The current manga has no released chapters. Chapters list fetched from the manga of source Source 3."),
        ([attempt(1, "empty"), attempt(2, "empty")], None, "The current manga has no released chapters. The other mangas have no released chapters either."),
        ([attempt(1, "error"), attempt(2, "error")], None, "Could not get the chapters list from the current manga. Maybe the source site is down or the manga URL changed in the source site. Could not get the chapters list from the other mangas either. Maybe their source sites are down or the manga URLs changed in the source sites."),
        ([attempt(1, "error", True, "Bad selector."), attempt(2, "empty")], None, "Could not get the chapters list from the current custom manga: Bad selector. The other mangas have no released chapters either."),
    ],
)
def test_chapters_list_warning(attempts, from_manga_id, want):
    assert chapters_list_warning(1, attempts, from_manga_id) == want


def test_failing_to_choose_the_manga_to_try_is_warned(api):
    api.errors["get_manga_chapters"] = user_error("Bad selector")
    api.errors["choose_current_manga"] = user_error("down")
    custom = custom_manga()
    api.multimanga.update(CurrentManga=custom, Mangas=[custom, fake_api._MANGA])
    at = run_app()
    assert_no_exception(at)
    assert chapter_warnings(at) == [
        "Could not get the chapters list from the current custom manga: Bad selector."
        " Could not get the chapters list from the other mangas either. Maybe their source sites are down or the manga URLs changed in the source sites."
    ]

    del api.errors["get_manga_chapters"]
    api.multimanga.update(CurrentManga=fake_api._MANGA, Mangas=[fake_api._MANGA])
    at.session_state["open_dialog"] = True
    at.run()
    assert_no_exception(at)
    assert chapter_warnings(at) == [
        "Could not get the chapters list from the multimanga mangas. Maybe the source sites are down or the manga URLs changed in the source sites."
    ]
