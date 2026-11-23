import pytest

from src.util import defaults
from src.util.util import is_unread_chapter


def chapter(name: str, url: str = "") -> dict:
    return {"Chapter": name, "URL": url}


@pytest.mark.parametrize(
    ("last_read", "last_released", "want"),
    [
        (chapter(""), chapter(""), False),
        (chapter("10"), chapter(""), False),
        (chapter(""), chapter("10"), True),
        (chapter("9"), chapter("10"), True),
        (chapter("10"), chapter("10"), False),
        (chapter("11"), chapter("10"), False),
        (chapter("Extra"), chapter("10"), True),
        (chapter(defaults.UNKNOWN_CHAPTER, "https://a/1"), chapter(defaults.UNKNOWN_CHAPTER, "https://a/2"), True),
        (chapter(defaults.UNKNOWN_CHAPTER, "https://a/1"), chapter(defaults.UNKNOWN_CHAPTER, "https://a/1"), False),
        # Same name, different chapter (e.g. a chapter re-released with the same number)
        (chapter("10", "https://a/10"), chapter("10", "https://a/10-v2"), True),
        (chapter("10", "https://a/10"), chapter("10", "https://a/10"), False),
        # Without both URLs there is nothing to compare
        (chapter("10"), chapter("10", "https://a/10"), False),
        (chapter("10", "https://a/10"), chapter("10"), False),
        # A multimanga read on one source and released on another: the URLs differ but it's the same chapter
        (chapter("10", "https://mangadex.org/chapter/x"), chapter("10", "https://mangaplus.shueisha.co.jp/viewer/y"), False),
    ],
)
def test_is_unread_chapter(last_read, last_released, want):
    """Same cases as TestIsUnreadChapter in the API (manga.IsUnreadChapter), keep them in sync."""
    assert is_unread_chapter(last_read, last_released) is want
