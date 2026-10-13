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
