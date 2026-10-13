import copy
import os

import pytest
from streamlit.testing.v1 import AppTest

import src.tests.src.util.fake_api as fake_api
from src.tests.src.util.test_custom_manga_form import assert_no_exception

RUNNER = os.path.join(os.path.dirname(__file__), "dashboard_runner.py")


@pytest.fixture
def api():
    fake_api.FAKE = fake_api.FakeAPI()
    return fake_api.FAKE


def run_page(**session_state) -> AppTest:
    at = AppTest.from_file(RUNNER, default_timeout=60)
    for key, value in session_state.items():
        at.session_state[key] = value
    return at.run()


def card_names(at: AppTest) -> list[str]:
    return [m.value for m in at.markdown if 'class="manga_header"' in m.value]


def test_page_renders_the_library(api):
    at = run_page(status_filter=0)
    assert_no_exception(at)
    assert not at.error, [e.value for e in at.error]
    assert len(card_names(at)) == len(api.library)


@pytest.mark.parametrize("display_mode", ["Grid View", "List View"])
def test_covers_link_to_the_manga_page(api, display_mode):
    api.dashboard_configs["display"]["displayMode"] = display_mode
    at = run_page(status_filter=0)
    assert_no_exception(at)
    links = {image.proto.imgs[0].alt: image.proto.link for image in at.image}
    assert len(links) == len(api.library)
    for manga in api.library:
        expected = manga["URL"] if manga["URL"].startswith("http") else ""
        assert links[manga["Name"]] == expected, manga["Name"]


def by_name(api, name):
    return next(m for m in api.library if m["Name"] == name)


@pytest.mark.parametrize("display_mode", ["Grid View", "List View"])
def test_card_actions(api, display_mode):
    api.dashboard_configs["display"]["displayMode"] = display_mode
    at = run_page(status_filter=0)
    assert_no_exception(at)
    unread, read = by_name(api, "Dandadan"), by_name(api, "Chainsaw Man")
    assert at.menu_button(key=f"manga_actions_{unread['ID']}").options == ["Set last read", "Highlight"]
    assert at.menu_button(key=f"manga_actions_{read['ID']}").options == ["Highlight"]

    at.menu_button(key=f"manga_actions_{unread['ID']}").click("Set last read").run()
    assert_no_exception(at)
    assert api.called("update_multimanga_last_read_chapter") == [((unread["MultiMangaID"], unread["ID"]), {})]

    at.menu_button(key=f"manga_actions_{read['ID']}").click("Highlight").run()
    assert_no_exception(at)
    assert at.session_state["highlighted_manga"]["ID"] == read["ID"]


def test_custom_manga_without_released_chapter_has_no_set_last_read(api):
    at = run_page(status_filter=0)
    manual = by_name(api, "Manual with text chapter")
    assert at.menu_button(key=f"manga_actions_{manual['ID']}").options == ["Highlight"]


def test_chapter_tags_are_markdown(api):
    at = run_page(status_filter=0)
    assert_no_exception(at)
    tags = [m.value for m in at.markdown if m.value.startswith(":color[")]
    ao_ashi = by_name(api, "Ao Ashi (URL only, chapter ?)")
    url = ao_ashi["LastReleasedChapter"]["URL"]
    assert f":color[[Ch\\. ?]({url})]" + '{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags
    # A chapter without URL has no link
    assert ':color[Ch\\. Prologue]{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags
    assert not [m for m in at.markdown if "display:inline-block" in m.value]


def test_tag_markdown_escapes_text_and_checks_links():
    from src.util.util import escape_markdown, tag_markdown

    assert escape_markdown(":red[x] [a](b) *b* $x$") == "\\:red\\[x\\] \\[a\\]\\(b\\) \\*b\\* \\$x\\$"
    assert tag_markdown("Ch. 3", "#000", "#fff", "https://a.com/x (1)") == (
        ':color[[Ch\\. 3](https://a.com/x%20%281%29)]{foreground="#fff" background="#000"}'
    )
    assert tag_markdown("x", "#000", "#fff", "javascript:alert(1)") == ':color[x]{foreground="#fff" background="#000"}'


def test_dialog_cards_use_markdown_tags(api):
    at = AppTest.from_file(os.path.join(os.path.dirname(__file__), "cards_runner.py"), default_timeout=30)
    at.session_state["search_result"] = {
        "Name": "Search result",
        "URL": "https://example.com/m",
        "CoverURL": "",
        "Status": "ongoing",
        "Year": 2020,
        "LastChapter": "12",
        "LastChapterURL": "https://example.com/m/12",
        "Description": "",
    }
    at.run()
    assert_no_exception(at)
    tags = [m.value for m in at.markdown if m.value.startswith(":color[")]
    assert ':color[Custom Manga]{foreground="black" background="white"}' in tags
    assert ':color[Current Manga]{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags
    assert ':color[Ongoing]{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags
    assert ':color[2020]{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags
    assert ':color[[12](https://example.com/m/12)]{foreground="rgb(59, 130, 246)" background="rgb(219, 233, 254)"}' in tags


def test_dialog_card_covers_link_to_the_manga(api):
    at = AppTest.from_file(os.path.join(os.path.dirname(__file__), "cards_runner.py"), default_timeout=30)
    at.session_state["search_result"] = {
        "Name": "Search result",
        "URL": "https://example.com/m",
        "CoverURL": "https://example.com/cover.jpg",
        "Status": "",
        "Year": "",
        "LastChapter": "",
        "LastChapterURL": "",
        "Description": "",
    }
    at.run()
    assert_no_exception(at)
    covers = {image.proto.imgs[0].alt: image.proto for image in at.image}
    assert covers["Search result"].link == "https://example.com/m"
    assert covers["Search result"].imgs[0].url == "https://example.com/cover.jpg"
    first = api.get_mangas()[0]
    assert covers[first["Name"]].link == first["URL"]


def pagination(at: AppTest):
    widgets = [w for w in at.get("pagination")]
    return widgets[0] if widgets else None


def go_to_page(at: AppTest, page: int, warnings: list[str]):
    """AppTest can't click st.pagination, so the page is written to its session state, which
    Streamlit warns about because the widget has a default. That warning is expected here."""
    key = pagination(at).key
    at.session_state[key] = page
    at.run()
    warnings[:] = [w for w in warnings if key not in w]


def make_library(api, size: int):
    template = api.library[1]
    api.library = []
    for i in range(size):
        manga = dict(copy.deepcopy(template), ID=1000 + i, MultiMangaID=2000 + i, Name=f"Manga {i:02d}", SearchNames=[f"Manga {i:02d}"])
        api.library.append(manga)


def test_grid_is_paged(api, widget_state_duplication_warnings):
    api.dashboard_configs["display"]["columns"] = 2
    make_library(api, 15)  # 2 columns x 3 rows = 6 per page -> 3 pages
    at = run_page(status_filter=0, mangas_sort="Name")
    assert_no_exception(at)
    assert len(card_names(at)) == 6
    page = pagination(at)
    assert page is not None and page.proto.num_pages == 3
    go_to_page(at, 3, widget_state_duplication_warnings)
    assert_no_exception(at)
    assert len(card_names(at)) == 3


def test_filters_and_sort_go_back_to_the_first_page(api):
    # AppTest doesn't keep a pagination value between runs, so this checks the mechanism:
    # going back to the first page gives the widget a new key, which starts at page 1
    api.dashboard_configs["display"]["columns"] = 2
    make_library(api, 15)
    at = run_page(status_filter=0)
    keys = [pagination(at).key]
    at.text_input(key="search_manga").input("Manga").run()
    keys.append(pagination(at).key)
    at.toggle(key="mangas_sort_reverse").set_value(True).run()
    keys.append(pagination(at).key)
    at.selectbox(key="mangas_sort").set_value("Name").run()
    keys.append(pagination(at).key)
    assert_no_exception(at)
    assert len(set(keys)) == 4, keys
    assert pagination(at).value == 1


def test_page_past_the_end_after_the_library_shrinks(api, widget_state_duplication_warnings):
    api.dashboard_configs["display"]["columns"] = 2
    make_library(api, 15)
    at = run_page(status_filter=0)
    go_to_page(at, 3, widget_state_duplication_warnings)
    api.library = api.library[:8]  # 2 pages now
    at.run()
    assert_no_exception(at)
    # Streamlit moves a page past the end back to the first page; what matters is that the
    # cards shown are the ones of the page the widget shows
    page = pagination(at).value
    assert page in (1, 2)
    assert len(card_names(at)) == (6 if page == 1 else 2)


def test_list_view_is_paged(api):
    api.dashboard_configs["display"]["displayMode"] = "List View"
    make_library(api, 25)
    at = run_page(status_filter=0)
    assert_no_exception(at)
    assert len(card_names(at)) == 20
    assert pagination(at).proto.num_pages == 2


def test_single_page_has_no_pagination(api):
    at = run_page(status_filter=0)
    assert pagination(at) is None


def test_skeleton_while_the_library_loads(api, monkeypatch):
    import streamlit as st

    shown = []
    original = st.skeleton

    def skeleton(*args, **kwargs):
        shown.append(kwargs.get("height"))
        return original(*args, **kwargs)

    monkeypatch.setattr(st, "skeleton", skeleton)
    at = run_page(status_filter=0)
    assert_no_exception(at)
    assert shown == [500]
    # The skeleton is gone once the library is loaded
    assert len(card_names(at)) == len(api.library)


def test_edit_dialogs_open_on_the_right(api, monkeypatch):
    import streamlit as st

    positions = {}
    original = st.dialog

    def dialog(title, *args, **kwargs):
        positions[title] = kwargs.get("position", "center")
        return original(title, *args, **kwargs)

    monkeypatch.setattr(st, "dialog", dialog)
    manga = by_name(api, "Dandadan")
    custom = by_name(api, "Manual with text chapter")
    at = run_page(status_filter=0)
    at.menu_button(key=f"manga_actions_{manga['ID']}").click("Highlight").run()
    assert_no_exception(at)
    assert positions[manga["Name"]] == "right"
    at = run_page(status_filter=0, highlighted_custom_manga=api.get_mangas()[[m["ID"] for m in api.library].index(custom["ID"])])
    assert_no_exception(at)
    assert positions["Update Custom Manga"] == "right"
